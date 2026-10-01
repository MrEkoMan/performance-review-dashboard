package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	roleManager  = "manager"
	roleEngineer = "engineer"

	sessionCookieName = "md_session"
	sessionDuration   = 7 * 24 * time.Hour

	// PBKDF2-SHA256 parameters. 600k iterations follows current OWASP
	// guidance for password storage; the salt and digest are stored inline
	// so the parameters can be raised later without rehashing explicitly.
	pbkdf2Iterations = 600000
	pbkdf2KeyLength  = 32
	pbkdf2SaltLength = 16

	minPasswordLength = 8
)

// AuthUser is the resolved identity of the caller. EngineerID is the
// engineer a login is bound to (zero for managers); OpenMode marks the
// pre-authentication state where no accounts exist yet and every caller is
// treated as the manager so the local tool keeps working unchanged.
type AuthUser struct {
	UserID     int64
	Email      string
	Role       string
	EngineerID int64
	OpenMode   bool
}

type authUserContextKey struct{}

func withAuthUser(ctx context.Context, user AuthUser) context.Context {
	return context.WithValue(ctx, authUserContextKey{}, user)
}

func currentUser(r *http.Request) (AuthUser, bool) {
	user, ok := r.Context().Value(authUserContextKey{}).(AuthUser)
	return user, ok
}

// authOpenMode reports whether authentication is dormant because no accounts
// exist yet. In that state every request is treated as the manager, matching
// the pre-auth behavior of the tool. As soon as any account exists (manager
// registered through the UI, seeded from environment variables, or an
// engineer granted portal access), sessions are required.
func authOpenMode() (bool, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return false, err
	}
	return count == 0, nil
}

