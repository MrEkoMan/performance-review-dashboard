package main

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func addRecencyEngineer(t *testing.T, name, team, reviewCycle string) int64 {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO engineers (name, role, level, team, career_goal, review_cycle)
		VALUES (?, 'Engineer', 'Senior', ?, 'Staff', ?)`,
		name, team, reviewCycle,
	)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	return id
}

func addRecencyNote(t *testing.T, engineerID int64, noteDate, reviewCycle string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO performance_notes
			(engineer_id, note_date, category, summary, review_cycle)
		VALUES (?, ?, 'Delivery', 'Evidence', ?)`,
		engineerID, noteDate, reviewCycle,
	); err != nil {
		t.Fatal(err)
	}
}

func fixedEvidenceDate(t *testing.T) {
	t.Helper()
	originalNow := dashboardNow
	dashboardNow = func() time.Time {
		return time.Date(2026, time.August, 10, 9, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() { dashboardNow = originalNow })
}

func TestEvidenceRecencyClassifiesBoundariesAndCountsEvidence(t *testing.T) {
	setupTestDatabase(t)
	fixedEvidenceDate(t)
	recentID := addRecencyEngineer(t, "Recent", "Platform", "2026-H2")
	agingID := addRecencyEngineer(t, "Aging", "Platform", "2026-H2")
	staleID := addRecencyEngineer(t, "Stale", "Product", "2026-H2")
	criticalID := addRecencyEngineer(t, "Critical", "Product", "2026-H1")
	addRecencyEngineer(t, "Never", "Product", "2026-H2")

	addRecencyNote(t, recentID, "2026-08-10", "2026-H2")
	addRecencyNote(t, recentID, "2026-07-20", "2026-H1")
	addRecencyNote(t, recentID, "2026-05-01", "2026-H2")
	addRecencyNote(t, agingID, "2026-07-10", "2026-H2")
	addRecencyNote(t, staleID, "2026-06-10", "2026-H2")
	addRecencyNote(t, criticalID, "2026-05-10", "2026-H1")

	response := request(t, newRouter(), http.MethodGet, "/api/dashboard/evidence-recency", nil)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{
		`"engineerName":"Never"`, `"daysSinceEvidence":-1`, `"recency":"never"`,
		`"engineerName":"Critical"`, `"daysSinceEvidence":92`, `"recency":"critical"`,
		`"engineerName":"Stale"`, `"daysSinceEvidence":61`, `"recency":"stale"`,
		`"engineerName":"Aging"`, `"daysSinceEvidence":31`, `"recency":"aging"`,
		`"engineerName":"Recent"`, `"daysSinceEvidence":0`, `"recency":"recent"`,
		`"totalEvidence":3`, `"evidenceLast30Days":2`, `"currentCycleEvidence":2`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %s in %s", expected, body)
		}
	}
	if strings.Index(body, "Never") > strings.Index(body, "Critical") ||
		strings.Index(body, "Critical") > strings.Index(body, "Stale") {
		t.Fatalf("expected least-covered engineers first: %s", body)
	}
}

func TestEvidenceRecencySupportsPortfolioFilters(t *testing.T) {
	setupTestDatabase(t)
	fixedEvidenceDate(t)
	adaID := addRecencyEngineer(t, "Ada", "Platform", "2026-H2")
	jordanID := addRecencyEngineer(t, "Jordan", "Product", "2027-H1")
	addRecencyNote(t, adaID, "2026-08-01", "2026-H2")
	addRecencyNote(t, jordanID, "2026-06-01", "2027-H1")

	tests := []struct {
		name, query, includes, excludes string
	}{
		{"recency", "?recency=stale", "Jordan", "Ada"},
		{"engineer", "?engineerId=" + strconv.FormatInt(adaID, 10), "Ada", "Jordan"},
		{"team", "?team=Product", "Jordan", "Ada"},
		{"review cycle", "?reviewCycle=2026-H2", "Ada", "Jordan"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := request(t, newRouter(), http.MethodGet, "/api/dashboard/evidence-recency"+test.query, nil)
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

func TestEvidenceRecencyRejectsInvalidFilters(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	for _, path := range []string{
		"/api/dashboard/evidence-recency?recency=unknown",
		"/api/dashboard/evidence-recency?engineerId=nope",
	} {
		response := request(t, router, http.MethodGet, path, nil)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d: %s", path, response.Code, response.Body.String())
		}
	}
}

func TestEvidenceRecencyHandlesEmptyInvalidDateAndDatabaseFailure(t *testing.T) {
	setupTestDatabase(t)
	fixedEvidenceDate(t)
	router := newRouter()
	response := request(t, router, http.MethodGet, "/api/dashboard/evidence-recency", nil)
	if response.Code != http.StatusOK || response.Body.String() != "[]\n" {
		t.Fatalf("expected empty array, got %d %q", response.Code, response.Body.String())
	}

	engineerID := addRecencyEngineer(t, "Bad Date", "Platform", "2026-H2")
	addRecencyNote(t, engineerID, "not-a-date", "2026-H2")
	response = request(t, router, http.MethodGet, "/api/dashboard/evidence-recency", nil)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected invalid date 500, got %d: %s", response.Code, response.Body.String())
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	response = request(t, router, http.MethodGet, "/api/dashboard/evidence-recency", nil)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected database 500, got %d: %s", response.Code, response.Body.String())
	}
}
