package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

var dashboardGoalHealth = map[string]bool{
	"blocked": true, "overdue": true, "at_risk": true, "on_track": true,
}

const dashboardGoalRiskGap = 20

func getDashboardGoals(w http.ResponseWriter, r *http.Request) {
	includeClosed := false
	if value := strings.TrimSpace(r.URL.Query().Get("includeClosed")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			http.Error(w, "Invalid includeClosed filter", http.StatusBadRequest)
			return
		}
		includeClosed = parsed
	}
	health := strings.TrimSpace(r.URL.Query().Get("health"))
	if health != "" && !dashboardGoalHealth[health] {
		http.Error(w, "Invalid goal health", http.StatusBadRequest)
		return
	}

	query := `SELECT
		g.id, g.engineer_id, e.name, g.title, g.goal_type, g.status, g.priority,
		COALESCE(g.start_date, ''), COALESCE(g.target_date, ''),
		g.progress_percentage, COALESCE(g.review_cycle, '')
		FROM goals g JOIN engineers e ON e.id = g.engineer_id WHERE 1 = 1`
	args := make([]any, 0)
	if !includeClosed {
		query += ` AND g.status IN ('not_started', 'in_progress', 'blocked')`
	}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !goalStatuses[status] {
			http.Error(w, "Invalid goal status", http.StatusBadRequest)
			return
		}
		query += ` AND g.status = ?`
		args = append(args, status)
	}
	if priority := strings.TrimSpace(r.URL.Query().Get("priority")); priority != "" {
		if !goalPriorities[priority] {
			http.Error(w, "Invalid goal priority", http.StatusBadRequest)
			return
		}
		query += ` AND g.priority = ?`
		args = append(args, priority)
	}
	if value := strings.TrimSpace(r.URL.Query().Get("engineerId")); value != "" {
		engineerID, err := positiveID(value)
		if err != nil {
			http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
			return
		}
		query += ` AND g.engineer_id = ?`
		args = append(args, engineerID)
	}
	if reviewCycle := strings.TrimSpace(r.URL.Query().Get("reviewCycle")); reviewCycle != "" {
		query += ` AND g.review_cycle = ?`
		args = append(args, reviewCycle)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve dashboard goals", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	today, _ := time.Parse("2006-01-02", dashboardNow().Format("2006-01-02"))
	items := make([]DashboardGoal, 0)
	for rows.Next() {
		var item DashboardGoal
		if err := rows.Scan(
			&item.ID, &item.EngineerID, &item.EngineerName, &item.Title,
			&item.GoalType, &item.Status, &item.Priority, &item.StartDate,
			&item.TargetDate, &item.ProgressPercent, &item.ReviewCycle,
		); err != nil {
			http.Error(w, "Failed to read dashboard goal", http.StatusInternalServerError)
			return
		}
		if err := deriveDashboardGoalHealth(&item, today); err != nil {
			http.Error(w, "Failed to read goal dates", http.StatusInternalServerError)
			return
		}
		if health == "" || item.Health == health {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading dashboard goals", http.StatusInternalServerError)
		return
	}
	sortDashboardGoals(items)
	writeJSON(w, http.StatusOK, items)
}

func deriveDashboardGoalHealth(item *DashboardGoal, today time.Time) error {
	item.Health = "on_track"
	if item.Status == "blocked" {
		item.Health = "blocked"
	}
	if item.TargetDate == "" {
		return nil
	}
	target, err := time.Parse("2006-01-02", item.TargetDate)
	if err != nil {
		return err
	}
	item.DaysToTarget = int(target.Sub(today).Hours() / 24)
	if item.Status != "blocked" && item.Status != "completed" &&
		item.Status != "cancelled" && target.Before(today) {
		item.Health = "overdue"
		return nil
	}
	if item.Health == "blocked" || item.StartDate == "" ||
		item.Status == "completed" || item.Status == "cancelled" {
		return nil
	}
	start, err := time.Parse("2006-01-02", item.StartDate)
	if err != nil {
		return err
	}
	totalDays := target.Sub(start).Hours() / 24
	elapsedDays := today.Sub(start).Hours() / 24
	if totalDays <= 0 || elapsedDays <= 0 {
		return nil
	}
	item.ExpectedProgress = min(100, int(elapsedDays/totalDays*100))
	if item.ExpectedProgress-item.ProgressPercent >= dashboardGoalRiskGap {
		item.Health = "at_risk"
	}
	return nil
}

func sortDashboardGoals(items []DashboardGoal) {
	healthRank := map[string]int{"blocked": 0, "overdue": 1, "at_risk": 2, "on_track": 3}
	priorityRank := map[string]int{"high": 0, "medium": 1, "low": 2}
	for i := 1; i < len(items); i++ {
		for j := i; j > 0; j-- {
			left, right := items[j-1], items[j]
			swap := healthRank[left.Health] > healthRank[right.Health] ||
				(healthRank[left.Health] == healthRank[right.Health] &&
					(priorityRank[left.Priority] > priorityRank[right.Priority] ||
						(priorityRank[left.Priority] == priorityRank[right.Priority] &&
							left.TargetDate > right.TargetDate)))
			if !swap {
				break
			}
			items[j-1], items[j] = items[j], items[j-1]
		}
	}
}
