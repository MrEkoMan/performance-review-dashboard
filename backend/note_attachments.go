package main

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func createNoteWithAttachment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentSize+(1<<20))
	if err := r.ParseMultipartForm(maxAttachmentSize + (1 << 20)); err != nil {
		http.Error(w, "Request is invalid or exceeds the 10 MB limit", http.StatusBadRequest)
		return
	}
	input, err := parseNoteWithAttachmentInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var engineerName string
	err = db.QueryRow(`SELECT name FROM engineers WHERE id = ?`, input.EngineerID).
		Scan(&engineerName)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Engineer not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to validate engineer", http.StatusInternalServerError)
		return
	}
	if _, err := attachmentStorageRoot(); err != nil {
		http.Error(w, "Attachment storage has not been configured", http.StatusPreconditionRequired)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Screenshot file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	stored, err := storeAttachmentFile(file, header, int64(input.EngineerID), engineerName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	note, attachment, err := persistNoteWithAttachment(input, engineerName, stored)
	if err != nil {
		_ = os.Remove(stored.absolutePath)
		http.Error(w, "Failed to save note and attachment", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		Note       PerformanceNote `json:"note"`
		Attachment Attachment      `json:"attachment"`
	}{note, attachment})
}

func parseNoteWithAttachmentInput(r *http.Request) (CreateNoteWithAttachmentInput, error) {
	engineerID, err := strconv.Atoi(r.FormValue("engineerId"))
	if err != nil || engineerID <= 0 {
		return CreateNoteWithAttachmentInput{}, errors.New("valid engineer ID is required")
	}
	input := CreateNoteWithAttachmentInput{
		EngineerID: engineerID, NoteDate: strings.TrimSpace(r.FormValue("noteDate")),
		Category:       strings.TrimSpace(r.FormValue("category")),
		Summary:        strings.TrimSpace(r.FormValue("summary")),
		Details:        strings.TrimSpace(r.FormValue("details")),
		Impact:         strings.TrimSpace(r.FormValue("impact")),
		ReviewCycle:    strings.TrimSpace(r.FormValue("reviewCycle")),
		SourceSystem:   strings.TrimSpace(r.FormValue("sourceSystem")),
		SourceAuthor:   strings.TrimSpace(r.FormValue("sourceAuthor")),
		SourceDate:     strings.TrimSpace(r.FormValue("sourceDate")),
		Caption:        strings.TrimSpace(r.FormValue("caption")),
		FollowUpNeeded: strings.EqualFold(r.FormValue("followUpNeeded"), "true"),
	}
	if input.NoteDate == "" || input.Category == "" || input.Summary == "" {
		return CreateNoteWithAttachmentInput{}, errors.New("date, category, and summary are required")
	}
	return input, nil
}

func persistNoteWithAttachment(
	input CreateNoteWithAttachmentInput,
	engineerName string,
	stored *storedAttachment,
) (PerformanceNote, Attachment, error) {
	transaction, err := db.Begin()
	if err != nil {
		return PerformanceNote{}, Attachment{}, err
	}
	defer transaction.Rollback()
	result, err := transaction.Exec(`
		INSERT INTO performance_notes
			(engineer_id, note_date, category, summary, details, impact,
			 follow_up_needed, review_cycle)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		input.EngineerID, input.NoteDate, input.Category, input.Summary,
		input.Details, input.Impact, input.FollowUpNeeded, input.ReviewCycle)
	if err != nil {
		return PerformanceNote{}, Attachment{}, err
	}
	noteID, err := result.LastInsertId()
	if err != nil {
		return PerformanceNote{}, Attachment{}, err
	}
	attachmentResult, err := transaction.Exec(`
		INSERT INTO attachments
			(original_filename, stored_filename, mime_type, file_size, sha256_hash,
			 source_system, source_author, source_date, caption)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		stored.originalName, stored.relativePath, stored.mimeType, stored.size,
		stored.hash, input.SourceSystem, input.SourceAuthor, input.SourceDate, input.Caption)
	if err != nil {
		return PerformanceNote{}, Attachment{}, err
	}
	attachmentID, err := attachmentResult.LastInsertId()
	if err != nil {
		return PerformanceNote{}, Attachment{}, err
	}
	if _, err := transaction.Exec(`
		INSERT INTO performance_note_attachments (note_id, attachment_id)
		VALUES (?, ?)`, noteID, attachmentID); err != nil {
		return PerformanceNote{}, Attachment{}, err
	}
	if err := transaction.Commit(); err != nil {
		return PerformanceNote{}, Attachment{}, err
	}
	note := PerformanceNote{
		ID: int(noteID), EngineerID: input.EngineerID, EngineerName: engineerName,
		NoteDate: input.NoteDate, Category: input.Category, Summary: input.Summary,
		Details: input.Details, Impact: input.Impact,
		FollowUpNeeded: input.FollowUpNeeded, ReviewCycle: input.ReviewCycle,
	}
	attachment := Attachment{
		ID: attachmentID, OriginalFilename: stored.originalName,
		MimeType: stored.mimeType, FileSize: stored.size, SHA256Hash: stored.hash,
		SourceSystem: input.SourceSystem, SourceAuthor: input.SourceAuthor,
		SourceDate: input.SourceDate, Caption: input.Caption,
		ContentURL: "/api/attachments/" + strconv.FormatInt(attachmentID, 10) + "/content",
	}
	return note, attachment, nil
}
