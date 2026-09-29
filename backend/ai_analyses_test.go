package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// withAIClient swaps the package-level AI HTTP client for one whose transport
// is the supplied function, mirroring how the integration-connection tests stub
// outbound HTTP. Restored on cleanup.
func withAIClient(t *testing.T, fn roundTripFunc) {
	t.Helper()
	previous := aiHTTPClient
	aiHTTPClient = &http.Client{Transport: fn}
	t.Cleanup(func() { aiHTTPClient = previous })
}

// storeAIProviderConfig inserts or replaces an AI provider configuration
// directly, bypassing the PUT handler's HTTPS/loopback URL validation so tests
// can point the provider at stub endpoints.
func storeAIProviderConfig(
	t *testing.T,
	provider, baseURL, model, apiVersion, apiKey string,
	enabled bool,
) {
	t.Helper()
	encrypted := ""
	if apiKey != "" {
		var err error
		encrypted, err = encryptSecret(apiKey)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`
		INSERT INTO ai_provider_configurations
			(provider, display_name, base_url, model, api_version,
			 encrypted_api_key, enabled, updated_at)
		VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, CURRENT_TIMESTAMP)
		ON CONFLICT(provider) DO UPDATE SET
			base_url = excluded.base_url,
			model = excluded.model,
			api_version = excluded.api_version,
			encrypted_api_key = excluded.encrypted_api_key,
			enabled = excluded.enabled`,
		provider, provider+" test", baseURL, model, apiVersion, encrypted, enabled,
	); err != nil {
		t.Fatal(err)
	}
}

func aiStubResponse(content string) []byte {
	return []byte(`{"stub":true,"text":"` + content + `"}`)
}

// stubAIProvider returns a stub response shaped for the given provider family.
func aiProviderStubResponse(provider, content string) []byte {
	switch provider {
	case "openai", "azure_openai", "openrouter":
		return []byte(`{"choices":[{"message":{"content":"` + content + `"}}]}`)
	case "anthropic":
		return []byte(`{"content":[{"type":"text","text":"` + content + `"}]}`)
	case "gemini":
		return []byte(`{"candidates":[{"content":{"parts":[{"text":"` + content + `"}]}}]}`)
	case "ollama":
		return []byte(`{"message":{"content":"` + content + `"}}`)
	default:
		return []byte(`{}`)
	}
}

func stubAIProvider(provider, content string, capture func(r *http.Request)) roundTripFunc {
	return func(request *http.Request) (*http.Response, error) {
		if capture != nil {
			capture(request)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(aiProviderStubResponse(provider, content)))),
			Header:     make(http.Header),
		}, nil
	}
}

func TestAIAnalysisLifecycle(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()
	engineerID := insertEngineer(t)
	storeAIProviderConfig(t, "openai", "https://api.openai.com", "gpt-test", "", "secret-key", true)

	var capturedHeaders http.Header
	withAIClient(t, stubAIProvider("openai", "## Summary: Looks solid.", func(r *http.Request) {
		capturedHeaders = r.Header.Clone()
	}))

	got := request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses",
		[]byte(`{"provider":"openai","reviewCycle":"2026-H1"}`))
	if got.Code != http.StatusCreated {
		t.Fatalf("create analysis = %d %s", got.Code, got.Body.String())
	}
	var created AIAnalysis
	if err := json.Unmarshal(got.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Provider != "openai" || created.Model != "gpt-test" {
		t.Errorf("created = %+v", created)
	}
	if created.ReviewCycle != "2026-H1" {
		t.Errorf("ReviewCycle = %q", created.ReviewCycle)
	}
	if created.ContextHash == "" || created.OutputMarkdown == "" {
		t.Errorf("expected hash and output, got %+v", created)
	}
	if capturedHeaders.Get("Authorization") != "Bearer secret-key" {
		t.Errorf("expected Bearer auth, got %v", capturedHeaders)
	}

	// The plaintext key must never appear in the response.
	if strings.Contains(got.Body.String(), "secret-key") {
		t.Error("plaintext API key leaked in response")
	}

	listed := request(t, router, http.MethodGet,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses", nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list analyses = %d", listed.Code)
	}
	var analyses []AIAnalysis
	if err := json.Unmarshal(listed.Body.Bytes(), &analyses); err != nil {
		t.Fatal(err)
	}
	if len(analyses) != 1 || analyses[0].OutputMarkdown != created.OutputMarkdown {
		t.Errorf("listed analyses = %+v", analyses)
	}

	deleted := request(t, router, http.MethodDelete,
		"/api/ai-analyses/"+strconv.FormatInt(created.ID, 10), nil)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete analysis = %d", deleted.Code)
	}
	again := request(t, router, http.MethodDelete,
		"/api/ai-analyses/"+strconv.FormatInt(created.ID, 10), nil)
	if again.Code != http.StatusNotFound {
		t.Fatalf("second delete = %d, want 404", again.Code)
	}
}

