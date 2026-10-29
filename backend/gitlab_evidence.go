package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// GitLabEvidence is one piece of GitLab activity surfaced as suggested review
// evidence. Merged merge requests and approvals are both represented: the
// roadmap forbids treating commit volume as productivity, so pushes are
// never harvested.
type GitLabEvidence struct {
	Repo       string `json:"repo"`
	Kind       string `json:"kind"` // "pr" (merged MR) or "review" (approval)
	Title      string `json:"title"`
	OccurredAt string `json:"occurredAt"`
	BrowseURL  string `json:"browseUrl"`
	// ProjectID and TargetIID are internal only: they locate the project web
	// path and merge-request number while the fetch resolves browse URLs and
	// are not serialized to the UI.
	ProjectID     int64 `json:"-"`
	TargetIID     int64 `json:"-"`
	SuggestedNote struct {
		Category string `json:"category"`
		Summary  string `json:"summary"`
		Details  string `json:"details"`
		Impact   string `json:"impact"`
	} `json:"suggestedNote"`
}

type gitlabEvidenceResponse struct {
	Evidence   []GitLabEvidence `json:"evidence"`
	Total      int              `json:"total"`
	FromDate   string           `json:"fromDate"`
	ToDate     string           `json:"toDate"`
	GitlabUser string           `json:"gitlabUser"`
	Engineer   string           `json:"engineer"`
	NextPage   int              `json:"nextPage,omitempty"`
	HasMore    bool             `json:"hasMore"`
}

