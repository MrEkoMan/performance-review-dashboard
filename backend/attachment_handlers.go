package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func uploadNoteAttachment(w http.ResponseWriter, r *http.Request) {
	noteID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || noteID <= 0 {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}
	var engineerID int64
	var engineerName string
	err = db.QueryRow(`
		SELECT e.id, e.name FROM performance_notes n
		JOIN engineers e ON e.id = n.engineer_id WHERE n.id = ?`, noteID).
		Scan(&engineerID, &engineerName)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Performance note or engineer not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to validate attachment owner", http.StatusInternalServerError)
		return
	}
	if _, err := attachmentStorageRoot(); err != nil {
		http.Error(w, "Attachment storage has not been configured", http.StatusPreconditionRequired)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentSize+(1<<20))
	if err := r.ParseMultipartForm(maxAttachmentSize + (1 << 20)); err != nil {
		http.Error(w, "Attachment is invalid or exceeds the 10 MB limit", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Screenshot file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	stored, err := storeAttachmentFile(file, header, engineerID, engineerName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	attachment, err := persistAttachment(parentNote, noteID, stored,
		strings.TrimSpace(r.FormValue("sourceSystem")),
		strings.TrimSpace(r.FormValue("sourceAuthor")),
		strings.TrimSpace(r.FormValue("sourceDate")),
		strings.TrimSpace(r.FormValue("caption")))
	if err != nil {
		_ = os.Remove(stored.absolutePath)
		http.Error(w, "Failed to save attachment metadata", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, attachment)
}

// attachmentParent identifies which owning entity an attachment is linked to.
type attachmentParent string

const (
	parentNote        attachmentParent = "note"
	parentRecognition attachmentParent = "recognition"
)

// persistAttachment inserts the attachment row and links it to its owning parent
// (note or recognition) in a single transaction. The parent junction table is
// chosen by parentKind; the attachments table itself is parent-agnostic.
func persistAttachment(
	parentKind attachmentParent,
	parentID int64,
	stored *storedAttachment,
	sourceSystem, sourceAuthor, sourceDate, caption string,
) (Attachment, error) {
	transaction, err := db.Begin()
	if err != nil {
		return Attachment{}, err
	}
	defer transaction.Rollback()
	result, err := transaction.Exec(`
		INSERT INTO attachments
			(original_filename, stored_filename, mime_type, file_size, sha256_hash,
			 source_system, source_author, source_date, caption)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		stored.originalName, stored.relativePath, stored.mimeType, stored.size,
		stored.hash, sourceSystem, sourceAuthor, sourceDate, caption)
	if err != nil {
		return Attachment{}, err
	}
	attachmentID, err := result.LastInsertId()
	if err != nil {
		return Attachment{}, err
	}
	junction, linkColumn := junctionTable(parentKind)
	if _, err := transaction.Exec(
		`INSERT INTO `+junction+` (`+linkColumn+`, attachment_id) VALUES (?, ?)`,
		parentID, attachmentID); err != nil {
		return Attachment{}, err
	}
	if err := transaction.Commit(); err != nil {
		return Attachment{}, err
	}
	return Attachment{
		ID: attachmentID, OriginalFilename: stored.originalName,
		MimeType: stored.mimeType, FileSize: stored.size, SHA256Hash: stored.hash,
		SourceSystem: sourceSystem, SourceAuthor: sourceAuthor,
		SourceDate: sourceDate, Caption: caption,
		ContentURL: fmt.Sprintf("/api/attachments/%d/content", attachmentID),
	}, nil
}

// junctionTable returns the link table and its parent-id column for a given parent kind.
func junctionTable(parentKind attachmentParent) (table, linkColumn string) {
	switch parentKind {
	case parentRecognition:
		return "recognition_attachments", "recognition_id"
	default:
		return "performance_note_attachments", "note_id"
	}
}

func getNoteAttachments(w http.ResponseWriter, r *http.Request) {
	noteID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || noteID <= 0 {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}
	rows, err := db.Query(`
		SELECT a.id, a.original_filename, a.mime_type, a.file_size, a.sha256_hash,
			COALESCE(a.source_system, ''), COALESCE(a.source_author, ''),
			COALESCE(a.source_date, ''), COALESCE(a.caption, ''), a.created_at
		FROM attachments a
		JOIN performance_note_attachments pna ON pna.attachment_id = a.id
		WHERE pna.note_id = ?
		ORDER BY COALESCE(a.source_date, a.created_at) DESC, a.created_at DESC, a.id DESC`,
		noteID)
	if err != nil {
		http.Error(w, "Failed to retrieve attachments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	attachments := make([]Attachment, 0)
	for rows.Next() {
		var attachment Attachment
		if err := rows.Scan(
			&attachment.ID, &attachment.OriginalFilename, &attachment.MimeType,
			&attachment.FileSize, &attachment.SHA256Hash, &attachment.SourceSystem,
			&attachment.SourceAuthor, &attachment.SourceDate, &attachment.Caption,
			&attachment.CreatedAt); err != nil {
			http.Error(w, "Failed to read attachment data", http.StatusInternalServerError)
			return
		}
		attachment.ContentURL = fmt.Sprintf("/api/attachments/%d/content", attachment.ID)
		attachments = append(attachments, attachment)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading attachments", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, attachments)
}

func getAttachmentContent(w http.ResponseWriter, r *http.Request) {
	attachmentID, err := parseAttachmentID(r)
	if err != nil || attachmentID <= 0 {
		http.Error(w, "Invalid attachment ID", http.StatusBadRequest)
		return
	}
	var storedPath, mimeType, originalFilename string
	err = db.QueryRow(`
		SELECT stored_filename, mime_type, original_filename
		FROM attachments WHERE id = ?`, attachmentID).
		Scan(&storedPath, &mimeType, &originalFilename)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to lookup attachment", http.StatusInternalServerError)
		return
	}
	resolvedPath, err := resolveAttachmentPath(storedPath)
	if err != nil {
		http.Error(w, "Attachment storage is not configured or invalid", http.StatusInternalServerError)
		return
	}
	if _, err := os.Stat(resolvedPath); errors.Is(err, os.ErrNotExist) {
		http.Error(w, "Attachment file not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to access attachment", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("inline; filename=%q", filepath.Base(originalFilename)))
	http.ServeFile(w, r, resolvedPath)
}

func deleteAttachment(w http.ResponseWriter, r *http.Request) {
	attachmentID, err := parseAttachmentID(r)
	if err != nil || attachmentID <= 0 {
		http.Error(w, "Invalid attachment ID", http.StatusBadRequest)
		return
	}
	var storedPath string
	err = db.QueryRow(`SELECT stored_filename FROM attachments WHERE id = ?`, attachmentID).
		Scan(&storedPath)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to lookup attachment", http.StatusInternalServerError)
		return
	}
	resolvedPath, err := resolveAttachmentPath(storedPath)
	if err != nil {
		http.Error(w, "Attachment storage is not configured or invalid", http.StatusInternalServerError)
		return
	}
	result, err := db.Exec(`DELETE FROM attachments WHERE id = ?`, attachmentID)
	if err != nil {
		http.Error(w, "Failed to delete attachment", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}
	if err := os.Remove(resolvedPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("attachment metadata deleted but file removal failed: %v", err)
	}
	w.WriteHeader(http.StatusNoContent)
}
