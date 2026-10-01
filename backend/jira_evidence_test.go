package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"strings"
	"testing"
)

// storeEngineerJiraUsername sets the Jira username on an engineer row.
func storeEngineerJiraUsername(t *testing.T, engineerID int64, username string) {
	t.Helper()
	if _, err := db.Exec(
		`UPDATE engineers SET jira_username = ? WHERE id = ?`, username, engineerID,
	); err != nil {
		t.Fatal(err)
	}
}

func jiraSearchResponse(keys ...string) []byte {
	issues := make([]string, 0, len(keys))
	for index, key := range keys {
		issues = append(issues, fmt.Sprintf(`{
			"key": "%s",
			"fields": {
				"summary": "Shipped feature %s",
				"status": {"name": "Done"},
				"priority": {"name": "High"},
				"resolution": {"name": "Done"},
				"resolutiondate": "2026-09-0%dT10:00:00.000+0000",
				"issuetype": {"name": "Story"}
			}
		}`, key, key, index+1))
	}
	return []byte(fmt.Sprintf(`{
		"startAt": 0, "total": %d, "maxResults": 50,
		"issues": [%s]
	}`, len(keys), strings.Join(issues, ",")))
}

func TestJiraEvidenceHarvestLifecycle(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerJiraUsername(t, engineerID, "brody.clark")

	var capturedAuth, capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedQuery = r.URL.RawQuery
		if !strings.Contains(r.URL.Path, "/rest/api/2/search") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jiraSearchResponse("PROJ-100", "PROJ-101"))
	}))
	defer server.Close()
	// Explicit server deployment type: the harvest tests v2 + Bearer auth,
	// and no serverInfo probe runs.
	storeConnectionCredentialWithType(
		t, "jira", "", server.URL, "jira-pat", true, "server")
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/jira-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusOK {
		t.Fatalf("harvest = %d %s", got.Code, got.Body.String())
	}
	var result jiraEvidenceResponse
	if err := json.Unmarshal(got.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || len(result.Evidence) != 2 {
		t.Fatalf("expected 2 issues, got total=%d len=%d", result.Total, len(result.Evidence))
	}
	first := result.Evidence[0]
	if first.IssueKey != "PROJ-100" || first.Resolution != "Done" ||
		!strings.HasPrefix(first.ResolvedDate, "2026-09-01") {
		t.Errorf("first evidence = %+v", first)
	}
	if !strings.Contains(first.BrowseURL, "/browse/PROJ-100") {
		t.Errorf("browse URL = %q", first.BrowseURL)
	}
	if result.JiraUser != "brody.clark" || result.Engineer == "" {
		t.Errorf("echo fields = %+v", result)
	}
	if !strings.Contains(capturedAuth, "Bearer jira-pat") {
		t.Errorf("expected Bearer auth for Server, got %q", capturedAuth)
	}
	if !strings.Contains(capturedQuery, "jql=assignee") ||
		!strings.Contains(capturedQuery, "resolved") {
		t.Errorf("query = %q", capturedQuery)
	}
	// Suggested note should be pre-populated for one-click accept.
	if first.SuggestedNote.Summary == "" || first.SuggestedNote.Category == "" {
		t.Errorf("suggested note = %+v", first.SuggestedNote)
	}
}

func TestJiraEvidenceRequiresConfiguredEnabledJira(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	storeEngineerJiraUsername(t, engineerID, "brody.clark")
	router := newRouter()

	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/jira-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusNotFound ||
		!strings.Contains(got.Body.String(), "not configured") {
		t.Fatalf("unconfigured = %d %s", got.Code, got.Body.String())
	}

	// Disabled: 409.
	storeConnectionCredential(t, "jira", "", "https://jira.example.com", "k", false)
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/jira-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusConflict ||
		!strings.Contains(got.Body.String(), "disabled") {
		t.Fatalf("disabled = %d %s", got.Code, got.Body.String())
	}
}

func TestJiraEvidenceValidatesInput(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	router := newRouter()

	// No username: 400.
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/jira-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "Jira username") {
		t.Fatalf("no username = %d %s", got.Code, got.Body.String())
	}

	// Missing engineer: 404.
	got = request(t, router, http.MethodGet,
		"/api/engineers/9999/jira-evidence?from=2026-09-01&to=2026-09-30", nil)
	if got.Code != http.StatusNotFound {
		t.Fatalf("missing engineer = %d, want 404", got.Code)
	}

	// Only one date: 400.
	storeEngineerJiraUsername(t, engineerID, "brody.clark")
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/jira-evidence?from=2026-09-01", engineerID), nil)
	if got.Code != http.StatusBadRequest {
		t.Fatalf("one date = %d, want 400", got.Code)
	}

	// Reversed range: 400.
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/jira-evidence?from=2026-10-01&to=2026-09-01", engineerID), nil)
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "cannot be before") {
		t.Fatalf("reversed = %d %s", got.Code, got.Body.String())
	}

	// Malformed: 400.
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/jira-evidence?from=09/01/2026&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusBadRequest {
		t.Fatalf("malformed = %d, want 400", got.Code)
	}

	// Range over one year: 400.
	got = request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/jira-evidence?from=2025-01-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "one year") {
		t.Fatalf("over one year = %d %s", got.Code, got.Body.String())
	}
}