// getGitLabEvidence handles GET /api/engineers/{engineerId}/gitlab-evidence:
// harvests the engineer's merged merge requests and approvals over a date
// range, using the stored (manager's) GitLab credential.
func getGitLabEvidence(w http.ResponseWriter, r *http.Request) {
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
	gitlabUser := strings.TrimSpace(engineer.GitlabUsername)
	if gitlabUser == "" {
		http.Error(w, "Engineer has no GitLab username", http.StatusBadRequest)
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
	// hammer the GitLab instance.
	if toTime.Sub(fromTime) > 366*24*time.Hour {
		http.Error(w, "Date range cannot exceed one year", http.StatusBadRequest)
		return
	}

	stored, credential, credentialErr := loadIntegrationCredential(r.Context(), "gitlab")
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
	result, fetchErr := fetchGitLabEvidence(r.Context(), stored, credential,
		gitlabUser, fromDate, toDate, page)
	if fetchErr != nil {
		http.Error(w, fetchErr.Message, fetchErr.Status)
		return
	}

	// Pre-populate suggested note fields so the UI can accept evidence with
	// one click: the note summary mirrors the activity, details reference the
	// merge request, and impact stays empty for the manager to write.
	for index := range result.Evidence {
		result.Evidence[index].SuggestedNote = buildGitLabSuggestedNote(result.Evidence[index])
	}
	result.GitlabUser = gitlabUser
	result.Engineer = engineer.Name
	result.FromDate = fromDate
	result.ToDate = toDate
	writeJSON(w, http.StatusOK, result)
}

const (
	gitlabEvidencePageSize = 50
	// Events pages are fetched to assemble one harvest page; the cap keeps a
	// sparse range from walking the whole history of a very active user.
	gitlabEvidenceMaxPages = 10
)

// fetchGitLabEvidence reads the GitLab user events feed — which supports
// after/before date filters — and keeps only merged merge requests and
// approvals. Pushes are excluded by design. The user is resolved from the
// username to a numeric ID first: gitlab.com restricts username-based profile
// access, so events must be addressed by ID. Project IDs are resolved to web
// paths once per fetch so browse URLs can be built.
func fetchGitLabEvidence(
	ctx context.Context,
	stored storedIntegration,
	secret, gitlabUser, fromDate, toDate string,
	page int,
) (gitlabEvidenceResponse, *evidenceError) {
	baseURL := strings.TrimRight(strings.TrimSpace(stored.BaseURL), "/")
	userID, resolveErr := gitLabResolveUserID(ctx, baseURL, secret, gitlabUser)
	if resolveErr != nil {
		return gitlabEvidenceResponse{}, resolveErr
	}
	// before is exclusive in the events API; the day after the to-date keeps
	// same-day events in range.
	beforeDate, _ := time.Parse("2006-01-02", toDate)
	before := beforeDate.AddDate(0, 0, 1).Format("2006-01-02")

	projectNames := map[int64]string{}
	var collected []GitLabEvidence
	seen := map[string]bool{}

	apiPage := page + 1
	for pagesFetched := 0; pagesFetched < gitlabEvidenceMaxPages; pagesFetched++ {
		endpoint := fmt.Sprintf(
			"%s/api/v4/users/%d/events?after=%s&before=%s&sort=desc&per_page=100&page=%d",
			baseURL, userID, fromDate, before, apiPage)
		body, next, requestErr := gitLabEventsRequest(ctx, endpoint, secret)
		if requestErr != nil {
			return gitlabEvidenceResponse{}, requestErr
		}
		var events []gitlabEvent
		if err := json.Unmarshal(body, &events); err != nil {
			return gitlabEvidenceResponse{}, &evidenceError{
				Status:  http.StatusBadGateway,
				Message: "GitLab returned an invalid response",
			}
		}
		for _, event := range events {
			evidence, ok := gitlabEventEvidence(event)
			if !ok {
				// Pushes, wiki activity, and other non-evidence actions.
				continue
			}
			// Dedupe only when the event carries its own URL: events without
			// target_url all share the empty key until the browse URLs are
			// resolved below, so keying on it would collapse distinct MRs.
			if evidence.BrowseURL != "" && seen[evidence.BrowseURL] {
				continue
			}
			if evidence.BrowseURL != "" {
				seen[evidence.BrowseURL] = true
			}
			collected = append(collected, evidence)
		}
		// The X-Next-Page header is authoritative in GitLab's paginated API.
		if next <= apiPage {
			break
		}
		apiPage = next
	}
	result := gitlabEvidenceResponse{
		FromDate: fromDate,
		ToDate:   toDate,
	}
	if page*gitlabEvidencePageSize < len(collected) {
		start := page * gitlabEvidencePageSize
		end := start + gitlabEvidencePageSize
		if end > len(collected) {
			end = len(collected)
		}
		result.Evidence = collected[start:end]
		if start+gitlabEvidencePageSize < len(collected) {
			result.HasMore = true
			result.NextPage = page + 2
		}
	} else {
		result.Evidence = []GitLabEvidence{}
	}

	// Resolve project web paths for any browse URL still missing one. Only
	// events without target_url need this; the per-fetch cache keeps it to
	// one lookup per distinct project.
	missing := map[int64]bool{}
	for index := range result.Evidence {
		if result.Evidence[index].BrowseURL == "" && result.Evidence[index].ProjectID > 0 {
			missing[result.Evidence[index].ProjectID] = true
		}
	}
	for projectID := range missing {
		path, lookupErr := gitLabProjectPath(ctx, baseURL, secret, projectID)
		if lookupErr != nil || path == "" {
			// Unresolvable projects keep URL-less evidence; the note details
			// still name the merge request.
			continue
		}
		projectNames[projectID] = path
	}
	for index := range result.Evidence {
		evidence := &result.Evidence[index]
		if path, ok := projectNames[evidence.ProjectID]; ok && path != "" {
			evidence.Repo = path
			evidence.BrowseURL = gitLabBrowseURL(baseURL, path, evidence.Kind, evidence.TargetIID)
		}
	}
	sortEvidence := result.Evidence
	sort.Slice(sortEvidence, func(i, j int) bool {
		return sortEvidence[i].OccurredAt > sortEvidence[j].OccurredAt
	})
	result.Evidence = sortEvidence
	result.Total = len(collected)
	return result, nil
}

type gitlabEvent struct {
	Action      string `json:"action_name"`
	TargetType  string `json:"target_type"`
	TargetTitle string `json:"target_title"`
	TargetIID   int64  `json:"target_iid"`
	TargetURL   string `json:"target_url"`
	ProjectID   int64  `json:"project_id"`
	CreatedAt   string `json:"created_at"`
}

// gitlabEventEvidence maps one event to evidence when the action is a merged
// merge request or an approval; pushes and everything else are skipped.
func gitlabEventEvidence(event gitlabEvent) (GitLabEvidence, bool) {
	var evidence GitLabEvidence
	if event.ProjectID == 0 || strings.TrimSpace(event.TargetTitle) == "" {
		return evidence, false
	}
	switch {
	case event.Action == "merged" && event.TargetType == "merge_request":
		evidence.Kind = "pr"
	case event.Action == "approved" && event.TargetType == "merge_request":
		evidence.Kind = "review"
	default:
		return evidence, false
	}
	evidence.ProjectID = event.ProjectID
	evidence.TargetIID = event.TargetIID
	evidence.Title = strings.TrimSpace(event.TargetTitle)
	evidence.OccurredAt = event.CreatedAt
	evidence.BrowseURL = strings.TrimSpace(event.TargetURL)
	return evidence, true
}

// gitLabEventsRequest performs one authenticated GitLab call with the shared
// integration transport and error mapping. It returns the body and the
// X-Next-Page header value (0 when there is no next page) so the caller can
// paginate without holding on to the response.
func gitLabEventsRequest(ctx context.Context, endpoint, secret string) ([]byte, int, *evidenceError) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, &evidenceError{
			Status:  http.StatusInternalServerError,
			Message: "Provider URL is invalid",
		}
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("PRIVATE-TOKEN", secret)
	response, err := integrationHTTPClient.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, 0, &evidenceError{
				Status:  http.StatusGatewayTimeout,
				Message: "GitLab did not respond before the timeout",
			}
		}
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			return nil, 0, &evidenceError{
				Status:  http.StatusGatewayTimeout,
				Message: "GitLab did not respond before the timeout",
			}
		}
		return nil, 0, &evidenceError{
			Status:  http.StatusBadGateway,
			Message: "GitLab could not be reached",
		}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, 0, &evidenceError{
			Status:  http.StatusBadGateway,
			Message: "GitLab returned an unreadable response",
		}
	}
	if response.StatusCode == http.StatusUnauthorized {
		return nil, 0, &evidenceError{
			Status:  http.StatusUnauthorized,
			Message: "The stored GitLab credential was rejected",
		}
	}
	if response.StatusCode == http.StatusForbidden {
		return nil, 0, &evidenceError{
			Status:  http.StatusForbidden,
			Message: "The stored GitLab credential lacks required access",
		}
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, 0, &evidenceError{
			Status:  http.StatusNotFound,
			Message: "Could not find GitLab user or the events feed",
		}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, 0, &evidenceError{
			Status:  http.StatusBadGateway,
			Message: fmt.Sprintf("GitLab returned HTTP %d", response.StatusCode),
		}
	}
	nextPage := 0
	if raw := strings.TrimSpace(response.Header.Get("X-Next-Page")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
			nextPage = parsed
		}
	}
	return body, nextPage, nil
}

