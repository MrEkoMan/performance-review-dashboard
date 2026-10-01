package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// TestEngineerAttachmentAccess covers the attachment ownership wrappers: an
// engineer may list/read attachments only on their own notes and records,
// while the manager sees everything.
func TestEngineerAttachmentAccess(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	seedManagerAccount(t, "manager@example.com", "sup3r-secret")

	adaID := insertEngineer(t)
	bobID := int64(0)
	if result, err := db.Exec(`
		INSERT INTO engineers (name, role, level, team, review_cycle)
		VALUES ('Bob', 'Engineer', 'Mid', 'Platform', '2026-H1')`); err != nil {
		t.Fatal(err)
	} else {
		bobID, _ = result.LastInsertId()
	}
	adaNoteID := insertNote(t, adaID)
	insertNote(t, bobID)

	// An attachment on Ada's note, linked through the join table.
	attachmentID := int64(0)
	if result, err := db.Exec(`
		INSERT INTO attachments (original_filename, stored_filename, mime_type,
			file_size, sha256_hash)
		VALUES ('evidence.png', 'stored-evidence.png', 'image/png', 10,
			'abc123')`); err != nil {
		t.Fatal(err)
	} else {
		attachmentID, _ = result.LastInsertId()
	}
	if _, err := db.Exec(`
		INSERT INTO performance_note_attachments (note_id, attachment_id)
		VALUES (?, ?)`, adaNoteID, attachmentID); err != nil {
		t.Fatal(err)
	}

	createEngineerUser(t, adaID, "ada@example.com", "ada-password")
	createEngineerUser(t, bobID, "bob@example.com", "bob-password")
	adaCookie := loginAs(t, router, "ada@example.com", "ada-password")
	bobCookie := loginAs(t, router, "bob@example.com", "bob-password")
	managerCookie := loginAs(t, router, "manager@example.com", "sup3r-secret")

	adaNote := strconv.FormatInt(adaNoteID, 10)
	attachment := strconv.FormatInt(attachmentID, 10)

	// The engineer who owns the note can list its attachments.
	if code := responseCode(requestAs(t, router, adaCookie, http.MethodGet,
		"/api/notes/"+adaNote+"/attachments", nil)); code != http.StatusOK {
		t.Fatalf("own note attachments = %d, want 200", code)
	}

	// Another engineer cannot enumerate them.
	if code := responseCode(requestAs(t, router, bobCookie, http.MethodGet,
		"/api/notes/"+adaNote+"/attachments", nil)); code != http.StatusForbidden {
		t.Fatalf("other note attachments = %d, want 403", code)
	}

	// The manager can.
	if code := responseCode(requestAs(t, router, managerCookie, http.MethodGet,
		"/api/notes/"+adaNote+"/attachments", nil)); code != http.StatusOK {
		t.Fatalf("manager note attachments = %d, want 200", code)
	}

	// Attachment content follows the same ownership rule through the join.
	if code := responseCode(requestAs(t, router, adaCookie, http.MethodGet,
		"/api/attachments/"+attachment+"/content", nil)); code == http.StatusForbidden {
		t.Fatalf("own attachment content = %d, want not-403 (storage may be unconfigured)", code)
	}
	if code := responseCode(requestAs(t, router, bobCookie, http.MethodGet,
		"/api/attachments/"+attachment+"/content", nil)); code != http.StatusForbidden {
		t.Fatalf("other attachment content = %d, want 403", code)
	}
}

func TestEngineerSingleOneOnOneGuard(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	seedManagerAccount(t, "manager@example.com", "sup3r-secret")

	adaID := insertEngineer(t)
	bobID := int64(0)
	if result, err := db.Exec(`
		INSERT INTO engineers (name, role, level, team, review_cycle)
		VALUES ('Bob', 'Engineer', 'Mid', 'Platform', '2026-H1')`); err != nil {
		t.Fatal(err)
	} else {
		bobID, _ = result.LastInsertId()
	}

	var adaMeetingID, bobMeetingID int64
	if result, err := db.Exec(`
		INSERT INTO one_on_ones (engineer_id, meeting_date, status,
			private_manager_notes, shared_notes)
		VALUES (?, '2026-09-01', 'completed', 'Ada private', 'Ada shared')`,
		adaID); err != nil {
		t.Fatal(err)
	} else {
		adaMeetingID, _ = result.LastInsertId()
	}
	if result, err := db.Exec(`
		INSERT INTO one_on_ones (engineer_id, meeting_date, status,
			private_manager_notes, shared_notes)
		VALUES (?, '2026-09-02', 'completed', 'Bob private', 'Bob shared')`,
		bobID); err != nil {
		t.Fatal(err)
	} else {
		bobMeetingID, _ = result.LastInsertId()
	}

	createEngineerUser(t, adaID, "ada@example.com", "ada-password")
	cookie := loginAs(t, router, "ada@example.com", "ada-password")

	// Cross-engineer single 1:1 access is rejected.
	if code := responseCode(requestAs(t, router, cookie, http.MethodGet,
		"/api/one-on-ones/"+strconv.FormatInt(bobMeetingID, 10), nil)); code != http.StatusForbidden {
		t.Fatalf("other meeting = %d, want 403", code)
	}

	// Own meeting is readable with private notes redacted.
	resp := requestAs(t, router, cookie, http.MethodGet,
		"/api/one-on-ones/"+strconv.FormatInt(adaMeetingID, 10), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("own meeting = %d", resp.StatusCode)
	}
	var meeting OneOnOne
	if err := json.NewDecoder(resp.Body).Decode(&meeting); err != nil {
		t.Fatal(err)
	}
	if meeting.PrivateManagerNotes != "" {
		t.Errorf("private notes leaked in single view: %q", meeting.PrivateManagerNotes)
	}
	if meeting.SharedNotes != "Ada shared" {
		t.Errorf("shared notes lost: %q", meeting.SharedNotes)
	}

	// A missing meeting still returns the canonical 404.
	if code := responseCode(requestAs(t, router, cookie, http.MethodGet,
		"/api/one-on-ones/9999", nil)); code != http.StatusNotFound {
		t.Fatalf("missing meeting = %d, want 404", code)
	}
}

func TestEngineerContextNoteBadgeFlow(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	seedManagerAccount(t, "manager@example.com", "sup3r-secret")
	engineerID := insertEngineer(t)
	createEngineerUser(t, engineerID, "ada@example.com", "ada-password")
	cookie := loginAs(t, router, "ada@example.com", "ada-password")

	// The engineer contributes context.
	created := requestAs(t, router, cookie, http.MethodPost, "/api/notes",
		[]byte(`{"engineerId":`+strconv.FormatInt(engineerID, 10)+
			`,"noteDate":"2026-09-21","category":"Team Contribution","summary":"Mentored the new hire on the release process"}`))
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create context note = %d %s", created.StatusCode, readBody(created))
	}

	// The manager sees the engineer-provided flag in the shared notes list.
	managerCookie := loginAs(t, router, "manager@example.com", "sup3r-secret")
	resp := requestAs(t, router, managerCookie, http.MethodGet,
		"/api/notes?engineerId="+strconv.FormatInt(engineerID, 10), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("manager notes list = %d", resp.StatusCode)
	}
	var notes []PerformanceNote
	if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("notes = %+v", notes)
	}
	if notes[0].AuthorRole != roleEngineer {
		t.Errorf("AuthorRole = %q, want engineer", notes[0].AuthorRole)
	}
	if !strings.Contains(notes[0].Summary, "Mentored") {
		t.Errorf("summary = %q", notes[0].Summary)
	}
}
