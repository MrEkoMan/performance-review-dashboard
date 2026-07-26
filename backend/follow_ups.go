package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var followUpStatuses = map[string]bool{
	"open": true, "in_progress": true, "completed": true, "cancelled": true,
}

var followUpPriorities = map[string]bool{
	"low": true, "medium": true, "high": true,
}

var followUpSourceTables = map[string]string{
	"note":       "performance_notes",
	"goal":       "goals",
	"one_on_one": "one_on_ones",
}

const followUpColumns = `
	id, engineer_id, source_type, source_id, description, owner,
	COALESCE(due_date, ''), status, priority, COALESCE(completion_date, ''),
	COALESCE(notes, ''), created_at, updated_at`

func getFollowUps(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	query := `SELECT ` + followUpColumns + ` FROM follow_ups WHERE engineer_id = ?`
	args := []any{engineerID}
	if status := r.URL.Query().Get("status"); status != "" {
		if !followUpStatuses[status] {
			http.Error(w, "Invalid follow-up status", http.StatusBadRequest)
			return
		}
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY
		CASE status WHEN 'open' THEN 0 WHEN 'in_progress' THEN 1
			WHEN 'completed' THEN 2 ELSE 3 END,
		CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END,
		COALESCE(due_date, '9999-12-31'), id DESC`
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve follow-ups", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	items := make([]FollowUp, 0)
	for rows.Next() {
		item, err := scanFollowUp(rows)
		if err != nil {
			http.Error(w, "Failed to read follow-up", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading follow-ups", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func getFollowUp(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid follow-up ID", http.StatusBadRequest)
		return
	}
	item, err := findFollowUp(id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Follow-up not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve follow-up", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func createFollowUp(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	item, err := decodeAndValidateFollowUp(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	item.EngineerID = engineerID
	if err := validateFollowUpSource(item); err != nil {
		writeFollowUpReferenceError(w, err)
		return
	}
	result, err := db.Exec(`
		INSERT INTO follow_ups
			(engineer_id, source_type, source_id, description, owner, due_date,
			 status, priority, completion_date, notes)
		VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?)`,
		item.EngineerID, item.SourceType, item.SourceID, item.Description,
		item.Owner, item.DueDate, item.Status, item.Priority,
		item.CompletionDate, item.Notes)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			http.Error(w, "Engineer not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to create follow-up", http.StatusInternalServerError)
		return
	}
	item.ID, err = result.LastInsertId()
	if err != nil {
		http.Error(w, "Follow-up created but ID could not be retrieved", http.StatusInternalServerError)
		return
	}
	created, err := findFollowUp(item.ID)
	if err != nil {
		http.Error(w, "Follow-up created but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func updateFollowUp(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid follow-up ID", http.StatusBadRequest)
		return
	}
	existing, err := findFollowUp(id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Follow-up not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve follow-up", http.StatusInternalServerError)
		return
	}
	item, err := decodeAndValidateFollowUp(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	item.EngineerID = existing.EngineerID
	if err := validateFollowUpSource(item); err != nil {
		writeFollowUpReferenceError(w, err)
		return
	}
	_, err = db.Exec(`
		UPDATE follow_ups SET source_type = ?, source_id = ?, description = ?,
			owner = ?, due_date = NULLIF(?, ''), status = ?, priority = ?,
			completion_date = NULLIF(?, ''), notes = ?,
			updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		item.SourceType, item.SourceID, item.Description, item.Owner,
		item.DueDate, item.Status, item.Priority, item.CompletionDate,
		item.Notes, id)
	if err != nil {
		http.Error(w, "Failed to update follow-up", http.StatusInternalServerError)
		return
	}
	updated, err := findFollowUp(id)
	if err != nil {
		http.Error(w, "Follow-up updated but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func deleteFollowUp(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid follow-up ID", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`DELETE FROM follow_ups WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Failed to delete follow-up", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm follow-up deletion", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Follow-up not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeAndValidateFollowUp(r *http.Request) (FollowUp, error) {
	var item FollowUp
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		return FollowUp{}, errors.New("invalid request body")
	}
	item.SourceType = strings.TrimSpace(item.SourceType)
	item.Description = strings.TrimSpace(item.Description)
	item.Owner = strings.TrimSpace(item.Owner)
	item.DueDate = strings.TrimSpace(item.DueDate)
	item.Status = strings.TrimSpace(item.Status)
	item.Priority = strings.TrimSpace(item.Priority)
	item.CompletionDate = strings.TrimSpace(item.CompletionDate)
	item.Notes = strings.TrimSpace(item.Notes)
	if item.SourceType == "" {
		item.SourceType = "manual"
	}
	if item.Description == "" {
		return FollowUp{}, errors.New("follow-up description is required")
	}
	if item.Owner == "" {
		return FollowUp{}, errors.New("follow-up owner is required")
	}
	if !followUpStatuses[item.Status] {
		return FollowUp{}, errors.New("invalid follow-up status")
	}
	if !followUpPriorities[item.Priority] {
		return FollowUp{}, errors.New("invalid follow-up priority")
	}
	if item.SourceType == "manual" {
		if item.SourceID != nil {
			return FollowUp{}, errors.New("manual follow-ups cannot have a source ID")
		}
	} else if _, ok := followUpSourceTables[item.SourceType]; !ok {
		return FollowUp{}, errors.New("invalid follow-up source type")
	} else if item.SourceID == nil || *item.SourceID <= 0 {
		return FollowUp{}, errors.New("linked follow-ups require a valid source ID")
	}
	for label, value := range map[string]string{
		"due date": item.DueDate, "completion date": item.CompletionDate,
	} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return FollowUp{}, errors.New(label + " must use YYYY-MM-DD")
			}
		}
	}
	if item.Status == "completed" && item.CompletionDate == "" {
		return FollowUp{}, errors.New("completed follow-ups require a completion date")
	}
	if item.Status != "completed" && item.CompletionDate != "" {
		return FollowUp{}, errors.New("only completed follow-ups can have a completion date")
	}
	return item, nil
}

func validateFollowUpSource(item FollowUp) error {
	var exists int
	if err := db.QueryRow(`SELECT 1 FROM engineers WHERE id = ?`, item.EngineerID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errEngineerReference
		}
		return err
	}
	if item.SourceType == "manual" {
		return nil
	}
	table := followUpSourceTables[item.SourceType]
	query := fmt.Sprintf(`SELECT 1 FROM %s WHERE id = ? AND engineer_id = ?`, table)
	if err := db.QueryRow(query, *item.SourceID, item.EngineerID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errSourceReference
		}
		return err
	}
	return nil
}

var (
	errEngineerReference = errors.New("engineer not found")
	errSourceReference   = errors.New("follow-up source not found for engineer")
)

func writeFollowUpReferenceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errEngineerReference):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, errSourceReference):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "Failed to validate follow-up source", http.StatusInternalServerError)
	}
}

func findFollowUp(id int64) (FollowUp, error) {
	return scanFollowUp(db.QueryRow(
		`SELECT `+followUpColumns+` FROM follow_ups WHERE id = ?`, id))
}

type followUpScanner interface {
	Scan(dest ...any) error
}

func scanFollowUp(scanner followUpScanner) (FollowUp, error) {
	var item FollowUp
	err := scanner.Scan(
		&item.ID, &item.EngineerID, &item.SourceType, &item.SourceID,
		&item.Description, &item.Owner, &item.DueDate, &item.Status,
		&item.Priority, &item.CompletionDate, &item.Notes,
		&item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}
