package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const onboardingProfileColumns = `
	id, engineer_id, COALESCE(answers, '{}'), COALESCE(meeting_date, ''),
	created_at, updated_at`

func getOnboardingProfile(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	profile, err := scanOnboardingProfile(db.QueryRow(
		`SELECT `+onboardingProfileColumns+
			` FROM onboarding_profiles WHERE engineer_id = ?`, engineerID))
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Onboarding profile not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve onboarding profile", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func upsertOnboardingProfile(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	profile, err := decodeAndValidateOnboardingProfile(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	profile.EngineerID = engineerID

	answersJSON, err := json.Marshal(profile.Answers)
	if err != nil {
		http.Error(w, "Failed to encode onboarding answers", http.StatusInternalServerError)
		return
	}
	_, err = db.Exec(`
		INSERT INTO onboarding_profiles (engineer_id, answers, meeting_date)
		VALUES (?, ?, NULLIF(?, ''))
		ON CONFLICT(engineer_id) DO UPDATE SET
			answers = excluded.answers,
			meeting_date = excluded.meeting_date,
			updated_at = CURRENT_TIMESTAMP`,
		profile.EngineerID, string(answersJSON), profile.MeetingDate)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			http.Error(w, "Engineer not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to save onboarding profile", http.StatusInternalServerError)
		return
	}
	saved, err := scanOnboardingProfile(db.QueryRow(
		`SELECT `+onboardingProfileColumns+
			` FROM onboarding_profiles WHERE engineer_id = ?`, profile.EngineerID))
	if err != nil {
		http.Error(w, "Onboarding profile saved but could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func parseOnboardingFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, "Upload is invalid or exceeds the 1 MB limit", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "A file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	buf, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read uploaded file", http.StatusBadRequest)
		return
	}
	answers := parseOnboardingDocument(string(buf))
	writeJSON(w, http.StatusOK, answers)
}

func decodeAndValidateOnboardingProfile(r *http.Request) (OnboardingProfile, error) {
	var profile OnboardingProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		return OnboardingProfile{}, errors.New("invalid request body")
	}
	profile.MeetingDate = strings.TrimSpace(profile.MeetingDate)
	if profile.MeetingDate != "" {
		if _, err := time.Parse("2006-01-02", profile.MeetingDate); err != nil {
			return OnboardingProfile{}, errors.New("meeting date must use YYYY-MM-DD")
		}
	}
	return profile, nil
}

type onboardingProfileScanner interface {
	Scan(dest ...any) error
}

func scanOnboardingProfile(scanner onboardingProfileScanner) (OnboardingProfile, error) {
	var profile OnboardingProfile
	var answersJSON string
	err := scanner.Scan(
		&profile.ID, &profile.EngineerID, &answersJSON, &profile.MeetingDate,
		&profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		return OnboardingProfile{}, err
	}
	if answersJSON != "" {
		if err := json.Unmarshal([]byte(answersJSON), &profile.Answers); err != nil {
			return OnboardingProfile{}, err
		}
	}
	return profile, nil
}
