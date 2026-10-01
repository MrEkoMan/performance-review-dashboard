package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func getNotes(w http.ResponseWriter, r *http.Request) {
	// Engineers may only list their own evidence; the manager may list any
	// engineer's. An engineer calling without an engineerId filter is
	// implicitly scoped to themselves.
	query := `
		SELECT n.id, n.engineer_id, e.name, n.note_date, n.category, n.summary,
			COALESCE(n.details, ''), COALESCE(n.impact, ''),
			COALESCE(n.follow_up_needed, 0), COALESCE(n.review_cycle, ''),
			COALESCE(n.author_role, 'manager')
		FROM performance_notes n
		JOIN engineers e ON e.id = n.engineer_id`
	args := []any{}
	user, _ := currentUser(r)
	if user.Role == roleEngineer {
		query += ` WHERE n.engineer_id = ?`
		args = append(args, user.EngineerID)
	} else if engineerID := r.URL.Query().Get("engineerId"); engineerID != "" {
		query += ` WHERE n.engineer_id = ?`
		args = append(args, engineerID)
	}
	query += ` ORDER BY n.note_date DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve notes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	notes := make([]PerformanceNote, 0)
	for rows.Next() {
		var note PerformanceNote
		if err := rows.Scan(&note.ID, &note.EngineerID, &note.EngineerName,
			&note.NoteDate, &note.Category, &note.Summary, &note.Details,
			&note.Impact, &note.FollowUpNeeded, &note.ReviewCycle,
			&note.AuthorRole); err != nil {
			http.Error(w, "Failed to read note data", http.StatusInternalServerError)
			return
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading notes", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

func createNote(w http.ResponseWriter, r *http.Request) {
	var note PerformanceNote
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if note.EngineerID <= 0 || note.NoteDate == "" || note.Category == "" || note.Summary == "" {
		http.Error(w, "Engineer, date, category, and summary are required", http.StatusBadRequest)
		return
	}

	authorRole := "manager"
	user, ok := currentUser(r)
	if ok && user.Role == roleEngineer {
		// Engineers can only contribute context to their own record, and the
		// follow-up decision stays with the manager.
		if int64(note.EngineerID) != user.EngineerID {
			http.Error(w, "You can only add context to your own record", http.StatusForbidden)
			return
		}
		authorRole = roleEngineer
		note.FollowUpNeeded = false
	}

	result, err := db.Exec(`
		INSERT INTO performance_notes
			(engineer_id, note_date, category, summary, details, impact,
			 follow_up_needed, review_cycle, author_role)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		note.EngineerID, note.NoteDate, note.Category, note.Summary,
		note.Details, note.Impact, note.FollowUpNeeded, note.ReviewCycle,
		authorRole)
	if err != nil {
		http.Error(w, "Failed to create note", http.StatusInternalServerError)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Note created but ID could not be retrieved", http.StatusInternalServerError)
		return
	}
	note.ID = int(id)
	note.AuthorRole = authorRole
	writeJSON(w, http.StatusCreated, note)
}

func updateNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	noteID, err := strconv.Atoi(id)
	if err != nil || noteID <= 0 {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}
	var note PerformanceNote
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if note.NoteDate == "" || note.Category == "" || note.Summary == "" {
		http.Error(w, "Date, category, and summary are required", http.StatusBadRequest)
		return
	}

	var storedEngineerID int64
	var storedAuthorRole string
	err = db.QueryRow(`
		SELECT engineer_id, COALESCE(author_role, 'manager')
		FROM performance_notes WHERE id = ?`, noteID).
		Scan(&storedEngineerID, &storedAuthorRole)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve note", http.StatusInternalServerError)
		return
	}
	user, ok := currentUser(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	if user.Role != roleManager {
		if int64(note.EngineerID) != user.EngineerID || storedEngineerID != user.EngineerID {
			http.Error(w, "You can only edit your own context entries", http.StatusForbidden)
			return
		}
		if storedAuthorRole != roleEngineer {
			http.Error(w, "Manager-recorded evidence is read-only for engineers", http.StatusForbidden)
			return
		}
	}

	result, err := db.Exec(`
		UPDATE performance_notes SET engineer_id = ?, note_date = ?, category = ?,
			summary = ?, details = ?, impact = ?, follow_up_needed = ?, review_cycle = ?
		WHERE id = ?`,
		note.EngineerID, note.NoteDate, note.Category, note.Summary, note.Details,
		note.Impact, note.FollowUpNeeded, note.ReviewCycle, noteID)
	if err != nil {
		http.Error(w, "Failed to update note", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm update", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}
	note.ID = noteID
	note.AuthorRole = storedAuthorRole
	writeJSON(w, http.StatusOK, note)
}

func deleteNote(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}
	var storedEngineerID int64
	var storedAuthorRole string
	err = db.QueryRow(`
		SELECT engineer_id, COALESCE(author_role, 'manager')
		FROM performance_notes WHERE id = ?`, id).
		Scan(&storedEngineerID, &storedAuthorRole)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve note", http.StatusInternalServerError)
		return
	}
	user, ok := currentUser(r)
	if ok && user.Role != roleManager {
		if storedEngineerID != user.EngineerID {
			http.Error(w, "You can only delete your own context entries", http.StatusForbidden)
			return
		}
		if storedAuthorRole != roleEngineer {
			http.Error(w, "Manager-recorded evidence is read-only for engineers", http.StatusForbidden)
			return
		}
	}
	result, err := db.Exec(`DELETE FROM performance_notes WHERE id = ?`, id)
	if err != nil {
		log.Printf("deleteNote failed: %v", err)
		http.Error(w, "Failed to delete note", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm deletion", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
