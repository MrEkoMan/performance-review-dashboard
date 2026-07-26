package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var timelineEventTypes = map[string]bool{
	"evidence": true, "goal": true, "one_on_one": true,
	"follow_up": true, "recognition": true,
}

const timelineQuery = `
	SELECT event_type, source_id, event_date, title, summary, status, review_cycle
	FROM (
		SELECT engineer_id, 'evidence' AS event_type, id AS source_id,
			note_date AS event_date, category AS title, summary,
			CASE WHEN follow_up_needed THEN 'follow_up_needed' ELSE '' END AS status,
			COALESCE(review_cycle, '') AS review_cycle
		FROM performance_notes
		UNION ALL
		SELECT engineer_id, 'goal', id,
			COALESCE(NULLIF(completion_date, ''), NULLIF(target_date, ''),
				NULLIF(start_date, ''), SUBSTR(created_at, 1, 10)),
			title, COALESCE(description, ''), status, COALESCE(review_cycle, '')
		FROM goals
		UNION ALL
		SELECT engineer_id, 'one_on_one', id, meeting_date,
			'1:1 meeting',
			COALESCE(NULLIF(shared_notes, ''), NULLIF(wins, ''),
				NULLIF(challenges, ''), ''),
			status, ''
		FROM one_on_ones
		UNION ALL
		SELECT engineer_id, 'follow_up', id,
			COALESCE(NULLIF(completion_date, ''), NULLIF(due_date, ''),
				SUBSTR(created_at, 1, 10)),
			description, COALESCE(notes, ''), status, ''
		FROM follow_ups
		UNION ALL
		SELECT engineer_id, 'recognition', id, recognition_date,
			summary, COALESCE(details, ''), category, COALESCE(review_cycle, '')
		FROM recognitions
	) AS events
	WHERE engineer_id = ?`

func getTimeline(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	query := timelineQuery
	args := []any{engineerID}
	if eventType := strings.TrimSpace(r.URL.Query().Get("eventType")); eventType != "" {
		if !timelineEventTypes[eventType] {
			http.Error(w, "Invalid timeline event type", http.StatusBadRequest)
			return
		}
		query += ` AND event_type = ?`
		args = append(args, eventType)
	}
	if reviewCycle := strings.TrimSpace(r.URL.Query().Get("reviewCycle")); reviewCycle != "" {
		query += ` AND review_cycle = ?`
		args = append(args, reviewCycle)
	}
	from, err := timelineDateFilter(r, "from")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	to, err := timelineDateFilter(r, "to")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if from != "" && to != "" && to < from {
		http.Error(w, "timeline to date cannot be before from date", http.StatusBadRequest)
		return
	}
	if from != "" {
		query += ` AND event_date >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND event_date <= ?`
		args = append(args, to)
	}
	query += ` ORDER BY event_date DESC, event_type, source_id DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve timeline", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	events := make([]TimelineEvent, 0)
	for rows.Next() {
		var event TimelineEvent
		if err := rows.Scan(
			&event.EventType, &event.SourceID, &event.EventDate, &event.Title,
			&event.Summary, &event.Status, &event.ReviewCycle,
		); err != nil {
			http.Error(w, "Failed to read timeline event", http.StatusInternalServerError)
			return
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading timeline", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func timelineDateFilter(r *http.Request, name string) (string, error) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return "", nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return "", &timelineDateError{name: name}
	}
	return value, nil
}

type timelineDateError struct {
	name string
}

func (e *timelineDateError) Error() string {
	return "timeline " + e.name + " date must use YYYY-MM-DD"
}
