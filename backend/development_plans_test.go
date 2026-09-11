package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

func TestDevelopmentPlanRoutes(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	router := newRouter()
	base := "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/development-plans"

	// List on missing plans returns an empty array, not 404.
	got := request(t, router, http.MethodGet, base, nil)
	if got.Code != http.StatusOK {
		t.Fatalf("list GET = %d", got.Code)
	}
	var list []DevelopmentPlan
	if err := json.Unmarshal(got.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("list len = %d, want 0", len(list))
	}

	// Invalid engineer id on create.
	if got := request(t, router, http.MethodPost, "/api/engineers/nope/development-plans", []byte(`{}`)); got.Code != http.StatusBadRequest {
		t.Fatalf("invalid engineer POST = %d, want 400", got.Code)
	}

	// Bad request body.
	if got := request(t, router, http.MethodPost, base, []byte("{")); got.Code != http.StatusBadRequest {
		t.Fatalf("bad body POST = %d, want 400", got.Code)
	}

	// Bad plan date.
	if got := request(t, router, http.MethodPost, base, []byte(`{"planDate":"tomorrow"}`)); got.Code != http.StatusBadRequest {
		t.Fatalf("bad date POST = %d, want 400", got.Code)
	}

	// Create a plan that persists raw markdown + parsed fields, with a default
	// title when none is supplied.
	payload := []byte(`{
		"planDate":"2026-09-01",
		"reviewCycle":"2026-H1",
		"rawMarkdown":"# Personal Development Plan",
		"fields":{"header":{"developer":"Ada"},"goals":[{"title":"Lead design","goal":"Lead it"}]}
	}`)
	got = request(t, router, http.MethodPost, base, payload)
	if got.Code != http.StatusCreated {
		t.Fatalf("create POST = %d %s", got.Code, got.Body.String())
	}
	var created DevelopmentPlan
	if err := json.Unmarshal(got.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.EngineerID != engineerID {
		t.Errorf("EngineerID = %d, want %d", created.EngineerID, engineerID)
	}
	if created.Title != "Personal Development Plan" {
		t.Errorf("Title = %q, want default", created.Title)
	}
	if created.RawMarkdown != "# Personal Development Plan" {
		t.Errorf("RawMarkdown = %q", created.RawMarkdown)
	}
	if created.Fields.Header.Developer != "Ada" {
		t.Errorf("Fields.Header.Developer = %q", created.Fields.Header.Developer)
	}
	if len(created.Fields.Goals) != 1 || created.Fields.Goals[0].Title != "Lead design" {
		t.Errorf("Fields.Goals = %+v", created.Fields.Goals)
	}
	if created.LinkedGoalID != nil {
		t.Errorf("LinkedGoalID = %v, want nil", *created.LinkedGoalID)
	}
	planID := created.ID

	// GET by id returns the saved plan.
	got = request(t, router, http.MethodGet, "/api/development-plans/"+strconv.FormatInt(planID, 10), nil)
	if got.Code != http.StatusOK {
		t.Fatalf("get GET = %d", got.Code)
	}

	// Update the plan: link it to a goal, change fields, and confirm the
	// engineer association is preserved (not moved to another engineer).
	goalID := insertGoal(t, engineerID)
	updatePayload := []byte(`{
		"title":"Updated plan",
		"planDate":"2026-09-02",
		"rawMarkdown":"# Updated",
		"fields":{"header":{"developer":"Grace"}},
		"linkedGoalId":` + strconv.FormatInt(goalID, 10) + `
	}`)
	got = request(t, router, http.MethodPut, "/api/development-plans/"+strconv.FormatInt(planID, 10), updatePayload)
	if got.Code != http.StatusOK {
		t.Fatalf("update PUT = %d %s", got.Code, got.Body.String())
	}
	var updated DevelopmentPlan
	if err := json.Unmarshal(got.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Updated plan" {
		t.Errorf("updated Title = %q", updated.Title)
	}
	if updated.Fields.Header.Developer != "Grace" {
		t.Errorf("updated Developer = %q", updated.Fields.Header.Developer)
	}
	if updated.EngineerID != engineerID {
		t.Errorf("updated EngineerID = %d, want %d (association preserved)", updated.EngineerID, engineerID)
	}
	if updated.LinkedGoalID == nil || *updated.LinkedGoalID != goalID {
		t.Errorf("updated LinkedGoalID = %v, want %d", updated.LinkedGoalID, goalID)
	}

	// List now returns one plan.
	got = request(t, router, http.MethodGet, base, nil)
	if err := json.Unmarshal(got.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("list len = %d, want 1", len(list))
	}

	// Delete the plan.
	if got := request(t, router, http.MethodDelete, "/api/development-plans/"+strconv.FormatInt(planID, 10), nil); got.Code != http.StatusNoContent {
		t.Fatalf("delete DELETE = %d, want 204", got.Code)
	}
	// Deleting again is 404.
	if got := request(t, router, http.MethodDelete, "/api/development-plans/"+strconv.FormatInt(planID, 10), nil); got.Code != http.StatusNotFound {
		t.Fatalf("re-delete DELETE = %d, want 404", got.Code)
	}

	// Create for a nonexistent engineer returns 404 (FK violation).
	if got := request(t, router, http.MethodPost, "/api/engineers/9999/development-plans", payload); got.Code != http.StatusNotFound {
		t.Fatalf("missing engineer POST = %d, want 404", got.Code)
	}
}

func TestDevelopmentPlanParse(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()

	req := multipartRequest(t, http.MethodPost, "/api/development-plans/parse", "plan.md",
		[]byte(filledPlan), nil)
	got := serveRequest(router, req)
	if got.Code != http.StatusOK {
		t.Fatalf("parse = %d %s", got.Code, got.Body.String())
	}
	var fields DevelopmentPlanFields
	if err := json.Unmarshal(got.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if fields.Header.Developer != "Ada Lovelace" {
		t.Errorf("Developer = %q", fields.Header.Developer)
	}
	if len(fields.Goals) != 2 {
		t.Errorf("Goals len = %d, want 2", len(fields.Goals))
	}
	if len(fields.Accomplishments) != 1 {
		t.Errorf("Accomplishments len = %d, want 1", len(fields.Accomplishments))
	}

	// Missing file field is a 400.
	req = multipartRequest(t, http.MethodPost, "/api/development-plans/parse", "", nil, nil)
	got = serveRequest(router, req)
	if got.Code != http.StatusBadRequest {
		t.Fatalf("missing file parse = %d, want 400", got.Code)
	}

	// Empty file does not crash; returns zero-value fields.
	req = multipartRequest(t, http.MethodPost, "/api/development-plans/parse", "empty.md", []byte(""), nil)
	got = serveRequest(router, req)
	if got.Code != http.StatusOK {
		t.Fatalf("empty file parse = %d, want 200", got.Code)
	}
}

// insertGoal creates a goal row for an engineer and returns its id, used to
// test the linked_goal_id foreign key on development plans.
func insertGoal(t *testing.T, engineerID int64) int64 {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO goals (engineer_id, title, goal_type, status, priority, progress_percentage)
		VALUES (?, 'Lead design', 'career_development', 'in_progress', 'high', 0)`,
		engineerID)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	return id
}
