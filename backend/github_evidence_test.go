package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// storeEngineerGithubUsername sets the GitHub username on an engineer row.
func storeEngineerGithubUsername(t *testing.T, engineerID int64, username string) {
	t.Helper()
	if _, err := db.Exec(
		`UPDATE engineers SET github_username = ? WHERE id = ?`, username, engineerID,
	); err != nil {
		t.Fatal(err)
	}
}

// githubSearchResponse builds one Search API payload. Each entry is
// "title|html_url|repo|closed_at|updated_at".
func githubSearchResponse(entries ...string) []byte {
	items := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, "|", 5)
		for len(parts) < 5 {
			parts = append(parts, "")
		}
		title, htmlURL, repo, closedAt, updatedAt :=
			parts[0], parts[1], parts[2], parts[3], parts[4]
		items = append(items, fmt.Sprintf(`{
			"title": %q, "html_url": %q, "state": "closed",
			"closed_at": %q, "updated_at": %q,
			"repository_url": "https://api.github.test/repos/%s"
		}`, title, htmlURL, closedAt, updatedAt, repo))
	}
	return []byte(fmt.Sprintf(`{
		"total_count": %d, "incomplete_results": false, "items": [%s]
	}`, len(entries), strings.Join(items, ",")))
}

func TestGitHubEvidenceHarvestLifecycle(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerGithubUsername(t, engineerID, "brody-dev")

	var capturedAuths []string
	var capturedQueries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuths = append(capturedAuths, r.Header.Get("Authorization"))
		capturedQueries = append(capturedQueries, r.URL.RawQuery)
		if r.URL.Path != "/search/issues" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// The merged query gets one PR; the review query gets another.
		if strings.Contains(r.URL.RawQuery, "is%3Amerged") {
			w.Write(githubSearchResponse(
				"Add caching layer|https://github.test/acme/api/pull/42|acme/api|2026-09-05T12:00:00Z|2026-09-05T12:00:00Z",
			))
			return
		}
		w.Write(githubSearchResponse(
			"Refactor billing|https://github.test/acme/web/pull/17|acme/web|2026-09-04T09:00:00Z|2026-09-06T10:00:00Z",
		))
	}))
	defer server.Close()
	storeConnectionCredential(t, "github", "", server.URL, "gh-token", true)
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/github-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusOK {
		t.Fatalf("harvest = %d %s", got.Code, got.Body.String())
	}
	var result githubEvidenceResponse
	if err := json.Unmarshal(got.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Evidence) != 2 {
		t.Fatalf("expected 2 items, got %d: %+v", len(result.Evidence), result.Evidence)
	}
	// Newest first: the review updated 09-06 precedes the merge on 09-05.
	if result.Evidence[0].Kind != "review" || result.Evidence[1].Kind != "pr" {
		t.Errorf("ordering = %s then %s", result.Evidence[0].Kind, result.Evidence[1].Kind)
	}
	merge := result.Evidence[1]
	if merge.Repo != "acme/api" || merge.Title != "Add caching layer" {
		t.Errorf("merge item = %+v", merge)
	}
	if !strings.HasPrefix(merge.OccurredAt, "2026-09-05") {
		t.Errorf("merge occurredAt = %q", merge.OccurredAt)
	}
	if !strings.Contains(merge.BrowseURL, "/pull/42") {
		t.Errorf("browse URL = %q", merge.BrowseURL)
	}
	if result.GitHubUser != "brody-dev" || result.Engineer == "" {
		t.Errorf("echo fields = %+v", result)
	}
	if len(capturedAuths) != 2 {
		t.Fatalf("expected 2 search requests, got %d", len(capturedAuths))
	}
	for _, auth := range capturedAuths {
		if auth != "Bearer gh-token" {
			t.Errorf("auth = %q", auth)
		}
	}
	for _, query := range capturedQueries {
		if !strings.Contains(query, "per_page=100") {
			t.Errorf("query = %q", query)
		}
	}
	// Suggested notes are pre-populated for one-click accept.
	for _, item := range result.Evidence {
		if item.SuggestedNote.Summary == "" || item.SuggestedNote.Category == "" {
			t.Errorf("suggested note = %+v", item.SuggestedNote)
		}
	}
	if result.Evidence[1].SuggestedNote.Category != "Technical Excellence" {
		t.Errorf("merge category = %q", result.Evidence[1].SuggestedNote.Category)
	}
	if result.Evidence[0].SuggestedNote.Category != "Team Contribution" {
		t.Errorf("review category = %q", result.Evidence[0].SuggestedNote.Category)
	}
}