func TestAIAnalysisAnthropicAdapter(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()
	engineerID := insertEngineer(t)
	storeAIProviderConfig(t, "anthropic", "https://api.anthropic.com", "claude-sonnet-5", "", "sk-ant-test", true)

	var capturedHeaders http.Header
	var capturedPath string
	withAIClient(t, stubAIProvider("anthropic", "Good evidence base.", func(r *http.Request) {
		capturedHeaders = r.Header.Clone()
		capturedPath = r.URL.Path
	}))

	got := request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses",
		[]byte(`{"provider":"anthropic"}`))
	if got.Code != http.StatusCreated {
		t.Fatalf("create analysis = %d %s", got.Code, got.Body.String())
	}
	if capturedHeaders.Get("x-api-key") != "sk-ant-test" {
		t.Errorf("x-api-key = %q", capturedHeaders.Get("x-api-key"))
	}
	if capturedHeaders.Get("Authorization") != "" {
		t.Error("Authorization header should not be set for anthropic")
	}
	if capturedHeaders.Get("anthropic-version") != "2023-06-01" {
		t.Errorf("anthropic-version = %q", capturedHeaders.Get("anthropic-version"))
	}
	if capturedPath != "/v1/messages" {
		t.Errorf("path = %q", capturedPath)
	}
}

func TestAIAnalysisGeminiAndOllamaAdapters(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()
	engineerID := insertEngineer(t)
	storeAIProviderConfig(t, "gemini", "https://generativelanguage.googleapis.com/v1beta", "gemini-test", "", "g-key", true)
	storeAIProviderConfig(t, "ollama", "http://localhost:11434", "llama3.2", "", "", true)

	var geminiURL string
	withAIClient(t, stubAIProvider("gemini", "Gemini says hi.", func(r *http.Request) {
		geminiURL = r.URL.String()
	}))
	got := request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses",
		[]byte(`{"provider":"gemini"}`))
	if got.Code != http.StatusCreated {
		t.Fatalf("gemini analysis = %d %s", got.Code, got.Body.String())
	}
	if !strings.Contains(geminiURL, "key=g-key") {
		t.Errorf("gemini URL missing key: %s", geminiURL)
	}
	if !strings.Contains(geminiURL, ":generateContent") {
		t.Errorf("gemini URL missing generateContent: %s", geminiURL)
	}

	var ollamaHeaders http.Header
	withAIClient(t, stubAIProvider("ollama", "Local model output.", func(r *http.Request) {
		ollamaHeaders = r.Header.Clone()
	}))
	got = request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses",
		[]byte(`{"provider":"ollama"}`))
	if got.Code != http.StatusCreated {
		t.Fatalf("ollama analysis = %d %s", got.Code, got.Body.String())
	}
	if ollamaHeaders.Get("Authorization") != "" {
		t.Error("ollama should not send an Authorization header")
	}
}

func TestAIAnalysisOllamaWorksWithoutEncryptionKey(t *testing.T) {
	setupTestDatabase(t)
	t.Setenv("MANAGER_DASHBOARD_ENCRYPTION_KEY", "")
	router := newRouter()
	engineerID := insertEngineer(t)
	storeAIProviderConfig(t, "ollama", "http://localhost:11434", "llama3.2", "", "", true)

	withAIClient(t, stubAIProvider("ollama", "Local model output.", nil))
	got := request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses",
		[]byte(`{"provider":"ollama"}`))
	if got.Code != http.StatusCreated {
		t.Fatalf("ollama analysis without encryption key = %d %s", got.Code, got.Body.String())
	}
}

