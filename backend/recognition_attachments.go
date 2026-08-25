package main

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// getRecognitionAttachments lists the screenshots attached to a recognition.
func getRecognitionAttachments(w http.ResponseWriter, r *http.Request) {
	recognitionID, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid recognition ID", http.StatusBadRequest)
		return
	}
	rows, err := db.Query(`
		SELECT a.id, a.original_filename, a.mime_type, a.file_size, a.sha256_hash,
			COALESCE(a.source_system, ''), COALESCE(a.source_author, ''),
			COALESCE(a.source_date, ''), COALESCE(a.caption, ''), a.created_at
		FROM attachments a
		JOIN recognition_attachments ra ON ra.attachment_id = a.id
		WHERE ra.recognition_id = ?
		ORDER BY COALESCE(a.source_date, a.created_at) DESC, a.created_at DESC, a.id DESC`,
		recognitionID)
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
		attachment.ContentURL = "/api/attachments/" + strconv.FormatInt(attachment.ID, 10) + "/content"
		attachments = append(attachments, attachment)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading attachments", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, attachments)
}

// uploadRecognitionAttachment attaches a screenshot to an existing recognition.
func uploadRecognitionAttachment(w http.ResponseWriter, r *http.Request) {
	recognitionID, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid recognition ID", http.StatusBadRequest)
		return
	}
	var engineerID int64
	var engineerName string
	err = db.QueryRow(`
		SELECT e.id, e.name FROM recognitions r
		JOIN engineers e ON e.id = r.engineer_id WHERE r.id = ?`, recognitionID).
		Scan(&engineerID, &engineerName)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Recognition or engineer not found", http.StatusNotFound)
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
	attachment, err := persistAttachment(parentRecognition, recognitionID, stored,
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

// createRecognitionWithAttachment creates a recognition and attaches a screenshot
// to it in a single request, so a manager can log recognition evidence in one step.
func createRecognitionWithAttachment(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentSize+(1<<20))
	if err := r.ParseMultipartForm(maxAttachmentSize + (1 << 20)); err != nil {
		http.Error(w, "Request is invalid or exceeds the 10 MB limit", http.StatusBadRequest)
		return
	}
	item, err := parseRecognitionWithAttachmentInput(r, engineerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var engineerName string
	err = db.QueryRow(`SELECT name FROM engineers WHERE id = ?`, engineerID).Scan(&engineerName)
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
	stored, err := storeAttachmentFile(file, header, engineerID, engineerName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	recognition, attachment, err := persistRecognitionWithAttachment(item, engineerName, stored)
	if err != nil {
		_ = os.Remove(stored.absolutePath)
		http.Error(w, "Failed to save recognition and attachment", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		Recognition Recognition `json:"recognition"`
		Attachment  Attachment  `json:"attachment"`
	}{recognition, attachment})
}

// parseRecognitionWithAttachmentInput reads recognition fields from multipart form
// values and validates them with the same rules as the JSON create endpoint.
func parseRecognitionWithAttachmentInput(r *http.Request, engineerID int64) (Recognition, error) {
	item := Recognition{
		EngineerID:      engineerID,
		RecognitionDate: strings.TrimSpace(r.FormValue("recognitionDate")),
		Source:          strings.TrimSpace(r.FormValue("source")),
		SourceType:      strings.TrimSpace(r.FormValue("sourceType")),
		Category:        strings.TrimSpace(r.FormValue("category")),
		Summary:         strings.TrimSpace(r.FormValue("summary")),
		Details:         strings.TrimSpace(r.FormValue("details")),
		RelatedWork:     strings.TrimSpace(r.FormValue("relatedWork")),
		ReviewCycle:     strings.TrimSpace(r.FormValue("reviewCycle")),
	}
	return item, validateRecognitionFields(item)
}

// validateRecognitionFields applies the same field rules as decodeAndValidateRecognition,
// factored out so the multipart create-with-attachment path shares one validation path.
func validateRecognitionFields(item Recognition) error {
	if item.RecognitionDate == "" {
		return errors.New("recognition date is required")
	}
	if _, err := time.Parse("2006-01-02", item.RecognitionDate); err != nil {
		return errors.New("recognition date must use YYYY-MM-DD")
	}
	if item.Source == "" {
		return errors.New("recognition source is required")
	}
	if !recognitionSourceTypes[item.SourceType] {
		return errors.New("invalid recognition source type")
	}
	if !recognitionCategories[item.Category] {
		return errors.New("invalid recognition category")
	}
	if item.Summary == "" {
		return errors.New("recognition summary is required")
	}
	return nil
}

// persistRecognitionWithAttachment inserts the recognition, the attachment, and the
// junction row linking them in a single transaction so the evidence and its parent
// are saved atomically.
func persistRecognitionWithAttachment(
	item Recognition,
	engineerName string,
	stored *storedAttachment,
) (Recognition, Attachment, error) {
	transaction, err := db.Begin()
	if err != nil {
		return Recognition{}, Attachment{}, err
	}
	defer transaction.Rollback()
	result, err := transaction.Exec(`
		INSERT INTO recognitions
			(engineer_id, recognition_date, source, source_type, category,
			 summary, details, related_work, review_cycle)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.EngineerID, item.RecognitionDate, item.Source, item.SourceType,
		item.Category, item.Summary, item.Details, item.RelatedWork, item.ReviewCycle)
	if err != nil {
		return Recognition{}, Attachment{}, err
	}
	recognitionID, err := result.LastInsertId()
	if err != nil {
		return Recognition{}, Attachment{}, err
	}
	attachmentResult, err := transaction.Exec(`
		INSERT INTO attachments
			(original_filename, stored_filename, mime_type, file_size, sha256_hash,
			 source_system, source_author, source_date, caption)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		stored.originalName, stored.relativePath, stored.mimeType, stored.size,
		stored.hash,
		strings.TrimSpace(""), strings.TrimSpace(""), strings.TrimSpace(""), strings.TrimSpace(""))
	if err != nil {
		return Recognition{}, Attachment{}, err
	}
	attachmentID, err := attachmentResult.LastInsertId()
	if err != nil {
		return Recognition{}, Attachment{}, err
	}
	if _, err := transaction.Exec(`
		INSERT INTO recognition_attachments (recognition_id, attachment_id)
		VALUES (?, ?)`, recognitionID, attachmentID); err != nil {
		return Recognition{}, Attachment{}, err
	}
	if err := transaction.Commit(); err != nil {
		return Recognition{}, Attachment{}, err
	}
	recognition, err := findRecognition(recognitionID)
	if err != nil {
		return Recognition{}, Attachment{}, err
	}
	attachment := Attachment{
		ID: attachmentID, OriginalFilename: stored.originalName,
		MimeType: stored.mimeType, FileSize: stored.size, SHA256Hash: stored.hash,
		ContentURL: "/api/attachments/" + strconv.FormatInt(attachmentID, 10) + "/content",
	}
	return recognition, attachment, nil
}
