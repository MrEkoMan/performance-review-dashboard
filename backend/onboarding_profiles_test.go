package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

func TestOnboardingProfileRoutes(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	router := newRouter()
	base := "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/onboarding-profile"

	// GET on missing profile returns 404.
	if got := request(t, router, http.MethodGet, base, nil); got.Code != http.StatusNotFound {
		t.Fatalf("missing profile GET = %d, want 404", got.Code)
	}

	// Invalid engineer id.
	if got := request(t, router, http.MethodGet, "/api/engineers/nope/onboarding-profile", nil); got.Code != http.StatusBadRequest {
		t.Fatalf("invalid engineer GET = %d, want 400", got.Code)
	}

	// Invalid request body / bad meeting date.
	for _, body := range [][]byte{
		[]byte("{"),
		[]byte(`{"meetingDate":"tomorrow"}`),
	} {
		if got := request(t, router, http.MethodPut, base, body); got.Code != http.StatusBadRequest {
			t.Fatalf("bad body PUT = %d, want 400", got.Code)
		}
	}

	// PUT creates the profile.
	payload := []byte(`{"answers":{"careerMotivation":{"enjoyMost":"Building tools."},"currentWork":{"proudOf":"The dashboard."}},"meetingDate":"2026-08-01"}`)
	got := request(t, router, http.MethodPut, base, payload)
	if got.Code != http.StatusOK {
		t.Fatalf("create PUT = %d %s", got.Code, got.Body.String())
	}
	var created OnboardingProfile
	if err := json.Unmarshal(got.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.EngineerID != engineerID {
		t.Errorf("EngineerID = %d, want %d", created.EngineerID, engineerID)
	}
	if created.Answers.CareerMotivation.EnjoyMost != "Building tools." {
		t.Errorf("EnjoyMost = %q", created.Answers.CareerMotivation.EnjoyMost)
	}
	if created.Answers.CurrentWork.ProudOf != "The dashboard." {
		t.Errorf("ProudOf = %q", created.Answers.CurrentWork.ProudOf)
	}
	if created.MeetingDate != "2026-08-01" {
		t.Errorf("MeetingDate = %q", created.MeetingDate)
	}

	// GET now returns the saved profile.
	got = request(t, router, http.MethodGet, base, nil)
	if got.Code != http.StatusOK {
		t.Fatalf("GET after create = %d", got.Code)
	}
	var fetched OnboardingProfile
	if err := json.Unmarshal(got.Body.Bytes(), &fetched); err != nil {
		t.Fatal(err)
	}
	if fetched.Answers.CareerMotivation.EnjoyMost != "Building tools." {
		t.Errorf("fetched EnjoyMost = %q", fetched.Answers.CareerMotivation.EnjoyMost)
	}

	// PUT replaces (upsert) — only one row, values updated.
	payload = []byte(`{"answers":{"careerMotivation":{"enjoyMost":"Updated answer.","careerNext2to3":"Staff"}}}`)
	got = request(t, router, http.MethodPut, base, payload)
	if got.Code != http.StatusOK {
		t.Fatalf("replace PUT = %d %s", got.Code, got.Body.String())
	}
	var replaced OnboardingProfile
	if err := json.Unmarshal(got.Body.Bytes(), &replaced); err != nil {
		t.Fatal(err)
	}
	if replaced.Answers.CareerMotivation.EnjoyMost != "Updated answer." {
		t.Errorf("replaced EnjoyMost = %q", replaced.Answers.CareerMotivation.EnjoyMost)
	}
	if replaced.Answers.CareerMotivation.CareerNext2to3 != "Staff" {
		t.Errorf("replaced CareerNext2to3 = %q", replaced.Answers.CareerMotivation.CareerNext2to3)
	}
	// Previously-set field is cleared on replace since the whole answers blob is replaced.
	if replaced.Answers.CurrentWork.ProudOf != "" {
		t.Errorf("replaced ProudOf should be cleared, got %q", replaced.Answers.CurrentWork.ProudOf)
	}

	// Only one row exists for the engineer.
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM onboarding_profiles WHERE engineer_id = ?`, engineerID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("row count = %d, want 1 (upsert should not insert duplicates)", count)
	}

	// PUT for a nonexistent engineer returns 404 (FK violation).
	if got := request(t, router, http.MethodPut, "/api/engineers/9999/onboarding-profile", payload); got.Code != http.StatusNotFound {
		t.Fatalf("missing engineer PUT = %d, want 404", got.Code)
	}
}

func TestOnboardingProfileParse(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()

	req := multipartRequest(t, http.MethodPost, "/api/onboarding-profile/parse", "notes.md",
		[]byte(sampleDocument), nil)
	got := serveRequest(router, req)
	if got.Code != http.StatusOK {
		t.Fatalf("parse = %d %s", got.Code, got.Body.String())
	}
	var answers OnboardingAnswers
	if err := json.Unmarshal(got.Body.Bytes(), &answers); err != nil {
		t.Fatal(err)
	}
	if answers.CareerMotivation.EnjoyMost != "Building internal tooling that engineers actually adopt." {
		t.Errorf("EnjoyMost = %q", answers.CareerMotivation.EnjoyMost)
	}
	if answers.OneThingToKnow != "I do my best thinking async and in writing." {
		t.Errorf("OneThingToKnow = %q", answers.OneThingToKnow)
	}

	// Missing file field is a 400.
	req = multipartRequest(t, http.MethodPost, "/api/onboarding-profile/parse", "", nil, nil)
	got = serveRequest(router, req)
	if got.Code != http.StatusBadRequest {
		t.Fatalf("missing file parse = %d, want 400", got.Code)
	}

	// Empty file does not crash; returns zero-value answers.
	req = multipartRequest(t, http.MethodPost, "/api/onboarding-profile/parse", "empty.md", []byte(""), nil)
	got = serveRequest(router, req)
	if got.Code != http.StatusOK {
		t.Fatalf("empty file parse = %d, want 200", got.Code)
	}
}
