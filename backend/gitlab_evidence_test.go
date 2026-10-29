package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// storeEngineerGitlabUsername sets the GitLab username on an engineer row.
func storeEngineerGitlabUsername(t *testing.T, engineerID int64, username string) {
	t.Helper()
	if _, err := db.Exec(
		`UPDATE engineers SET gitlab_username = ? WHERE id = ?`, username, engineerID,
	); err != nil {
		t.Fatal(err)
	}
}

// gitlabEventsResponse builds one events payload from "action|type|title|iid|project|created" rows.
func gitlabEventsResponse(entries ...string) []byte {
	events := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, "|", 6)
		for len(parts) < 6 {
			parts = append(parts, "")
		}
		action, targetType, title, iid, project, createdAt :=
			parts[0], parts[1], parts[2], parts[3], parts[4], parts[5]
		events = append(events, fmt.Sprintf(`{
			"action_name": %q, "target_type": %q, "target_title": %q,
			"target_iid": %s, "project_id": %s, "created_at": %q
		}`, action, targetType, title, iid, project, createdAt))
	}
	return []byte("[" + strings.Join(events, ",") + "]")
}

// stubGitLabUserResolution answers the /users?username= resolution call used
// before the events feed can be addressed by numeric ID.
func stubGitLabUserResolution(username string, userID int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/v4/users") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("username") != username {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[]`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`[{"id": %d, "username": %q}]`, userID, username)))
	}
}

func TestGitLabEvidenceHarvestLifecycle(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerGitlabUsername(t, engineerID, "brody.dev")

	var capturedAuths []string
	var capturedPaths []string
	var capturedQueries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuths = append(capturedAuths, r.Header.Get("PRIVATE-TOKEN"))
		capturedPaths = append(capturedPaths, r.URL.Path)
		capturedQueries = append(capturedQueries, r.URL.RawQuery)
		if strings.Contains(r.URL.Path, "/api/v4/projects/") {
			// One distinct project, so exactly one lookup is expected.
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"path_with_namespace": "acme/api"}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/api/v4/users") {
			stubGitLabUserResolution("brody.dev", 31415)(w, r)
			return
		}
		if !strings.Contains(r.URL.Path, "/users/31415/events") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(gitlabEventsResponse(
			"merged|merge_request|Add caching layer|42|101|2026-09-05T12:00:00Z",
			"approved|merge_request|Refactor billing|17|101|2026-09-06T10:00:00Z",
			"pushed|push||0|101|2026-09-07T10:00:00Z",
			"created|milestone|Rename epics group|0|0|2026-09-08T10:00:00Z",
		))
	}))
	defer server.Close()
	storeConnectionCredential(t, "gitlab", "", server.URL, "gl-token", true)
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusOK {
		t.Fatalf("harvest = %d %s", got.Code, got.Body.String())
	}
	var result gitlabEvidenceResponse
	if err := json.Unmarshal(got.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	// Pushes and non-MR activity are filtered out; only merges + approvals remain.
	if len(result.Evidence) != 2 {
		t.Fatalf("expected 2 items, got %d: %+v", len(result.Evidence), result.Evidence)
	}
	// Newest first: the approval on 09-06 precedes the merge on 09-05.
	if result.Evidence[0].Kind != "review" || result.Evidence[1].Kind != "pr" {
		t.Errorf("ordering = %s then %s", result.Evidence[0].Kind, result.Evidence[1].Kind)
	}
	merge := result.Evidence[1]
	if merge.Repo != "acme/api" || merge.Title != "Add caching layer" {
		t.Errorf("merge item = %+v", merge)
	}
	if !strings.Contains(merge.BrowseURL, "acme/api/-/merge_requests/42") {
		t.Errorf("browse URL = %q", merge.BrowseURL)
	}
	if !strings.HasPrefix(merge.OccurredAt, "2026-09-05") {
		t.Errorf("merge occurredAt = %q", merge.OccurredAt)
	}
	if result.GitlabUser != "brody.dev" || result.Engineer == "" {
		t.Errorf("echo fields = %+v", result)
	}
	if len(capturedAuths) == 0 {
		t.Fatal("no event requests captured")
	}
	for _, auth := range capturedAuths {
		if auth != "gl-token" {
			t.Errorf("auth = %q", auth)
		}
	}
	for i, query := range capturedQueries {
		// Project lookups carry no query string; only events requests do.
		if !strings.Contains(capturedPaths[i], "/events") {
			continue
		}
		if !strings.Contains(query, "after=2026-09-01") ||
			!strings.Contains(query, "before=2026-10-01") ||
			!strings.Contains(query, "per_page=100") {
			t.Errorf("query = %q", query)
		}
	}
	// Suggested notes are pre-populated for one-click accept.
	if result.Evidence[1].SuggestedNote.Category != "Technical Excellence" {
		t.Errorf("merge category = %q", result.Evidence[1].SuggestedNote.Category)
	}
	if result.Evidence[0].SuggestedNote.Category != "Team Contribution" {
		t.Errorf("review category = %q", result.Evidence[0].SuggestedNote.Category)
	}
	// One distinct project: the project lookup ran exactly once.
	lookups := 0
	for _, path := range capturedPaths {
		if strings.Contains(path, "/api/v4/projects/") {
			lookups++
		}
	}
	if lookups != 1 {
		t.Errorf("project lookups = %d, want 1", lookups)
	}
}

