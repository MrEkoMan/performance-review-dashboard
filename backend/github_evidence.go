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

// GitHubEvidence is one piece of GitHub activity surfaced as suggested review
// evidence. Merges and reviews are both represented: the roadmap forbids
// treating commit volume as productivity, so pushes are never harvested.
type GitHubEvidence struct {
	Repo          string `json:"repo"`
	Kind          string `json:"kind"` // "pr" (merged) or "review"
	Title         string `json:"title"`
	OccurredAt    string `json:"occurredAt"`
	State         string `json:"state,omitempty"`
	BrowseURL     string `json:"browseUrl"`
	SuggestedNote struct {
		Category string `json:"category"`
		Summary  string `json:"summary"`
		Details  string `json:"details"`
		Impact   string `json:"impact"`
	} `json:"suggestedNote"`
}

type githubEvidenceResponse struct {
	Evidence   []GitHubEvidence `json:"evidence"`
	Total      int              `json:"total"`
	FromDate   string           `json:"fromDate"`
	ToDate     string           `json:"toDate"`
	GitHubUser string           `json:"gitHubUser"`
	Engineer   string           `json:"engineer"`
	NextPage   int              `json:"nextPage,omitempty"`
	HasMore    bool             `json:"hasMore"`
}

// getGitHubEvidence handles GET /api/engineers/{engineerId}/github-evidence:
// harvests the engineer's merged pull requests and review activity over a
// date range, using the stored (manager's) GitHub credential.
func getGitHubEvidence(w http.ResponseWriter, r *http.Request) {
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
	gitHubUser := strings.TrimSpace(engineer.GitHubUsername)
	if gitHubUser == "" {
		http.Error(w, "Engineer has no GitHub username", http.StatusBadRequest)
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
	// hammer the GitHub API.
	if toTime.Sub(fromTime) > 366*24*time.Hour {
		http.Error(w, "Date range cannot exceed one year", http.StatusBadRequest)
		return
	}

	stored, credential, credentialErr := loadIntegrationCredential(r.Context(), "github")
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
	result, fetchErr := fetchGitHubEvidence(r.Context(), stored, credential,
		gitHubUser, fromDate, toDate, page)
	if fetchErr != nil {
		http.Error(w, fetchErr.Message, fetchErr.Status)
		return
	}

	// Pre-populate suggested note fields so the UI can accept evidence with
	// one click: the note summary mirrors the activity, details reference the
	// pull request, and impact stays empty for the manager to write.
	for index := range result.Evidence {
		result.Evidence[index].SuggestedNote = buildGitHubSuggestedNote(result.Evidence[index])
	}
	result.GitHubUser = gitHubUser
	result.Engineer = engineer.Name
	result.FromDate = fromDate
	result.ToDate = toDate
	writeJSON(w, http.StatusOK, result)
}

const gitHubEvidencePageSize = 50

// fetchGitHubEvidence queries the GitHub Search API — not the events API,
// which only covers the last 30 days and public activity — for merged pull
// requests authored by the engineer and pull requests they reviewed, merges
// the results, dedupes them, and applies client-side pagination. Each search
// query is capped at 1000 results by GitHub; the per-fetch page size bounds
// what reaches the UI.
func fetchGitHubEvidence(
	ctx context.Context,
	stored storedIntegration,
	secret, gitHubUser, fromDate, toDate string,
	page int,
) (githubEvidenceResponse, *evidenceError) {
	queries := []struct {
		qualifiers string
		kind       string
		dateField  string
	}{
		{fmt.Sprintf("author:%s is:pr is:merged merged:%s..%s", gitHubUser, fromDate, toDate),
			"pr", "closed_at"},
		{fmt.Sprintf("reviewed-by:%s is:pr -author:%s updated:%s..%s", gitHubUser, gitHubUser, fromDate, toDate),
			"review", "updated_at"},
	}

	baseURL := strings.TrimSpace(stored.BaseURL)
	endpoint, err := integrationEndpoint(baseURL, "/search/issues")
	if err != nil {
		return githubEvidenceResponse{}, &evidenceError{
			Status:  http.StatusBadRequest,
			Message: err.Error(),
		}
	}

	var collected []GitHubEvidence
	seen := map[string]bool{}
	total := 0
	for _, search := range queries {
		query := url.Values{}
		query.Set("q", search.qualifiers)
		query.Set("sort", "updated")
		query.Set("order", "desc")
		query.Set("per_page", "100")
		query.Set("page", strconv.Itoa(page+1))
		response, fetchErr := gitHubSearchRequest(ctx, endpoint+"?"+query.Encode(), secret)
		if fetchErr != nil {
			return githubEvidenceResponse{}, fetchErr
		}
		var payload struct {
			TotalCount int `json:"total_count"`
			Items      []struct {
				Title      string `json:"title"`
				HTMLURL    string `json:"html_url"`
				State      string `json:"state"`
				ClosedAt   string `json:"closed_at"`
				UpdatedAt  string `json:"updated_at"`
				Repository struct {
					FullName string `json:"full_name"`
				} `json:"repository"`
				RepositoryURL string `json:"repository_url"`
			} `json:"items"`
		}
		if err := json.Unmarshal(response, &payload); err != nil {
			return githubEvidenceResponse{}, &evidenceError{
				Status:  http.StatusBadGateway,
				Message: "GitHub returned an invalid response",
			}
		}
		total += payload.TotalCount
		for _, item := range payload.Items {
			// The review search can surface PRs the engineer authored too
			// (GitHub's indexed field set shifts); the -author qualifier is
			// best-effort, so dedupe by URL here rather than trusting it.
			if seen[item.HTMLURL] {
				continue
			}
			seen[item.HTMLURL] = true
			occurredAt := item.UpdatedAt
			if search.dateField == "closed_at" && item.ClosedAt != "" {
				occurredAt = item.ClosedAt
			}
			collected = append(collected, GitHubEvidence{
				Repo:       repositoryName(item.Repository.FullName, item.RepositoryURL),
				Kind:       search.kind,
				Title:      item.Title,
				OccurredAt: occurredAt,
				State:      item.State,
				BrowseURL:  item.HTMLURL,
			})
		}
	}
	// Newest first, matching the Jira harvest ordering.
	sort.Slice(collected, func(i, j int) bool {
		return collected[i].OccurredAt > collected[j].OccurredAt
	})

	start := page * gitHubEvidencePageSize
	result := githubEvidenceResponse{
		Total:    len(collected),
		FromDate: fromDate,
		ToDate:   toDate,
	}
	if start < len(collected) {
		end := start + gitHubEvidencePageSize
		if end > len(collected) {
			end = len(collected)
		}
		result.Evidence = collected[start:end]
	} else {
		result.Evidence = []GitHubEvidence{}
	}
	if start+gitHubEvidencePageSize < len(collected) {
		result.HasMore = true
		result.NextPage = page + 2
	}
	return result, nil
}

// repositoryName prefers the search payload's repository.full_name and falls
// back to parsing ".../repos/{owner}/{repo}" out of repository_url.
func repositoryName(fullName, repositoryURL string) string {
	if trimmed := strings.TrimSpace(fullName); trimmed != "" {
		return trimmed
	}
	marker := "/repos/"
	if index := strings.LastIndex(repositoryURL, marker); index >= 0 {
		return repositoryURL[index+len(marker):]
	}
	return ""
}

// gitHubSearchRequest performs one authenticated GitHub search call with the
// shared integration transport and error mapping.
func gitHubSearchRequest(ctx context.Context, endpoint, secret string) ([]byte, *evidenceError) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, &evidenceError{
			Status:  http.StatusInternalServerError,
			Message: "Provider URL is invalid",
		}
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("Authorization", "Bearer "+secret)
	response, err := integrationHTTPClient.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, &evidenceError{
				Status:  http.StatusGatewayTimeout,
				Message: "GitHub did not respond before the timeout",
			}
		}
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			return nil, &evidenceError{
				Status:  http.StatusGatewayTimeout,
				Message: "GitHub did not respond before the timeout",
			}
		}
		return nil, &evidenceError{
			Status:  http.StatusBadGateway,
			Message: "GitHub could not be reached",
		}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, &evidenceError{
			Status:  http.StatusBadGateway,
			Message: "GitHub returned an unreadable response",
		}
	}
	if response.StatusCode == http.StatusUnauthorized {
		return nil, &evidenceError{
			Status:  http.StatusUnauthorized,
			Message: "The stored GitHub credential was rejected",
		}
	}
	if response.StatusCode == http.StatusForbidden {
		return nil, &evidenceError{
			Status:  http.StatusForbidden,
			Message: "The stored GitHub credential lacks required access",
		}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, &evidenceError{
			Status:  http.StatusBadGateway,
			Message: fmt.Sprintf("GitHub returned HTTP %d", response.StatusCode),
		}
	}
	return body, nil
}