func TestGitHubEvidenceRequiresConfiguredEnabledGithub(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	storeEngineerGithubUsername(t, engineerID, "brody-dev")
	router := newRouter()

	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/github-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusNotFound ||
		!strings.Contains(got.Body.String(), "not configured") {
		t.Fatalf("unconfigured = %d %s", got.Code, got.Body.String())
	}

	// Disabled: 409.
	storeConnectionCredential(t, "github", "", "https://api.github.com", "k", false)
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/github-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusConflict ||
		!strings.Contains(got.Body.String(), "disabled") {
		t.Fatalf("disabled = %d %s", got.Code, got.Body.String())
	}
}

func TestGitHubEvidenceValidatesInput(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	router := newRouter()

	// No username: 400.
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/github-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "GitHub username") {
		t.Fatalf("no username = %d %s", got.Code, got.Body.String())
	}

	// Missing engineer: 404.
	got = request(t, router, http.MethodGet,
		"/api/engineers/9999/github-evidence?from=2026-09-01&to=2026-09-30", nil)
	if got.Code != http.StatusNotFound {
		t.Fatalf("missing engineer = %d, want 404", got.Code)
	}

	// Only one date: 400.
	storeEngineerGithubUsername(t, engineerID, "brody-dev")
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/github-evidence?from=2026-09-01", engineerID), nil)
	if got.Code != http.StatusBadRequest {
		t.Fatalf("one date = %d, want 400", got.Code)
	}

	// Reversed range: 400.
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/github-evidence?from=2026-10-01&to=2026-09-01", engineerID), nil)
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "cannot be before") {
		t.Fatalf("reversed = %d %s", got.Code, got.Body.String())
	}

	// Malformed: 400.
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/github-evidence?from=09/01/2026&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusBadRequest {
		t.Fatalf("malformed = %d, want 400", got.Code)
	}

	// Range over one year: 400.
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/github-evidence?from=2025-01-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "one year") {
		t.Fatalf("over one year = %d %s", got.Code, got.Body.String())
	}
}

func TestGitHubEvidenceDedupesSamePullRequest(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerGithubUsername(t, engineerID, "brody-dev")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Both searches surface the same pull request URL.
		w.Write(githubSearchResponse(
			"Shared PR|https://github.test/acme/api/pull/7|acme/api|2026-09-05T12:00:00Z|2026-09-06T10:00:00Z",
		))
	}))
	defer server.Close()
	storeConnectionCredential(t, "github", "", server.URL, "gh-token", true)
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/github-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusOK {
		t.Fatalf("harvest = %d %s", got.Code, got.Body.String())
	}
	var result githubEvidenceResponse
	if err := json.Unmarshal(got.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Evidence) != 1 {
		t.Fatalf("expected 1 deduped item, got %d", len(result.Evidence))
	}
	if result.Total != 1 {
		t.Errorf("total = %d", result.Total)
	}
}