func TestGitLabEvidenceRequiresConfiguredEnabledGitlab(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	storeEngineerGitlabUsername(t, engineerID, "brody.dev")
	router := newRouter()

	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusNotFound ||
		!strings.Contains(got.Body.String(), "not configured") {
		t.Fatalf("unconfigured = %d %s", got.Code, got.Body.String())
	}

	// Disabled: 409.
	storeConnectionCredential(t, "gitlab", "", "https://gitlab.test", "k", false)
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusConflict ||
		!strings.Contains(got.Body.String(), "disabled") {
		t.Fatalf("disabled = %d %s", got.Code, got.Body.String())
	}
}

func TestGitLabEvidenceValidatesInput(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	router := newRouter()

	// No username: 400.
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "GitLab username") {
		t.Fatalf("no username = %d %s", got.Code, got.Body.String())
	}

	// Missing engineer: 404.
	got = request(t, router, http.MethodGet,
		"/api/engineers/9999/gitlab-evidence?from=2026-09-01&to=2026-09-30", nil)
	if got.Code != http.StatusNotFound {
		t.Fatalf("missing engineer = %d, want 404", got.Code)
	}

	// Only one date: 400.
	storeEngineerGitlabUsername(t, engineerID, "brody.dev")
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=2026-09-01", engineerID), nil)
	if got.Code != http.StatusBadRequest {
		t.Fatalf("one date = %d, want 400", got.Code)
	}

	// Reversed range: 400.
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=2026-10-01&to=2026-09-01", engineerID), nil)
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "cannot be before") {
		t.Fatalf("reversed = %d %s", got.Code, got.Body.String())
	}

	// Malformed: 400.
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=09/01/2026&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusBadRequest {
		t.Fatalf("malformed = %d, want 400", got.Code)
	}

	// Range over one year: 400.
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=2025-01-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "one year") {
		t.Fatalf("over one year = %d %s", got.Code, got.Body.String())
	}
}

func TestGitLabEvidenceFollowsNextPage(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerGitlabUsername(t, engineerID, "brody.dev")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/api/v4/users") {
			stubGitLabUserResolution("brody.dev", 31415)(w, r)
			return
		}
		if strings.HasSuffix(r.URL.RawQuery, "page=1") {
			w.Header().Set("X-Next-Page", "2")
			// First page has one merge; second page adds another.
			w.Write(gitlabEventsResponse(
				"merged|merge_request|Page one merge|42|101|2026-09-05T12:00:00Z",
			))
			return
		}
		w.Write(gitlabEventsResponse(
			"merged|merge_request|Page two merge|43|101|2026-09-01T12:00:00Z",
		))
	}))
	defer server.Close()
	storeConnectionCredential(t, "gitlab", "", server.URL, "gl-token", true)
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusOK {
		t.Fatalf("harvest = %d %s", got.Code, got.Body.String())
	}
	var result gitlabEvidenceResponse
	if err := json.Unmarshal(got.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 {
		t.Fatalf("total = %d, want 2 (both pages merged)", result.Total)
	}
}

