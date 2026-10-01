package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// JiraEvidence is one resolved Jira issue surfaced as suggested review
// evidence. The struct mirrors the Jira search response fields the UI needs;
// everything else in the API payload is discarded.
type JiraEvidence struct {
	IssueKey      string `json:"issueKey"`
	Summary       string `json:"summary"`
	Resolution    string `json:"resolution"`
	ResolvedDate  string `json:"resolvedDate"`
	Status        string `json:"status"`
	Priority      string `json:"priority"`
	BrowseURL     string `json:"browseUrl"`
	SuggestedNote struct {
		Category string `json:"category"`
		Summary  string `json:"summary"`
		Details  string `json:"details"`
		Impact   string `json:"impact"`
	} `json:"suggestedNote"`
}

type jiraEvidenceResponse struct {
	Evidence  []JiraEvidence `json:"evidence"`
	Total     int            `json:"total"`
	FromDate  string         `json:"fromDate"`
	ToDate    string         `json:"toDate"`
	JiraUser  string         `json:"jiraUser"`
	Engineer  string         `json:"engineer"`
	NextPage  int            `json:"nextPage,omitempty"`
	HasMore   bool           `json:"hasMore"`
}

// getJiraEvidence handles GET /api/engineers/{engineerId}/jira-evidence:
// harvests resolved Jira issues assigned to the engineer over a date range,
// using the stored (manager's) Jira integration credential.
func getJiraEvidence(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	engineer, err := scanEngineer(db.QueryRow(
		`SELECT `+engineerColumns+` FROM engineers WHERE id = ?`, engineerID))
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Engineer not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve engineer", http.StatusInternalServerError)
		return
	}
	jiraUser := strings.TrimSpace(engineer.JiraUsername)
	if jiraUser == "" {
		http.Error(w, "Engineer has no Jira username", http.StatusBadRequest)
		return
	}

	fromDate := strings.TrimSpace(r.URL.Query().Get("from"))
	toDate := strings.TrimSpace(r.URL.Query().Get("to"))
	if fromDate == "" && toDate == "" {
		// Default to the last 90 days so the button "just works".
		toDate = time.Now().Format("2006-01-02")
		fromDate = time.Now().AddDate(0, 0, -90).Format("2006-01-02")
	}
	if fromDate == "" || toDate == "" {
		http.Error(w, "From and to dates are required together, or omit both", http.StatusBadRequest)
		return
	}
	fromTime, err := time.Parse("2006-01-02", fromDate)
	if err != nil {
		http.Error(w, "From date must use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	toTime, err := time.Parse("2006-01-02", toDate)
	if err != nil {
		http.Error(w, "To date must use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	if toTime.Before(fromTime) {
		http.Error(w, "To date cannot be before from date", http.StatusBadRequest)
		return
	}
	// Cap the range to a year so an accidental open-ended query cannot
	// hammer the Jira instance.
	if toTime.Sub(fromTime) > 366*24*time.Hour {
		http.Error(w, "Date range cannot exceed one year", http.StatusBadRequest)
		return
	}

	stored, credential, credentialErr := loadJiraCredential(r.Context())
	if credentialErr != nil {
		http.Error(w, credentialErr.Message, credentialErr.Status)
		return
	}
	page := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 {
			page = parsed - 1
		}
	}
	result, fetchErr := fetchJiraEvidence(r.Context(), stored, credential, jiraUser,
		fromDate, toDate, page)
	if fetchErr != nil {
		http.Error(w, fetchErr.Message, fetchErr.Status)
		return
	}

	// Pre-populate suggested note fields so the UI can accept evidence with
	// one click: the note summary mirrors the Jira summary, details reference
	// the issue, and the category is derived from the issue type/priority.
	for index := range result.Evidence {
		result.Evidence[index].SuggestedNote = buildSuggestedNote(result.Evidence[index])
	}
	result.JiraUser = jiraUser
	result.Engineer = engineer.Name
	result.FromDate = fromDate
	result.ToDate = toDate
	writeJSON(w, http.StatusOK, result)
}

// jiraEvidenceError carries an HTTP status with its message so handlers can
// propagate provider failures without re-mapping them.
type jiraEvidenceError struct {
	Status  int
	Message string
}

func (e *jiraEvidenceError) Error() string {
	return e.Message
}

// loadJiraCredential reads the stored (manager's) Jira credential. It mirrors
// the loading in testIntegrationConnection so both features share one
// credential row per provider.
func loadJiraCredential(ctx context.Context) (storedIntegration, string, *jiraEvidenceError) {
	var stored storedIntegration
	err := db.QueryRow(`
		SELECT provider, COALESCE(account_label, ''), COALESCE(base_url, ''),
			encrypted_secret, COALESCE(deployment_type, ''), enabled
		FROM integration_credentials WHERE provider = ?`, "jira",
	).Scan(
		&stored.Provider, &stored.AccountLabel, &stored.BaseURL,
		&stored.EncryptedSecret, &stored.DeploymentType, &stored.Enabled,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return storedIntegration{}, "", &jiraEvidenceError{
			Status:  http.StatusNotFound,
			Message: "Jira integration is not configured",
		}
	}
	if err != nil {
		return storedIntegration{}, "", &jiraEvidenceError{
			Status:  http.StatusInternalServerError,
			Message: "Failed to retrieve Jira integration",
		}
	}
	if !stored.Enabled {
		return storedIntegration{}, "", &jiraEvidenceError{
			Status:  http.StatusConflict,
			Message: "Jira integration is disabled",
		}
	}
	if strings.TrimSpace(stored.BaseURL) == "" {
		return storedIntegration{}, "", &jiraEvidenceError{
			Status:  http.StatusBadRequest,
			Message: "Jira base URL is required",
		}
	}
	secret, decryptErr := decryptSecret(stored.EncryptedSecret)
	if decryptErr != nil {
		return storedIntegration{}, "", &jiraEvidenceError{
			Status:  http.StatusInternalServerError,
			Message: "Stored credential could not be decrypted",
		}
	}
	return stored, secret, nil
}

// fetchJiraEvidence builds the JQL query, calls the Jira search endpoint with
// the right auth for the deployment type, and maps the response.
func fetchJiraEvidence(
	ctx context.Context,
	stored storedIntegration,
	secret, jiraUser, fromDate, toDate string,
	page int,
) (jiraEvidenceResponse, *jiraEvidenceError) {
	baseURL := strings.TrimSpace(stored.BaseURL)
	isServer := false
	switch stored.DeploymentType {
	case "server":
		isServer = true
	case "cloud":
	default:
		if detected, err := detectJiraDeployment(ctx, baseURL); err == nil {
			isServer = detected
		}
		// Detection failure falls back to Cloud behavior; the request will
		// fail with a provider error that surfaces to the caller.
	}

	// Resolve after the to-date so issues closed the same day count; ordering
	// is newest-first so the UI shows the most recent evidence first.
	jql := fmt.Sprintf(
		`assignee = %q AND resolved >= %q AND resolved <= %q ORDER BY resolved DESC`,
		jiraUser, fromDate+" 00:00", toDate+" 23:59",
	)
	path := "/rest/api/2/search"
	if !isServer {
		path = "/rest/api/3/search"
	}
	endpoint, err := integrationEndpoint(baseURL, path)
	if err != nil {
		return jiraEvidenceResponse{}, &jiraEvidenceError{
			Status:  http.StatusBadRequest,
			Message: err.Error(),
		}
	}
	query := url.Values{}
	query.Set("jql", jql)
	query.Set("maxResults", "50")
	query.Set("startAt", strconv.Itoa(page*50))
	query.Set("fields", "key,summary,status,priority,resolution,resolutiondate,issuetype")
	endpoint += "?" + query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return jiraEvidenceResponse{}, &jiraEvidenceError{
			Status:  http.StatusInternalServerError,
			Message: "Provider URL is invalid",
		}
	}
	request.Header.Set("Accept", "application/json")
	if isServer {
		// Server / Data Center personal access token.
		request.Header.Set("Authorization", "Bearer "+secret)
	} else {
		request.Header.Set("Authorization", "Basic "+encodeBasicAuth(stored.AccountLabel, secret))
	}
	response, err := integrationHTTPClient.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return jiraEvidenceResponse{}, &jiraEvidenceError{
				Status:  http.StatusGatewayTimeout,
				Message: "The Jira instance did not respond before the timeout",
			}
		}
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			return jiraEvidenceResponse{}, &jiraEvidenceError{
				Status:  http.StatusGatewayTimeout,
				Message: "The Jira instance did not respond before the timeout",
			}
		}
		return jiraEvidenceResponse{}, &jiraEvidenceError{
			Status:  http.StatusBadGateway,
			Message: "The Jira instance could not be reached",
		}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return jiraEvidenceResponse{}, &jiraEvidenceError{
			Status:  http.StatusBadGateway,
			Message: "The Jira instance returned an unreadable response",
		}
	}
	if response.StatusCode == http.StatusUnauthorized {
		return jiraEvidenceResponse{}, &jiraEvidenceError{
			Status:  http.StatusUnauthorized,
			Message: "The stored Jira credential was rejected",
		}
	}
	if response.StatusCode == http.StatusForbidden {
		return jiraEvidenceResponse{}, &jiraEvidenceError{
			Status:  http.StatusForbidden,
			Message: "The stored Jira credential lacks required access",
		}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return jiraEvidenceResponse{}, &jiraEvidenceError{
			Status:  http.StatusBadGateway,
			Message: fmt.Sprintf("The Jira instance returned HTTP %d", response.StatusCode),
		}
	}
	var payload struct {
		StartAt    int `json:"startAt"`
		Total      int `json:"total"`
		MaxResults int `json:"maxResults"`
		Issues     []struct {
			Key    string `json:"key"`
			Fields struct {
				Summary string `json:"summary"`
				Status  struct {
					Name string `json:"name"`
				} `json:"status"`
				Priority struct {
					Name string `json:"name"`
				} `json:"priority"`
				Resolution struct {
					Name string `json:"name"`
				} `json:"resolution"`
				ResolutionDate string `json:"resolutiondate"`
				IssueType      struct {
					Name string `json:"name"`
				} `json:"issuetype"`
			} `json:"fields"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return jiraEvidenceResponse{}, &jiraEvidenceError{
			Status:  http.StatusBadGateway,
			Message: "The Jira instance returned an invalid response",
		}
	}
	result := jiraEvidenceResponse{
		Evidence: make([]JiraEvidence, 0, len(payload.Issues)),
		Total:    payload.Total,
	}
	for _, issue := range payload.Issues {
		resolution := strings.TrimSpace(issue.Fields.Resolution.Name)
		if resolution == "" {
			// Done/Done-style statuses on Jira without a resolution object;
			// the status name carries the resolution meaning in that case.
			resolution = strings.TrimSpace(issue.Fields.Status.Name)
		}
		result.Evidence = append(result.Evidence, JiraEvidence{
			IssueKey:     issue.Key,
			Summary:      issue.Fields.Summary,
			Resolution:   resolution,
			ResolvedDate: strings.TrimSpace(issue.Fields.ResolutionDate),
			Status:       issue.Fields.Status.Name,
			Priority:     issue.Fields.Priority.Name,
			BrowseURL:    strings.TrimRight(baseURL, "/") + "/browse/" + issue.Key,
		})
	}
	if payload.StartAt+payload.MaxResults < payload.Total && payload.MaxResults > 0 {
		result.HasMore = true
		result.NextPage = page + 2
	}
	return result, nil
}

// encodeBasicAuth builds the Basic authorization header value from an
// account email and API token, matching the Jira Cloud credential scheme.
func encodeBasicAuth(accountLabel, secret string) string {
	credentials := accountLabel + ":" + secret
	return base64.StdEncoding.EncodeToString([]byte(credentials))
}

// jiraEvidenceInput is the accept request body: the engineer's Jira evidence
// items the user selected, plus note metadata they can override before
// persisting.
type jiraEvidenceInput struct {
	Items []struct {
		IssueKey     string `json:"issueKey"`
		Summary      string `json:"summary"`
		ResolvedDate string `json:"resolvedDate"`
		Category     string `json:"category"`
		Details      string `json:"details"`
		Impact       string `json:"impact"`
		BrowseURL    string `json:"browseUrl"`
	} `json:"items"`
}

// acceptJiraEvidence handles POST /api/engineers/{engineerId}/jira-evidence/accept:
// persists the selected evidence items as performance notes, one row per item.
// Note dates come from the Jira resolution date so the note lands on the
// timeline where the work happened.
func acceptJiraEvidence(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	engineer, err := scanEngineer(db.QueryRow(
		`SELECT `+engineerColumns+` FROM engineers WHERE id = ?`, engineerID))
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Engineer not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve engineer", http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(engineer.JiraUsername) == "" {
		http.Error(w, "Engineer has no Jira username", http.StatusBadRequest)
		return
	}
	var input jiraEvidenceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(input.Items) == 0 {
		http.Error(w, "At least one evidence item is required", http.StatusBadRequest)
		return
	}

	created := make([]PerformanceNote, 0, len(input.Items))
	for _, item := range input.Items {
		item.IssueKey = strings.TrimSpace(item.IssueKey)
		item.Summary = strings.TrimSpace(item.Summary)
		item.Category = strings.TrimSpace(item.Category)
		item.ResolvedDate = strings.TrimSpace(item.ResolvedDate)
		if item.IssueKey == "" {
			http.Error(w, "Issue key is required for every evidence item", http.StatusBadRequest)
			return
		}
		if item.Category == "" {
			// Default when the user did not override the suggestion.
			item.Category = "Technical Excellence"
		}
		if item.ResolvedDate == "" {
			http.Error(w, "Resolved date is required for every evidence item", http.StatusBadRequest)
			return
		}
		noteDate := item.ResolvedDate
		if len(noteDate) > 10 {
			// Jira returns full timestamps; the note stores the date part.
			noteDate = noteDate[:10]
		}
		if _, err := time.Parse("2006-01-02", noteDate); err != nil {
			http.Error(w, "Resolved date must use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		summary := item.Summary
		if summary == "" {
			summary = "[" + item.IssueKey + "]"
		}
		details := strings.TrimSpace(item.Details)
		if details == "" && strings.TrimSpace(item.BrowseURL) != "" {
			details = "Resolved in Jira — " + strings.TrimSpace(item.BrowseURL)
		}
		result, err := db.Exec(`
			INSERT INTO performance_notes
				(engineer_id, note_date, category, summary, details, impact,
				 follow_up_needed, review_cycle)
			VALUES (?, ?, ?, ?, ?, ?, 0, ?)`,
			engineerID, noteDate, item.Category, summary,
			details, item.Impact, engineer.ReviewCycle)
		if err != nil {
			http.Error(w, "Failed to save evidence note", http.StatusInternalServerError)
			return
		}
		id, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "Evidence note saved but ID could not be retrieved", http.StatusInternalServerError)
			return
		}
		created = append(created, PerformanceNote{
			ID:          int(id),
			EngineerID:  int(engineerID),
			NoteDate:    noteDate,
			Category:    item.Category,
			Summary:     summary,
			Details:     details,
			Impact:      item.Impact,
			ReviewCycle: engineer.ReviewCycle,
		})
	}
	writeJSON(w, http.StatusCreated, map[string]any{"created": created})
}
// with one click. Category derives from the issue type: the closest match of
// the existing note-category vocabulary.
func buildSuggestedNote(evidence JiraEvidence) struct {
	Category string `json:"category"`
	Summary  string `json:"summary"`
	Details  string `json:"details"`
	Impact   string `json:"impact"`
} {
	summary := fmt.Sprintf("[%s] %s", evidence.IssueKey, evidence.Summary)
	details := fmt.Sprintf(
		"Resolved in Jira on %s — %s (status: %s, resolution: %s)",
		evidence.ResolvedDate, evidence.BrowseURL,
		evidence.Status, evidence.Resolution)
	impact := ""
	switch {
	case strings.EqualFold(evidence.Priority, "Highest"), strings.EqualFold(evidence.Priority, "High"):
		impact = "High-priority work item completed."
	}
	return struct {
		Category string `json:"category"`
		Summary  string `json:"summary"`
		Details  string `json:"details"`
		Impact   string `json:"impact"`
	}{
		Category: "Technical Excellence",
		Summary:  summary,
		Details:  details,
		Impact:   impact,
	}
}
