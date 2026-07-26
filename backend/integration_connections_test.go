package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func configureConnectionTest(t *testing.T) {
	t.Helper()
	key := bytes.Repeat([]byte{9}, 32)
	t.Setenv(
		"MANAGER_DASHBOARD_ENCRYPTION_KEY",
		base64.StdEncoding.EncodeToString(key),
	)
	originalClient := integrationHTTPClient
	originalInsecure := integrationAllowInsecureHTTP
	originalNow := integrationTestNow
	integrationAllowInsecureHTTP = true
	integrationTestNow = func() time.Time {
		return time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		integrationHTTPClient = originalClient
		integrationAllowInsecureHTTP = originalInsecure
		integrationTestNow = originalNow
	})
}

func storeConnectionCredential(
	t *testing.T,
	provider, account, baseURL, secret string,
	enabled bool,
) {
	t.Helper()
	encrypted, err := encryptSecret(secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO integration_credentials
			(provider, account_label, base_url, encrypted_secret, enabled)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(provider) DO UPDATE SET
			account_label = excluded.account_label,
			base_url = excluded.base_url,
			encrypted_secret = excluded.encrypted_secret,
			enabled = excluded.enabled`,
		provider, account, baseURL, encrypted, enabled,
	); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationConnectionTestsSupportedProviders(t *testing.T) {
	setupTestDatabase(t)
	configureConnectionTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			if r.Method != http.MethodGet ||
				r.Header.Get("Authorization") != "Bearer github-token" ||
				r.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
				t.Errorf("unexpected GitHub request: %s %#v", r.Method, r.Header)
			}
			w.Write([]byte(`{"login":"octocat"}`))
		case "/rest/api/3/myself":
			expected := "Basic " + base64.StdEncoding.EncodeToString(
				[]byte("manager@example.com:jira-token"),
			)
			if r.Method != http.MethodGet || r.Header.Get("Authorization") != expected {
				t.Errorf("unexpected Jira request: %s %#v", r.Method, r.Header)
			}
			w.Write([]byte(`{"displayName":"Manager","emailAddress":"manager@example.com"}`))
		case "/api/auth.test":
			if r.Method != http.MethodPost ||
				r.Header.Get("Authorization") != "Bearer slack-token" {
				t.Errorf("unexpected Slack request: %s %#v", r.Method, r.Header)
			}
			w.Write([]byte(`{"ok":true,"team":"Engineering","user":"manager"}`))
		case "/me":
			if r.Method != http.MethodGet ||
				r.Header.Get("Authorization") != "Bearer teams-token" {
				t.Errorf("unexpected Teams request: %s %#v", r.Method, r.Header)
			}
			w.Write([]byte(`{"displayName":"Manager","userPrincipalName":"manager@example.com"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	integrationHTTPClient = server.Client()

	for _, credential := range []struct {
		provider, account, secret, identity string
	}{
		{"github", "Work", "github-token", "octocat"},
		{"jira", "manager@example.com", "jira-token", "Manager"},
		{"slack", "Engineering", "slack-token", "Engineering"},
		{"teams", "Tenant", "teams-token", "Manager"},
	} {
		storeConnectionCredential(
			t, credential.provider, credential.account, server.URL,
			credential.secret, true,
		)
		response := request(
			t, newRouter(), http.MethodPost,
			"/api/integrations/"+credential.provider+"/test", nil,
		)
		if response.Code != http.StatusOK ||
			!strings.Contains(response.Body.String(), `"success":true`) ||
			!strings.Contains(response.Body.String(), credential.identity) ||
			!strings.Contains(response.Body.String(), `"testedAt":"2026-08-10T12:00:00Z"`) {
			t.Fatalf("%s test = %d %s", credential.provider, response.Code, response.Body.String())
		}
	}
}

func TestIntegrationConnectionNormalizesProviderFailures(t *testing.T) {
	setupTestDatabase(t)
	configureConnectionTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Authorization") {
		case "Bearer unauthorized":
			w.WriteHeader(http.StatusUnauthorized)
		case "Bearer forbidden":
			w.WriteHeader(http.StatusForbidden)
		case "Bearer limited":
			w.WriteHeader(http.StatusTooManyRequests)
		case "Bearer invalid-json":
			w.Write([]byte("not json"))
		case "Bearer slack-rejected":
			w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
		default:
			w.WriteHeader(http.StatusBadGateway)
		}
	}))
	defer server.Close()
	integrationHTTPClient = server.Client()

	tests := []struct {
		provider, secret, category string
	}{
		{"github", "unauthorized", "authentication"},
		{"github", "forbidden", "authorization"},
		{"github", "limited", "rate_limit"},
		{"github", "invalid-json", "invalid_response"},
		{"slack", "slack-rejected", "authentication"},
		{"teams", "provider-error", "provider"},
	}
	for _, test := range tests {
		storeConnectionCredential(t, test.provider, "Account", server.URL, test.secret, true)
		response := request(
			t, newRouter(), http.MethodPost,
			"/api/integrations/"+test.provider+"/test", nil,
		)
		if response.Code != 200 ||
			!strings.Contains(response.Body.String(), `"success":false`) ||
			!strings.Contains(response.Body.String(), `"category":"`+test.category+`"`) {
			t.Fatalf("%s/%s = %d %s", test.provider, test.secret, response.Code, response.Body.String())
		}
	}
}

