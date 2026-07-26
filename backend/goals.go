package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var goalTypes = map[string]bool{
	"delivery": true, "technical_growth": true, "leadership": true,
	"communication": true, "operational_excellence": true, "mentoring": true,
	"career_development": true, "stretch_assignment": true,
}

var goalStatuses = map[string]bool{
	"not_started": true, "in_progress": true, "blocked": true,
	"completed": true, "cancelled": true,
}

var goalPriorities = map[string]bool{"low": true, "medium": true, "high": true}

const goalColumns = `
	id, engineer_id, title, COALESCE(description, ''), goal_type, status,
	priority, COALESCE(start_date, ''), COALESCE(target_date, ''),
	COALESCE(completion_date, ''), progress_percentage,
	COALESCE(success_criteria, ''), COALESCE(manager_notes, ''),
	COALESCE(engineer_notes, ''), COALESCE(review_cycle, ''),
	created_at, updated_at`

func getGoals(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	query := `SELECT ` + goalColumns + ` FROM goals WHERE engineer_id = ?`
	args := []any{engineerID}
	if status := r.URL.Query().Get("status"); status != "" {
		if !goalStatuses[status] {
			http.Error(w, "Invalid goal status", http.StatusBadRequest)
			return
		}
		query += ` AND status = ?`
		args = append(args, status)
	}
	if reviewCycle := r.URL.Query().Get("reviewCycle"); reviewCycle != "" {
		query += ` AND review_cycle = ?`
		args = append(args, reviewCycle)
	}
	query += ` ORDER BY
		CASE status WHEN 'blocked' THEN 0 WHEN 'in_progress' THEN 1
			WHEN 'not_started' THEN 2 WHEN 'completed' THEN 3 ELSE 4 END,
		CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END,
		COALESCE(target_date, '9999-12-31'), id DESC`
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve goals", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	goals := make([]Goal, 0)
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			http.Error(w, "Failed to read goal", http.StatusInternalServerError)
			return
		}
		goals = append(goals, goal)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading goals", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, goals)
}

func getGoal(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid goal ID", http.StatusBadRequest)
		return
	}
	goal, err := scanGoal(db.QueryRow(`SELECT `+goalColumns+` FROM goals WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Goal not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve goal", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, goal)
}

func createGoal(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	goal, err := decodeAndValidateGoal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	goal.EngineerID = engineerID
	result, err := db.Exec(`
		INSERT INTO goals
			(engineer_id, title, description, goal_type, status, priority,
			 start_date, target_date, completion_date, progress_percentage,
			 success_criteria, manager_notes, engineer_notes, review_cycle)
		VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''),
			?, ?, ?, ?, ?)`,
		goal.EngineerID, goal.Title, goal.Description, goal.GoalType, goal.Status,
		goal.Priority, goal.StartDate, goal.TargetDate, goal.CompletionDate,
		goal.ProgressPercent, goal.SuccessCriteria, goal.ManagerNotes,
		goal.EngineerNotes, goal.ReviewCycle)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			http.Error(w, "Engineer not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to create goal", http.StatusInternalServerError)
		return
	}
	goal.ID, err = result.LastInsertId()
	if err != nil {
		http.Error(w, "Goal created but ID could not be retrieved", http.StatusInternalServerError)
		return
	}
	created, err := scanGoal(db.QueryRow(`SELECT `+goalColumns+` FROM goals WHERE id = ?`, goal.ID))
	if err != nil {
		http.Error(w, "Goal created but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func updateGoal(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid goal ID", http.StatusBadRequest)
		return
	}
	goal, err := decodeAndValidateGoal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`
		UPDATE goals SET title = ?, description = ?, goal_type = ?, status = ?,
			priority = ?, start_date = NULLIF(?, ''), target_date = NULLIF(?, ''),
			completion_date = NULLIF(?, ''), progress_percentage = ?,
			success_criteria = ?, manager_notes = ?, engineer_notes = ?,
			review_cycle = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		goal.Title, goal.Description, goal.GoalType, goal.Status, goal.Priority,
		goal.StartDate, goal.TargetDate, goal.CompletionDate, goal.ProgressPercent,
		goal.SuccessCriteria, goal.ManagerNotes, goal.EngineerNotes,
		goal.ReviewCycle, id)
	if err != nil {
		http.Error(w, "Failed to update goal", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm goal update", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Goal not found", http.StatusNotFound)
		return
	}
	updated, err := scanGoal(db.QueryRow(`SELECT `+goalColumns+` FROM goals WHERE id = ?`, id))
	if err != nil {
		http.Error(w, "Goal updated but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func deleteGoal(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid goal ID", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`DELETE FROM goals WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Failed to delete goal", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm goal deletion", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Goal not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeAndValidateGoal(r *http.Request) (Goal, error) {
	var goal Goal
	if err := json.NewDecoder(r.Body).Decode(&goal); err != nil {
		return Goal{}, errors.New("invalid request body")
	}
	goal.Title = strings.TrimSpace(goal.Title)
	goal.Description = strings.TrimSpace(goal.Description)
	goal.GoalType = strings.TrimSpace(goal.GoalType)
	goal.Status = strings.TrimSpace(goal.Status)
	goal.Priority = strings.TrimSpace(goal.Priority)
	goal.SuccessCriteria = strings.TrimSpace(goal.SuccessCriteria)
	goal.ManagerNotes = strings.TrimSpace(goal.ManagerNotes)
	goal.EngineerNotes = strings.TrimSpace(goal.EngineerNotes)
	goal.ReviewCycle = strings.TrimSpace(goal.ReviewCycle)
	if goal.Title == "" {
		return Goal{}, errors.New("goal title is required")
	}
	if !goalTypes[goal.GoalType] {
		return Goal{}, errors.New("invalid goal type")
	}
	if !goalStatuses[goal.Status] {
		return Goal{}, errors.New("invalid goal status")
	}
	if !goalPriorities[goal.Priority] {
		return Goal{}, errors.New("invalid goal priority")
	}
	if goal.ProgressPercent < 0 || goal.ProgressPercent > 100 {
		return Goal{}, errors.New("progress percentage must be between 0 and 100")
	}
	for label, value := range map[string]string{
		"start date": goal.StartDate, "target date": goal.TargetDate,
		"completion date": goal.CompletionDate,
	} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return Goal{}, errors.New(label + " must use YYYY-MM-DD")
			}
		}
	}
	if goal.StartDate != "" && goal.TargetDate != "" && goal.TargetDate < goal.StartDate {
		return Goal{}, errors.New("target date cannot be before start date")
	}
	if goal.Status == "completed" {
		if goal.ProgressPercent != 100 {
			return Goal{}, errors.New("completed goals must have 100 percent progress")
		}
		if goal.CompletionDate == "" {
			return Goal{}, errors.New("completed goals require a completion date")
		}
	}
	return goal, nil
}

func positiveID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid ID")
	}
	return id, nil
}

type goalScanner interface {
	Scan(dest ...any) error
}

func scanGoal(scanner goalScanner) (Goal, error) {
	var goal Goal
	err := scanner.Scan(
		&goal.ID, &goal.EngineerID, &goal.Title, &goal.Description,
		&goal.GoalType, &goal.Status, &goal.Priority, &goal.StartDate,
		&goal.TargetDate, &goal.CompletionDate, &goal.ProgressPercent,
		&goal.SuccessCriteria, &goal.ManagerNotes, &goal.EngineerNotes,
		&goal.ReviewCycle, &goal.CreatedAt, &goal.UpdatedAt,
	)
	return goal, err
}
