package main

import (
	"net/http"
	"strings"
	"time"
)

var dashboardNow = time.Now

var attentionTypes = map[string]bool{
	"overdue_follow_up": true, "blocked_goal": true, "overdue_goal": true,
	"stale_evidence": true, "upcoming_one_on_one": true,
	"legacy_follow_up": true, "stale_recognition": true,
}

var attentionSeverities = map[string]bool{
	"high": true, "medium": true, "low": true,
}

const attentionQuery = `
	SELECT item_type, severity, engineer_id, engineer_name, title, reason,
		due_date, source_type, source_id, target_tab
	FROM (
		SELECT 'overdue_follow_up' AS item_type, 'high' AS severity,
			f.engineer_id, e.name AS engineer_name, f.description AS title,
			'Structured follow-up is overdue' AS reason,
			COALESCE(f.due_date, '') AS due_date, 'follow_up' AS source_type,
			f.id AS source_id, 'follow-ups' AS target_tab
		FROM follow_ups f JOIN engineers e ON e.id = f.engineer_id
		WHERE f.status IN ('open', 'in_progress') AND f.due_date < ?
		UNION ALL
		SELECT 'blocked_goal', 'high', g.engineer_id, e.name, g.title,
			'Goal is blocked', COALESCE(g.target_date, ''), 'goal', g.id, 'goals'
		FROM goals g JOIN engineers e ON e.id = g.engineer_id
		WHERE g.status = 'blocked'
		UNION ALL
		SELECT 'overdue_goal', 'high', g.engineer_id, e.name, g.title,
			'Goal target date has passed', g.target_date, 'goal', g.id, 'goals'
		FROM goals g JOIN engineers e ON e.id = g.engineer_id
		WHERE g.status IN ('not_started', 'in_progress')
			AND g.target_date < ?
		UNION ALL
		SELECT 'stale_evidence', 'medium', e.id, e.name,
			'Evidence needs attention',
			'No performance evidence recorded in the last 30 days',
			'', 'engineer', e.id, 'evidence'
		FROM engineers e
		LEFT JOIN performance_notes n ON n.engineer_id = e.id
		GROUP BY e.id, e.name
		HAVING MAX(n.note_date) IS NULL OR MAX(n.note_date) < ?
		UNION ALL
		SELECT 'upcoming_one_on_one', 'medium', o.engineer_id, e.name,
			'Upcoming 1:1', 'Prepare for the scheduled 1:1', o.meeting_date,
			'one_on_one', o.id, 'one-on-ones'
		FROM one_on_ones o JOIN engineers e ON e.id = o.engineer_id
		WHERE o.status = 'scheduled' AND o.meeting_date BETWEEN ? AND ?
		UNION ALL
		SELECT 'legacy_follow_up', 'medium', n.engineer_id, e.name, n.summary,
			'Performance evidence is marked for follow-up', n.note_date,
			'evidence', n.id, 'evidence'
		FROM performance_notes n JOIN engineers e ON e.id = n.engineer_id
		WHERE n.follow_up_needed = TRUE
		UNION ALL
		SELECT 'stale_recognition', 'low', e.id, e.name,
			'Recognition may be overdue',
			'No recognition recorded in the last 90 days',
			'', 'engineer', e.id, 'recognition'
		FROM engineers e
		LEFT JOIN recognitions r ON r.engineer_id = e.id
		GROUP BY e.id, e.name
		HAVING MAX(r.recognition_date) IS NULL OR MAX(r.recognition_date) < ?
	) AS attention
	WHERE 1 = 1`

func getDashboardAttention(w http.ResponseWriter, r *http.Request) {
	now := dashboardNow()
	today := now.Format("2006-01-02")
	evidenceCutoff := now.AddDate(0, 0, -30).Format("2006-01-02")
	upcomingCutoff := now.AddDate(0, 0, 7).Format("2006-01-02")
	recognitionCutoff := now.AddDate(0, 0, -90).Format("2006-01-02")
	query := attentionQuery
	args := []any{
		today, today, evidenceCutoff, today, upcomingCutoff, recognitionCutoff,
	}
	if itemType := strings.TrimSpace(r.URL.Query().Get("type")); itemType != "" {
		if !attentionTypes[itemType] {
			http.Error(w, "Invalid attention type", http.StatusBadRequest)
			return
		}
		query += ` AND item_type = ?`
		args = append(args, itemType)
	}
	if severity := strings.TrimSpace(r.URL.Query().Get("severity")); severity != "" {
		if !attentionSeverities[severity] {
			http.Error(w, "Invalid attention severity", http.StatusBadRequest)
			return
		}
		query += ` AND severity = ?`
		args = append(args, severity)
	}
	query += ` ORDER BY
		CASE severity WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END,
		CASE WHEN due_date = '' THEN 1 ELSE 0 END, due_date, engineer_name`

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve attention items", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	items := make([]AttentionItem, 0)
	for rows.Next() {
		var item AttentionItem
		if err := rows.Scan(
			&item.ItemType, &item.Severity, &item.EngineerID,
			&item.EngineerName, &item.Title, &item.Reason, &item.DueDate,
			&item.SourceType, &item.SourceID, &item.TargetTab,
		); err != nil {
			http.Error(w, "Failed to read attention item", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading attention items", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