func TestIntegrationConnectionValidatesConfigurationAndStoredState(t *testing.T) {
	setupTestDatabase(t)
	configureConnectionTest(t)
	router := newRouter()
	for _, test := range []struct {
		provider, account, baseURL string
		code                       int
		contains                   string
	}{
		{"dropbox", "", "", 400, "not supported"},
		{"github", "", "", 404, "not configured"},
	} {
		response := request(t, router, http.MethodPost, "/api/integrations/"+test.provider+"/test", nil)
		if response.Code != test.code ||
			!strings.Contains(strings.ToLower(response.Body.String()), test.contains) {
			t.Fatalf("%s = %d %s", test.provider, response.Code, response.Body.String())
		}
	}

	storeConnectionCredential(t, "github", "", "", "token", false)
	if got := request(t, router, http.MethodPost, "/api/integrations/github/test", nil); got.Code != http.StatusConflict {
		t.Fatalf("disabled = %d %s", got.Code, got.Body.String())
	}
	storeConnectionCredential(t, "jira", "", "", "token", true)
	if got := request(t, router, http.MethodPost, "/api/integrations/jira/test", nil); got.Code != 200 ||
		!strings.Contains(got.Body.String(), `"category":"configuration"`) {
		t.Fatalf("jira config = %d %s", got.Code, got.Body.String())
	}
	storeConnectionCredential(t, "github", "", "http://example.com", "token", true)
	integrationAllowInsecureHTTP = false
	if got := request(t, router, http.MethodPost, "/api/integrations/github/test", nil); got.Code != 200 ||
		!strings.Contains(got.Body.String(), `"category":"configuration"`) {
		t.Fatalf("insecure URL = %d %s", got.Code, got.Body.String())
	}
	integrationAllowInsecureHTTP = true

	if _, err := db.Exec(`
		UPDATE integration_credentials SET encrypted_secret = 'invalid'
		WHERE provider = 'github'`); err != nil {
		t.Fatal(err)
	}
	if got := request(t, router, http.MethodPost, "/api/integrations/github/test", nil); got.Code != 500 {
		t.Fatalf("decrypt failure = %d %s", got.Code, got.Body.String())
	}
}

func TestIntegrationConnectionNormalizesNetworkAndTimeoutFailures(t *testing.T) {
	setupTestDatabase(t)
	configureConnectionTest(t)
	storeConnectionCredential(t, "github", "", "https://api.example.com", "token", true)
	router := newRouter()

	integrationHTTPClient = &http.Client{Transport: roundTripFunc(
		func(*http.Request) (*http.Response, error) {
			return nil, errors.New("offline")
		},
	)}
	if got := request(t, router, http.MethodPost, "/api/integrations/github/test", nil); got.Code != 200 ||
		!strings.Contains(got.Body.String(), `"category":"network"`) {
		t.Fatalf("network = %d %s", got.Code, got.Body.String())
	}
	integrationHTTPClient = &http.Client{Transport: roundTripFunc(
		func(*http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		},
	)}
	if got := request(t, router, http.MethodPost, "/api/integrations/github/test", nil); got.Code != 200 ||
		!strings.Contains(got.Body.String(), `"category":"timeout"`) {
		t.Fatalf("timeout = %d %s", got.Code, got.Body.String())
	}
}

func TestIntegrationConnectionHandlesDatabaseFailure(t *testing.T) {
	setupTestDatabase(t)
	configureConnectionTest(t)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	got := request(t, newRouter(), http.MethodPost, "/api/integrations/github/test", nil)
	if got.Code != http.StatusInternalServerError {
		t.Fatalf("database failure = %d %s", got.Code, got.Body.String())
	}
}
