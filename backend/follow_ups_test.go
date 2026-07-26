package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestFollowUpRoutes(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	noteID := insertNote(t, engineerID)
	router := newRouter()
	base := "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/follow-ups"

	if got := request(t, router, http.MethodGet, "/api/engineers/nope/follow-ups", nil); got.Code != 400 {
		t.Fatalf("invalid engineer follow-ups = %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, base+"?status=unknown", nil); got.Code != 400 {
		t.Fatalf("invalid filter = %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, base, nil); got.Code != 200 || got.Body.String() != "[]\n" {
		t.Fatalf("empty follow-ups = %d %s", got.Code, got.Body.String())
	}

	invalidBodies := [][]byte{
		[]byte("{"),
		[]byte(`{"owner":"Manager","status":"open","priority":"medium"}`),
		[]byte(`{"description":"Act","status":"open","priority":"medium"}`),
		[]byte(`{"description":"Act","owner":"Manager","status":"bad","priority":"medium"}`),
		[]byte(`{"description":"Act","owner":"Manager","status":"open","priority":"bad"}`),
		[]byte(`{"description":"Act","owner":"Manager","status":"open","priority":"medium","sourceType":"bad"}`),
		[]byte(`{"description":"Act","owner":"Manager","status":"open","priority":"medium","sourceType":"note"}`),
		[]byte(`{"description":"Act","owner":"Manager","status":"open","priority":"medium","sourceType":"manual","sourceId":1}`),
		[]byte(`{"description":"Act","owner":"Manager","status":"open","priority":"medium","dueDate":"tomorrow"}`),
		[]byte(`{"description":"Act","owner":"Manager","status":"completed","priority":"medium"}`),
		[]byte(`{"description":"Act","owner":"Manager","status":"open","priority":"medium","completionDate":"2026-08-01"}`),
	}
	for index, body := range invalidBodies {
		if got := request(t, router, http.MethodPost, base, body); got.Code != 400 {
			t.Errorf("invalid follow-up %d = %d %s", index, got.Code, got.Body.String())
		}
	}

	sourceID := noteID
	item := FollowUp{
		SourceType: "note", SourceID: &sourceID,
		Description: "Share the architecture draft", Owner: "Manager",
		DueDate: "2026-08-01", Status: "open", Priority: "high",
		Notes: "Discuss at the next 1:1",
	}
	body, _ := json.Marshal(item)
	if got := request(t, router, http.MethodPost, "/api/engineers/999/follow-ups", body); got.Code != 404 {
		t.Fatalf("missing engineer = %d %s", got.Code, got.Body.String())
	}
	badSource := item
	missingSourceID := int64(999)
	badSource.SourceID = &missingSourceID
	badSourceBody, _ := json.Marshal(badSource)
	if got := request(t, router, http.MethodPost, base, badSourceBody); got.Code != 400 {
		t.Fatalf("missing source = %d %s", got.Code, got.Body.String())
	}

	created := request(t, router, http.MethodPost, base, body)
	if created.Code != 201 {
		t.Fatalf("create follow-up = %d %s", created.Code, created.Body.String())
	}
	var saved FollowUp
	if err := json.Unmarshal(created.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.ID <= 0 || saved.EngineerID != engineerID ||
		saved.SourceID == nil || *saved.SourceID != noteID {
		t.Fatalf("saved follow-up = %#v", saved)
	}
	id := strconv.FormatInt(saved.ID, 10)
	for _, target := range []string{
		base, base + "?status=open", "/api/follow-ups/" + id,
	} {
		if got := request(t, router, http.MethodGet, target, nil); got.Code != 200 ||
			!strings.Contains(got.Body.String(), "architecture draft") {
			t.Errorf("get %s = %d %s", target, got.Code, got.Body.String())
		}
	}
	if got := request(t, router, http.MethodGet, "/api/follow-ups/nope", nil); got.Code != 400 {
		t.Fatalf("invalid get = %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, "/api/follow-ups/999", nil); got.Code != 404 {
		t.Fatalf("missing get = %d", got.Code)
	}
	if got := request(t, router, http.MethodPut, "/api/follow-ups/nope", body); got.Code != 400 {
		t.Fatalf("invalid update = %d", got.Code)
	}
	if got := request(t, router, http.MethodPut, "/api/follow-ups/999", body); got.Code != 404 {
		t.Fatalf("missing update = %d", got.Code)
	}
	item.Status = "completed"
	item.CompletionDate = "2026-07-31"
	item.Notes = "Shared"
	updatedBody, _ := json.Marshal(item)
	if got := request(t, router, http.MethodPut, "/api/follow-ups/"+id, updatedBody); got.Code != 200 ||
		!strings.Contains(got.Body.String(), `"status":"completed"`) {
		t.Fatalf("update = %d %s", got.Code, got.Body.String())
	}
	if got := request(t, router, http.MethodDelete, "/api/follow-ups/nope", nil); got.Code != 400 {
		t.Fatalf("invalid delete = %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/follow-ups/999", nil); got.Code != 404 {
		t.Fatalf("missing delete = %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/follow-ups/"+id, nil); got.Code != 204 {
		t.Fatalf("delete = %d %s", got.Code, got.Body.String())
	}
}

func TestFollowUpSourceMustBelongToEngineer(t *testing.T) {
	setupTestDatabase(t)
	firstEngineer := insertEngineer(t)
	secondEngineer := insertEngineer(t)
	noteID := insertNote(t, firstEngineer)
	sourceID := noteID
	item := FollowUp{
		SourceType: "note", SourceID: &sourceID, Description: "Cross-link",
		Owner: "Manager", Status: "open", Priority: "low",
	}
	body, _ := json.Marshal(item)
	target := "/api/engineers/" + strconv.FormatInt(secondEngineer, 10) + "/follow-ups"
	if got := request(t, newRouter(), http.MethodPost, target, body); got.Code != 400 {
		t.Fatalf("cross-engineer source = %d %s", got.Code, got.Body.String())
	}
}

func TestFollowUpDatabaseErrors(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	item := FollowUp{
		SourceType: "manual", Description: "Act", Owner: "Manager",
		Status: "open", Priority: "medium",
	}
	body, _ := json.Marshal(item)
	result, err := db.Exec(`
		INSERT INTO follow_ups
			(engineer_id, source_type, description, owner, status, priority)
		VALUES (?, 'manual', 'Existing', 'Manager', 'open', 'medium')`, engineerID)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	router := newRouter()
	db.Close()
	cases := []struct {
		method string
		target string
		body   []byte
	}{
		{http.MethodGet, "/api/engineers/1/follow-ups", nil},
		{http.MethodPost, "/api/engineers/1/follow-ups", body},
		{http.MethodGet, "/api/follow-ups/" + strconv.FormatInt(id, 10), nil},
		{http.MethodPut, "/api/follow-ups/" + strconv.FormatInt(id, 10), body},
		{http.MethodDelete, "/api/follow-ups/" + strconv.FormatInt(id, 10), nil},
	}
	for _, test := range cases {
		if got := request(t, router, test.method, test.target, test.body); got.Code != 500 {
			t.Errorf("%s %s = %d, want 500 (%s)", test.method, test.target, got.Code, got.Body.String())
		}
	}
}
