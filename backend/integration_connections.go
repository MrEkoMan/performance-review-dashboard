package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var integrationHTTPClient = &http.Client{Timeout: 8 * time.Second}
var integrationAllowInsecureHTTP = false
var integrationTestNow = time.Now

type storedIntegration struct {
	Provider, AccountLabel, BaseURL, EncryptedSecret string
	DeploymentType                                   string
	Enabled                                          bool
}

func testIntegrationConnection(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if provider != "github" && provider != "gitlab" && provider != "jira" &&
		provider != "slack" && provider != "teams" {
		http.Error(w, "Connection testing is not supported for this provider", http.StatusBadRequest)
		return
	}
	var stored storedIntegration
	err := db.QueryRow(`
		SELECT provider, COALESCE(account_label, ''), COALESCE(base_url, ''),
			encrypted_secret, COALESCE(deployment_type, ''), enabled
		FROM integration_credentials WHERE provider = ?`, provider,
	).Scan(
		&stored.Provider, &stored.AccountLabel, &stored.BaseURL,
		&stored.EncryptedSecret, &stored.DeploymentType, &stored.Enabled,
	)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Integration is not configured", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve integration", http.StatusInternalServerError)
		return
	}
	if !stored.Enabled {
		http.Error(w, "Integration is disabled", http.StatusConflict)
		return
	}
	secret, err := decryptSecret(stored.EncryptedSecret)
	if err != nil {
		http.Error(w, "Stored credential could not be decrypted", http.StatusInternalServerError)
		return
	}
	request, err := buildIntegrationTestRequest(r.Context(), stored, secret)
	if err != nil {
		writeJSON(w, http.StatusOK, connectionFailure(provider, "configuration", err.Error(), 0))
		return
	}
	response, err := integrationHTTPClient.Do(request)
	if err != nil {
		category := "network"
		message := "The provider could not be reached"
		if errors.Is(err, context.DeadlineExceeded) {
			category = "timeout"
			message = "The provider did not respond before the timeout"
		} else {
			var networkError net.Error
			if errors.As(err, &networkError) && networkError.Timeout() {
				category = "timeout"
				message = "The provider did not respond before the timeout"
			}
		}
		writeJSON(w, http.StatusOK, connectionFailure(provider, category, message, 0))
		return
	}
	defer response.Body.Close()
	result := evaluateIntegrationResponse(provider, response)
	writeJSON(w, http.StatusOK, result)
}

