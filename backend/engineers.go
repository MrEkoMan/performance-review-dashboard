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

const engineerColumns = `
	id, name, role, level, team, COALESCE(career_goal, ''), review_cycle,
	COALESCE(archived, 0), COALESCE(departure_date, ''),
	COALESCE(departure_reason, ''), COALESCE(departure_notes, '')`

// departureReasons is the allowed set of departure reasons. Empty means the
// engineer is active (not archived).
var departureReasons = map[string]bool{
	"resigned": true, "terminated": true, "other": true,
}

func getEngineers(w http.ResponseWriter, r *http.Request) {
	query := `SELECT ` + engineerColumns + ` FROM engineers WHERE 1 = 1`
	args := []any{}
	// Archived engineers are excluded by default so dashboards, filters, and
	// note creation target active staff only; ?archived=all or =only opts in.
	switch strings.TrimSpace(r.URL.Query().Get("archived")) {
	case "all":
	case "only":
		query += ` AND COALESCE(archived, 0) = 1`
	default:
		query += ` AND COALESCE(archived, 0) = 0`
	}
	query += ` ORDER BY name`
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve engineers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	engineers := make([]Engineer, 0)
	for rows.Next() {
		engineer, err := scanEngineer(rows)
		if err != nil {
			http.Error(w, "Failed to read engineer", http.StatusInternalServerError)
			return
		}
		engineers = append(engineers, engineer)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading engineers", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, engineers)
}

func getEngineer(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	engineer, err := scanEngineer(db.QueryRow(
		`SELECT `+engineerColumns+` FROM engineers WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Engineer not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve engineer", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, engineer)
}

func createEngineer(w http.ResponseWriter, r *http.Request) {
	var engineer Engineer
	if err := json.NewDecoder(r.Body).Decode(&engineer); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if engineer.Name == "" || engineer.Role == "" || engineer.Level == "" ||
		engineer.Team == "" || engineer.ReviewCycle == "" {
		http.Error(w, "Name, role, level, team, and review cycle are required", http.StatusBadRequest)
		return
	}
	if engineer.Archived {
		http.Error(w, "A new engineer cannot be archived at creation", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`
		INSERT INTO engineers (name, role, level, team, career_goal, review_cycle)
		VALUES (?, ?, ?, ?, ?, ?)`,
		engineer.Name, engineer.Role, engineer.Level, engineer.Team,
		engineer.CareerGoal, engineer.ReviewCycle)
	if err != nil {
		http.Error(w, "Failed to create engineer", http.StatusInternalServerError)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Engineer created but ID could not be retrieved", http.StatusInternalServerError)
		return
	}
	engineer.ID = int(id)
	saved, err := scanEngineer(db.QueryRow(
		`SELECT `+engineerColumns+` FROM engineers WHERE id = ?`, engineer.ID))
	if err != nil {
		writeJSON(w, http.StatusCreated, engineer)
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

// updateEngineer handles PUT /api/engineers/{engineerId}: editing profile
// fields and archiving/unarchiving. Archive requests require a departure
// reason; unarchive clears the departure metadata.
func updateEngineer(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	var engineer Engineer
	if err := json.NewDecoder(r.Body).Decode(&engineer); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if engineer.Name == "" || engineer.Role == "" || engineer.Level == "" ||
		engineer.Team == "" || engineer.ReviewCycle == "" {
		http.Error(w, "Name, role, level, team, and review cycle are required", http.StatusBadRequest)
		return
	}
	if engineer.Archived {
		engineer.DepartureReason = strings.TrimSpace(engineer.DepartureReason)
		if !departureReasons[engineer.DepartureReason] {
			http.Error(w, "Departure reason must be one of: resigned, terminated, other", http.StatusBadRequest)
			return
		}
		if engineer.DepartureDate != "" {
			if _, err := time.Parse("2006-01-02", engineer.DepartureDate); err != nil {
				http.Error(w, "Departure date must use YYYY-MM-DD", http.StatusBadRequest)
				return
			}
		}
	}
	result, err := db.Exec(`
		UPDATE engineers SET name = ?, role = ?, level = ?, team = ?,
			career_goal = ?, review_cycle = ?, archived = ?,
			departure_date = NULLIF(?, ''), departure_reason = NULLIF(?, ''),
			departure_notes = NULLIF(?, '')
		WHERE id = ?`,
		engineer.Name, engineer.Role, engineer.Level, engineer.Team,
		engineer.CareerGoal, engineer.ReviewCycle, engineer.Archived,
		engineer.DepartureDate, engineer.DepartureReason, engineer.DepartureNotes, id)
	if err != nil {
		http.Error(w, "Failed to update engineer", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm engineer update", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Engineer not found", http.StatusNotFound)
		return
	}
	updated, err := scanEngineer(db.QueryRow(
		`SELECT `+engineerColumns+` FROM engineers WHERE id = ?`, id))
	if err != nil {
		http.Error(w, "Engineer updated but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// archiveEngineer handles POST /api/engineers/{engineerId}/archive: marks the
// engineer archived with departure metadata without deleting any history.
func archiveEngineer(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	var payload struct {
		DepartureDate   string `json:"departureDate"`
		DepartureReason string `json:"departureReason"`
		DepartureNotes  string `json:"departureNotes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	payload.DepartureReason = strings.TrimSpace(payload.DepartureReason)
	if !departureReasons[payload.DepartureReason] {
		http.Error(w, "Departure reason must be one of: resigned, terminated, other", http.StatusBadRequest)
		return
	}
	if payload.DepartureDate != "" {
		if _, err := time.Parse("2006-01-02", payload.DepartureDate); err != nil {
			http.Error(w, "Departure date must use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
	}
	result, err := db.Exec(`
		UPDATE engineers SET archived = 1,
			departure_date = COALESCE(NULLIF(?, ''), DATE('now')),
			departure_reason = ?, departure_notes = NULLIF(?, '')
		WHERE id = ?`,
		payload.DepartureDate, payload.DepartureReason, payload.DepartureNotes, id)
	if err != nil {
		http.Error(w, "Failed to archive engineer", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm archive", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Engineer not found", http.StatusNotFound)
		return
	}
	updated, err := scanEngineer(db.QueryRow(
		`SELECT `+engineerColumns+` FROM engineers WHERE id = ?`, id))
	if err != nil {
		http.Error(w, "Engineer archived but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// restoreEngineer handles POST /api/engineers/{engineerId}/restore: clears the
// archived flag and departure metadata.
func restoreEngineer(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`
		UPDATE engineers SET archived = 0,
			departure_date = NULL, departure_reason = NULL, departure_notes = NULL
		WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Failed to restore engineer", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm restore", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Engineer not found", http.StatusNotFound)
		return
	}
	updated, err := scanEngineer(db.QueryRow(
		`SELECT `+engineerColumns+` FROM engineers WHERE id = ?`, id))
	if err != nil {
		http.Error(w, "Engineer restored but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

type engineerScanner interface {
	Scan(dest ...any) error
}

func scanEngineer(scanner engineerScanner) (Engineer, error) {
	var engineer Engineer
	var archived int
	var departureDate, departureReason, departureNotes sql.NullString
	if err := scanner.Scan(
		&engineer.ID, &engineer.Name, &engineer.Role, &engineer.Level,
		&engineer.Team, &engineer.CareerGoal, &engineer.ReviewCycle,
		&archived, &departureDate, &departureReason, &departureNotes,
	); err != nil {
		return Engineer{}, err
	}
	engineer.Archived = archived == 1
	engineer.DepartureDate = departureDate.String
	engineer.DepartureReason = departureReason.String
	engineer.DepartureNotes = departureNotes.String
	return engineer, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
