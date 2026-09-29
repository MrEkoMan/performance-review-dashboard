package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// TestEngineerArchiveRoutes covers the archive/restore lifecycle: archived
// engineers drop out of the default list and dashboard queries but remain
// reachable by ID, and restore clears the departure metadata.
func TestEngineerArchiveRoutes(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	engineerID := insertEngineer(t)
	id := strconv.FormatInt(engineerID, 10)

	// Create a second engineer so filtering behavior is observable.
	if got := request(t, router, http.MethodPost, "/api/engineers",
		[]byte(`{"name":"Bob","role":"Engineer","level":"Mid","team":"Platform","reviewCycle":"2026-H1"}`)); got.Code != http.StatusCreated {
		t.Fatalf("create second engineer: %d %s", got.Code, got.Body.String())
	}

	// PUT update with archive but no reason is rejected.
	updateBody := []byte(`{"name":"Ada","role":"Engineer","level":"Senior","team":"Platform","careerGoal":"Staff","reviewCycle":"2026-H1","archived":true}`)
	if got := request(t, router, http.MethodPut, "/api/engineers/"+id, updateBody); got.Code != http.StatusBadRequest {
		t.Fatalf("archive via PUT without reason = %d, want 400", got.Code)
	}

	// Archive with an invalid reason is rejected.
	if got := request(t, router, http.MethodPost, "/api/engineers/"+id+"/archive",
		[]byte(`{"departureReason":"vacation"}`)); got.Code != http.StatusBadRequest {
		t.Fatalf("invalid departure reason = %d, want 400", got.Code)
	}

	// Archive with a bad date is rejected.
	if got := request(t, router, http.MethodPost, "/api/engineers/"+id+"/archive",
		[]byte(`{"departureReason":"resigned","departureDate":"2026-13-01"}`)); got.Code != http.StatusBadRequest {
		t.Fatalf("bad departure date = %d, want 400", got.Code)
	}

	// Archive with an invalid ID is 400.
	if got := request(t, router, http.MethodPost, "/api/engineers/nope/archive",
		[]byte(`{"departureReason":"resigned"}`)); got.Code != http.StatusBadRequest {
		t.Fatalf("invalid archive id = %d, want 400", got.Code)
	}

	// Archive for a missing engineer is 404.
	if got := request(t, router, http.MethodPost, "/api/engineers/9999/archive",
		[]byte(`{"departureReason":"resigned"}`)); got.Code != http.StatusNotFound {
		t.Fatalf("missing engineer archive = %d, want 404", got.Code)
	}

	// Successful archive defaults the departure date to today.
	got := request(t, router, http.MethodPost, "/api/engineers/"+id+"/archive",
		[]byte(`{"departureReason":"resigned","departureNotes":"Left for another role."}`))
	if got.Code != http.StatusOK {
		t.Fatalf("archive = %d %s", got.Code, got.Body.String())
	}
	var archived Engineer
	if err := json.Unmarshal(got.Body.Bytes(), &archived); err != nil {
		t.Fatal(err)
	}
	if !archived.Archived {
		t.Error("archived flag should be true")
	}
	if archived.DepartureReason != "resigned" {
		t.Errorf("DepartureReason = %q", archived.DepartureReason)
	}
	if archived.DepartureDate == "" {
		t.Error("DepartureDate should default to today when omitted")
	}
	if archived.DepartureNotes != "Left for another role." {
		t.Errorf("DepartureNotes = %q", archived.DepartureNotes)
	}

	// Default engineer list excludes the archived engineer but includes Bob.
	got = request(t, router, http.MethodGet, "/api/engineers", nil)
	var active []Engineer
	if err := json.Unmarshal(got.Body.Bytes(), &active); err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].Name != "Bob" {
		t.Errorf("active engineers = %+v, want only Bob", active)
	}

	// archived=only returns just the archived engineer.
	got = request(t, router, http.MethodGet, "/api/engineers?archived=only", nil)
	var archivedOnly []Engineer
	if err := json.Unmarshal(got.Body.Bytes(), &archivedOnly); err != nil {
		t.Fatal(err)
	}
	if len(archivedOnly) != 1 || archivedOnly[0].Name != "Ada" {
		t.Errorf("archived-only engineers = %+v, want only Ada", archivedOnly)
	}

	// archived=all returns both.
	got = request(t, router, http.MethodGet, "/api/engineers?archived=all", nil)
	var everyone []Engineer
	if err := json.Unmarshal(got.Body.Bytes(), &everyone); err != nil {
		t.Fatal(err)
	}
	if len(everyone) != 2 {
		t.Errorf("archived=all engineers len = %d, want 2", len(everyone))
	}

	// The archived engineer is still directly retrievable by ID.
	got = request(t, router, http.MethodGet, "/api/engineers/"+id, nil)
	if got.Code != http.StatusOK {
		t.Fatalf("get archived engineer by id = %d", got.Code)
	}

	// Dashboard aggregates exclude the archived engineer: their note should
	// not surface in attention or evidence recency.
	if got := request(t, router, http.MethodGet, "/api/dashboard/evidence-recency", nil); got.Code != http.StatusOK || strings.Contains(got.Body.String(), "Ada") {
		t.Fatalf("evidence recency should exclude archived engineers: %d %s", got.Code, got.Body.String())
	}

	// Restore clears the archive flag and departure metadata.
	got = request(t, router, http.MethodPost, "/api/engineers/"+id+"/restore", nil)
	if got.Code != http.StatusOK {
		t.Fatalf("restore = %d %s", got.Code, got.Body.String())
	}
	var restored Engineer
	if err := json.Unmarshal(got.Body.Bytes(), &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Archived {
		t.Error("restored engineer should not be archived")
	}
	if restored.DepartureReason != "" || restored.DepartureDate != "" || restored.DepartureNotes != "" {
		t.Errorf("restored departure metadata should be cleared, got %+v", restored)
	}

	// After restore, the default list includes the engineer again.
	got = request(t, router, http.MethodGet, "/api/engineers", nil)
	var reactivated []Engineer
	if err := json.Unmarshal(got.Body.Bytes(), &reactivated); err != nil {
		t.Fatal(err)
	}
	if len(reactivated) != 2 {
		t.Errorf("post-restore engineer count = %d, want 2", len(reactivated))
	}

	// Restore on a missing engineer is 404.
	if got := request(t, router, http.MethodPost, "/api/engineers/9999/restore", nil); got.Code != http.StatusNotFound {
		t.Fatalf("missing engineer restore = %d, want 404", got.Code)
	}
}

