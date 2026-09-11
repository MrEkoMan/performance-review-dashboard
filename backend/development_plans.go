package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const developmentPlanColumns = `
	id, engineer_id, title, COALESCE(plan_date, ''), COALESCE(review_cycle, ''),
	COALESCE(raw_markdown, ''), COALESCE(fields, '{}'), linked_goal_id,
	created_at, updated_at`

func getDevelopmentPlans(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	rows, err := db.Query(
		`SELECT `+developmentPlanColumns+
			` FROM development_plans WHERE engineer_id = ? ORDER BY id DESC`, engineerID)
	if err != nil {
		http.Error(w, "Failed to retrieve development plans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	plans := make([]DevelopmentPlan, 0)
	for rows.Next() {
		plan, err := scanDevelopmentPlan(rows)
		if err != nil {
			http.Error(w, "Failed to read development plan", http.StatusInternalServerError)
			return
		}
		plans = append(plans, plan)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading development plans", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, plans)
}

func getDevelopmentPlan(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid development plan ID", http.StatusBadRequest)
		return
	}
	plan, err := scanDevelopmentPlan(db.QueryRow(
		`SELECT `+developmentPlanColumns+` FROM development_plans WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Development plan not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve development plan", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func createDevelopmentPlan(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	plan, err := decodeAndValidateDevelopmentPlan(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plan.EngineerID = engineerID
	saved, err := saveDevelopmentPlan(plan)
	if err != nil {
		writePlanError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func updateDevelopmentPlan(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid development plan ID", http.StatusBadRequest)
		return
	}
	plan, err := decodeAndValidateDevelopmentPlan(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plan.ID = id
	// Preserve the engineer association; the plan is scoped to one engineer.
	var existingEngineerID int64
	err = db.QueryRow(`SELECT engineer_id FROM development_plans WHERE id = ?`, id).Scan(&existingEngineerID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Development plan not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve development plan", http.StatusInternalServerError)
		return
	}
	plan.EngineerID = existingEngineerID
	saved, err := saveDevelopmentPlan(plan)
	if err != nil {
		writePlanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func deleteDevelopmentPlan(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid development plan ID", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`DELETE FROM development_plans WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Failed to delete development plan", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm development plan deletion", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Development plan not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseDevelopmentPlanFile accepts a multipart markdown upload and returns the
// structured fields parsed from it, mirroring the onboarding parse endpoint.
// It does not persist anything; the manager reviews the parsed fields in the
// form before saving.
func parseDevelopmentPlanFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, "Upload is invalid or exceeds the 1 MB limit", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "A file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	buf, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read uploaded file", http.StatusBadRequest)
		return
	}
	fields := parseDevelopmentPlan(string(buf))
	writeJSON(w, http.StatusOK, fields)
}

func decodeAndValidateDevelopmentPlan(r *http.Request) (DevelopmentPlan, error) {
	var plan DevelopmentPlan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		return DevelopmentPlan{}, errors.New("invalid request body")
	}
	plan.Title = strings.TrimSpace(plan.Title)
	if plan.Title == "" {
		plan.Title = "Personal Development Plan"
	}
	plan.PlanDate = strings.TrimSpace(plan.PlanDate)
	if plan.PlanDate != "" {
		if _, err := time.Parse("2006-01-02", plan.PlanDate); err != nil {
			return DevelopmentPlan{}, errors.New("plan date must use YYYY-MM-DD")
		}
	}
	plan.ReviewCycle = strings.TrimSpace(plan.ReviewCycle)
	plan.RawMarkdown = strings.TrimSpace(plan.RawMarkdown)
	if plan.LinkedGoalID != nil && *plan.LinkedGoalID <= 0 {
		plan.LinkedGoalID = nil
	}
	return plan, nil
}

// planSaveError distinguishes the recoverable error cases saveDevelopmentPlan
// can surface so the HTTP layer can map them to the right status code.
type planSaveError struct {
	notFound bool
	message  string
}

func (e planSaveError) Error() string { return e.message }

func writePlanError(w http.ResponseWriter, err error) {
	var pse planSaveError
	if errors.As(err, &pse) {
		if pse.notFound {
			http.Error(w, pse.message, http.StatusNotFound)
			return
		}
		http.Error(w, pse.message, http.StatusBadRequest)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// saveDevelopmentPlan inserts or updates the plan and returns the persisted
// record. A non-zero ID means update; otherwise insert.
func saveDevelopmentPlan(plan DevelopmentPlan) (DevelopmentPlan, error) {
	fieldsJSON, _ := json.Marshal(plan.Fields)
	linkedGoal := nullableInt64(plan.LinkedGoalID)
	if plan.ID == 0 {
		result, err := db.Exec(`
			INSERT INTO development_plans
				(engineer_id, title, plan_date, review_cycle, raw_markdown, fields, linked_goal_id)
			VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)`,
			plan.EngineerID, plan.Title, plan.PlanDate, plan.ReviewCycle,
			plan.RawMarkdown, string(fieldsJSON), linkedGoal)
		if err != nil {
			if strings.Contains(err.Error(), "FOREIGN KEY") {
				return plan, planSaveError{notFound: true, message: "Engineer not found"}
			}
			return plan, errors.New("Failed to create development plan")
		}
		plan.ID, _ = result.LastInsertId()
	} else {
		result, err := db.Exec(`
			UPDATE development_plans SET title = ?, plan_date = NULLIF(?, ''),
				review_cycle = NULLIF(?, ''), raw_markdown = ?, fields = ?,
				linked_goal_id = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			plan.Title, plan.PlanDate, plan.ReviewCycle, plan.RawMarkdown,
			string(fieldsJSON), linkedGoal, plan.ID)
		if err != nil {
			return plan, errors.New("Failed to update development plan")
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return plan, errors.New("Failed to confirm development plan update")
		}
		if affected == 0 {
			return plan, planSaveError{notFound: true, message: "Development plan not found"}
		}
	}
	saved, err := scanDevelopmentPlan(db.QueryRow(
		`SELECT `+developmentPlanColumns+` FROM development_plans WHERE id = ?`, plan.ID))
	if err != nil {
		return plan, errors.New("Development plan saved but could not be retrieved")
	}
	return saved, nil
}

// nullableInt64 converts a *int64 to a sql.NullInt64 suitable for binding.
func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

type developmentPlanScanner interface {
	Scan(dest ...any) error
}

func scanDevelopmentPlan(scanner developmentPlanScanner) (DevelopmentPlan, error) {
	var plan DevelopmentPlan
	var fieldsJSON string
	var linkedGoal sql.NullInt64
	err := scanner.Scan(
		&plan.ID, &plan.EngineerID, &plan.Title, &plan.PlanDate, &plan.ReviewCycle,
		&plan.RawMarkdown, &fieldsJSON, &linkedGoal, &plan.CreatedAt, &plan.UpdatedAt)
	if err != nil {
		return DevelopmentPlan{}, err
	}
	if fieldsJSON != "" {
		if err := json.Unmarshal([]byte(fieldsJSON), &plan.Fields); err != nil {
			return DevelopmentPlan{}, err
		}
	}
	if linkedGoal.Valid {
		id := linkedGoal.Int64
		plan.LinkedGoalID = &id
	}
	return plan, nil
}