func TestAIAnalysisAzureOpenAIUsesAPIVersion(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()
	engineerID := insertEngineer(t)
	storeAIProviderConfig(
		t, "azure_openai", "https://example.openai.azure.com/openai",
		"deployment-x", "2024-10-01", "azure-key", true)

	var capturedURL string
	var capturedHeaders http.Header
	withAIClient(t, stubAIProvider("azure_openai", "Azure output.", func(r *http.Request) {
		capturedURL = r.URL.String()
		capturedHeaders = r.Header.Clone()
	}))

	got := request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses",
		[]byte(`{"provider":"azure_openai"}`))
	if got.Code != http.StatusCreated {
		t.Fatalf("azure analysis = %d %s", got.Code, got.Body.String())
	}
	if !strings.Contains(capturedURL, "api-version=2024-10-01") {
		t.Errorf("azure URL missing api-version: %s", capturedURL)
	}
	if capturedHeaders.Get("api-key") != "azure-key" {
		t.Errorf("api-key = %q", capturedHeaders.Get("api-key"))
	}
	if capturedHeaders.Get("Authorization") != "" {
		t.Error("Authorization header should not be set when api-version is configured")
	}
}

func TestAIAnalysisRequiresConfiguredEnabledProvider(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()
	engineerID := insertEngineer(t)

	got := request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses",
		[]byte(`{"provider":"openai"}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "not configured") {
		t.Fatalf("unconfigured provider = %d %s", got.Code, got.Body.String())
	}

	storeAIProviderConfig(t, "openai", "https://api.openai.com", "gpt-test", "", "k", false)
	got = request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses",
		[]byte(`{"provider":"openai"}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "disabled") {
		t.Fatalf("disabled provider = %d %s", got.Code, got.Body.String())
	}

	got = request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses",
		[]byte(`{"provider":"mystery"}`))
	if got.Code != http.StatusBadRequest {
		t.Fatalf("unknown provider = %d, want 400", got.Code)
	}
}

func TestAIAnalysisMissingEngineer(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()
	storeAIProviderConfig(t, "openai", "https://api.openai.com", "gpt-test", "", "k", true)

	got := request(t, router, http.MethodPost,
		"/api/engineers/9999/ai-analyses", []byte(`{"provider":"openai"}`))
	if got.Code != http.StatusNotFound {
		t.Fatalf("missing engineer = %d, want 404", got.Code)
	}
}

func TestAIAnalysisProviderFailure(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()
	engineerID := insertEngineer(t)
	storeAIProviderConfig(t, "openai", "https://api.openai.com", "gpt-test", "", "k", true)

	withAIClient(t, func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`{"error":"bad key"}`)),
			Header:     make(http.Header),
		}, nil
	})

	got := request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses",
		[]byte(`{"provider":"openai"}`))
	if got.Code != http.StatusBadGateway ||
		!strings.Contains(got.Body.String(), "HTTP 401") {
		t.Fatalf("provider failure = %d %s", got.Code, got.Body.String())
	}
	if strings.Contains(got.Body.String(), "bad key") {
		t.Error("provider response body should not be echoed to the client")
	}

	listed := request(t, router, http.MethodGet,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses", nil)
	if listed.Body.String() != "[]\n" {
		t.Errorf("failed analyses must not be persisted: %s", listed.Body.String())
	}
}

func TestAIAnalysisTimeout(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()
	engineerID := insertEngineer(t)
	storeAIProviderConfig(t, "openai", "https://api.openai.com", "gpt-test", "", "k", true)

	withAIClient(t, func(_ *http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})

	got := request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses",
		[]byte(`{"provider":"openai"}`))
	if got.Code != http.StatusGatewayTimeout {
		t.Fatalf("timeout = %d, want 504", got.Code)
	}
}

func TestAIAnalysisInvalidRequestBody(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()
	engineerID := insertEngineer(t)

	got := request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/ai-analyses",
		[]byte(`{`))
	if got.Code != http.StatusBadRequest {
		t.Fatalf("invalid body = %d, want 400", got.Code)
	}
}

func TestAIAnalysisHandlesDatabaseFailure(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	router := newRouter()
	db.Close()

	got := request(t, router, http.MethodGet, "/api/engineers/1/ai-analyses", nil)
	if got.Code != http.StatusInternalServerError {
		t.Fatalf("list with closed db = %d, want 500", got.Code)
	}
	got = request(t, router, http.MethodDelete, "/api/ai-analyses/1", nil)
	if got.Code != http.StatusInternalServerError {
		t.Fatalf("delete with closed db = %d, want 500", got.Code)
	}
	got = request(t, router, http.MethodPost, "/api/engineers/1/ai-analyses",
		[]byte(`{"provider":"openai"}`))
	if got.Code != http.StatusInternalServerError {
		t.Fatalf("create with closed db = %d, want 500", got.Code)
	}
}

