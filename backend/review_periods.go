package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const reviewPeriodColumns = `
	id, label, start_date, end_date, created_at, updated_at`

func getReviewPeriods(w http.ResponseWriter, _ *http.Request) {
	rows, err := db.Query(`SELECT ` + reviewPeriodColumns + `
		FROM review_periods ORDER BY start_date DESC, label`)
	if err != nil {
		http.Error(w, "Failed to retrieve review periods", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	items := make([]ReviewPeriod, 0)
	today := dashboardNow()
	for rows.Next() {
		item, err := scanReviewPeriod(rows, today)
		if err != nil {
			http.Error(w, "Failed to read review period", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading review periods", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func createReviewPeriod(w http.ResponseWriter, r *http.Request) {
	item, err := decodeReviewPeriod(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`
		INSERT INTO review_periods (label, start_date, end_date)
		VALUES (?, ?, ?)`, item.Label, item.StartDate, item.EndDate)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "Review period label already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create review period", http.StatusInternalServerError)
		return
	}
	item.ID, err = result.LastInsertId()
	if err != nil {
		http.Error(w, "Review period created but ID could not be retrieved", http.StatusInternalServerError)
		return
	}
	created, err := getReviewPeriodByID(item.ID)
	if err != nil {
		http.Error(w, "Review period created but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func updateReviewPeriod(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid review period ID", http.StatusBadRequest)
		return
	}
	item, err := decodeReviewPeriod(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var existingLabel string
	if err := db.QueryRow(`SELECT label FROM review_periods WHERE id = ?`, id).Scan(&existingLabel); errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Review period not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve review period", http.StatusInternalServerError)
		return
	}
	if item.Label != existingLabel {
		var assignments int
		if err := db.QueryRow(`SELECT COUNT(*) FROM engineers WHERE review_cycle = ?`, existingLabel).Scan(&assignments); err != nil {
			http.Error(w, "Failed to check review period assignments", http.StatusInternalServerError)
			return
		}
		if assignments > 0 {
			http.Error(w, "Assigned review period labels cannot be changed", http.StatusConflict)
			return
		}
	}
	result, err := db.Exec(`
		UPDATE review_periods SET label = ?, start_date = ?, end_date = ?,
			updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		item.Label, item.StartDate, item.EndDate, id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "Review period label already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to update review period", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		http.Error(w, "Failed to confirm review period update", http.StatusInternalServerError)
		return
	}
	updated, err := getReviewPeriodByID(id)
	if err != nil {
		http.Error(w, "Review period updated but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func deleteReviewPeriod(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid review period ID", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`DELETE FROM review_periods WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Failed to delete review period", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm review period deletion", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Review period not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeReviewPeriod(r *http.Request) (ReviewPeriod, error) {
	var item ReviewPeriod
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		return ReviewPeriod{}, errors.New("invalid request body")
	}
	item.Label = strings.TrimSpace(item.Label)
	item.StartDate = strings.TrimSpace(item.StartDate)
	item.EndDate = strings.TrimSpace(item.EndDate)
	if item.Label == "" || item.StartDate == "" || item.EndDate == "" {
		return ReviewPeriod{}, errors.New("label, start date, and end date are required")
	}
	for _, value := range []string{item.StartDate, item.EndDate} {
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return ReviewPeriod{}, errors.New("review period dates must use YYYY-MM-DD")
		}
	}
	if item.EndDate < item.StartDate {
		return ReviewPeriod{}, errors.New("end date cannot be before start date")
	}
	return item, nil
}

type reviewPeriodScanner interface {
	Scan(dest ...any) error
}

func scanReviewPeriod(scanner reviewPeriodScanner, now time.Time) (ReviewPeriod, error) {
	var item ReviewPeriod
	err := scanner.Scan(
		&item.ID, &item.Label, &item.StartDate, &item.EndDate,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err == nil {
		today := now.Format("2006-01-02")
		switch {
		case today < item.StartDate:
			item.Phase = "planned"
		case today > item.EndDate:
			item.Phase = "closed"
		default:
			item.Phase = "active"
		}
	}
	return item, err
}

func getReviewPeriodByID(id int64) (ReviewPeriod, error) {
	return scanReviewPeriod(
		db.QueryRow(`SELECT `+reviewPeriodColumns+` FROM review_periods WHERE id = ?`, id),
		dashboardNow(),
	)
}