// TestDashboardExcludesArchivedEngineers verifies the dashboard aggregates
// (attention, upcoming 1:1s, follow-ups, goals, review readiness) skip
// archived engineers.
func TestDashboardExcludesArchivedEngineers(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	engineerID := insertEngineer(t)
	id := strconv.FormatInt(engineerID, 10)

	// Seed a note, goal, follow-up, and 1:1 for the engineer so dashboard
	// queries would surface them when active.
	if got := request(t, router, http.MethodPost, "/api/notes",
		[]byte(`{"engineerId":`+id+`,"noteDate":"2026-07-25","category":"Delivery","summary":"Shipped","details":"D","impact":"High","followUpNeeded":true,"reviewCycle":"2026-H1"}`)); got.Code != http.StatusCreated {
		t.Fatalf("create note: %d %s", got.Code, got.Body.String())
	}
	if got := request(t, router, http.MethodPost, "/api/engineers/"+id+"/goals",
		[]byte(`{"title":"Blocked goal","goalType":"delivery","status":"blocked","priority":"high","progressPercent":0}`)); got.Code != http.StatusCreated {
		t.Fatalf("create goal: %d %s", got.Code, got.Body.String())
	}
	if got := request(t, router, http.MethodPost, "/api/engineers/"+id+"/follow-ups",
		[]byte(`{"description":"Overdue action","owner":"Ada","dueDate":"2026-01-01","status":"open","priority":"high"}`)); got.Code != http.StatusCreated {
		t.Fatalf("create follow-up: %d %s", got.Code, got.Body.String())
	}

	// Before archiving, the dashboards surface the engineer's items.
	if got := request(t, router, http.MethodGet, "/api/dashboard/attention", nil); !strings.Contains(got.Body.String(), "Ada") {
		t.Fatalf("attention should include active engineer: %s", got.Body.String())
	}
	if got := request(t, router, http.MethodGet, "/api/dashboard/goals", nil); !strings.Contains(got.Body.String(), "Blocked goal") {
		t.Fatalf("dashboard goals should include active engineer's goal: %s", got.Body.String())
	}

	// Archive the engineer.
	if got := request(t, router, http.MethodPost, "/api/engineers/"+id+"/archive",
		[]byte(`{"departureReason":"terminated"}`)); got.Code != http.StatusOK {
		t.Fatalf("archive = %d %s", got.Code, got.Body.String())
	}

	// After archiving, the dashboards no longer surface the engineer.
	for _, endpoint := range []string{
		"/api/dashboard/attention",
		"/api/dashboard/upcoming-one-on-ones",
		"/api/dashboard/follow-ups",
		"/api/dashboard/goals",
		"/api/dashboard/evidence-recency",
		"/api/dashboard/review-readiness",
	} {
		if got := request(t, router, http.MethodGet, endpoint, nil); strings.Contains(got.Body.String(), "Ada") {
			t.Errorf("%s should exclude archived engineers: %s", endpoint, got.Body.String())
		}
	}
}