// managerExists reports whether a manager account has been registered.
func managerExists() (bool, error) {
	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE role = ?`, roleManager).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// authenticate resolves the session cookie into an AuthUser. Requests made
// while authentication is open-mode are treated as the manager. The auth
// endpoints are skipped by the session gate but still get the resolved user
// attached when a valid session exists, so /api/auth/me works.
func authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		open, err := authOpenMode()
		if err != nil {
			http.Error(w, "Failed to verify authentication state", http.StatusInternalServerError)
			return
		}
		if open {
			if isPublicAuthPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(withAuthUser(r.Context(),
				AuthUser{Role: roleManager, OpenMode: true})))
			return
		}
		user, found, err := resolveSessionUser(r)
		if err != nil {
			http.Error(w, "Failed to resolve session", http.StatusInternalServerError)
			return
		}
		if found {
			r = r.WithContext(withAuthUser(r.Context(), user))
		} else if !isPublicAuthPath(r.URL.Path) {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// registrationStatusHandler handles GET /api/auth/registration-status:
// tells the login page whether the manager account can still be claimed
// (no accounts exist, or no manager exists yet) and whether an engineer
// account may be registered by the signed-in manager.
func registrationStatusHandler(w http.ResponseWriter, r *http.Request) {
	open, err := authOpenMode()
	if err != nil {
		http.Error(w, "Failed to verify authentication state", http.StatusInternalServerError)
		return
	}
	hasManager, err := managerExists()
	if err != nil {
		http.Error(w, "Failed to verify manager state", http.StatusInternalServerError)
		return
	}
	// Claiming is possible when the database holds no accounts at all —
	// that is the lockout-recovery path. If engineer-only accounts exist
	// without a manager, registration is deliberately not offered (that
	// state is recoverable only through the environment variables).
	user, _ := currentUser(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"openMode":          open,
		"canClaimManager":   open,
		"canRegisterUsers":  hasManager && user.Role == roleManager,
	})
}

// isPublicAuthPath matches the auth endpoints that must be reachable without
// a session. registration-status is public so the login page can decide
// whether to offer the manager claiming flow.
func isPublicAuthPath(path string) bool {
	switch path {
	case "/api/auth/login", "/api/auth/logout", "/api/auth/me",
		"/api/auth/register", "/api/auth/registration-status":
		return true
	}
	return false
}

// managerOnly guards handlers that only the manager may call.
func managerOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok || user.Role != roleManager {
			http.Error(w, "Manager access required", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// engineerSelf guards engineer-scoped routes: the manager may read any
// engineer, a logged-in engineer only their own record. Routes without an
// engineerId path parameter (like /api/notes) rely on handler-level scoping.
func engineerSelf(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		if user.Role != roleManager {
			if param := chi.URLParam(r, "engineerId"); param != "" {
				requested, err := positiveID(param)
				if err != nil {
					http.Error(w, "Invalid engineer ID", http.StatusBadRequest)
					return
				}
				if requested != user.EngineerID {
					http.Error(w, "You can only access your own records", http.StatusForbidden)
					return
				}
			}
		}
		next(w, r)
	}
}

// engineerOwnNote allows only the note's engineer or the manager — used for
// note attachment reads, where the note ID is in the URL but the engineer is
// not.
func engineerOwnNote(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		if user.Role == roleManager {
			next(w, r)
			return
		}
		noteID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil || noteID <= 0 {
			http.Error(w, "Invalid note ID", http.StatusBadRequest)
			return
		}
		var owner int64
		if err := db.QueryRow(
			`SELECT engineer_id FROM performance_notes WHERE id = ?`, noteID).
			Scan(&owner); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Note not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to retrieve note", http.StatusInternalServerError)
			return
		}
		if owner != user.EngineerID {
			http.Error(w, "You can only access your own records", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// engineerOwnAttachment guards attachment content reads: the attachment must
// belong to a note or recognition owned by the calling engineer.
func engineerOwnAttachment(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		if user.Role == roleManager {
			next(w, r)
			return
		}
		attachmentID, err := positiveID(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "Invalid attachment ID", http.StatusBadRequest)
			return
		}
		var engineerID int64
		err = db.QueryRow(`
			SELECT n.engineer_id
			FROM performance_note_attachments pna
			JOIN performance_notes n ON n.id = pna.note_id
			WHERE pna.attachment_id = ?
			UNION ALL
			SELECT rec.engineer_id
			FROM recognition_attachments ra
			JOIN recognitions rec ON rec.id = ra.recognition_id
			WHERE ra.attachment_id = ?`, attachmentID, attachmentID).Scan(&engineerID)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Attachment not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Failed to resolve attachment owner", http.StatusInternalServerError)
			return
		}
		if engineerID != user.EngineerID {
			http.Error(w, "You can only access your own records", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// --- Password hashing -------------------------------------------------------

func hashPassword(password string) (string, error) {
	salt := make([]byte, pbkdf2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	digest := pbkdf2SHA256([]byte(password), salt, pbkdf2Iterations, pbkdf2KeyLength)
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s",
		pbkdf2Iterations,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(digest)), nil
}

func verifyPassword(stored, password string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < 1 {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expected, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	actual := pbkdf2SHA256([]byte(password), salt, iterations, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

// pbkdf2SHA256 is the PBKDF2 key-derivation function with HMAC-SHA256 as the
// pseudo-random function (RFC 8018). Implemented in-package so the tool keeps
// its dependency set unchanged.
func pbkdf2SHA256(password, salt []byte, iterations, keyLength int) []byte {
	prf := hmac.New(sha256.New, password)
	hashLength := prf.Size()
	blocks := (keyLength + hashLength - 1) / hashLength

	var counter [4]byte
	derived := make([]byte, 0, blocks*hashLength)
	block := make([]byte, hashLength)
	for i := 1; i <= blocks; i++ {
		prf.Reset()
		prf.Write(salt)
		binary.BigEndian.PutUint32(counter[:], uint32(i))
		prf.Write(counter[:])
		derived = prf.Sum(derived)
		target := derived[len(derived)-hashLength:]
		copy(block, target)
		for n := 2; n <= iterations; n++ {
			prf.Reset()
			prf.Write(block)
			u := prf.Sum(nil)
			for x := range u {
				target[x] ^= u[x]
			}
			copy(block, u)
		}
	}
	return derived
}

// --- Sessions ----------------------------------------------------------------

func sessionTokenHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", digest)
}

// createSession mints a random opaque token and stores only its SHA-256 so a
// database compromise cannot resurrect live sessions.
func createSession(userID int64) (string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expires := time.Now().Add(sessionDuration)
	if _, err := db.Exec(`
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES (?, ?, ?)`,
		sessionTokenHash(token), userID, expires.UTC().Format(time.RFC3339)); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

func resolveSessionUser(r *http.Request) (AuthUser, bool, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return AuthUser{}, false, nil
	}
	tokenHash := sessionTokenHash(cookie.Value)
	var userID int64
	var expiresAt string
	err = db.QueryRow(`
		SELECT user_id, expires_at FROM sessions WHERE token_hash = ?`, tokenHash).
		Scan(&userID, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return AuthUser{}, false, nil
	}
	if err != nil {
		return AuthUser{}, false, err
	}
	if expiry, parseErr := time.Parse(time.RFC3339, expiresAt); parseErr == nil &&
		time.Now().After(expiry) {
		_, _ = db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
		return AuthUser{}, false, nil
	}
	var user AuthUser
	var engineerID sql.NullInt64
	err = db.QueryRow(`
		SELECT id, email, role, engineer_id FROM users WHERE id = ?`, userID).
		Scan(&user.UserID, &user.Email, &user.Role, &engineerID)
	if errors.Is(err, sql.ErrNoRows) {
		return AuthUser{}, false, nil
	}
	if err != nil {
		return AuthUser{}, false, err
	}
	user.EngineerID = engineerID.Int64
	return user, true, nil
}

func setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// --- Auth endpoints -----------------------------------------------------------

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(input.Email)
	if email == "" || input.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}
	var user AuthUser
	var engineerID sql.NullInt64
	var engineerName string
	var storedHash string
	err := db.QueryRow(`
		SELECT u.id, u.email, u.role, u.engineer_id, u.password_hash,
			COALESCE(e.name, '')
		FROM users u
		LEFT JOIN engineers e ON e.id = u.engineer_id
		WHERE u.email = ?`, email).
		Scan(&user.UserID, &user.Email, &user.Role, &engineerID, &storedHash, &engineerName)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "Failed to look up account", http.StatusInternalServerError)
		return
	}
	if !verifyPassword(storedHash, input.Password) {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}
	token, expires, err := createSession(user.UserID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}
	user.EngineerID = engineerID.Int64
	setSessionCookie(w, token, expires)
	writeJSON(w, http.StatusOK, SessionUserResponse{
		ID: user.UserID, Email: user.Email, Role: user.Role,
		EngineerID: nullableInt(user.EngineerID), Name: engineerName,
	})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		if _, err := db.Exec(`DELETE FROM sessions WHERE token_hash = ?`,
			sessionTokenHash(cookie.Value)); err != nil {
			http.Error(w, "Failed to end session", http.StatusInternalServerError)
			return
		}
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	response := SessionUserResponse{
		ID: user.UserID, Email: user.Email, Role: user.Role,
		OpenMode: user.OpenMode,
	}
	if user.Role == roleEngineer {
		response.EngineerID = nullableInt(user.EngineerID)
		var name string
		if err := db.QueryRow(`SELECT name FROM engineers WHERE id = ?`,
			user.EngineerID).Scan(&name); err == nil {
			response.Name = name
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func nullableInt(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

// registerHandler handles POST /api/auth/register: the only self-service
// registration path. While no manager account exists, anyone reaching the
// local app can claim the manager role once — this is the recovery path when
// authentication is on but no admin was seeded. Once a manager exists,
// registration requires an authenticated manager and the new account's role
// is chosen by that manager.
func registerHandler(w http.ResponseWriter, r *http.Request) {
	open, err := authOpenMode()
	if err != nil {
		http.Error(w, "Failed to verify authentication state", http.StatusInternalServerError)
		return
	}
	if !open {
		user, ok := currentUser(r)
		if !ok || user.Role != roleManager {
			http.Error(w, "Only a signed-in manager can register accounts", http.StatusForbidden)
			return
		}
	}

	var input RegisterInput
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
	role := strings.TrimSpace(input.Role)
	if role == "" {
		// Claiming flow: the first registered account becomes the manager.
		role = roleManager
	}
	if role != roleManager && role != roleEngineer {
		http.Error(w, "Role must be manager or engineer", http.StatusBadRequest)
		return
	}
	if role == roleEngineer && input.EngineerID == nil {
		http.Error(w, "Engineer accounts must reference an engineer", http.StatusBadRequest)
		return
	}

	var engineerID sql.NullInt64
	if input.EngineerID != nil {
		if _, err := scanEngineer(db.QueryRow(
			`SELECT `+engineerColumns+` FROM engineers WHERE id = ?`, *input.EngineerID)); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Engineer not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to verify engineer", http.StatusInternalServerError)
			return
		}
		engineerID = sql.NullInt64{Int64: *input.EngineerID, Valid: true}
	}

	// A second manager account can only be created by an existing manager
	// (or in open mode); the claiming flow itself must stay first-come.
	var managerCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE role = ?`, roleManager).Scan(&managerCount); err != nil {
		http.Error(w, "Failed to check manager accounts", http.StatusInternalServerError)
		return
	}
	if role == roleManager && managerCount > 0 && (open || !isRequestingManager(r)) {
		http.Error(w, "A manager account already exists", http.StatusConflict)
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
		VALUES (?, ?, ?, ?)`, email, passwordHash, role, engineerID)
	if err != nil {
		http.Error(w, "Failed to create account", http.StatusInternalServerError)
		return
	}
	userID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Account created but ID could not be retrieved", http.StatusInternalServerError)
		return
	}
	// Registering signs the caller in immediately so the claiming flow lands
	// the new admin on their dashboard without a second form.
	token, expires, err := createSession(userID)
	if err != nil {
		http.Error(w, "Account created but session could not be started", http.StatusInternalServerError)
		return
	}
	name := ""
	if engineerID.Valid {
		_ = db.QueryRow(`SELECT name FROM engineers WHERE id = ?`,
			engineerID.Int64).Scan(&name)
	}
	setSessionCookie(w, token, expires)
	writeJSON(w, http.StatusCreated, SessionUserResponse{
		ID: userID, Email: email, Role: role,
		EngineerID: nullableInt(engineerID.Int64), Name: name,
	})
}

// isRequestingManager reports whether the caller of the current request is an
// authenticated manager. Used by registerHandler to distinguish a manager
// delegating account creation from an anonymous claiming attempt.
func isRequestingManager(r *http.Request) bool {
	if user, ok := currentUser(r); ok && user.Role == roleManager {
		return true
	}
	return false
}