func TestGitHubEvidenceAcceptCreatesNotes(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	storeEngineerGithubUsername(t, engineerID, "brody-dev")
	router := newRouter()

	body := []byte(`{
		"items": [
			{
				"repo": "acme/api",
				"kind": "pr",
				"title": "Merged PR: Add caching layer",
				"occurredAt": "2026-09-05T12:00:00Z",
				"url": "https://github.test/acme/api/pull/42"
			},
			{
				"repo": "acme/web",
				"kind": "review",
				"title": "Reviewed PR: Refactor billing",
				"occurredAt": "2026-09-06",
				"category": "Business Impact",
				"details": "Custom details preserved",
				"impact": "Unblocked the release"
			}
		]
	}`)
	got := request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/github-evidence/accept", engineerID), body)
	if got.Code != http.StatusCreated {
		t.Fatalf("accept = %d %s", got.Code, got.Body.String())
	}
	var response struct {
		Created []PerformanceNote `json:"created"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Created) != 2 {
		t.Fatalf("created = %+v", response.Created)
	}
	first := response.Created[0]
	if first.NoteDate != "2026-09-05" || first.Category != "Technical Excellence" {
		t.Errorf("first note = %+v", first)
	}
	// Default details name the repo and link the pull request.
	if !strings.Contains(first.Details, "acme/api") ||
		!strings.Contains(first.Details, "pull/42") {
		t.Errorf("first details = %q", first.Details)
	}
	second := response.Created[1]
	if second.Category != "Business Impact" || second.Details != "Custom details preserved" {
		t.Errorf("second note = %+v", second)
	}

	// The notes are real performance_notes rows and appear in the timeline.
	timeline := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/timeline", engineerID), nil)
	if timeline.Code != http.StatusOK ||
		!strings.Contains(timeline.Body.String(), "Add caching layer") {
		t.Fatalf("timeline = %d %s", timeline.Code, timeline.Body.String())
	}
}

func TestGitHubEvidenceAcceptValidatesItems(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	storeEngineerGithubUsername(t, engineerID, "brody-dev")
	router := newRouter()

	// Empty items array.
	got := request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/github-evidence/accept", engineerID),
		[]byte(`{"items":[]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "At least one") {
		t.Fatalf("empty = %d %s", got.Code, got.Body.String())
	}

	// Missing title.
	got = request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/github-evidence/accept", engineerID),
		[]byte(`{"items":[{"kind":"pr","occurredAt":"2026-09-01"}]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "Title") {
		t.Fatalf("no title = %d %s", got.Code, got.Body.String())
	}

	// Missing occurred date.
	got = request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/github-evidence/accept", engineerID),
		[]byte(`{"items":[{"kind":"pr","title":"no date"}]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "Occurred date") {
		t.Fatalf("no date = %d %s", got.Code, got.Body.String())
	}

	// Malformed occurred date.
	got = request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/github-evidence/accept", engineerID),
		[]byte(`{"items":[{"kind":"pr","title":"x","occurredAt":"Sept 1"}]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "YYYY-MM-DD") {
		t.Fatalf("malformed = %d %s", got.Code, got.Body.String())
	}

	// Engineer with no GitHub username.
	otherID := insertEngineer(t)
	got = request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/github-evidence/accept", otherID),
		[]byte(`{"items":[{"kind":"pr","title":"x","occurredAt":"2026-09-01"}]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "GitHub username") {
		t.Fatalf("no github username = %d %s", got.Code, got.Body.String())
	}
}

func TestGitHubEvidenceProviderFailure(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerGithubUsername(t, engineerID, "brody-dev")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusUnauthorized)
	}))
	defer server.Close()
	storeConnectionCredential(t, "github", "", server.URL, "k", true)
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/github-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusUnauthorized ||
		!strings.Contains(got.Body.String(), "rejected") {
		t.Fatalf("provider failure = %d %s", got.Code, got.Body.String())
	}
}

func TestGitHubEvidenceMalformedResponse(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerGithubUsername(t, engineerID, "brody-dev")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not-json`))
	}))
	defer server.Close()
	storeConnectionCredential(t, "github", "", server.URL, "k", true)
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/github-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusBadGateway ||
		!strings.Contains(got.Body.String(), "invalid response") {
		t.Fatalf("malformed = %d %s", got.Code, got.Body.String())
	}
}

func TestGitHubEvidenceTimeout(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerGithubUsername(t, engineerID, "brody-dev")
	storeConnectionCredential(t, "github", "", "https://api.github.test", "k", true)
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	originalClient := integrationHTTPClient
	integrationHTTPClient = &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		}),
	}
	t.Cleanup(func() { integrationHTTPClient = originalClient })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/github-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusGatewayTimeout ||
		!strings.Contains(got.Body.String(), "timeout") {
		t.Fatalf("timeout = %d %s", got.Code, got.Body.String())
	}
	// Satisfy the unused-import check when context is only used here.
	_ = errors.Is
}
