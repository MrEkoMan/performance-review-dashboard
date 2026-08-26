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

var oneOnOneStatuses = map[string]bool{
	"scheduled": true, "completed": true, "cancelled": true,
}

const oneOnOneColumns = `
	id, engineer_id, meeting_date, COALESCE(wins, ''),
	COALESCE(challenges, ''), COALESCE(career_discussion, ''),
	COALESCE(feedback, ''), COALESCE(manager_topics, ''),
	COALESCE(engineer_topics, ''), COALESCE(private_manager_notes, ''),
	COALESCE(shared_notes, ''), COALESCE(follow_up_date, ''),
	status, created_at, updated_at`

func getOneOnOnes(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	query := `SELECT ` + oneOnOneColumns +
		` FROM one_on_ones WHERE engineer_id = ?`
	args := []any{engineerID}
	if status := r.URL.Query().Get("status"); status != "" {
		if !oneOnOneStatuses[status] {
			http.Error(w, "Invalid 1:1 status", http.StatusBadRequest)
			return
		}
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY meeting_date DESC, id DESC`
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve 1:1 records", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	meetings := make([]OneOnOne, 0)
	for rows.Next() {
		meeting, err := scanOneOnOne(rows)
		if err != nil {
			http.Error(w, "Failed to read 1:1 record", http.StatusInternalServerError)
			return
		}
		meetings = append(meetings, meeting)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading 1:1 records", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, meetings)
}

func getOneOnOne(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid 1:1 ID", http.StatusBadRequest)
		return
	}
	meeting, err := scanOneOnOne(db.QueryRow(
		`SELECT `+oneOnOneColumns+` FROM one_on_ones WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "1:1 record not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve 1:1 record", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, meeting)
}

func createOneOnOne(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	meeting, err := decodeAndValidateOneOnOne(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	meeting.EngineerID = engineerID
	result, err := db.Exec(`
		INSERT INTO one_on_ones
			(engineer_id, meeting_date, wins, challenges, career_discussion,
			 feedback, manager_topics, engineer_topics, private_manager_notes,
			 shared_notes, follow_up_date, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), ?)`,
		meeting.EngineerID, meeting.MeetingDate, meeting.Wins,
		meeting.Challenges, meeting.CareerDiscussion, meeting.Feedback,
		meeting.ManagerTopics, meeting.EngineerTopics,
		meeting.PrivateManagerNotes, meeting.SharedNotes,
		meeting.FollowUpDate, meeting.Status)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			http.Error(w, "Engineer not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to create 1:1 record", http.StatusInternalServerError)
		return
	}
	meeting.ID, err = result.LastInsertId()
	if err != nil {
		http.Error(w, "1:1 created but ID could not be retrieved", http.StatusInternalServerError)
		return
	}
	created, err := scanOneOnOne(db.QueryRow(
		`SELECT `+oneOnOneColumns+` FROM one_on_ones WHERE id = ?`, meeting.ID))
	if err != nil {
		http.Error(w, "1:1 created but could not be retrieved", http.StatusInternalServerError)
		return
	}
	ensureNextScheduledOneOnOne(meeting.EngineerID, meeting.FollowUpDate, meeting.ID)
	writeJSON(w, http.StatusCreated, created)
}

