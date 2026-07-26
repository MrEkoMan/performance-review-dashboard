package main

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func addDashboardGoal(
	t *testing.T,
	engineerID int64,
	title, status, priority, startDate, targetDate string,
	progress int,
	reviewCycle string,
) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO goals
			(engineer_id, title, goal_type, status, priority, start_date,
			 target_date, progress_percentage, review_cycle)
		VALUES (?, ?, 'leadership', ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?)`,
		engineerID, title, status, priority, startDate, targetDate, progress,
		reviewCycle,
	); err != nil {
		t.Fatal(err)
	}
}

func fixedDashboardGoalDate(t *testing.T) {
	t.Helper()
	originalNow := dashboardNow
	dashboardNow = func() time.Time {
		return time.Date(2026, time.August, 10, 9, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() { dashboardNow = originalNow })
}

func TestDashboardGoalsDerivesHealthAndSortsAttentionFirst(t *testing.T) {
	setupTestDatabase(t)
	fixedDashboardGoalDate(t)
	engineerID := insertEngineer(t)
	addDashboardGoal(t, engineerID, "Blocked goal", "blocked", "medium", "2026-07-01", "2026-08-01", 50, "2026-H2")
	addDashboardGoal(t, engineerID, "Overdue goal", "in_progress", "high", "2026-07-01", "2026-08-09", 90, "2026-H2")
	addDashboardGoal(t, engineerID, "At-risk goal", "in_progress", "high", "2026-08-01", "2026-08-21", 20, "2026-H2")
	addDashboardGoal(t, engineerID, "Boundary goal", "in_progress", "medium", "2026-08-01", "2026-08-21", 25, "2026-H2")
	addDashboardGoal(t, engineerID, "Undated goal", "not_started", "low", "", "", 0, "2026-H2")
	addDashboardGoal(t, engineerID, "Completed goal", "completed", "high", "2026-07-01", "2026-08-01", 100, "2026-H2")

	response := request(t, newRouter(), http.MethodGet, "/api/dashboard/goals", nil)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{
		`"title":"Blocked goal"`, `"health":"blocked"`,
		`"title":"Overdue goal"`, `"health":"overdue"`,
		`"title":"At-risk goal"`, `"expectedProgress":45`, `"health":"at_risk"`,
		`"title":"Boundary goal"`, `"health":"on_track"`,
		`"title":"Undated goal"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %s in %s", expected, body)
		}
	}
	if strings.Contains(body, "Completed goal") {
		t.Fatalf("closed goals should be excluded by default: %s", body)
	}
	if strings.Index(body, "Blocked goal") > strings.Index(body, "Overdue goal") ||
		strings.Index(body, "Overdue goal") > strings.Index(body, "At-risk goal") {
		t.Fatalf("expected blocked, overdue, then at-risk order: %s", body)
	}
}

func TestDashboardGoalsSupportsPortfolioFilters(t *testing.T) {
	setupTestDatabase(t)
	fixedDashboardGoalDate(t)
	adaID := insertEngineer(t)
	result, err := db.Exec(`
		INSERT INTO engineers (name, role, level, team, career_goal, review_cycle)
		VALUES ('Jordan', 'Engineer', 'Senior', 'Product', 'Staff', '2027-H1')`)
	if err != nil {
		t.Fatal(err)
	}
	jordanID, _ := result.LastInsertId()
	addDashboardGoal(t, adaID, "Ada risk", "in_progress", "high", "2026-08-01", "2026-08-21", 20, "2026-H2")
	addDashboardGoal(t, jordanID, "Jordan blocked", "blocked", "low", "2026-08-01", "2026-09-01", 20, "2027-H1")
	addDashboardGoal(t, jordanID, "Jordan completed", "completed", "high", "2026-07-01", "2026-08-01", 100, "2027-H1")

	tests := []struct {
		name, query, includes, excludes string
	}{
		{"health", "?health=at_risk", "Ada risk", "Jordan blocked"},
		{"status", "?status=blocked", "Jordan blocked", "Ada risk"},
		{"priority", "?priority=low", "Jordan blocked", "Ada risk"},
		{"engineer", "?engineerId=" + strconv.FormatInt(adaID, 10), "Ada risk", "Jordan blocked"},
		{"review cycle", "?reviewCycle=2027-H1", "Jordan blocked", "Ada risk"},
		{"include closed", "?includeClosed=true&status=completed", "Jordan completed", "Ada risk"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := request(t, newRouter(), http.MethodGet, "/api/dashboard/goals"+test.query, nil)
			if response.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), test.includes) ||
				strings.Contains(response.Body.String(), test.excludes) {
				t.Fatalf("unexpected filtered response: %s", response.Body.String())
			}
		})
	}
}

func TestDashboardGoalsRejectsInvalidFilters(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	for _, path := range []string{
		"/api/dashboard/goals?includeClosed=maybe",
		"/api/dashboard/goals?health=unknown",
		"/api/dashboard/goals?status=unknown",
		"/api/dashboard/goals?priority=unknown",
		"/api/dashboard/goals?engineerId=nope",
	} {
		response := request(t, router, http.MethodGet, path, nil)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d: %s", path, response.Code, response.Body.String())
		}
	}
}

func TestDashboardGoalsHandlesEmptyInvalidDatesAndDatabaseFailure(t *testing.T) {
	setupTestDatabase(t)
	fixedDashboardGoalDate(t)
	router := newRouter()
	response := request(t, router, http.MethodGet, "/api/dashboard/goals", nil)
	if response.Code != http.StatusOK || response.Body.String() != "[]\n" {
		t.Fatalf("expected empty array, got %d %q", response.Code, response.Body.String())
	}

	engineerID := insertEngineer(t)
	addDashboardGoal(t, engineerID, "Bad date", "in_progress", "high", "bad", "2026-08-20", 20, "")
	response = request(t, router, http.MethodGet, "/api/dashboard/goals", nil)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected invalid date 500, got %d: %s", response.Code, response.Body.String())
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	response = request(t, router, http.MethodGet, "/api/dashboard/goals", nil)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected database 500, got %d: %s", response.Code, response.Body.String())
	}
}