func TestGitLabEvidenceUnknownUser(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerGitlabUsername(t, engineerID, "ghost.user")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/api/v4/users") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[]`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	storeConnectionCredential(t, "gitlab", "", server.URL, "gl-token", true)
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusNotFound ||
		!strings.Contains(got.Body.String(), "Could not find GitLab user") {
		t.Fatalf("unknown user = %d %s", got.Code, got.Body.String())
	}
}

func TestGitLabEvidenceAcceptCreatesNotes(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	storeEngineerGitlabUsername(t, engineerID, "brody.dev")
	router := newRouter()

	body := []byte(`{
		"items": [
			{
				"repo": "acme/api",
				"kind": "pr",
				"title": "Merged MR: Add caching layer",
				"occurredAt": "2026-09-05T12:00:00Z",
				"url": "https://gitlab.test/acme/api/-/merge_requests/42"
			},
			{
				"repo": "acme/web",
				"kind": "review",
				"title": "Approved MR: Refactor billing",
				"occurredAt": "2026-09-06",
				"category": "Business Impact",
				"details": "Custom details preserved",
				"impact": "Unblocked the release"
			}
		]
	}`)
	got := request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence/accept", engineerID), body)
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
	if !strings.Contains(first.Details, "acme/api") ||
		!strings.Contains(first.Details, "merge_requests/42") {
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

func TestGitLabEvidenceAcceptValidatesItems(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	storeEngineerGitlabUsername(t, engineerID, "brody.dev")
	router := newRouter()

	// Empty items array.
	got := request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence/accept", engineerID),
		[]byte(`{"items":[]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "At least one") {
		t.Fatalf("empty = %d %s", got.Code, got.Body.String())
	}

	// Missing title.
	got = request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence/accept", engineerID),
		[]byte(`{"items":[{"kind":"pr","occurredAt":"2026-09-01"}]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "Title") {
		t.Fatalf("no title = %d %s", got.Code, got.Body.String())
	}

	// Missing occurred date.
	got = request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence/accept", engineerID),
		[]byte(`{"items":[{"kind":"pr","title":"no date"}]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "Occurred date") {
		t.Fatalf("no date = %d %s", got.Code, got.Body.String())
	}

	// Malformed occurred date.
	got = request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence/accept", engineerID),
		[]byte(`{"items":[{"kind":"pr","title":"x","occurredAt":"Sept 1"}]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "YYYY-MM-DD") {
		t.Fatalf("malformed = %d %s", got.Code, got.Body.String())
	}

	// Engineer with no GitLab username.
	otherID := insertEngineer(t)
	got = request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence/accept", otherID),
		[]byte(`{"items":[{"kind":"pr","title":"x","occurredAt":"2026-09-01"}]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "GitLab username") {
		t.Fatalf("no gitlab username = %d %s", got.Code, got.Body.String())
	}
}

func TestGitLabEvidenceProviderFailure(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerGitlabUsername(t, engineerID, "brody.dev")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusUnauthorized)
	}))
	defer server.Close()
	storeConnectionCredential(t, "gitlab", "", server.URL, "k", true)
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusUnauthorized ||
		!strings.Contains(got.Body.String(), "rejected") {
		t.Fatalf("provider failure = %d %s", got.Code, got.Body.String())
	}
}

func TestGitLabEvidenceMalformedResponse(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerGitlabUsername(t, engineerID, "brody.dev")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not-json`))
	}))
	defer server.Close()
	storeConnectionCredential(t, "gitlab", "", server.URL, "k", true)
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusBadGateway ||
		!strings.Contains(got.Body.String(), "invalid response") {
		t.Fatalf("malformed = %d %s", got.Code, got.Body.String())
	}
}

func TestGitLabEvidenceTimeout(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerGitlabUsername(t, engineerID, "brody.dev")
	storeConnectionCredential(t, "gitlab", "", "https://gitlab.test", "k", true)
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
		fmt.Sprintf("/api/engineers/%d/gitlab-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusGatewayTimeout ||
		!strings.Contains(got.Body.String(), "timeout") {
		t.Fatalf("timeout = %d %s", got.Code, got.Body.String())
	}
}
