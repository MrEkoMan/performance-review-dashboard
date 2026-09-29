package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

const aiAnalysisColumns = `
	id, engineer_id, provider, model, COALESCE(review_cycle, ''),
	context_summary, context_hash, output_markdown, created_at`

func getAIAnalyses(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	rows, err := db.Query(`
		SELECT ` + aiAnalysisColumns + ` FROM ai_analyses
		WHERE engineer_id = ?
		ORDER BY created_at DESC, id DESC`, engineerID)
	if err != nil {
		http.Error(w, "Failed to retrieve AI analyses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	items := make([]AIAnalysis, 0)
	for rows.Next() {
		item, err := scanAIAnalysis(rows)
		if err != nil {
			http.Error(w, "Failed to read AI analysis", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading AI analyses", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func deleteAIAnalysis(w http.ResponseWriter, r *http.Request) {
	id, err := positiveID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid analysis ID", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`DELETE FROM ai_analyses WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Failed to remove AI analysis", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm AI analysis removal", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Analysis is not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func createAIAnalysis(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	var input AIAnalysisInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	input.Provider = strings.TrimSpace(input.Provider)
	input.ReviewCycle = strings.TrimSpace(input.ReviewCycle)
	if !allowedAIProviders[input.Provider] {
		http.Error(w, "Unsupported AI provider", http.StatusBadRequest)
		return
	}
	if _, err := scanEngineer(db.QueryRow(
		`SELECT ` + engineerColumns + ` FROM engineers WHERE id = ?`, engineerID,
	)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Engineer is not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to retrieve engineer", http.StatusInternalServerError)
		return
	}

	var config aiProviderConfig
	var enabled bool
	err = db.QueryRow(`
		SELECT base_url, model, COALESCE(api_version, ''),
			COALESCE(encrypted_api_key, ''), enabled
		FROM ai_provider_configurations WHERE provider = ?`,
		input.Provider,
	).Scan(&config.BaseURL, &config.Model, &config.APIVersion,
		&config.APIKey, &enabled)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "AI provider is not configured", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve AI provider", http.StatusInternalServerError)
		return
	}
	if !enabled {
		http.Error(w, "AI provider is disabled", http.StatusBadRequest)
		return
	}

	prompt, summary, hash, err := buildAnalysisContext(engineerID, input.ReviewCycle)
	if err != nil {
		http.Error(w, "Failed to gather engineer evidence", http.StatusInternalServerError)
		return
	}

	if input.Provider != "ollama" {
		apiKey, err := decryptSecret(config.APIKey)
		if err != nil {
			http.Error(w, "Stored credential could not be decrypted", http.StatusInternalServerError)
			return
		}
		config.APIKey = apiKey
	} else {
		config.APIKey = ""
	}
	client, err := newAIClient(input.Provider, config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	output, err := client.analyze(r.Context(), prompt)
	if err != nil {
		category := http.StatusBadGateway
		if errors.Is(err, context.DeadlineExceeded) {
			category = http.StatusGatewayTimeout
		} else {
			var networkError net.Error
			if errors.As(err, &networkError) && networkError.Timeout() {
				category = http.StatusGatewayTimeout
			}
		}
		http.Error(w, err.Error(), category)
		return
	}

	result, err := db.Exec(`
		INSERT INTO ai_analyses
			(engineer_id, provider, model, review_cycle, context_summary,
			 context_hash, output_markdown)
		VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, ?)`,
		engineerID, input.Provider, config.Model, input.ReviewCycle,
		summary, hash, output)
	if err != nil {
		http.Error(w, "Failed to save AI analysis", http.StatusInternalServerError)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Analysis saved but ID could not be retrieved", http.StatusInternalServerError)
		return
	}
	analysis, err := scanAIAnalysis(db.QueryRow(
		`SELECT ` + aiAnalysisColumns + ` FROM ai_analyses WHERE id = ?`, id))
	if err != nil {
		http.Error(w, "Analysis saved but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, analysis)
}

type aiAnalysisScanner interface {
	Scan(dest ...any) error
}

func scanAIAnalysis(scanner aiAnalysisScanner) (AIAnalysis, error) {
	var analysis AIAnalysis
	err := scanner.Scan(
		&analysis.ID, &analysis.EngineerID, &analysis.Provider,
		&analysis.Model, &analysis.ReviewCycle, &analysis.ContextSummary,
		&analysis.ContextHash, &analysis.OutputMarkdown,
		&analysis.CreatedAt,
	)
	return analysis, err
}

// buildAnalysisContext gathers the engineer's evidence into a prompt and
// returns it alongside a machine-readable summary (per-source counts, cycle,
// truncation flags) and a hash of the prompt for provenance.
func buildAnalysisContext(
	engineerID int64,
	reviewCycle string,
) (string, string, string, error) {
	engineer, err := scanEngineer(db.QueryRow(
		`SELECT ` + engineerColumns + ` FROM engineers WHERE id = ?`, engineerID))
	if err != nil {
		return "", "", "", err
	}

	notes, err := analysisNotes(engineerID, reviewCycle)
	if err != nil {
		return "", "", "", err
	}
	goals, err := analysisGoals(engineerID, reviewCycle)
	if err != nil {
		return "", "", "", err
	}
	oneOnOnes, err := analysisOneOnOnes(engineerID)
	if err != nil {
		return "", "", "", err
	}
	followUps, err := analysisFollowUps(engineerID)
	if err != nil {
		return "", "", "", err
	}
	recognitions, err := analysisRecognitions(engineerID, reviewCycle)
	if err != nil {
		return "", "", "", err
	}
	plans, err := analysisDevelopmentPlanTitles(engineerID)
	if err != nil {
		return "", "", "", err
	}

	cycleClause := "all review cycles"
	if reviewCycle != "" {
		cycleClause = "review cycle " + reviewCycle
	}
	prompt := strings.Builder{}
	prompt.WriteString(fmt.Sprintf(
		"You are analyzing the performance record of %s, %s %s on the %s team. "+
			"Evidence below covers %s. Analyze only this evidence — do not "+
			"invent facts. Respond in GitHub-flavored markdown with these "+
			"sections: ## Summary, ## Strengths, ## Growth Areas, ## Goal "+
			"Progress, ## One-on-One Themes, ## Follow-up Hygiene, ## Overall "+
			"Assessment, ## Suggested Talking Points.\n\n",
		engineer.Name, engineer.Level, engineer.Role, engineer.Team,
		cycleClause))

	prompt.WriteString("## Evidence: Engineer Profile\n")
	prompt.WriteString(fmt.Sprintf("- Role: %s %s\n- Team: %s\n",
		engineer.Level, engineer.Role, engineer.Team))
	if engineer.CareerGoal != "" {
		prompt.WriteString(fmt.Sprintf("- Stated career goal: %s\n", engineer.CareerGoal))
	}
	if engineer.ReviewCycle != "" {
		prompt.WriteString(fmt.Sprintf("- Current review cycle: %s\n", engineer.ReviewCycle))
	}

	prompt.WriteString("\n## Evidence: Performance Notes\n")
	if len(notes) == 0 {
		prompt.WriteString("None recorded.\n")
	} else {
		for _, note := range notes {
			prompt.WriteString(fmt.Sprintf(
				"- %s [%s] %s%s\n",
				note.NoteDate, note.Category, note.Summary,
				truncateForAnalysis(note.Impact, "")))
		}
	}

	prompt.WriteString("\n## Evidence: Goals\n")
	if len(goals) == 0 {
		prompt.WriteString("None recorded.\n")
	} else {
		for _, goal := range goals {
			prompt.WriteString(fmt.Sprintf(
				"- [%s/%s, %d%%] %s (target %s)%s\n",
				goal.Status, goal.Priority, goal.ProgressPercent, goal.Title,
				orNone(goal.TargetDate),
				truncateForAnalysis(goal.ManagerNotes, " | Manager: ")))
		}
	}

	prompt.WriteString("\n## Evidence: One-on-One Themes\n")
	if len(oneOnOnes) == 0 {
		prompt.WriteString("None recorded.\n")
	} else {
		for _, meeting := range oneOnOnes {
			prompt.WriteString(fmt.Sprintf(
				"- %s [status: %s]\n", meeting.MeetingDate, meeting.Status))
			if meeting.Wins != "" {
				prompt.WriteString(fmt.Sprintf("  Wins: %s\n",
					truncatePlain(meeting.Wins)))
			}
			if meeting.Challenges != "" {
				prompt.WriteString(fmt.Sprintf("  Challenges: %s\n",
					truncatePlain(meeting.Challenges)))
			}
			if meeting.CareerDiscussion != "" {
				prompt.WriteString(fmt.Sprintf("  Career: %s\n",
					truncatePlain(meeting.CareerDiscussion)))
			}
			if meeting.Feedback != "" {
				prompt.WriteString(fmt.Sprintf("  Feedback: %s\n",
					truncatePlain(meeting.Feedback)))
			}
			if meeting.PrivateManagerNotes != "" {
				prompt.WriteString(fmt.Sprintf("  Private notes: %s\n",
					truncatePlain(meeting.PrivateManagerNotes)))
			}
		}
	}

	prompt.WriteString("\n## Evidence: Follow-ups\n")
	if len(followUps) == 0 {
		prompt.WriteString("None recorded.\n")
	} else {
		for _, item := range followUps {
			prompt.WriteString(fmt.Sprintf(
				"- [%s/%s, due %s] %s (owner: %s)\n",
				item.Status, item.Priority, orNone(item.DueDate),
				item.Description, orNone(item.Owner)))
		}
	}

	prompt.WriteString("\n## Evidence: Recognition\n")
	if len(recognitions) == 0 {
		prompt.WriteString("None recorded.\n")
	} else {
		for _, recognition := range recognitions {
			prompt.WriteString(fmt.Sprintf(
				"- %s [%s from %s] %s%s\n",
				recognition.RecognitionDate, recognition.Category,
				recognition.Source, recognition.Summary,
				truncateForAnalysis(recognition.RelatedWork, " | Related: ")))
		}
	}

	if len(plans) > 0 {
		prompt.WriteString("\n## Evidence: Development Plans\n")
		for _, plan := range plans {
			prompt.WriteString(fmt.Sprintf(
				"- %s (planned %s, cycle %s)\n",
				plan.Title, orNone(plan.PlanDate), orNone(plan.ReviewCycle)))
		}
	}

	summary := analysisContextSummary{
		Notes:                len(notes),
		Goals:                len(goals),
		OneOnOnes:            len(oneOnOnes),
		FollowUps:            len(followUps),
		Recognitions:         len(recognitions),
		DevelopmentPlans:     len(plans),
		ReviewCycle:          reviewCycle,
		EngineerName:         engineer.Name,
	}
	encoded, err := json.Marshal(summary)
	if err != nil {
		return "", "", "", err
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(prompt.String())))
	return prompt.String(), string(encoded), hash, nil
}

type analysisContextSummary struct {
	Notes            int    `json:"notes"`
	Goals            int    `json:"goals"`
	OneOnOnes        int    `json:"oneOnOnes"`
	FollowUps        int    `json:"followUps"`
	Recognitions     int    `json:"recognitions"`
	DevelopmentPlans int    `json:"developmentPlans"`
	ReviewCycle      string `json:"reviewCycle"`
	EngineerName     string `json:"engineerName"`
}

const analysisTextLimit = 500

func truncatePlain(value string) string {
	if len(value) <= analysisTextLimit {
		return value
	}
	return value[:analysisTextLimit] + "… [truncated]"
}

func truncateForAnalysis(value, prefix string) string {
	if value == "" {
		return ""
	}
	return prefix + truncatePlain(value)
}

func orNone(value string) string {
	if value == "" {
		return "none"
	}
	return value
}

// Per-engineer data queries. Free-text fields are truncated in Go after
// selection so the prompt stays bounded regardless of database content.

const analysisNoteLimit = 50
const analysisOneOnOneLimit = 20
const analysisFollowUpLimit = 30
const analysisRecognitionLimit = 30

func analysisNotes(engineerID int64, reviewCycle string) ([]PerformanceNote, error) {
	query := `
		SELECT id, engineer_id, '', note_date, category,
			summary, COALESCE(details, ''), COALESCE(impact, ''),
			COALESCE(follow_up_needed, 0), COALESCE(review_cycle, '')
		FROM performance_notes WHERE engineer_id = ?`
	args := []any{engineerID}
	if reviewCycle != "" {
		query += ` AND COALESCE(review_cycle, '') = ?`
		args = append(args, reviewCycle)
	}
	query += ` ORDER BY note_date DESC LIMIT ?`
	args = append(args, analysisNoteLimit)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	notes := make([]PerformanceNote, 0)
	for rows.Next() {
		var note PerformanceNote
		if err := rows.Scan(
			&note.ID, &note.EngineerID, &note.EngineerName, &note.NoteDate,
			&note.Category, &note.Summary, &note.Details, &note.Impact,
			&note.FollowUpNeeded, &note.ReviewCycle,
		); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func analysisGoals(engineerID int64, reviewCycle string) ([]Goal, error) {
	query := `SELECT ` + goalColumns + ` FROM goals WHERE engineer_id = ?`
	args := []any{engineerID}
	if reviewCycle != "" {
		query += ` AND COALESCE(review_cycle, '') = ?`
		args = append(args, reviewCycle)
	}
	query += ` ORDER BY
		CASE status WHEN 'blocked' THEN 0 WHEN 'in_progress' THEN 1
			WHEN 'not_started' THEN 2 WHEN 'completed' THEN 3 ELSE 4 END,
		COALESCE(target_date, '9999-12-31'), id DESC`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	goals := make([]Goal, 0)
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}
	return goals, rows.Err()
}

func analysisOneOnOnes(engineerID int64) ([]OneOnOne, error) {
	rows, err := db.Query(`
		SELECT ` + oneOnOneColumns + ` FROM one_on_ones
		WHERE engineer_id = ?
		ORDER BY meeting_date DESC LIMIT ?`,
		engineerID, analysisOneOnOneLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	meetings := make([]OneOnOne, 0)
	for rows.Next() {
		meeting, err := scanOneOnOne(rows)
		if err != nil {
			return nil, err
		}
		meetings = append(meetings, meeting)
	}
	return meetings, rows.Err()
}

func analysisFollowUps(engineerID int64) ([]FollowUp, error) {
	rows, err := db.Query(`
		SELECT ` + followUpColumns + ` FROM follow_ups
		WHERE engineer_id = ?
		ORDER BY
			CASE status WHEN 'open' THEN 0 WHEN 'in_progress' THEN 1
				WHEN 'completed' THEN 2 ELSE 3 END,
			CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END,
			COALESCE(due_date, '9999-12-31'), id DESC
		LIMIT ?`,
		engineerID, analysisFollowUpLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]FollowUp, 0)
	for rows.Next() {
		item, err := scanFollowUp(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func analysisRecognitions(engineerID int64, reviewCycle string) ([]Recognition, error) {
	query := `SELECT ` + recognitionColumns + ` FROM recognitions WHERE engineer_id = ?`
	args := []any{engineerID}
	if reviewCycle != "" {
		query += ` AND COALESCE(review_cycle, '') = ?`
		args = append(args, reviewCycle)
	}
	query += ` ORDER BY recognition_date DESC LIMIT ?`
	args = append(args, analysisRecognitionLimit)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Recognition, 0)
	for rows.Next() {
		item, err := scanRecognition(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func analysisDevelopmentPlanTitles(engineerID int64) ([]DevelopmentPlan, error) {
	rows, err := db.Query(`
		SELECT id, engineer_id, title, COALESCE(plan_date, ''),
			COALESCE(review_cycle, '')
		FROM development_plans
		WHERE engineer_id = ?
		ORDER BY plan_date DESC, id DESC`,
		engineerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plans := make([]DevelopmentPlan, 0)
	for rows.Next() {
		var plan DevelopmentPlan
		if err := rows.Scan(
			&plan.ID, &plan.EngineerID, &plan.Title, &plan.PlanDate,
			&plan.ReviewCycle,
		); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}