func TestBuildAnalysisContextBounds(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	for index := 0; index < 60; index++ {
		// Stagger dates so ordering is deterministic: the newest note is
		// index 59, the oldest index 0.
		date := "2026-01-" + padDay(index)
		if _, err := db.Exec(`
			INSERT INTO performance_notes
				(engineer_id, note_date, category, summary, review_cycle)
			VALUES (?, ?, 'Business Impact', ?, '2026-H1')`,
			engineerID, date, "Note number "+strconv.Itoa(index),
		); err != nil {
			t.Fatal(err)
		}
	}
	for index := 0; index < 25; index++ {
		if _, err := db.Exec(`
			INSERT INTO one_on_ones (engineer_id, meeting_date, status)
			VALUES (?, ?, 'completed')`,
			engineerID, "2026-02-01",
		); err != nil {
			t.Fatal(err)
		}
	}

	prompt, summary, hash, err := buildAnalysisContext(engineerID, "")
	if err != nil {
		t.Fatal(err)
	}
	var decoded analysisContextSummary
	if err := json.Unmarshal([]byte(summary), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Notes != 50 || decoded.OneOnOnes != 20 {
		t.Errorf("summary counts = %+v, want 50 notes / 20 one-on-ones", decoded)
	}
	if !strings.Contains(prompt, "Note number 59") {
		t.Error("newest note should be included")
	}
	if strings.Contains(prompt, "Note number 5 ") || strings.Contains(prompt, "Note number 5\n") {
		t.Error("oldest notes beyond the cap should be excluded")
	}
	if hash == "" || len(hash) != 64 {
		t.Errorf("hash = %q", hash)
	}
}

func TestBuildAnalysisContextReviewCycleFilter(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	if _, err := db.Exec(`
		INSERT INTO performance_notes
			(engineer_id, note_date, category, summary, review_cycle)
		VALUES (?, '2026-01-15', 'Business Impact', 'Current cycle note', '2026-H1')`,
		engineerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO performance_notes
			(engineer_id, note_date, category, summary, review_cycle)
		VALUES (?, '2025-07-15', 'Business Impact', 'Old cycle note', '2025-H2')`,
		engineerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO one_on_ones (engineer_id, meeting_date, status)
		VALUES (?, '2026-02-01', 'completed')`, engineerID); err != nil {
		t.Fatal(err)
	}

	prompt, _, _, err := buildAnalysisContext(engineerID, "2026-H1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "Current cycle note") {
		t.Error("current cycle note should be included")
	}
	if strings.Contains(prompt, "Old cycle note") {
		t.Error("other cycle note should be excluded")
	}
	if !strings.Contains(prompt, "2026-02-01") {
		t.Error("one-on-ones have no cycle column and should always be included")
	}
}

func TestExtractContentHelpers(t *testing.T) {
	if content, err := extractOpenAIContent(aiProviderStubResponse("openai", "hello")); err != nil || content != "hello" {
		t.Errorf("openai extract = %q, %v", content, err)
	}
	if content, err := extractAnthropicContent(aiProviderStubResponse("anthropic", "hi")); err != nil || content != "hi" {
		t.Errorf("anthropic extract = %q, %v", content, err)
	}
	if content, err := extractGeminiContent(aiProviderStubResponse("gemini", "hey")); err != nil || content != "hey" {
		t.Errorf("gemini extract = %q, %v", content, err)
	}
	if content, err := extractOllamaContent(aiProviderStubResponse("ollama", "yo")); err != nil || content != "yo" {
		t.Errorf("ollama extract = %q, %v", content, err)
	}
	if _, err := extractOpenAIContent([]byte(`{"choices":[]}`)); err == nil {
		t.Error("empty choices should error")
	}
	if _, err := extractAnthropicContent([]byte(`{"content":[]}`)); err == nil {
		t.Error("empty content should error")
	}
	if _, err := extractOllamaContent(aiStubResponse("ignored")); err == nil {
		t.Error("unrelated payload should error")
	}
}

// padDay renders a 1-based zero-padded day string for the bounds test.
func padDay(index int) string {
	if index < 9 {
		return "0" + strconv.Itoa(index+1)
	}
	return strconv.Itoa(index + 1)
}