// gitLabResolveUserID resolves a GitLab username to the numeric user ID the
// events API needs. On gitlab.com the username-addressed user and events
// endpoints can 404 for other users even with valid credentials (profile
// visibility), while the search API still finds the account. A username that
// matches multiple accounts resolves to the exact case-insensitive match when
// one exists, otherwise 400.
func gitLabResolveUserID(ctx context.Context, baseURL, secret, gitlabUser string) (int64, *evidenceError) {
	endpoint := baseURL + "/api/v4/users?username=" + url.QueryEscape(gitlabUser)
	body, _, requestErr := gitLabEventsRequest(ctx, endpoint, secret)
	if requestErr != nil {
		return 0, requestErr
	}
	var matches []struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(body, &matches); err != nil {
		return 0, &evidenceError{
			Status:  http.StatusBadGateway,
			Message: "GitLab returned an invalid response",
		}
	}
	for _, match := range matches {
		// The username= filter is exact but not case-sensitive; prefer the
		// literal match so "John.Smith" does not silently pick "john.smith".
		if strings.EqualFold(match.Username, gitlabUser) {
			return match.ID, nil
		}
	}
	return 0, &evidenceError{
		Status:  http.StatusNotFound,
		Message: fmt.Sprintf("Could not find GitLab user %s", gitlabUser),
	}
}

