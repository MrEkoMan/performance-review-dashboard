package main

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func addDashboardFollowUp(
	t *testing.T,
	engineerID int64,
	description, owner, dueDate, status, priority string,
) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO follow_ups
			(engineer_id, source_type, description, owner, due_date, status, priority)
		VALUES (?, 'manual', ?, ?, ?, ?, ?)`,
		engineerID, description, owner, dueDate, status, priority,
	); err != nil {
		t.Fatal(err)
	}
}

func TestDashboardFollowUpsReturnsOnlyActiveOverdueItemsByDefault(t *testing.T) {
	setupTestDatabase(t)
	originalNow := dashboardNow
	dashboardNow = func() time.Time {
		return time.Date(2026, time.August, 10, 9, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() { dashboardNow = originalNow })
	engineerID := insertEngineer(t)
	addDashboardFollowUp(t, engineerID, "Prepare promotion evidence", "Manager", "2026-08-05", "open", "high")
	addDashboardFollowUp(t, engineerID, "Resolve delivery risk", "Ada", "2026-07-01", "in_progress", "critical")
	addDashboardFollowUp(t, engineerID, "Future action", "Manager", "2026-08-20", "open", "medium")
	addDashboardFollowUp(t, engineerID, "Already completed", "Manager", "2026-08-01", "completed", "low")

	response := request(t, newRouter(), http.MethodGet, "/api/dashboard/follow-ups", nil)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `"description":"Prepare promotion evidence"`) ||
		!strings.Contains(body, `"daysOverdue":5`) {
		t.Fatalf("expected five-day overdue follow-up, got %s", body)
	}
	if !strings.Contains(body, `"description":"Resolve delivery risk"`) ||
		!strings.Contains(body, `"daysOverdue":40`) {
		t.Fatalf("expected forty-day overdue follow-up, got %s", body)
	}
	if strings.Contains(body, "Future action") || strings.Contains(body, "Already completed") {
		t.Fatalf("default response should exclude future and completed items: %s", body)
	}
	if strings.Index(body, "Resolve delivery risk") > strings.Index(body, "Prepare promotion evidence") {
		t.Fatalf("expected critical priority before high priority: %s", body)
	}
}

func TestDashboardFollowUpsSupportsPortfolioFilters(t *testing.T) {
	setupTestDatabase(t)
	originalNow := dashboardNow
	dashboardNow = func() time.Time {
		return time.Date(2026, time.August, 10, 9, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() { dashboardNow = originalNow })
	adaID := insertEngineer(t)
	result, err := db.Exec(`
		INSERT INTO engineers (name, role, level, team, career_goal, review_cycle)
		VALUES ('Jordan', 'Engineer', 'Senior', 'Product', 'Staff', '2026-H1')`)
	if err != nil {
		t.Fatal(err)
	}
	jordanID, _ := result.LastInsertId()
	addDashboardFollowUp(t, adaID, "Ada high item", "Taylor", "2026-08-01", "open", "high")
	addDashboardFollowUp(t, jordanID, "Jordan low item", "Morgan", "2026-08-01", "open", "low")
	addDashboardFollowUp(t, jordanID, "Jordan completed item", "Morgan", "2026-08-01", "completed", "high")

	tests := []struct {
		name, query, includes, excludes string
	}{
		{"priority", "?priority=high", "Ada high item", "Jordan low item"},
		{"engineer", "?engineerId=" + strconv.FormatInt(jordanID, 10), "Jordan low item", "Ada high item"},
		{"owner case insensitive", "?owner=tAyLoR", "Ada high item", "Jordan low item"},
		{"completed history", "?overdue=false&status=completed", "Jordan completed item", "Ada high item"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := request(t, newRouter(), http.MethodGet, "/api/dashboard/follow-ups"+test.query, nil)
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

func TestDashboardFollowUpsRejectsInvalidFilters(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	for _, path := range []string{
		"/api/dashboard/follow-ups?overdue=maybe",
		"/api/dashboard/follow-ups?status=unknown",
		"/api/dashboard/follow-ups?priority=unknown",
		"/api/dashboard/follow-ups?engineerId=nope",
	} {
		response := request(t, router, http.MethodGet, path, nil)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d: %s", path, response.Code, response.Body.String())
		}
	}
}

func TestDashboardFollowUpsHandlesEmptyAndFailedDatabase(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	response := request(t, router, http.MethodGet, "/api/dashboard/follow-ups", nil)
	if response.Code != http.StatusOK || response.Body.String() != "[]\n" {
		t.Fatalf("expected empty array, got %d %q", response.Code, response.Body.String())
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	response = request(t, router, http.MethodGet, "/api/dashboard/follow-ups", nil)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", response.Code, response.Body.String())
	}
}