// gitHubEvidenceInput is the accept request body: the engineer's GitHub
// evidence items the user selected, plus note metadata they can override
// before saving.
type gitHubEvidenceInput struct {
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

// acceptGitHubEvidence handles POST /api/engineers/{engineerId}/github-evidence/accept:
// persists the selected evidence items as performance notes, one row per item.
// Note dates come from when the activity happened so the note lands on the
// timeline where the work happened.
func acceptGitHubEvidence(w http.ResponseWriter, r *http.Request) {
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
	if strings.TrimSpace(engineer.GitHubUsername) == "" {
		http.Error(w, "Engineer has no GitHub username", http.StatusBadRequest)
		return
	}
	var input gitHubEvidenceInput
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
			// GitHub returns full timestamps; the note stores the date part.
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
			details = gitHubNoteDetails(item.Repo, strings.TrimSpace(item.URL), item.Kind)
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

// gitHubNoteDetails builds the traceable details line for a note whose
// details the user did not override, naming the repository and linking the
// pull request.
func gitHubNoteDetails(repo, browseURL, kind string) string {
	place := repo
	if place == "" {
		place = "GitHub"
	}
	verb := "Merged in"
	if kind == "review" {
		verb = "Reviewed"
	}
	details := verb + " " + place
	if browseURL != "" {
		details += " — " + browseURL
	}
	return details
}

// buildGitHubSuggestedNote pre-populates the note fields for one piece of
// GitHub evidence. Merges are technical excellence; reviews are team
// contribution. Impact stays empty so the manager records it themselves
// rather than the system inferring it from activity volume.
func buildGitHubSuggestedNote(evidence GitHubEvidence) struct {
	Category string `json:"category"`
	Summary  string `json:"summary"`
	Details  string `json:"details"`
	Impact   string `json:"impact"`
} {
	var category, summary string
	details := gitHubNoteDetails(evidence.Repo, evidence.BrowseURL, evidence.Kind)
	if evidence.OccurredAt != "" {
		details += " on " + strings.TrimSuffix(evidence.OccurredAt, "T00:00:00Z")
	}
	if evidence.Kind == "review" {
		category = "Team Contribution"
		summary = "Reviewed PR: " + evidence.Title
	} else {
		category = "Technical Excellence"
		summary = "Merged PR: " + evidence.Title
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
