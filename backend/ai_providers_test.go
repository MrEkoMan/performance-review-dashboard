package main

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

func setAIEncryptionKey(t *testing.T) {
	t.Helper()
	t.Setenv(
		"MANAGER_DASHBOARD_ENCRYPTION_KEY",
		base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{11}, 32)),
	)
}

func TestAIProviderConfigurationLifecycle(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()

	if got := request(t, router, http.MethodGet, "/api/ai-providers", nil); got.Code != 200 ||
		got.Body.String() != "[]\n" {
		t.Fatalf("empty AI settings = %d %s", got.Code, got.Body.String())
	}
	body := []byte(`{
		"displayName":"Primary OpenAI",
		"baseUrl":"https://api.openai.com/v1",
		"model":"model-name",
		"apiKey":"super-secret",
		"enabled":true
	}`)
	if got := request(t, router, http.MethodPut, "/api/ai-providers/openai", body); got.Code != http.StatusNoContent {
		t.Fatalf("save AI settings = %d %s", got.Code, got.Body.String())
	}
	list := request(t, router, http.MethodGet, "/api/ai-providers", nil)
	if list.Code != 200 ||
		!strings.Contains(list.Body.String(), `"hasApiKey":true`) ||
		!strings.Contains(list.Body.String(), `"model":"model-name"`) ||
		strings.Contains(list.Body.String(), "super-secret") {
		t.Fatalf("listed AI settings = %d %s", list.Code, list.Body.String())
	}
	var encrypted string
	if err := db.QueryRow(`
		SELECT encrypted_api_key FROM ai_provider_configurations
		WHERE provider = 'openai'`).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if encrypted == "" || strings.Contains(encrypted, "super-secret") {
		t.Fatalf("API key was not encrypted: %q", encrypted)
	}

	update := []byte(`{
		"displayName":"Primary OpenAI",
		"baseUrl":"https://api.openai.com/v1",
		"model":"new-model",
		"apiKey":"",
		"enabled":false
	}`)
	if got := request(t, router, http.MethodPut, "/api/ai-providers/openai", update); got.Code != 204 {
		t.Fatalf("update AI settings = %d %s", got.Code, got.Body.String())
	}
	var updatedEncrypted, model string
	if err := db.QueryRow(`
		SELECT encrypted_api_key, model FROM ai_provider_configurations
		WHERE provider = 'openai'`).Scan(&updatedEncrypted, &model); err != nil {
		t.Fatal(err)
	}
	if updatedEncrypted != encrypted || model != "new-model" {
		t.Fatalf("preserved key = %t, model = %q", updatedEncrypted == encrypted, model)
	}
	if got := request(t, router, http.MethodDelete, "/api/ai-providers/openai", nil); got.Code != 204 {
		t.Fatalf("delete AI settings = %d %s", got.Code, got.Body.String())
	}
	if got := request(t, router, http.MethodDelete, "/api/ai-providers/openai", nil); got.Code != 404 {
		t.Fatalf("missing AI settings = %d %s", got.Code, got.Body.String())
	}
}

func TestOllamaConfigurationDoesNotRequireEncryptionOrAPIKey(t *testing.T) {
	setupTestDatabase(t)
	t.Setenv("MANAGER_DASHBOARD_ENCRYPTION_KEY", "")
	body := []byte(`{
		"displayName":"Local",
		"baseUrl":"http://localhost:11434",
		"model":"llama3.2",
		"enabled":true
	}`)
	got := request(t, newRouter(), http.MethodPut, "/api/ai-providers/ollama", body)
	if got.Code != http.StatusNoContent {
		t.Fatalf("save Ollama = %d %s", got.Code, got.Body.String())
	}
	list := request(t, newRouter(), http.MethodGet, "/api/ai-providers", nil)
	if list.Code != 200 || !strings.Contains(list.Body.String(), `"hasApiKey":false`) {
		t.Fatalf("list Ollama = %d %s", list.Code, list.Body.String())
	}
}

func TestAIProviderConfigurationValidation(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()
	tests := []struct {
		provider, body string
	}{
		{"unknown", `{}`},
		{"openai", `{`},
		{"openai", `{"baseUrl":"","model":"x","apiKey":"key"}`},
		{"openai", `{"baseUrl":"http://api.example.com","model":"x","apiKey":"key"}`},
		{"openai", `{"baseUrl":"https://user@example.com","model":"x","apiKey":"key"}`},
		{"openai", `{"baseUrl":"https://api.example.com","model":"x"}`},
		{"ollama", `{"baseUrl":"ftp://localhost:11434","model":"x"}`},
		{"ollama", `{"baseUrl":"http://example.com:11434","model":"x"}`},
		{"ollama", `{"baseUrl":"http://192.168.1.20:11434","model":"x"}`},
	}
	for index, test := range tests {
		got := request(
			t, router, http.MethodPut,
			"/api/ai-providers/"+test.provider, []byte(test.body),
		)
		if got.Code != http.StatusBadRequest {
			t.Errorf("invalid AI setting %d = %d %s", index, got.Code, got.Body.String())
		}
	}
	if got := request(t, router, http.MethodDelete, "/api/ai-providers/unknown", nil); got.Code != 400 {
		t.Fatalf("unsupported delete = %d %s", got.Code, got.Body.String())
	}

	t.Setenv("MANAGER_DASHBOARD_ENCRYPTION_KEY", "")
	body := []byte(`{"baseUrl":"https://api.example.com","model":"x","apiKey":"key"}`)
	if got := request(t, router, http.MethodPut, "/api/ai-providers/openai", body); got.Code != 500 {
		t.Fatalf("encryption failure = %d %s", got.Code, got.Body.String())
	}
}

func TestAIProviderConfigurationHandlesDatabaseFailure(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"baseUrl":"https://api.example.com","model":"x","apiKey":"key"}`)
	for _, test := range []struct {
		method, path string
		body         []byte
	}{
		{http.MethodGet, "/api/ai-providers", nil},
		{http.MethodPut, "/api/ai-providers/openai", body},
		{http.MethodDelete, "/api/ai-providers/openai", nil},
	} {
		got := request(t, router, test.method, test.path, test.body)
		if got.Code != http.StatusInternalServerError {
			t.Fatalf("%s %s = %d %s", test.method, test.path, got.Code, got.Body.String())
		}
	}
}
