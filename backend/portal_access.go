package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
)

func lookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// createPortalAccess handles POST /api/engineers/{engineerId}/portal-access:
// the manager grants an engineer portal access by assigning an email and
// initial password.
func createPortalAccess(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	if _, err := scanEngineer(db.QueryRow(
		`SELECT `+engineerColumns+` FROM engineers WHERE id = ?`, engineerID)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Engineer not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to retrieve engineer", http.StatusInternalServerError)
		return
	}
	var input PortalAccessInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(input.Email)
	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	if len(input.Password) < minPasswordLength {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	var existing int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE email = ?`, email).Scan(&existing); err != nil {
		http.Error(w, "Failed to check email", http.StatusInternalServerError)
		return
	}
	if existing > 0 {
		http.Error(w, "Email is already in use", http.StatusConflict)
		return
	}
	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		http.Error(w, "Failed to secure password", http.StatusInternalServerError)
		return
	}
	result, err := db.Exec(`
		INSERT INTO users (email, password_hash, role, engineer_id)
		VALUES (?, ?, ?, ?)`, email, passwordHash, roleEngineer, engineerID)
	if err != nil {
		http.Error(w, "Failed to create portal access", http.StatusInternalServerError)
		return
	}
	userID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Portal access created but ID could not be retrieved", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, SessionUserResponse{
		ID: userID, Email: email, Role: roleEngineer, EngineerID: &engineerID,
	})
}

// updatePortalAccess handles PUT /api/engineers/{engineerId}/portal-access:
// reset the engineer's password (email updates are not supported in v1 —
// revoke and recreate instead).
func updatePortalAccess(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	var input PortalAccessInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	var userID int64
	err = db.QueryRow(
		`SELECT id FROM users WHERE engineer_id = ? AND role = ?`,
		engineerID, roleEngineer).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Engineer has no portal access", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to look up portal access", http.StatusInternalServerError)
		return
	}
	if len(input.Password) < minPasswordLength {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		http.Error(w, "Failed to secure password", http.StatusInternalServerError)
		return
	}
	// Resetting the password revokes existing sessions so an old session
	// cannot survive a credential hand-back.
	if _, err := db.Exec(`
		UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, passwordHash, userID); err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}
	if _, err := db.Exec(`
		DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		http.Error(w, "Password updated but sessions could not be revoked", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deletePortalAccess handles DELETE /api/engineers/{engineerId}/portal-access:
// removes the engineer's login; sessions cascade away with the user row.
func deletePortalAccess(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(
		`DELETE FROM users WHERE engineer_id = ? AND role = ?`,
		engineerID, roleEngineer)
	if err != nil {
		http.Error(w, "Failed to remove portal access", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm portal access removal", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Engineer has no portal access", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// getPortalAccess handles GET /api/engineers/{engineerId}/portal-access:
// returns the engineer's login email (or hasAccess=false) so the profile page
// can render the current state without exposing credentials.
func getPortalAccess(w http.ResponseWriter, r *http.Request) {
	engineerID, err := positiveID(chi.URLParam(r, "engineerId"))
	if err != nil {
		http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
		return
	}
	var userID int64
	var email string
	err = db.QueryRow(
		`SELECT id, email FROM users WHERE engineer_id = ? AND role = ?`,
		engineerID, roleEngineer).Scan(&userID, &email)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusOK, map[string]any{"hasAccess": false})
		return
	}
	if err != nil {
		http.Error(w, "Failed to look up portal access", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hasAccess": true, "email": email})
}

// seedManagerFromEnv bootstraps the manager account from environment
// variables so a fresh database is immediately usable in open mode (no
// accounts) and becomes authenticated mode on first start with credentials.
func seedManagerFromEnv() error {
	email := strings.TrimSpace(getenvDefault("MANAGER_EMAIL", ""))
	password := getenvDefault("MANAGER_PASSWORD", "")
	if email == "" || password == "" {
		return nil
	}
	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE role = ?`, roleManager).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO users (email, password_hash, role, engineer_id)
		VALUES (?, ?, ?, NULL)`, email, passwordHash, roleManager)
	return err
}

func getenvDefault(key, fallback string) string {
	value, ok := lookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	return value
}
