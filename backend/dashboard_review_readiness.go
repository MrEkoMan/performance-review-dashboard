package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

var reviewReadinessStates = map[string]bool{
	"ready": true, "needs_attention": true, "unconfigured": true,
}

const (
	minimumReviewEvidence   = 3
	minimumReviewCategories = 2
)

func getReviewReadiness(w http.ResponseWriter, r *http.Request) {
	readiness := strings.TrimSpace(r.URL.Query().Get("readiness"))
	if readiness != "" && !reviewReadinessStates[readiness] {
		http.Error(w, "Invalid review readiness", http.StatusBadRequest)
		return
	}
	endingWithinDays := 0
	if value := strings.TrimSpace(r.URL.Query().Get("endingWithinDays")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 365 {
			http.Error(w, "Ending window must be between 1 and 365 days", http.StatusBadRequest)
			return
		}
		endingWithinDays = parsed
	}
	today := dashboardNow().Format("2006-01-02")
	query := `SELECT
		e.id, e.name, e.team, e.review_cycle,
		COALESCE(rp.start_date, ''), COALESCE(rp.end_date, ''),
		(SELECT COUNT(*) FROM performance_notes n
			WHERE n.engineer_id = e.id AND n.review_cycle = e.review_cycle),
		(SELECT COUNT(DISTINCT n.category) FROM performance_notes n
			WHERE n.engineer_id = e.id AND n.review_cycle = e.review_cycle),
		(SELECT COUNT(*) FROM goals g
			WHERE g.engineer_id = e.id AND g.review_cycle = e.review_cycle),
		(SELECT COUNT(*) FROM recognitions r
			WHERE r.engineer_id = e.id AND r.review_cycle = e.review_cycle),
		(SELECT COUNT(*) FROM one_on_ones o
			WHERE o.engineer_id = e.id AND o.status = 'completed'
				AND rp.id IS NOT NULL
				AND o.meeting_date BETWEEN rp.start_date AND rp.end_date),
		(SELECT COUNT(*) FROM follow_ups f
			WHERE f.engineer_id = e.id
				AND f.status IN ('open', 'in_progress')
				AND f.due_date IS NOT NULL AND f.due_date < ?)
		FROM engineers e
		LEFT JOIN review_periods rp ON rp.label = e.review_cycle
		WHERE 1 = 1`
	args := []any{today}
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

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve review readiness", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	todayDate, _ := time.Parse("2006-01-02", today)
	items := make([]ReviewReadiness, 0)
	for rows.Next() {
		var item ReviewReadiness
		if err := rows.Scan(
			&item.EngineerID, &item.EngineerName, &item.Team, &item.ReviewCycle,
			&item.PeriodStart, &item.PeriodEnd, &item.EvidenceCount,
			&item.EvidenceCategoryCount, &item.GoalCount, &item.RecognitionCount,
			&item.CompletedOneOnOnes, &item.OverdueFollowUps,
		); err != nil {
			http.Error(w, "Failed to read review readiness", http.StatusInternalServerError)
			return
		}
		if err := deriveReviewReadiness(&item, todayDate); err != nil {
			http.Error(w, "Failed to read review period dates", http.StatusInternalServerError)
			return
		}
		if readiness != "" && item.Readiness != readiness {
			continue
		}
		if endingWithinDays > 0 &&
			(item.PeriodEnd == "" || item.DaysUntilEnd < 0 ||
				item.DaysUntilEnd > endingWithinDays) {
			continue
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading review readiness", http.StatusInternalServerError)
		return
	}
	sortReviewReadiness(items)
	writeJSON(w, http.StatusOK, items)
}

func deriveReviewReadiness(item *ReviewReadiness, today time.Time) error {
	item.MissingItems = make([]string, 0)
	if item.PeriodStart == "" || item.PeriodEnd == "" {
		item.Readiness = "unconfigured"
		item.PeriodPhase = "unconfigured"
		item.MissingItems = append(item.MissingItems, "Configure review period dates")
		return nil
	}
	start, err := time.Parse("2006-01-02", item.PeriodStart)
	if err != nil {
		return err
	}
	end, err := time.Parse("2006-01-02", item.PeriodEnd)
	if err != nil {
		return err
	}
	item.DaysUntilEnd = int(end.Sub(today).Hours() / 24)
	switch {
	case today.Before(start):
		item.PeriodPhase = "planned"
	case today.After(end):
		item.PeriodPhase = "closed"
	default:
		item.PeriodPhase = "active"
	}
	if item.EvidenceCount < minimumReviewEvidence {
		item.MissingItems = append(item.MissingItems, "Record at least 3 evidence notes")
	}
	if item.EvidenceCategoryCount < minimumReviewCategories {
		item.MissingItems = append(item.MissingItems, "Cover at least 2 evidence categories")
	}
	if item.GoalCount == 0 {
		item.MissingItems = append(item.MissingItems, "Link at least 1 goal")
	}
	if item.RecognitionCount == 0 {
		item.MissingItems = append(item.MissingItems, "Record at least 1 recognition")
	}
	if item.CompletedOneOnOnes == 0 {
		item.MissingItems = append(item.MissingItems, "Complete at least 1 one-on-one")
	}
	if item.OverdueFollowUps > 0 {
		item.MissingItems = append(item.MissingItems, "Resolve overdue follow-ups")
	}
	if len(item.MissingItems) == 0 {
		item.Readiness = "ready"
	} else {
		item.Readiness = "needs_attention"
	}
	return nil
}

func sortReviewReadiness(items []ReviewReadiness) {
	rank := map[string]int{"unconfigured": 0, "needs_attention": 1, "ready": 2}
	for i := 1; i < len(items); i++ {
		for j := i; j > 0; j-- {
			left, right := items[j-1], items[j]
			if rank[left.Readiness] < rank[right.Readiness] ||
				(rank[left.Readiness] == rank[right.Readiness] &&
					(left.DaysUntilEnd < right.DaysUntilEnd ||
						(left.DaysUntilEnd == right.DaysUntilEnd &&
							left.EngineerName <= right.EngineerName))) {
				break
			}
			items[j-1], items[j] = items[j], items[j-1]
		}
	}
}
