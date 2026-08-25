package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

var recognitionSourceTypes = map[string]bool{
	"manager": true, "peer": true, "product": true, "customer": true,
	"leadership": true, "cross_functional": true, "external_partner": true,
}

var recognitionCategories = map[string]bool{
	"business_impact": true, "technical_excellence": true,
	"operational_excellence": true, "mentoring": true,
	"collaboration": true, "leadership": true, "innovation": true,
	"customer_focus": true,
}

const recognitionColumns = `
	id, engineer_id, recognition_date, source, source_type, category, summary,
	COALESCE(details, ''), COALESCE(related_work, ''),
	COALESCE(review_cycle, ''), created_at, updated_at`

func getRecognitions(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	query := `SELECT ` + recognitionColumns + ` FROM recognitions WHERE engineer_id = ?`
	args := []any{engineerID}
	if category := r.URL.Query().Get("category"); category != "" {
		if !recognitionCategories[category] {
			http.Error(w, "Invalid recognition category", http.StatusBadRequest)
			return
		}
		query += ` AND category = ?`
		args = append(args, category)
	}
	if reviewCycle := r.URL.Query().Get("reviewCycle"); reviewCycle != "" {
		query += ` AND review_cycle = ?`
		args = append(args, reviewCycle)
	}
	query += ` ORDER BY recognition_date DESC, id DESC`
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve recognitions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	items := make([]Recognition, 0)
	for rows.Next() {
		item, err := scanRecognition(rows)
		if err != nil {
			http.Error(w, "Failed to read recognition", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading recognitions", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func getRecognition(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid recognition ID", http.StatusBadRequest)
		return
	}
	item, err := findRecognition(id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Recognition not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve recognition", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func createRecognition(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	item, err := decodeAndValidateRecognition(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	item.EngineerID = engineerID
	result, err := db.Exec(`
		INSERT INTO recognitions
			(engineer_id, recognition_date, source, source_type, category,
			 summary, details, related_work, review_cycle)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.EngineerID, item.RecognitionDate, item.Source, item.SourceType,
		item.Category, item.Summary, item.Details, item.RelatedWork,
		item.ReviewCycle)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			http.Error(w, "Engineer not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to create recognition", http.StatusInternalServerError)
		return
	}
	item.ID, err = result.LastInsertId()
	if err != nil {
		http.Error(w, "Recognition created but ID could not be retrieved", http.StatusInternalServerError)
		return
	}
	created, err := findRecognition(item.ID)
	if err != nil {
		http.Error(w, "Recognition created but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func updateRecognition(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid recognition ID", http.StatusBadRequest)
		return
	}
	item, err := decodeAndValidateRecognition(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`
		UPDATE recognitions SET recognition_date = ?, source = ?,
			source_type = ?, category = ?, summary = ?, details = ?,
			related_work = ?, review_cycle = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		item.RecognitionDate, item.Source, item.SourceType, item.Category,
		item.Summary, item.Details, item.RelatedWork, item.ReviewCycle, id)
	if err != nil {
		http.Error(w, "Failed to update recognition", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm recognition update", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Recognition not found", http.StatusNotFound)
		return
	}
	updated, err := findRecognition(id)
	if err != nil {
		http.Error(w, "Recognition updated but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func deleteRecognition(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid recognition ID", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`DELETE FROM recognitions WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Failed to delete recognition", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm recognition deletion", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Recognition not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeAndValidateRecognition(r *http.Request) (Recognition, error) {
	var item Recognition
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		return Recognition{}, errors.New("invalid request body")
	}
	item.RecognitionDate = strings.TrimSpace(item.RecognitionDate)
	item.Source = strings.TrimSpace(item.Source)
	item.SourceType = strings.TrimSpace(item.SourceType)
	item.Category = strings.TrimSpace(item.Category)
	item.Summary = strings.TrimSpace(item.Summary)
	item.Details = strings.TrimSpace(item.Details)
	item.RelatedWork = strings.TrimSpace(item.RelatedWork)
	item.ReviewCycle = strings.TrimSpace(item.ReviewCycle)
	if err := validateRecognitionFields(item); err != nil {
		return Recognition{}, err
	}
	return item, nil
}

func findRecognition(id int64) (Recognition, error) {
	return scanRecognition(db.QueryRow(
		`SELECT `+recognitionColumns+` FROM recognitions WHERE id = ?`, id))
}

type recognitionScanner interface {
	Scan(dest ...any) error
}

func scanRecognition(scanner recognitionScanner) (Recognition, error) {
	var item Recognition
	err := scanner.Scan(
		&item.ID, &item.EngineerID, &item.RecognitionDate, &item.Source,
		&item.SourceType, &item.Category, &item.Summary, &item.Details,
		&item.RelatedWork, &item.ReviewCycle, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}
