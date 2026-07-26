package main

import (
	"net/http"
	"strconv"
	"time"
)

const upcomingOneOnOnesQuery = `
	SELECT o.id, o.engineer_id, e.name, o.meeting_date,
		COALESCE((
			SELECT MAX(previous.meeting_date) FROM one_on_ones previous
			WHERE previous.engineer_id = o.engineer_id
				AND previous.status = 'completed'
				AND previous.meeting_date < o.meeting_date
		), ''),
		(SELECT COUNT(*) FROM follow_ups f
			WHERE f.engineer_id = o.engineer_id
				AND f.status IN ('open', 'in_progress')),
		(SELECT COUNT(*) FROM goals blocked
			WHERE blocked.engineer_id = o.engineer_id
				AND blocked.status = 'blocked'),
		(SELECT COUNT(*) FROM goals overdue
			WHERE overdue.engineer_id = o.engineer_id
				AND overdue.status IN ('not_started', 'in_progress')
				AND overdue.target_date < ?),
		(SELECT COUNT(*) FROM performance_notes n
			WHERE n.engineer_id = o.engineer_id AND n.note_date >= ?),
		(SELECT COUNT(*) FROM recognitions r
			WHERE r.engineer_id = o.engineer_id AND r.recognition_date >= ?)
	FROM one_on_ones o
	JOIN engineers e ON e.id = o.engineer_id
	WHERE o.status = 'scheduled' AND o.meeting_date <= ?
	ORDER BY o.meeting_date, e.name, o.id`

func getUpcomingOneOnOnes(w http.ResponseWriter, r *http.Request) {
	windowDays := 14
	if value := r.URL.Query().Get("days"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 90 {
			http.Error(w, "Upcoming 1:1 days must be between 1 and 90", http.StatusBadRequest)
			return
		}
		windowDays = parsed
	}
	now := dashboardNow()
	today := now.Format("2006-01-02")
	upcomingCutoff := now.AddDate(0, 0, windowDays).Format("2006-01-02")
	evidenceCutoff := now.AddDate(0, 0, -30).Format("2006-01-02")
	recognitionCutoff := now.AddDate(0, 0, -90).Format("2006-01-02")
	rows, err := db.Query(
		upcomingOneOnOnesQuery,
		today, evidenceCutoff, recognitionCutoff, upcomingCutoff,
	)
	if err != nil {
		http.Error(w, "Failed to retrieve upcoming 1:1s", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	items := make([]UpcomingOneOnOne, 0)
	for rows.Next() {
		var item UpcomingOneOnOne
		if err := rows.Scan(
			&item.MeetingID, &item.EngineerID, &item.EngineerName,
			&item.MeetingDate, &item.LastCompletedDate, &item.OpenFollowUps,
			&item.BlockedGoals, &item.OverdueGoals, &item.RecentEvidenceCount,
			&item.RecentRecognitionCount,
		); err != nil {
			http.Error(w, "Failed to read upcoming 1:1", http.StatusInternalServerError)
			return
		}
		meetingDate, err := time.Parse("2006-01-02", item.MeetingDate)
		if err != nil {
			http.Error(w, "Failed to read upcoming 1:1 date", http.StatusInternalServerError)
			return
		}
		todayDate, _ := time.Parse("2006-01-02", today)
		item.DaysUntil = int(meetingDate.Sub(todayDate).Hours() / 24)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading upcoming 1:1s", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
