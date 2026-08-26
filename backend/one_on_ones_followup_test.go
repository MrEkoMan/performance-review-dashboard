package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"
)

// TestOneOnOneFollowUpSchedulesNextMeeting confirms that setting a follow-up date
// on a 1:1 makes it the engineer's next scheduled 1:1 by creating a scheduled row
// for that date — but only when no future scheduled 1:1 already exists.
func TestOneOnOneFollowUpSchedulesNextMeeting(t *testing.T) {
	originalNow := dashboardNow
	dashboardNow = func() time.Time { return time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { dashboardNow = originalNow })

	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	router := newRouter()
	base := "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/one-on-ones"

	// Create a 1:1 dated today (2026-07-01) with a follow-up date in the future
	// (2026-08-19). Since the engineer has no future scheduled 1:1, a scheduled
	// row for the follow-up date should be auto-created.
	meeting := OneOnOne{
		MeetingDate: "2026-07-01", FollowUpDate: "2026-08-19", Status: "scheduled",
	}
	body, _ := json.Marshal(meeting)
	created := request(t, router, http.MethodPost, base, body)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}

	listed := request(t, router, http.MethodGet, base+"?status=scheduled", nil)
	var scheduled []OneOnOne
	if err := json.Unmarshal(listed.Body.Bytes(), &scheduled); err != nil {
		t.Fatal(err)
	}
	if len(scheduled) != 2 {
		t.Fatalf("expected 2 scheduled rows after follow-up, got %d", len(scheduled))
	}
	foundFollowUp := false
	for _, m := range scheduled {
		if m.MeetingDate == "2026-08-19" {
			foundFollowUp = true
		}
	}
	if !foundFollowUp {
		t.Fatalf("auto-created scheduled row for follow-up date not found: %#v", scheduled)
	}

	// Now set a later follow-up date on the same meeting. A future scheduled 1:1
	// (08-19) already exists, so NO new row should be created — the existing
	// arrangement is left untouched even though the new follow-up is later.
	var saved OneOnOne
	json.Unmarshal(created.Body.Bytes(), &saved)
	saved.FollowUpDate = "2026-09-01"
	updateBody, _ := json.Marshal(saved)
	updated := request(t, router, http.MethodPut, "/api/one-on-ones/"+strconv.FormatInt(saved.ID, 10), updateBody)
	if updated.Code != http.StatusOK {
		t.Fatalf("update = %d %s", updated.Code, updated.Body.String())
	}
	listed = request(t, router, http.MethodGet, base+"?status=scheduled", nil)
	json.Unmarshal(listed.Body.Bytes(), &scheduled)
	if len(scheduled) != 2 {
		t.Fatalf("expected still 2 scheduled rows (no overwrite of existing next 1:1), got %d", len(scheduled))
	}

	// A 1:1 with no follow-up date creates no extra scheduled row.
	noFollowUp := OneOnOne{MeetingDate: "2026-07-02", Status: "scheduled"}
	body, _ = json.Marshal(noFollowUp)
	if got := request(t, router, http.MethodPost, base, body); got.Code != http.StatusCreated {
		t.Fatalf("create no-followup = %d %s", got.Code, got.Body.String())
	}
	listed = request(t, router, http.MethodGet, base+"?status=scheduled", nil)
	json.Unmarshal(listed.Body.Bytes(), &scheduled)
	if len(scheduled) != 3 {
		t.Fatalf("expected 3 scheduled rows (one per meeting, no auto-create without follow-up), got %d", len(scheduled))
	}
}
