package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

const dashboardFollowUpColumns = `
	f.id, f.engineer_id, e.name, f.source_type, f.source_id,
	f.description, f.owner, COALESCE(f.due_date, ''), f.status, f.priority,
	COALESCE(f.notes, '')`

func getDashboardFollowUps(w http.ResponseWriter, r *http.Request) {
	today := dashboardNow().Format("2006-01-02")
	overdueOnly := true
	if value := strings.TrimSpace(r.URL.Query().Get("overdue")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			http.Error(w, "Invalid overdue filter", http.StatusBadRequest)
			return
		}
		overdueOnly = parsed
	}
	query := `SELECT ` + dashboardFollowUpColumns + `
		FROM follow_ups f JOIN engineers e ON e.id = f.engineer_id WHERE 1 = 1`
	args := make([]any, 0)
	if overdueOnly {
		query += ` AND f.status IN ('open', 'in_progress')
			AND f.due_date IS NOT NULL AND f.due_date < ?`
		args = append(args, today)
	}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !followUpStatuses[status] {
			http.Error(w, "Invalid follow-up status", http.StatusBadRequest)
			return
		}
		query += ` AND f.status = ?`
		args = append(args, status)
	}
	if priority := strings.TrimSpace(r.URL.Query().Get("priority")); priority != "" {
		if !followUpPriorities[priority] {
			http.Error(w, "Invalid follow-up priority", http.StatusBadRequest)
			return
		}
		query += ` AND f.priority = ?`
		args = append(args, priority)
	}
	if value := strings.TrimSpace(r.URL.Query().Get("engineerId")); value != "" {
		engineerID, err := positiveID(value)
		if err != nil {
			http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
			return
		}
		query += ` AND f.engineer_id = ?`
		args = append(args, engineerID)
	}
	if owner := strings.TrimSpace(r.URL.Query().Get("owner")); owner != "" {
		query += ` AND LOWER(f.owner) = LOWER(?)`
		args = append(args, owner)
	}
	query += ` ORDER BY
		CASE f.priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1
			WHEN 'medium' THEN 2 ELSE 3 END,
		CASE f.status WHEN 'open' THEN 0 WHEN 'in_progress' THEN 1
			WHEN 'completed' THEN 2 ELSE 3 END,
		COALESCE(f.due_date, '9999-12-31'), e.name, f.id`

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve dashboard follow-ups", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	items := make([]DashboardFollowUp, 0)
	todayDate, _ := time.Parse("2006-01-02", today)
	for rows.Next() {
		var item DashboardFollowUp
		if err := rows.Scan(
			&item.ID, &item.EngineerID, &item.EngineerName, &item.SourceType,
			&item.SourceID, &item.Description, &item.Owner, &item.DueDate,
			&item.Status, &item.Priority, &item.Notes,
		); err != nil {
			http.Error(w, "Failed to read dashboard follow-up", http.StatusInternalServerError)
			return
		}
		if item.DueDate != "" {
			dueDate, err := time.Parse("2006-01-02", item.DueDate)
			if err != nil {
				http.Error(w, "Failed to read follow-up due date", http.StatusInternalServerError)
				return
			}
			if dueDate.Before(todayDate) {
				item.DaysOverdue = int(todayDate.Sub(dueDate).Hours() / 24)
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading dashboard follow-ups", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
