package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func getNotes(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT n.id, n.engineer_id, e.name, n.note_date, n.category, n.summary,
			COALESCE(n.details, ''), COALESCE(n.impact, ''),
			COALESCE(n.follow_up_needed, 0), COALESCE(n.review_cycle, '')
		FROM performance_notes n
		JOIN engineers e ON e.id = n.engineer_id`
	args := []any{}
	if engineerID := r.URL.Query().Get("engineerId"); engineerID != "" {
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
			&note.Impact, &note.FollowUpNeeded, &note.ReviewCycle); err != nil {
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
	result, err := db.Exec(`
		INSERT INTO performance_notes
			(engineer_id, note_date, category, summary, details, impact,
			 follow_up_needed, review_cycle)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		note.EngineerID, note.NoteDate, note.Category, note.Summary,
		note.Details, note.Impact, note.FollowUpNeeded, note.ReviewCycle)
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
	writeJSON(w, http.StatusOK, note)
}

func deleteNote(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
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
