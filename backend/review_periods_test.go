package main

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func clearReviewPeriodSeeds(t *testing.T) {
	t.Helper()
	if _, err := db.Exec(`
		DELETE FROM review_periods;
		DELETE FROM sqlite_sequence WHERE name = 'review_periods'`); err != nil {
		t.Fatal(err)
	}
}

func TestReviewPeriodRoutesAndAssignmentProtection(t *testing.T) {
	setupTestDatabase(t)
	clearReviewPeriodSeeds(t)
	originalNow := dashboardNow
	dashboardNow = func() time.Time {
		return time.Date(2026, time.August, 10, 9, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() { dashboardNow = originalNow })
	router := newRouter()

	if got := request(t, router, http.MethodGet, "/api/review-periods", nil); got.Code != 200 ||
		got.Body.String() != "[]\n" {
		t.Fatalf("empty periods = %d %s", got.Code, got.Body.String())
	}
	body := []byte(`{"label":"2026 H2","startDate":"2026-07-01","endDate":"2026-12-31"}`)
	created := request(t, router, http.MethodPost, "/api/review-periods", body)
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), `"phase":"active"`) {
		t.Fatalf("create period = %d %s", created.Code, created.Body.String())
	}
	if duplicate := request(t, router, http.MethodPost, "/api/review-periods", body); duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate period = %d %s", duplicate.Code, duplicate.Body.String())
	}

	update := []byte(`{"label":"2026 H2","startDate":"2026-07-01","endDate":"2027-01-15"}`)
	if got := request(t, router, http.MethodPut, "/api/review-periods/1", update); got.Code != 200 ||
		!strings.Contains(got.Body.String(), "2027-01-15") {
		t.Fatalf("update period = %d %s", got.Code, got.Body.String())
	}
	if got := request(t, router, http.MethodGet, "/api/review-periods", nil); got.Code != 200 ||
		!strings.Contains(got.Body.String(), "2026 H2") {
		t.Fatalf("list periods = %d %s", got.Code, got.Body.String())
	}

	if _, err := db.Exec(`
		INSERT INTO engineers (name, role, level, team, career_goal, review_cycle)
		VALUES ('Ada', 'Engineer', 'Senior', 'Platform', 'Staff', '2026 H2')`); err != nil {
		t.Fatal(err)
	}
	renamed := []byte(`{"label":"Renamed","startDate":"2026-07-01","endDate":"2027-01-15"}`)
	if got := request(t, router, http.MethodPut, "/api/review-periods/1", renamed); got.Code != http.StatusConflict {
		t.Fatalf("assigned rename = %d %s", got.Code, got.Body.String())
	}
	if got := request(t, router, http.MethodDelete, "/api/review-periods/1", nil); got.Code != http.StatusNoContent {
		t.Fatalf("assigned delete = %d %s", got.Code, got.Body.String())
	}

	unassigned := []byte(`{"label":"2027 H1","startDate":"2027-01-01","endDate":"2027-06-30"}`)
	if got := request(t, router, http.MethodPost, "/api/review-periods", unassigned); got.Code != 201 {
		t.Fatalf("second create = %d %s", got.Code, got.Body.String())
	}
	if got := request(t, router, http.MethodDelete, "/api/review-periods/2", nil); got.Code != http.StatusNoContent {
		t.Fatalf("unassigned delete = %d %s", got.Code, got.Body.String())
	}
}

func TestDefaultReviewPeriodsSeedOnlyOnce(t *testing.T) {
	setupTestDatabase(t)
	year := time.Now().Year()
	label := strconv.Itoa(year) + " H2"
	var count int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM review_periods WHERE label = ?`, label).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected seeded %s period, count = %d", label, count)
	}
	if _, err := db.Exec(`DELETE FROM review_periods WHERE label = ?`, label); err != nil {
		t.Fatal(err)
	}
	if err := seedDefaultReviewPeriods(db, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM review_periods WHERE label = ?`, label).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deleted default %s was unexpectedly recreated", label)
	}
}

func TestReviewPeriodsRejectInvalidRequestsAndMissingRecords(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	for _, body := range [][]byte{
		[]byte("{"),
		[]byte(`{"label":"","startDate":"2026-01-01","endDate":"2026-06-30"}`),
		[]byte(`{"label":"Cycle","startDate":"tomorrow","endDate":"2026-06-30"}`),
		[]byte(`{"label":"Cycle","startDate":"2026-07-01","endDate":"2026-06-30"}`),
	} {
		if got := request(t, router, http.MethodPost, "/api/review-periods", body); got.Code != http.StatusBadRequest {
			t.Fatalf("invalid create = %d %s", got.Code, got.Body.String())
		}
	}
	valid := []byte(`{"label":"Cycle","startDate":"2026-01-01","endDate":"2026-06-30"}`)
	for _, test := range []struct {
		method, path string
		code         int
	}{
		{http.MethodPut, "/api/review-periods/nope", 400},
		{http.MethodDelete, "/api/review-periods/nope", 400},
		{http.MethodPut, "/api/review-periods/999", 404},
		{http.MethodDelete, "/api/review-periods/999", 404},
	} {
		if got := request(t, router, test.method, test.path, valid); got.Code != test.code {
			t.Fatalf("%s %s = %d %s", test.method, test.path, got.Code, got.Body.String())
		}
	}
}

func TestReviewPeriodsHandleDatabaseFailure(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		method, path string
		body         []byte
	}{
		{http.MethodGet, "/api/review-periods", nil},
		{http.MethodPost, "/api/review-periods", []byte(`{"label":"Cycle","startDate":"2026-01-01","endDate":"2026-06-30"}`)},
		{http.MethodPut, "/api/review-periods/1", []byte(`{"label":"Cycle","startDate":"2026-01-01","endDate":"2026-06-30"}`)},
		{http.MethodDelete, "/api/review-periods/1", nil},
	} {
		if got := request(t, router, test.method, test.path, test.body); got.Code != http.StatusInternalServerError {
			t.Fatalf("%s %s = %d %s", test.method, test.path, got.Code, got.Body.String())
		}
	}
}