// gitLabProjectPath resolves one project ID to its path_with_namespace so
// browse URLs can be built for events that lack target_url.
func gitLabProjectPath(ctx context.Context, baseURL string, secret string, projectID int64) (string, *evidenceError) {
	endpoint := fmt.Sprintf("%s/api/v4/projects/%d", baseURL, projectID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", nil
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("PRIVATE-TOKEN", secret)
	response, err := integrationHTTPClient.Do(request)
	if err != nil {
		return "", nil
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", nil
	}
	var payload struct {
		PathWithNamespace string `json:"path_with_namespace"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil
	}
	return payload.PathWithNamespace, nil
}

// gitLabBrowseURL builds the web URL for a merge request so accepted notes
// stay traceable to their source.
func gitLabBrowseURL(baseURL, projectPath, kind string, targetIID int64) string {
	if projectPath == "" || targetIID <= 0 {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + "/" + projectPath + "/-/merge_requests/" + strconv.FormatInt(targetIID, 10)
}

// gitlabEvidenceInput is the accept request body: the engineer's GitLab
// evidence items the user selected, plus note metadata they can override
// before saving.
type gitlabEvidenceInput struct {
	Items []struct {
		Repo       string `json:"repo"`
		Kind       string `json:"kind"`
		Title      string `json:"title"`
		URL        string `json:"url"`
		OccurredAt string `json:"occurredAt"`
		Category   string `json:"category"`
		Details    string `json:"details"`
		Impact     string `json:"impact"`
	} `json:"items"`
}

// acceptGitLabEvidence handles POST /api/engineers/{engineerId}/gitlab-evidence/accept:
// persists the selected evidence items as performance notes, one row per item.
// Note dates come from when the activity happened so the note lands on the
// timeline where the work happened.
func acceptGitLabEvidence(w http.ResponseWriter, r *http.Request) {
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
	if strings.TrimSpace(engineer.GitlabUsername) == "" {
		http.Error(w, "Engineer has no GitLab username", http.StatusBadRequest)
		return
	}
	var input gitlabEvidenceInput
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
		item.Kind = strings.TrimSpace(item.Kind)
		item.Title = strings.TrimSpace(item.Title)
		item.Category = strings.TrimSpace(item.Category)
		item.OccurredAt = strings.TrimSpace(item.OccurredAt)
		if item.Kind == "" {
			http.Error(w, "Kind is required for every evidence item", http.StatusBadRequest)
			return
		}
		if item.Title == "" {
			http.Error(w, "Title is required for every evidence item", http.StatusBadRequest)
			return
		}
		if item.Category == "" {
			// Default when the user did not override the suggestion.
			item.Category = "Technical Excellence"
		}
		if item.OccurredAt == "" {
			http.Error(w, "Occurred date is required for every evidence item", http.StatusBadRequest)
			return
		}
		noteDate := item.OccurredAt
		if len(noteDate) > 10 {
			// GitLab returns full timestamps; the note stores the date part.
			noteDate = noteDate[:10]
		}
		if _, err := time.Parse("2006-01-02", noteDate); err != nil {
			http.Error(w, "Occurred date must use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		summary := item.Title
		if summary == "" {
			summary = item.Kind
		}
		details := strings.TrimSpace(item.Details)
		if details == "" {
			details = gitLabNoteDetails(item.Repo, strings.TrimSpace(item.URL), item.Kind)
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

// gitLabNoteDetails builds the traceable details line for a note whose
// details the user did not override, naming the project and linking the
// merge request.
func gitLabNoteDetails(repo, browseURL, kind string) string {
	place := repo
	if place == "" {
		place = "GitLab"
	}
	verb := "Merged MR in"
	if kind == "review" {
		verb = "Approved MR in"
	}
	details := verb + " " + place
	if browseURL != "" {
		details += " — " + browseURL
	}
	return details
}

// buildGitLabSuggestedNote pre-populates the note fields for one piece of
// GitLab evidence. Merges are technical excellence; approvals are team
// contribution. Impact stays empty so the manager records it themselves
// rather than the system inferring it from activity volume.
func buildGitLabSuggestedNote(evidence GitLabEvidence) struct {
	Category string `json:"category"`
	Summary  string `json:"summary"`
	Details  string `json:"details"`
	Impact   string `json:"impact"`
} {
	var category, summary string
	details := gitLabNoteDetails(evidence.Repo, evidence.BrowseURL, evidence.Kind)
	if evidence.OccurredAt != "" {
		details += " on " + strings.TrimSuffix(evidence.OccurredAt, "T00:00:00Z")
	}
	if evidence.Kind == "review" {
		category = "Team Contribution"
		summary = "Approved MR: " + evidence.Title
	} else {
		category = "Technical Excellence"
		summary = "Merged MR: " + evidence.Title
	}
	return struct {
		Category string `json:"category"`
		Summary  string `json:"summary"`
		Details  string `json:"details"`
		Impact   string `json:"impact"`
	}{
		Category: category,
		Summary:  summary,
		Details:  details,
		Impact:   "",
	}
}
