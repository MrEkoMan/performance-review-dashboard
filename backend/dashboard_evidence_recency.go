package main

import (
	"net/http"
	"strings"
	"time"
)

var evidenceRecencyStates = map[string]bool{
	"recent": true, "aging": true, "stale": true,
	"critical": true, "never": true,
}

func getEvidenceRecency(w http.ResponseWriter, r *http.Request) {
	recency := strings.TrimSpace(r.URL.Query().Get("recency"))
	if recency != "" && !evidenceRecencyStates[recency] {
		http.Error(w, "Invalid evidence recency", http.StatusBadRequest)
		return
	}
	query := `SELECT
		e.id, e.name, e.team, e.review_cycle, COALESCE(MAX(n.note_date), ''),
		COUNT(n.id),
		COALESCE(SUM(CASE WHEN n.note_date >= ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN n.review_cycle = e.review_cycle THEN 1 ELSE 0 END), 0)
		FROM engineers e
		LEFT JOIN performance_notes n ON n.engineer_id = e.id
		WHERE 1 = 1`
	now := dashboardNow()
	args := []any{now.AddDate(0, 0, -30).Format("2006-01-02")}
	if value := strings.TrimSpace(r.URL.Query().Get("engineerId")); value != "" {
		engineerID, err := positiveID(value)
		if err != nil {
			http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
			return
		}
		query += ` AND e.id = ?`
		args = append(args, engineerID)
	}
	if team := strings.TrimSpace(r.URL.Query().Get("team")); team != "" {
		query += ` AND e.team = ?`
		args = append(args, team)
	}
	if reviewCycle := strings.TrimSpace(r.URL.Query().Get("reviewCycle")); reviewCycle != "" {
		query += ` AND e.review_cycle = ?`
		args = append(args, reviewCycle)
	}
	query += ` GROUP BY e.id, e.name, e.team, e.review_cycle`

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve evidence recency", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	today, _ := time.Parse("2006-01-02", now.Format("2006-01-02"))
	items := make([]EvidenceRecency, 0)
	for rows.Next() {
		var item EvidenceRecency
		if err := rows.Scan(
			&item.EngineerID, &item.EngineerName, &item.Team, &item.ReviewCycle,
			&item.LastEvidenceDate, &item.TotalEvidence, &item.EvidenceLast30Days,
			&item.CurrentCycleEvidence,
		); err != nil {
			http.Error(w, "Failed to read evidence recency", http.StatusInternalServerError)
			return
		}
		if err := deriveEvidenceRecency(&item, today); err != nil {
			http.Error(w, "Failed to read evidence date", http.StatusInternalServerError)
			return
		}
		if recency == "" || item.Recency == recency {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading evidence recency", http.StatusInternalServerError)
		return
	}
	sortEvidenceRecency(items)
	writeJSON(w, http.StatusOK, items)
}

func deriveEvidenceRecency(item *EvidenceRecency, today time.Time) error {
	if item.LastEvidenceDate == "" {
		item.DaysSinceEvidence = -1
		item.Recency = "never"
		return nil
	}
	lastEvidence, err := time.Parse("2006-01-02", item.LastEvidenceDate)
	if err != nil {
		return err
	}
	item.DaysSinceEvidence = max(0, int(today.Sub(lastEvidence).Hours()/24))
	switch {
	case item.DaysSinceEvidence > 90:
		item.Recency = "critical"
	case item.DaysSinceEvidence > 60:
		item.Recency = "stale"
	case item.DaysSinceEvidence > 30:
		item.Recency = "aging"
	default:
		item.Recency = "recent"
	}
	return nil
}

func sortEvidenceRecency(items []EvidenceRecency) {
	rank := map[string]int{
		"never": 0, "critical": 1, "stale": 2, "aging": 3, "recent": 4,
	}
	for i := 1; i < len(items); i++ {
		for j := i; j > 0; j-- {
			left, right := items[j-1], items[j]
			if rank[left.Recency] < rank[right.Recency] ||
				(rank[left.Recency] == rank[right.Recency] &&
					(left.DaysSinceEvidence > right.DaysSinceEvidence ||
						(left.DaysSinceEvidence == right.DaysSinceEvidence &&
							left.EngineerName <= right.EngineerName))) {
				break
			}
			items[j-1], items[j] = items[j], items[j-1]
		}
	}
}
