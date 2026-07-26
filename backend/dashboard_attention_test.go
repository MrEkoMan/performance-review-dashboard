package main

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDashboardAttentionRulesAndFilters(t *testing.T) {
	setupTestDatabase(t)
	originalNow := dashboardNow
	dashboardNow = func() time.Time {
		return time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { dashboardNow = originalNow })

	engineerID := insertEngineer(t)
	statements := []string{
		`INSERT INTO performance_notes
			(engineer_id, note_date, category, summary, follow_up_needed)
			VALUES (?, '2026-06-01', 'Delivery', 'Legacy action', TRUE)`,
		`INSERT INTO goals
			(engineer_id, title, goal_type, status, priority,
			 target_date, progress_percentage)
			VALUES (?, 'Blocked design', 'leadership', 'blocked', 'high',
			 '2026-09-01', 20)`,
		`INSERT INTO goals
			(engineer_id, title, goal_type, status, priority,
			 target_date, progress_percentage)
			VALUES (?, 'Late delivery', 'delivery', 'in_progress', 'high',
			 '2026-08-01', 50)`,
		`INSERT INTO follow_ups
			(engineer_id, source_type, description, owner, due_date, status, priority)
			VALUES (?, 'manual', 'Send feedback', 'Manager',
			 '2026-08-05', 'open', 'high')`,
		`INSERT INTO one_on_ones
			(engineer_id, meeting_date, status)
			VALUES (?, '2026-08-14', 'scheduled')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement, engineerID); err != nil {
			t.Fatal(err)
		}
	}
	router := newRouter()

	for _, target := range []string{
		"/api/dashboard/attention?type=unknown",
		"/api/dashboard/attention?severity=urgent",
	} {
		if got := request(t, router, http.MethodGet, target, nil); got.Code != 400 {
			t.Errorf("invalid filter %s = %d %s", target, got.Code, got.Body.String())
		}
	}

	all := request(t, router, http.MethodGet, "/api/dashboard/attention", nil)
	if all.Code != 200 {
		t.Fatalf("attention = %d %s", all.Code, all.Body.String())
	}
	for _, itemType := range []string{
		"overdue_follow_up", "blocked_goal", "overdue_goal",
		"stale_evidence", "upcoming_one_on_one", "legacy_follow_up",
		"stale_recognition",
	} {
		if !strings.Contains(all.Body.String(), `"itemType":"`+itemType+`"`) {
			t.Errorf("attention missing %s: %s", itemType, all.Body.String())
		}
	}
	high := request(t, router, http.MethodGet, "/api/dashboard/attention?severity=high", nil)
	if high.Code != 200 || !strings.Contains(high.Body.String(), "Send feedback") ||
		strings.Contains(high.Body.String(), "Upcoming 1:1") {
		t.Fatalf("high filter = %d %s", high.Code, high.Body.String())
	}
	upcoming := request(t, router, http.MethodGet,
		"/api/dashboard/attention?type=upcoming_one_on_one", nil)
	if upcoming.Code != 200 || !strings.Contains(upcoming.Body.String(), "2026-08-14") ||
		strings.Contains(upcoming.Body.String(), "Late delivery") {
		t.Fatalf("type filter = %d %s", upcoming.Code, upcoming.Body.String())
	}
}

func TestDashboardAttentionHealthyAndDatabaseError(t *testing.T) {
	setupTestDatabase(t)
	originalNow := dashboardNow
	dashboardNow = func() time.Time {
		return time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { dashboardNow = originalNow })
	engineerID := insertEngineer(t)
	if _, err := db.Exec(`
		INSERT INTO performance_notes
			(engineer_id, note_date, category, summary, follow_up_needed)
		VALUES (?, '2026-08-09', 'Delivery', 'Recent', FALSE)`, engineerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO recognitions
			(engineer_id, recognition_date, source, source_type, category, summary)
		VALUES (?, '2026-08-01', 'Peer', 'peer', 'collaboration', 'Great work')`,
		engineerID); err != nil {
		t.Fatal(err)
	}
	router := newRouter()
	if got := request(t, router, http.MethodGet, "/api/dashboard/attention", nil); got.Code != 200 ||
		got.Body.String() != "[]\n" {
		t.Fatalf("healthy attention = %d %s", got.Code, got.Body.String())
	}
	db.Close()
	if got := request(t, router, http.MethodGet, "/api/dashboard/attention", nil); got.Code != 500 {
		t.Fatalf("database error = %d %s", got.Code, got.Body.String())
	}
}
