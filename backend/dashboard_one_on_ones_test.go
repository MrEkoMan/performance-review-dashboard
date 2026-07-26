package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestUpcomingOneOnOnesRoutesAndContext(t *testing.T) {
	setupTestDatabase(t)
	originalNow := dashboardNow
	dashboardNow = func() time.Time {
		return time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { dashboardNow = originalNow })
	engineerID := insertEngineer(t)

	statements := []string{
		`INSERT INTO one_on_ones (engineer_id, meeting_date, status)
			VALUES (?, '2026-08-01', 'completed')`,
		`INSERT INTO one_on_ones (engineer_id, meeting_date, status)
			VALUES (?, '2026-08-09', 'scheduled')`,
		`INSERT INTO one_on_ones (engineer_id, meeting_date, status)
			VALUES (?, '2026-08-20', 'scheduled')`,
		`INSERT INTO one_on_ones (engineer_id, meeting_date, status)
			VALUES (?, '2026-09-20', 'scheduled')`,
		`INSERT INTO follow_ups
			(engineer_id, source_type, description, owner, status, priority)
			VALUES (?, 'manual', 'Discuss growth', 'Manager', 'open', 'high')`,
		`INSERT INTO goals
			(engineer_id, title, goal_type, status, priority, progress_percentage)
			VALUES (?, 'Blocked goal', 'leadership', 'blocked', 'high', 10)`,
		`INSERT INTO goals
			(engineer_id, title, goal_type, status, priority, target_date,
			 progress_percentage)
			VALUES (?, 'Overdue goal', 'delivery', 'in_progress', 'high',
			 '2026-08-01', 50)`,
		`INSERT INTO performance_notes
			(engineer_id, note_date, category, summary)
			VALUES (?, '2026-08-05', 'Delivery', 'Recent evidence')`,
		`INSERT INTO recognitions
			(engineer_id, recognition_date, source, source_type, category, summary)
			VALUES (?, '2026-07-15', 'Peer', 'peer', 'collaboration', 'Helpful')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement, engineerID); err != nil {
			t.Fatal(err)
		}
	}
	router := newRouter()
	for _, target := range []string{
		"/api/dashboard/upcoming-one-on-ones?days=abc",
		"/api/dashboard/upcoming-one-on-ones?days=0",
		"/api/dashboard/upcoming-one-on-ones?days=91",
	} {
		if got := request(t, router, http.MethodGet, target, nil); got.Code != 400 {
			t.Errorf("invalid window %s = %d %s", target, got.Code, got.Body.String())
		}
	}

	defaultWindow := request(t, router, http.MethodGet,
		"/api/dashboard/upcoming-one-on-ones", nil)
	if defaultWindow.Code != 200 {
		t.Fatalf("upcoming = %d %s", defaultWindow.Code, defaultWindow.Body.String())
	}
	var meetings []UpcomingOneOnOne
	if err := json.Unmarshal(defaultWindow.Body.Bytes(), &meetings); err != nil {
		t.Fatal(err)
	}
	if len(meetings) != 2 {
		t.Fatalf("meetings = %#v", meetings)
	}
	overdue := meetings[0]
	upcoming := meetings[1]
	if overdue.DaysUntil != -1 || upcoming.DaysUntil != 10 ||
		upcoming.LastCompletedDate != "2026-08-01" ||
		upcoming.OpenFollowUps != 1 || upcoming.BlockedGoals != 1 ||
		upcoming.OverdueGoals != 1 || upcoming.RecentEvidenceCount != 1 ||
		upcoming.RecentRecognitionCount != 1 {
		t.Fatalf("meeting context: overdue=%#v upcoming=%#v", overdue, upcoming)
	}

	sevenDays := request(t, router, http.MethodGet,
		"/api/dashboard/upcoming-one-on-ones?days=7", nil)
	if sevenDays.Code != 200 || strings.Contains(sevenDays.Body.String(), "2026-08-20") ||
		!strings.Contains(sevenDays.Body.String(), "2026-08-09") {
		t.Fatalf("seven-day window = %d %s", sevenDays.Code, sevenDays.Body.String())
	}
}

func TestUpcomingOneOnOnesEmptyAndDatabaseError(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	target := "/api/dashboard/upcoming-one-on-ones"
	if got := request(t, router, http.MethodGet, target, nil); got.Code != 200 ||
		got.Body.String() != "[]\n" {
		t.Fatalf("empty upcoming = %d %s", got.Code, got.Body.String())
	}
	db.Close()
	if got := request(t, router, http.MethodGet, target, nil); got.Code != 500 {
		t.Fatalf("database error = %d %s", got.Code, got.Body.String())
	}
}