// detectJiraDeployment distinguishes Jira Cloud from an internally hosted
// Jira Server / Data Center instance. The serverInfo endpoint exists on
// Server / Data Center only. Cloud instances respond with 404; Server / Data
// Center respond with 200 (deploymentType in the body) or 401/403 when the
// endpoint itself requires authentication — which still identifies the
// deployment as self-hosted.
func detectJiraDeployment(ctx context.Context, baseURL string) (server bool, err error) {
	endpoint, err := integrationEndpoint(baseURL, "/rest/api/latest/serverInfo")
	if err != nil {
		return false, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, errors.New("Provider URL is invalid")
	}
	request.Header.Set("Accept", "application/json")
	response, err := integrationHTTPClient.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
		return false, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// 401/403 and other non-success statuses: the serverInfo endpoint
		// exists but is locked down, which only self-hosted instances do.
		return true, nil
	}
	var payload struct {
		DeploymentType string `json:"deploymentType"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return false, nil
	}
	// Server / Data Center report "Server"; Cloud never reaches this path.
	return strings.EqualFold(payload.DeploymentType, "Server"), nil
}

func buildIntegrationTestRequest(
	context context.Context,
	stored storedIntegration,
	secret string,
) (*http.Request, error) {
	baseURL := strings.TrimSpace(stored.BaseURL)
	method := http.MethodGet
	path := ""
	switch stored.Provider {
	case "github":
		if baseURL == "" {
			baseURL = "https://api.github.com"
		}
		path = "/user"
	case "gitlab":
		if baseURL == "" {
			baseURL = "https://gitlab.com"
		}
		path = "/api/v4/user"
	case "jira":
		if baseURL == "" {
			return nil, errors.New("Jira base URL is required")
		}
		// Internally hosted Jira (Server / Data Center) uses REST API v2 and
		// PAT bearer auth; Jira Cloud uses REST API v3 and Basic auth with
		// the account email. An explicit deployment type skips the probe.
		isServer := false
		switch stored.DeploymentType {
		case "server":
			isServer = true
		case "cloud":
		default:
			// Auto-detect: probe once and adapt.
			if detected, err := detectJiraDeployment(context, baseURL); err != nil {
				return nil, err
			} else if detected {
				isServer = true
			}
		}
		if isServer {
			path = "/rest/api/2/myself"
		} else {
			if strings.TrimSpace(stored.AccountLabel) == "" {
				return nil, errors.New("Jira account email is required")
			}
			path = "/rest/api/3/myself"
		}
	case "slack":
		if baseURL == "" {
			baseURL = "https://slack.com"
		}
		path = "/api/auth.test"
		method = http.MethodPost
	case "teams":
		if baseURL == "" {
			baseURL = "https://graph.microsoft.com/v1.0"
		}
		path = "/me"
	}
	endpoint, err := integrationEndpoint(baseURL, path)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(context, method, endpoint, nil)
	if err != nil {
		return nil, errors.New("Provider URL is invalid")
	}
	request.Header.Set("Accept", "application/json")
	switch stored.Provider {
	case "jira":
		if path == "/rest/api/2/myself" {
			// Server / Data Center personal access token.
			request.Header.Set("Authorization", "Bearer "+secret)
		} else {
			credentials := base64.StdEncoding.EncodeToString(
				[]byte(strings.TrimSpace(stored.AccountLabel) + ":" + secret),
			)
			request.Header.Set("Authorization", "Basic "+credentials)
		}
	default:
		request.Header.Set("Authorization", "Bearer "+secret)
	}
	if stored.Provider == "github" {
		request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
	return request, nil
}

func integrationEndpoint(baseURL, path string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Host == "" ||
		(parsed.Scheme != "https" && !(integrationAllowInsecureHTTP && parsed.Scheme == "http")) ||
		parsed.User != nil {
		return "", errors.New("Provider base URL must be a valid HTTPS URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func evaluateIntegrationResponse(provider string, response *http.Response) IntegrationConnectionResult {
	if response.StatusCode == http.StatusUnauthorized {
		return connectionFailure(provider, "authentication", "The credential was rejected", response.StatusCode)
	}
	if response.StatusCode == http.StatusForbidden {
		return connectionFailure(provider, "authorization", "The credential lacks required access", response.StatusCode)
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return connectionFailure(provider, "rate_limit", "The provider rate limit was reached", response.StatusCode)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return connectionFailure(
			provider, "provider",
			fmt.Sprintf("The provider returned HTTP %d", response.StatusCode),
			response.StatusCode,
		)
	}
	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return connectionFailure(provider, "invalid_response", "The provider returned an invalid response", response.StatusCode)
	}
	if provider == "slack" {
		ok, _ := payload["ok"].(bool)
		if !ok {
			return connectionFailure(provider, "authentication", "Slack rejected the credential", response.StatusCode)
		}
	}
	identity := integrationIdentity(provider, payload)
	return IntegrationConnectionResult{
		Provider: provider, Success: true, Category: "success",
		Message: "Connection successful", Identity: identity,
		StatusCode: response.StatusCode,
		TestedAt:   integrationTestNow().UTC().Format(time.RFC3339),
	}
}

func integrationIdentity(provider string, payload map[string]any) string {
	keys := map[string][]string{
		"github": {"login"},
		"gitlab": {"username", "email"},
		"jira":   {"displayName", "emailAddress"},
		"slack":  {"team", "user"},
		"teams":  {"displayName", "userPrincipalName"},
	}
	values := make([]string, 0, 2)
	for _, key := range keys[provider] {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			values = append(values, value)
		}
	}
	return strings.Join(values, " · ")
}

func connectionFailure(
	provider, category, message string,
	statusCode int,
) IntegrationConnectionResult {
	return IntegrationConnectionResult{
		Provider: provider, Success: false, Category: category,
		Message: message, StatusCode: statusCode,
		TestedAt: integrationTestNow().UTC().Format(time.RFC3339),
	}
}
