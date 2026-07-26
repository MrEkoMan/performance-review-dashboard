package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestRecognitionRoutes(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	router := newRouter()
	base := "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/recognitions"

	if got := request(t, router, http.MethodGet, "/api/engineers/nope/recognitions", nil); got.Code != 400 {
		t.Fatalf("invalid engineer recognitions = %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, base+"?category=unknown", nil); got.Code != 400 {
		t.Fatalf("invalid category filter = %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, base, nil); got.Code != 200 || got.Body.String() != "[]\n" {
		t.Fatalf("empty recognitions = %d %s", got.Code, got.Body.String())
	}

	invalidBodies := [][]byte{
		[]byte("{"),
		[]byte(`{"source":"Peer","sourceType":"peer","category":"collaboration","summary":"Helped"}`),
		[]byte(`{"recognitionDate":"07/25/2026","source":"Peer","sourceType":"peer","category":"collaboration","summary":"Helped"}`),
		[]byte(`{"recognitionDate":"2026-07-25","sourceType":"peer","category":"collaboration","summary":"Helped"}`),
		[]byte(`{"recognitionDate":"2026-07-25","source":"Peer","sourceType":"bad","category":"collaboration","summary":"Helped"}`),
		[]byte(`{"recognitionDate":"2026-07-25","source":"Peer","sourceType":"peer","category":"bad","summary":"Helped"}`),
		[]byte(`{"recognitionDate":"2026-07-25","source":"Peer","sourceType":"peer","category":"collaboration"}`),
	}
	for index, body := range invalidBodies {
		if got := request(t, router, http.MethodPost, base, body); got.Code != 400 {
			t.Errorf("invalid recognition %d = %d %s", index, got.Code, got.Body.String())
		}
	}

	item := Recognition{
		RecognitionDate: "2026-07-25", Source: "Platform team",
		SourceType: "peer", Category: "technical_excellence",
		Summary: "Led the architecture review",
		Details: "Made a complex migration safer", RelatedWork: "Storage migration",
		ReviewCycle: "2026-H2",
	}
	body, _ := json.Marshal(item)
	if got := request(t, router, http.MethodPost, "/api/engineers/999/recognitions", body); got.Code != 404 {
		t.Fatalf("missing engineer = %d %s", got.Code, got.Body.String())
	}
	created := request(t, router, http.MethodPost, base, body)
	if created.Code != 201 {
		t.Fatalf("create recognition = %d %s", created.Code, created.Body.String())
	}
	var saved Recognition
	if err := json.Unmarshal(created.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.ID <= 0 || saved.EngineerID != engineerID ||
		saved.Category != "technical_excellence" {
		t.Fatalf("saved recognition = %#v", saved)
	}
	id := strconv.FormatInt(saved.ID, 10)
	for _, target := range []string{
		base,
		base + "?category=technical_excellence",
		base + "?reviewCycle=2026-H2",
		"/api/recognitions/" + id,
	} {
		if got := request(t, router, http.MethodGet, target, nil); got.Code != 200 ||
			!strings.Contains(got.Body.String(), "architecture review") {
			t.Errorf("get %s = %d %s", target, got.Code, got.Body.String())
		}
	}
	if got := request(t, router, http.MethodGet, "/api/recognitions/nope", nil); got.Code != 400 {
		t.Fatalf("invalid get = %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, "/api/recognitions/999", nil); got.Code != 404 {
		t.Fatalf("missing get = %d", got.Code)
	}
	if got := request(t, router, http.MethodPut, "/api/recognitions/nope", body); got.Code != 400 {
		t.Fatalf("invalid update = %d", got.Code)
	}
	if got := request(t, router, http.MethodPut, "/api/recognitions/999", body); got.Code != 404 {
		t.Fatalf("missing update = %d", got.Code)
	}
	item.Summary = "Led two architecture reviews"
	updatedBody, _ := json.Marshal(item)
	if got := request(t, router, http.MethodPut, "/api/recognitions/"+id, updatedBody); got.Code != 200 ||
		!strings.Contains(got.Body.String(), "two architecture reviews") {
		t.Fatalf("update = %d %s", got.Code, got.Body.String())
	}
	if got := request(t, router, http.MethodDelete, "/api/recognitions/nope", nil); got.Code != 400 {
		t.Fatalf("invalid delete = %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/recognitions/999", nil); got.Code != 404 {
		t.Fatalf("missing delete = %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/recognitions/"+id, nil); got.Code != 204 {
		t.Fatalf("delete = %d %s", got.Code, got.Body.String())
	}
}

func TestRecognitionDatabaseErrors(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	item := Recognition{
		RecognitionDate: "2026-07-25", Source: "Peer", SourceType: "peer",
		Category: "collaboration", Summary: "Helped",
	}
	body, _ := json.Marshal(item)
	result, err := db.Exec(`
		INSERT INTO recognitions
			(engineer_id, recognition_date, source, source_type, category, summary)
		VALUES (?, '2026-07-25', 'Peer', 'peer', 'collaboration', 'Helped')`,
		engineerID)
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
		{http.MethodGet, "/api/engineers/1/recognitions", nil},
		{http.MethodPost, "/api/engineers/1/recognitions", body},
		{http.MethodGet, "/api/recognitions/" + strconv.FormatInt(id, 10), nil},
		{http.MethodPut, "/api/recognitions/" + strconv.FormatInt(id, 10), body},
		{http.MethodDelete, "/api/recognitions/" + strconv.FormatInt(id, 10), nil},
	}
	for _, test := range cases {
		if got := request(t, router, test.method, test.target, test.body); got.Code != 500 {
			t.Errorf("%s %s = %d, want 500 (%s)", test.method, test.target, got.Code, got.Body.String())
		}
	}
}
