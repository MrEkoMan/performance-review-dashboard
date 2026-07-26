package main

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func addReadinessPeriod(t *testing.T, label, startDate, endDate string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO review_periods (label, start_date, end_date) VALUES (?, ?, ?)`,
		label, startDate, endDate,
	); err != nil {
		t.Fatal(err)
	}
}

func TestReviewReadinessBuildsChecklistAndCounts(t *testing.T) {
	setupTestDatabase(t)
	clearReviewPeriodSeeds(t)
	fixedEvidenceDate(t)
	addReadinessPeriod(t, "2026 H2", "2026-07-01", "2026-08-20")
	readyID := addRecencyEngineer(t, "Ready", "Platform", "2026 H2")
	needsID := addRecencyEngineer(t, "Needs Work", "Product", "2026 H2")
	addRecencyEngineer(t, "No Period", "Product", "Unknown")

	for _, note := range []struct {
		category string
	}{
		{"Delivery"}, {"Leadership"}, {"Delivery"},
	} {
		if _, err := db.Exec(`
			INSERT INTO performance_notes
				(engineer_id, note_date, category, summary, review_cycle)
			VALUES (?, '2026-08-01', ?, 'Evidence', '2026 H2')`,
			readyID, note.category,
		); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`
		INSERT INTO goals
			(engineer_id, title, goal_type, status, priority, progress_percentage, review_cycle)
		VALUES (?, 'Growth', 'leadership', 'in_progress', 'high', 50, '2026 H2');
		INSERT INTO recognitions
			(engineer_id, recognition_date, source, source_type, category, summary, review_cycle)
		VALUES (?, '2026-08-01', 'Manager', 'manager', 'leadership', 'Led well', '2026 H2');
		INSERT INTO one_on_ones (engineer_id, meeting_date, status)
		VALUES (?, '2026-07-15', 'completed')`,
		readyID, readyID, readyID,
	); err != nil {
		t.Fatal(err)
	}
	addRecencyNote(t, needsID, "2026-08-01", "2026 H2")
	if _, err := db.Exec(`
		INSERT INTO follow_ups
			(engineer_id, source_type, description, owner, due_date, status, priority)
		VALUES (?, 'manual', 'Late action', 'Manager', '2026-08-01', 'open', 'high')`,
		needsID,
	); err != nil {
		t.Fatal(err)
	}

	response := request(t, newRouter(), http.MethodGet, "/api/dashboard/review-readiness", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{
		`"engineerName":"Ready"`, `"daysUntilEnd":10`, `"readiness":"ready"`,
		`"evidenceCount":3`, `"evidenceCategoryCount":2`, `"goalCount":1`,
		`"recognitionCount":1`, `"completedOneOnOnes":1`,
		`"engineerName":"Needs Work"`, `"readiness":"needs_attention"`,
		`"overdueFollowUps":1`, `"Resolve overdue follow-ups"`,
		`"engineerName":"No Period"`, `"readiness":"unconfigured"`,
		`"Configure review period dates"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %s in %s", expected, body)
		}
	}
	if strings.Index(body, "No Period") > strings.Index(body, "Needs Work") ||
		strings.Index(body, "Needs Work") > strings.Index(body, "Ready") {
		t.Fatalf("expected unconfigured and incomplete reviews first: %s", body)
	}
}

func TestReviewReadinessSupportsFilters(t *testing.T) {
	setupTestDatabase(t)
	fixedEvidenceDate(t)
	addReadinessPeriod(t, "Soon", "2026-01-01", "2026-08-20")
	addReadinessPeriod(t, "Later", "2026-01-01", "2026-12-31")
	soonID := addRecencyEngineer(t, "Soon Engineer", "Platform", "Soon")
	addRecencyEngineer(t, "Later Engineer", "Product", "Later")

	tests := []struct {
		name, query, includes, excludes string
	}{
		{"readiness", "?readiness=needs_attention", "Soon Engineer", ""},
		{"engineer", "?engineerId=" + strconv.FormatInt(soonID, 10), "Soon Engineer", "Later Engineer"},
		{"team", "?team=Product", "Later Engineer", "Soon Engineer"},
		{"cycle", "?reviewCycle=Soon", "Soon Engineer", "Later Engineer"},
		{"ending window", "?endingWithinDays=30", "Soon Engineer", "Later Engineer"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := request(t, newRouter(), http.MethodGet, "/api/dashboard/review-readiness"+test.query, nil)
			if response.Code != 200 || !strings.Contains(response.Body.String(), test.includes) ||
				(test.excludes != "" && strings.Contains(response.Body.String(), test.excludes)) {
				t.Fatalf("unexpected filtered response: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestReviewReadinessRejectsInvalidFiltersAndDates(t *testing.T) {
	setupTestDatabase(t)
	fixedEvidenceDate(t)
	router := newRouter()
	for _, path := range []string{
		"/api/dashboard/review-readiness?readiness=unknown",
		"/api/dashboard/review-readiness?endingWithinDays=0",
		"/api/dashboard/review-readiness?endingWithinDays=366",
		"/api/dashboard/review-readiness?engineerId=nope",
	} {
		if got := request(t, router, http.MethodGet, path, nil); got.Code != http.StatusBadRequest {
			t.Fatalf("%s = %d %s", path, got.Code, got.Body.String())
		}
	}
	if _, err := db.Exec(`
		INSERT INTO review_periods (label, start_date, end_date)
		VALUES ('Bad', 'not-a-date', 'zzz');
		INSERT INTO engineers (name, role, level, team, career_goal, review_cycle)
		VALUES ('Bad Period', 'Engineer', 'Senior', 'Platform', 'Staff', 'Bad')`); err != nil {
		t.Fatal(err)
	}
	if got := request(t, router, http.MethodGet, "/api/dashboard/review-readiness", nil); got.Code != http.StatusInternalServerError {
		t.Fatalf("bad period = %d %s", got.Code, got.Body.String())
	}
}

func TestReviewReadinessHandlesEmptyAndDatabaseFailure(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	if got := request(t, router, http.MethodGet, "/api/dashboard/review-readiness", nil); got.Code != 200 ||
		got.Body.String() != "[]\n" {
		t.Fatalf("empty = %d %s", got.Code, got.Body.String())
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if got := request(t, router, http.MethodGet, "/api/dashboard/review-readiness", nil); got.Code != 500 {
		t.Fatalf("database failure = %d %s", got.Code, got.Body.String())
	}
}