func TestJiraEvidenceAcceptCreatesNotes(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerJiraUsername(t, engineerID, "brody.clark")
	router := newRouter()

	body := []byte(`{
		"items": [
			{
				"issueKey": "PROJ-100",
				"summary": "[PROJ-100] Shipped feature A",
				"resolvedDate": "2026-09-01T10:00:00.000+0000",
				"browseUrl": "https://jira/browse/PROJ-100"
			},
			{
				"issueKey": "PROJ-101",
				"summary": "[PROJ-101] Shipped feature B",
				"resolvedDate": "2026-09-02",
				"category": "Business Impact",
				"details": "Custom details preserved",
				"impact": "Unblocked the release"
			}
		]
	}`)
	got := request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/jira-evidence/accept", engineerID), body)
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
	if first.NoteDate != "2026-09-01" || first.Category != "Technical Excellence" {
		t.Errorf("first note = %+v", first)
	}
	// Default details from browse URL when not overridden.
	if !strings.Contains(first.Details, "PROJ-100") {
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
		!strings.Contains(timeline.Body.String(), "PROJ-100") {
		t.Fatalf("timeline = %d %s", timeline.Code, timeline.Body.String())
	}
}

func TestJiraEvidenceAcceptValidatesItems(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	storeEngineerJiraUsername(t, engineerID, "brody.clark")
	router := newRouter()

	// Empty items array.
	got := request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/jira-evidence/accept", engineerID),
		[]byte(`{"items":[]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "At least one") {
		t.Fatalf("empty = %d %s", got.Code, got.Body.String())
	}

	// Missing issue key.
	got = request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/jira-evidence/accept", engineerID),
		[]byte(`{"items":[{"summary":"no key","resolvedDate":"2026-09-01"}]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "Issue key") {
		t.Fatalf("no key = %d %s", got.Code, got.Body.String())
	}

	// Missing resolved date.
	got = request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/jira-evidence/accept", engineerID),
		[]byte(`{"items":[{"issueKey":"PROJ-1","summary":"no date"}]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "Resolved date") {
		t.Fatalf("no date = %d %s", got.Code, got.Body.String())
	}

	// Malformed resolved date.
	got = request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/jira-evidence/accept", engineerID),
		[]byte(`{"items":[{"issueKey":"PROJ-1","resolvedDate":"Sept 1"}]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "YYYY-MM-DD") {
		t.Fatalf("malformed = %d %s", got.Code, got.Body.String())
	}

	// Engineer with no Jira username.
	otherID := insertEngineer(t)
	got = request(t, router, http.MethodPost,
		fmt.Sprintf("/api/engineers/%d/jira-evidence/accept", otherID),
		[]byte(`{"items":[{"issueKey":"PROJ-1","resolvedDate":"2026-09-01"}]}`))
	if got.Code != http.StatusBadRequest ||
		!strings.Contains(got.Body.String(), "Jira username") {
		t.Fatalf("no jira username = %d %s", got.Code, got.Body.String())
	}
}

func TestJiraEvidenceProviderFailure(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerJiraUsername(t, engineerID, "brody.clark")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusUnauthorized)
	}))
	defer server.Close()
	storeConnectionCredential(t, "jira", "", server.URL, "k", true)
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/jira-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusUnauthorized ||
		!strings.Contains(got.Body.String(), "rejected") {
		t.Fatalf("provider failure = %d %s", got.Code, got.Body.String())
	}
}

func TestJiraEvidenceCloudUsesBasicAuth(t *testing.T) {
	setupTestDatabase(t)
	setAIEncryptionKey(t)
	engineerID := insertEngineer(t)
	storeEngineerJiraUsername(t, engineerID, "brody.clark")

	var capturedAuth, capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedPath = r.URL.Path
		if strings.Contains(r.URL.Path, "/rest/api/3/search") {
			w.Write(jiraSearchResponse("PROJ-200"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	// Explicit Cloud deployment type, so no serverInfo probe runs.
	storeConnectionCredentialWithType(
		t, "jira", "manager@example.com", server.URL, "cloud-token", true, "cloud")
	integrationAllowInsecureHTTP = true
	t.Cleanup(func() { integrationAllowInsecureHTTP = false })

	router := newRouter()
	got := request(t, router, http.MethodGet,
		fmt.Sprintf("/api/engineers/%d/jira-evidence?from=2026-09-01&to=2026-09-30", engineerID), nil)
	if got.Code != http.StatusOK {
		t.Fatalf("cloud harvest = %d %s", got.Code, got.Body.String())
	}
	expected := "Basic " + base64.StdEncoding.EncodeToString(
		[]byte("manager@example.com:cloud-token"))
	if capturedAuth != expected {
		t.Errorf("auth = %q, want %q", capturedAuth, expected)
	}
	if !strings.Contains(capturedPath, "/rest/api/3/search") {
		t.Errorf("path = %q", capturedPath)
	}
}