func updateOneOnOne(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid 1:1 ID", http.StatusBadRequest)
		return
	}
	meeting, err := decodeAndValidateOneOnOne(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`
		UPDATE one_on_ones SET meeting_date = ?, wins = ?, challenges = ?,
			career_discussion = ?, feedback = ?, manager_topics = ?,
			engineer_topics = ?, private_manager_notes = ?, shared_notes = ?,
			follow_up_date = NULLIF(?, ''), status = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		meeting.MeetingDate, meeting.Wins, meeting.Challenges,
		meeting.CareerDiscussion, meeting.Feedback, meeting.ManagerTopics,
		meeting.EngineerTopics, meeting.PrivateManagerNotes,
		meeting.SharedNotes, meeting.FollowUpDate, meeting.Status, id)
	if err != nil {
		http.Error(w, "Failed to update 1:1 record", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm 1:1 update", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "1:1 record not found", http.StatusNotFound)
		return
	}
	updated, err := scanOneOnOne(db.QueryRow(
		`SELECT `+oneOnOneColumns+` FROM one_on_ones WHERE id = ?`, id))
	if err != nil {
		http.Error(w, "1:1 updated but could not be retrieved", http.StatusInternalServerError)
		return
	}
	ensureNextScheduledOneOnOne(updated.EngineerID, updated.FollowUpDate, updated.ID)
	writeJSON(w, http.StatusOK, updated)
}

// ensureNextScheduledOneOnOne makes the follow-up date the engineer's next
// scheduled 1:1. If followUpDate is set and the engineer has no future scheduled
// 1:1 (a scheduled row dated after today, other than the meeting just saved,
// identified by excludeID), a scheduled one_on_ones row is created for that date.
// An existing future scheduled 1:1 is left untouched so an already-arranged
// meeting is never overwritten.
func ensureNextScheduledOneOnOne(engineerID int64, followUpDate string, excludeID int64) {
	if followUpDate == "" {
		return
	}
	today := dashboardNow().Format("2006-01-02")
	var existing int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM one_on_ones
		WHERE engineer_id = ? AND status = 'scheduled'
			AND meeting_date > ? AND id <> ?`,
		engineerID, today, excludeID).Scan(&existing)
	if err != nil || existing > 0 {
		return
	}
	_, _ = db.Exec(`
		INSERT INTO one_on_ones (engineer_id, meeting_date, status)
		VALUES (?, ?, 'scheduled')`, engineerID, followUpDate)
}

func deleteOneOnOne(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid 1:1 ID", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`DELETE FROM one_on_ones WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Failed to delete 1:1 record", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm 1:1 deletion", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "1:1 record not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeAndValidateOneOnOne(r *http.Request) (OneOnOne, error) {
	var meeting OneOnOne
	if err := json.NewDecoder(r.Body).Decode(&meeting); err != nil {
		return OneOnOne{}, errors.New("invalid request body")
	}
	meeting.MeetingDate = strings.TrimSpace(meeting.MeetingDate)
	meeting.Wins = strings.TrimSpace(meeting.Wins)
	meeting.Challenges = strings.TrimSpace(meeting.Challenges)
	meeting.CareerDiscussion = strings.TrimSpace(meeting.CareerDiscussion)
	meeting.Feedback = strings.TrimSpace(meeting.Feedback)
	meeting.ManagerTopics = strings.TrimSpace(meeting.ManagerTopics)
	meeting.EngineerTopics = strings.TrimSpace(meeting.EngineerTopics)
	meeting.PrivateManagerNotes = strings.TrimSpace(meeting.PrivateManagerNotes)
	meeting.SharedNotes = strings.TrimSpace(meeting.SharedNotes)
	meeting.FollowUpDate = strings.TrimSpace(meeting.FollowUpDate)
	meeting.Status = strings.TrimSpace(meeting.Status)
	if meeting.MeetingDate == "" {
		return OneOnOne{}, errors.New("meeting date is required")
	}
	if _, err := time.Parse("2006-01-02", meeting.MeetingDate); err != nil {
		return OneOnOne{}, errors.New("meeting date must use YYYY-MM-DD")
	}
	if meeting.FollowUpDate != "" {
		if _, err := time.Parse("2006-01-02", meeting.FollowUpDate); err != nil {
			return OneOnOne{}, errors.New("follow-up date must use YYYY-MM-DD")
		}
	}
	if !oneOnOneStatuses[meeting.Status] {
		return OneOnOne{}, errors.New("invalid 1:1 status")
	}
	return meeting, nil
}

type oneOnOneScanner interface {
	Scan(dest ...any) error
}

func scanOneOnOne(scanner oneOnOneScanner) (OneOnOne, error) {
	var meeting OneOnOne
	err := scanner.Scan(
		&meeting.ID, &meeting.EngineerID, &meeting.MeetingDate,
		&meeting.Wins, &meeting.Challenges, &meeting.CareerDiscussion,
		&meeting.Feedback, &meeting.ManagerTopics, &meeting.EngineerTopics,
		&meeting.PrivateManagerNotes, &meeting.SharedNotes,
		&meeting.FollowUpDate, &meeting.Status,
		&meeting.CreatedAt, &meeting.UpdatedAt,
	)
	return meeting, err
}
