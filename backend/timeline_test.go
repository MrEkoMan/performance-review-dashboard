package main

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestTimelineRoutesAndFilters(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	insertNote(t, engineerID)
	statements := []string{
		`INSERT INTO goals (engineer_id, title, goal_type, status, priority,
			target_date, progress_percentage, review_cycle)
			VALUES (?, 'Lead design', 'leadership', 'in_progress', 'high',
			'2026-08-15', 40, '2026-H2')`,
		`INSERT INTO one_on_ones (engineer_id, meeting_date, wins, status)
			VALUES (?, '2026-07-20', 'Mentored a teammate', 'completed')`,
		`INSERT INTO follow_ups
			(engineer_id, source_type, description, owner, due_date, status, priority)
			VALUES (?, 'manual', 'Share RFC', 'Manager', '2026-08-01', 'open', 'high')`,
		`INSERT INTO recognitions
			(engineer_id, recognition_date, source, source_type, category,
			summary, details, review_cycle)
			VALUES (?, '2026-07-30', 'Peer', 'peer', 'collaboration',
			'Unblocked the team', 'Helped across boundaries', '2026-H2')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement, engineerID); err != nil {
			t.Fatal(err)
		}
	}
	router := newRouter()
	base := "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/timeline"

	if got := request(t, router, http.MethodGet, "/api/engineers/nope/timeline", nil); got.Code != 400 {
		t.Fatalf("invalid engineer timeline = %d", got.Code)
	}
	for _, target := range []string{
		base + "?eventType=unknown",
		base + "?from=07/01/2026",
		base + "?to=tomorrow",
		base + "?from=2026-08-01&to=2026-07-01",
	} {
		if got := request(t, router, http.MethodGet, target, nil); got.Code != 400 {
			t.Errorf("invalid filter %s = %d %s", target, got.Code, got.Body.String())
		}
	}

	all := request(t, router, http.MethodGet, base, nil)
	if all.Code != 200 {
		t.Fatalf("timeline = %d %s", all.Code, all.Body.String())
	}
	for _, eventType := range []string{
		`"eventType":"evidence"`, `"eventType":"goal"`,
		`"eventType":"one_on_one"`, `"eventType":"follow_up"`,
		`"eventType":"recognition"`,
	} {
		if !strings.Contains(all.Body.String(), eventType) {
			t.Errorf("timeline missing %s: %s", eventType, all.Body.String())
		}
	}
	if strings.Index(all.Body.String(), "2026-08-15") >
		strings.Index(all.Body.String(), "2026-07-20") {
		t.Errorf("timeline is not newest first: %s", all.Body.String())
	}

	tests := []struct {
		target  string
		include string
		exclude string
	}{
		{base + "?eventType=recognition", "Unblocked the team", "Lead design"},
		{base + "?reviewCycle=2026-H2", "Unblocked the team", "Mentored a teammate"},
		{base + "?from=2026-08-01", "Lead design", "Unblocked the team"},
		{base + "?to=2026-07-25", "Mentored a teammate", "Unblocked the team"},
	}
	for _, test := range tests {
		got := request(t, router, http.MethodGet, test.target, nil)
		if got.Code != 200 || !strings.Contains(got.Body.String(), test.include) ||
			strings.Contains(got.Body.String(), test.exclude) {
			t.Errorf("filter %s = %d %s", test.target, got.Code, got.Body.String())
		}
	}
}

func TestTimelineEmptyAndDatabaseError(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	router := newRouter()
	target := "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/timeline"
	if got := request(t, router, http.MethodGet, target, nil); got.Code != 200 || got.Body.String() != "[]\n" {
		t.Fatalf("empty timeline = %d %s", got.Code, got.Body.String())
	}
	db.Close()
	if got := request(t, router, http.MethodGet, target, nil); got.Code != 500 {
		t.Fatalf("database error = %d %s", got.Code, got.Body.String())
	}
}
