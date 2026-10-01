package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

// seedManagerAccount creates the manager account for tests that exercise
// authenticated behavior.
func seedManagerAccount(t *testing.T, email, password string) {
	t.Helper()
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO users (email, password_hash, role, engineer_id)
		VALUES (?, ?, ?, NULL)`, email, hash, roleManager); err != nil {
		t.Fatal(err)
	}
}

// createEngineerUser grants portal access to an engineer directly through the
// data layer, the way the portal-access endpoint does.
func createEngineerUser(t *testing.T, engineerID int64, email, password string) {
	t.Helper()
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO users (email, password_hash, role, engineer_id)
		VALUES (?, ?, ?, ?)`, email, hash, roleEngineer, engineerID); err != nil {
		t.Fatal(err)
	}
}

// loginAs performs a real login against the router and returns the session
// cookie so subsequent requests can carry it.
func loginAs(t *testing.T, router http.Handler, email, password string) string {
	t.Helper()
	got := request(t, router, http.MethodPost, "/api/auth/login",
		[]byte(`{"email":"`+email+`","password":"`+password+`"}`))
	if got.Code != http.StatusOK {
		t.Fatalf("login as %s = %d %s", email, got.Code, got.Body.String())
	}
	cookies := got.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			return cookie.Value
		}
	}
	t.Fatal("login response did not set the session cookie")
	return ""
}

// requestAs runs a request with the given session cookie.
func requestAs(t *testing.T, handler http.Handler, cookie, method, target string, body []byte) *http.Response {
	t.Helper()
	req := newRequest(method, target, body)
	req.Header.Set("Cookie", sessionCookieName+"="+cookie)
	recorder := newRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder.Result()
}

// responseCode collapses the response into a status code for assertions.
func responseCode(resp *http.Response) int {
	return resp.StatusCode
}

func TestPasswordHashingRoundTrip(t *testing.T) {
	hash, err := hashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyPassword(hash, "correct horse battery staple") {
		t.Error("expected the correct password to verify")
	}
	if verifyPassword(hash, "wrong password") {
		t.Error("expected the wrong password to be rejected")
	}
	// Malformed hashes must never verify.
	if verifyPassword("not-a-hash", "anything") {
		t.Error("malformed hash verified")
	}
}

func TestAuthLoginLifecycle(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()

	// No accounts exist: open mode lets everything through as manager.
	if got := request(t, router, http.MethodGet, "/api/engineers", nil); got.Code != http.StatusOK {
		t.Fatalf("open mode engineer list = %d %s", got.Code, got.Body.String())
	}

	seedManagerAccount(t, "manager@example.com", "sup3r-secret")

	// Now that an account exists, unauthenticated calls are rejected.
	if got := request(t, router, http.MethodGet, "/api/engineers", nil); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated engineer list = %d, want 401", got.Code)
	}

	// Wrong password.
	if got := request(t, router, http.MethodPost, "/api/auth/login",
		[]byte(`{"email":"manager@example.com","password":"nope"}`)); got.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password = %d, want 401", got.Code)
	}
	// Unknown email.
	if got := request(t, router, http.MethodPost, "/api/auth/login",
		[]byte(`{"email":"ghost@example.com","password":"nope"}`)); got.Code != http.StatusUnauthorized {
		t.Fatalf("unknown email = %d, want 401", got.Code)
	}

	// Successful login sets the cookie and /me reflects the manager role.
	cookie := loginAs(t, router, "manager@example.com", "sup3r-secret")
	me := requestAs(t, router, cookie, http.MethodGet, "/api/auth/me", nil)
	if me.StatusCode != http.StatusOK {
		t.Fatalf("me = %d", me.StatusCode)
	}
	var session SessionUserResponse
	if err := json.NewDecoder(me.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}
	if session.Role != roleManager || session.Email != "manager@example.com" {
		t.Errorf("me = %+v", session)
	}

	// Logout ends the session.
	logout := requestAs(t, router, cookie, http.MethodPost, "/api/auth/logout", nil)
	if logout.StatusCode != http.StatusNoContent {
		t.Fatalf("logout = %d", logout.StatusCode)
	}
	if got := requestAs(t, router, cookie, http.MethodGet, "/api/auth/me", nil); got.StatusCode != http.StatusUnauthorized {
		t.Fatalf("me after logout = %d, want 401", got.StatusCode)
	}
}

func TestEngineerScopedAccess(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	seedManagerAccount(t, "manager@example.com", "sup3r-secret")

	adaID := insertEngineer(t)
	bobID := int64(0)
	if result, err := db.Exec(`
		INSERT INTO engineers (name, role, level, team, review_cycle)
		VALUES ('Bob', 'Engineer', 'Mid', 'Platform', '2026-H1')`); err != nil {
		t.Fatal(err)
	} else {
		bobID, _ = result.LastInsertId()
	}
	insertNote(t, adaID)
	insertNote(t, bobID)

	if _, err := db.Exec(`
		INSERT INTO one_on_ones (engineer_id, meeting_date, status,
			private_manager_notes, shared_notes)
		VALUES (?, '2026-09-01', 'completed', ' candid private note ', 'Shared summary')`,
		adaID); err != nil {
		t.Fatal(err)
	}

	createEngineerUser(t, adaID, "ada@example.com", "ada-password")
	adaCookie := loginAs(t, router, "ada@example.com", "ada-password")

	// Engineer list collapses to self.
	resp := requestAs(t, router, adaCookie, http.MethodGet, "/api/engineers", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("engineer list = %d", resp.StatusCode)
	}
	var engineers []Engineer
	if err := json.NewDecoder(resp.Body).Decode(&engineers); err != nil {
		t.Fatal(err)
	}
	if len(engineers) != 1 || engineers[0].ID != int(adaID) {
		t.Errorf("engineer list for engineer = %+v, want only Ada", engineers)
	}

	// Own profile is readable; another engineer's is not.
	if code := responseCode(requestAs(t, router, adaCookie, http.MethodGet,
		"/api/engineers/"+strconv.FormatInt(adaID, 10), nil)); code != http.StatusOK {
		t.Fatalf("own profile = %d, want 200", code)
	}
	if code := responseCode(requestAs(t, router, adaCookie, http.MethodGet,
		"/api/engineers/"+strconv.FormatInt(bobID, 10), nil)); code != http.StatusForbidden {
		t.Fatalf("other profile = %d, want 403", code)
	}

	// Notes are scoped to self.
	resp = requestAs(t, router, adaCookie, http.MethodGet, "/api/notes", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notes list = %d", resp.StatusCode)
	}
	var notes []PerformanceNote
	if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].EngineerID != int(adaID) {
		t.Errorf("notes for engineer = %+v, want only Ada's note", notes)
	}

	// Cross-engineer note creation is rejected.
	if code := responseCode(requestAs(t, router, adaCookie, http.MethodPost,
		"/api/notes", []byte(`{"engineerId":`+strconv.FormatInt(bobID, 10)+
			`,"noteDate":"2026-09-15","category":"Team Contribution","summary":"Helped Bob"}`))); code != http.StatusForbidden {
		t.Fatalf("note for other engineer = %d, want 403", code)
	}

	// Own-context creation works and is flagged as engineer-authored, with
	// follow-up ownership forced to the manager.
	created := requestAs(t, router, adaCookie, http.MethodPost, "/api/notes",
		[]byte(`{"engineerId":`+strconv.FormatInt(adaID, 10)+
			`,"noteDate":"2026-09-20","category":"Technical Excellence","summary":"Refactored the auth module","followUpNeeded":true}`))
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("own note create = %d %s", created.StatusCode, readBody(created))
	}
	var note PerformanceNote
	if err := json.NewDecoder(created.Body).Decode(&note); err != nil {
		t.Fatal(err)
	}
	if note.AuthorRole != roleEngineer {
		t.Errorf("AuthorRole = %q, want engineer", note.AuthorRole)
	}
	if note.FollowUpNeeded {
		t.Error("engineer-created notes must not set followUpNeeded")
	}

	// Engineers cannot edit or delete manager-recorded evidence.
	managerNoteID := int64(0)
	if err := db.QueryRow(
		`SELECT id FROM performance_notes WHERE engineer_id = ? AND author_role = 'manager'`,
		adaID).Scan(&managerNoteID); err != nil {
		t.Fatal(err)
	}
	if code := responseCode(requestAs(t, router, adaCookie, http.MethodPut,
		"/api/notes/"+strconv.FormatInt(managerNoteID, 10),
		[]byte(`{"engineerId":`+strconv.FormatInt(adaID, 10)+
			`,"noteDate":"2026-07-25","category":"Delivery","summary":"Tampered"}`))); code != http.StatusForbidden {
		t.Fatalf("edit manager note = %d, want 403", code)
	}
	if code := responseCode(requestAs(t, router, adaCookie, http.MethodDelete,
		"/api/notes/"+strconv.FormatInt(managerNoteID, 10), nil)); code != http.StatusForbidden {
		t.Fatalf("delete manager note = %d, want 403", code)
	}

	// Engineers can edit and delete their own context entries.
	ownNoteID := int64(note.ID)
	if code := responseCode(requestAs(t, router, adaCookie, http.MethodPut,
		"/api/notes/"+strconv.FormatInt(ownNoteID, 10),
		[]byte(`{"engineerId":`+strconv.FormatInt(adaID, 10)+
			`,"noteDate":"2026-09-21","category":"Technical Excellence","summary":"Refactored the auth module end to end"}`))); code != http.StatusOK {
		t.Fatalf("edit own note = %d", code)
	}
	if code := responseCode(requestAs(t, router, adaCookie, http.MethodDelete,
		"/api/notes/"+strconv.FormatInt(ownNoteID, 10), nil)); code != http.StatusNoContent {
		t.Fatalf("delete own note = %d", code)
	}

	// Private manager notes are redacted in the engineer's 1:1 view.
	resp = requestAs(t, router, adaCookie, http.MethodGet,
		"/api/engineers/"+strconv.FormatInt(adaID, 10)+"/one-on-ones", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("one-on-ones = %d", resp.StatusCode)
	}
	var meetings []OneOnOne
	if err := json.NewDecoder(resp.Body).Decode(&meetings); err != nil {
		t.Fatal(err)
	}
	if len(meetings) != 1 {
		t.Fatalf("meetings = %+v", meetings)
	}
	if meetings[0].PrivateManagerNotes != "" {
		t.Errorf("private manager notes leaked to engineer: %q", meetings[0].PrivateManagerNotes)
	}
	if meetings[0].SharedNotes != "Shared summary" {
		t.Errorf("shared notes should remain visible: %q", meetings[0].SharedNotes)
	}

	// The manager still sees the private notes.
	managerCookie := loginAs(t, router, "manager@example.com", "sup3r-secret")
	resp = requestAs(t, router, managerCookie, http.MethodGet,
		"/api/engineers/"+strconv.FormatInt(adaID, 10)+"/one-on-ones", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("manager one-on-ones = %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&meetings); err != nil {
		t.Fatal(err)
	}
	if len(meetings) != 1 || meetings[0].PrivateManagerNotes != " candid private note " {
		t.Errorf("manager view of private notes = %+v", meetings)
	}
}

func TestEngineerBlockedFromManagerSurfaces(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	seedManagerAccount(t, "manager@example.com", "sup3r-secret")
	engineerID := insertEngineer(t)
	createEngineerUser(t, engineerID, "ada@example.com", "ada-password")
	cookie := loginAs(t, router, "ada@example.com", "ada-password")

	managerOnlyRoutes := []struct {
		method, target string
	}{
		{http.MethodGet, "/api/settings"},
		{http.MethodGet, "/api/integrations"},
		{http.MethodGet, "/api/ai-providers"},
		{http.MethodGet, "/api/dashboard/attention"},
		{http.MethodGet, "/api/dashboard/goals"},
		{http.MethodGet, "/api/dashboard/review-readiness"},
		{http.MethodGet, "/api/dashboard/evidence-recency"},
		{http.MethodGet, "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/ai-analyses"},
		{http.MethodGet, "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/jira-evidence"},
		{http.MethodPut, "/api/engineers/" + strconv.FormatInt(engineerID, 10)},
		{http.MethodPost, "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/archive"},
		{http.MethodPost, "/api/engineers"},
		{http.MethodPost, "/api/review-periods"},
		{http.MethodPut, "/api/settings/theme"},
		{http.MethodPost, "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/goals"},
		{http.MethodPost, "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/one-on-ones"},
		{http.MethodPost, "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/recognitions"},
		{http.MethodPost, "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/development-plans"},
		{http.MethodPut, "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/onboarding-profile"},
		{http.MethodPost, "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/follow-ups"},
	}
	for _, route := range managerOnlyRoutes {
		if code := responseCode(requestAs(t, router, cookie, route.method, route.target, []byte(`{}`))); code != http.StatusForbidden {
			t.Errorf("%s %s = %d, want 403", route.method, route.target, code)
		}
	}

	// Engineer-readable surfaces still work.
	readable := []string{
		"/api/engineers/" + strconv.FormatInt(engineerID, 10),
		"/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/goals",
		"/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/one-on-ones",
		"/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/follow-ups",
		"/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/recognitions",
		"/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/timeline",
		"/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/development-plans",
		"/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/onboarding-profile",
		"/api/review-periods",
	}
	for _, target := range readable {
		// Onboarding-profile is 404 when none exists yet, which is normal.
		code := responseCode(requestAs(t, router, cookie, http.MethodGet, target, nil))
		if code != http.StatusOK && code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 200 or 404", target, code)
		}
	}
}

func TestPortalAccessLifecycle(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	seedManagerAccount(t, "manager@example.com", "sup3r-secret")
	engineerID := insertEngineer(t)
	id := strconv.FormatInt(engineerID, 10)
	managerCookie := loginAs(t, router, "manager@example.com", "sup3r-secret")

	// Creating portal access.
	create := requestAs(t, router, managerCookie, http.MethodPost,
		"/api/engineers/"+id+"/portal-access",
		[]byte(`{"email":"ada@example.com","password":"initial-pass"}`))
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("create portal access = %d %s", create.StatusCode, readBody(create))
	}

	// Duplicate email is rejected.
	duplicate := requestAs(t, router, managerCookie, http.MethodPost,
		"/api/engineers/"+id+"/portal-access",
		[]byte(`{"email":"ada@example.com","password":"initial-pass"}`))
	if duplicate.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate email = %d, want 409", duplicate.StatusCode)
	}

	// Short password is rejected.
	short := requestAs(t, router, managerCookie, http.MethodPost,
		"/api/engineers/"+id+"/portal-access",
		[]byte(`{"email":"other@example.com","password":"short"}`))
	if short.StatusCode != http.StatusBadRequest {
		t.Fatalf("short password = %d, want 400", short.StatusCode)
	}

	// Engineer can log in with the granted credentials.
	if _, err := loginAsErr(t, router, "ada@example.com", "initial-pass"); err != nil {
		t.Fatalf("engineer login: %v", err)
	}

	// GET reports access with the email but never a hash.
	status := requestAs(t, router, managerCookie, http.MethodGet,
		"/api/engineers/"+id+"/portal-access", nil)
	if status.StatusCode != http.StatusOK {
		t.Fatalf("portal status = %d", status.StatusCode)
	}
	var access map[string]any
	if err := json.NewDecoder(status.Body).Decode(&access); err != nil {
		t.Fatal(err)
	}
	if access["hasAccess"] != true || access["email"] != "ada@example.com" {
		t.Errorf("portal access = %+v", access)
	}

	// Password reset revokes the old session and accepts the new one.
	oldCookie := loginAs(t, router, "ada@example.com", "initial-pass")
	reset := requestAs(t, router, managerCookie, http.MethodPut,
		"/api/engineers/"+id+"/portal-access",
		[]byte(`{"password":"new-password-1"}`))
	if reset.StatusCode != http.StatusNoContent {
		t.Fatalf("password reset = %d %s", reset.StatusCode, readBody(reset))
	}
	if code := responseCode(requestAs(t, router, oldCookie, http.MethodGet, "/api/auth/me", nil)); code != http.StatusUnauthorized {
		t.Fatalf("old session after reset = %d, want 401", code)
	}
	if _, err := loginAsErr(t, router, "ada@example.com", "new-password-1"); err != nil {
		t.Fatalf("login with new password: %v", err)
	}

	// Engineers cannot manage portal access.
	adaCookie := loginAs(t, router, "ada@example.com", "new-password-1")
	if code := responseCode(requestAs(t, router, adaCookie, http.MethodGet,
		"/api/engineers/"+id+"/portal-access", nil)); code != http.StatusForbidden {
		t.Fatalf("engineer portal status = %d, want 403", code)
	}

	// Revoking portal access removes the login.
	revoke := requestAs(t, router, managerCookie, http.MethodDelete,
		"/api/engineers/"+id+"/portal-access", nil)
	if revoke.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke = %d", revoke.StatusCode)
	}
	if code := responseCode(requestAs(t, router, adaCookie, http.MethodGet, "/api/auth/me", nil)); code != http.StatusUnauthorized {
		t.Fatalf("session after revoke = %d, want 401", code)
	}
	after := requestAs(t, router, managerCookie, http.MethodGet,
		"/api/engineers/"+id+"/portal-access", nil)
	if err := json.NewDecoder(after.Body).Decode(&access); err != nil {
		t.Fatal(err)
	}
	if access["hasAccess"] != false {
		t.Errorf("portal access after revoke = %+v", access)
	}
}

func TestManagerSeededFromEnv(t *testing.T) {
	setupTestDatabase(t)
	t.Setenv("MANAGER_EMAIL", "boss@example.com")
	t.Setenv("MANAGER_PASSWORD", "bootstrap-pass")
	if err := seedManagerFromEnv(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE role = 'manager'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("manager count = %d, want 1", count)
	}
	// Second call is idempotent.
	if err := seedManagerFromEnv(); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE role = 'manager'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("manager count after re-seed = %d, want 1", count)
	}
}

func TestPBKDF2Vector(t *testing.T) {
	// Reference vector: PBKDF2-HMAC-SHA256 with password "password", salt
	// "salt", 1 iteration, dkLen 32 equals
	// 120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b
	// (from the published PBKDF2-HMAC-SHA256 test corpus).
	derived := pbkdf2SHA256([]byte("password"), []byte("salt"), 1, 32)
	expected := "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b"
	if hexEncode(derived) != expected {
		t.Fatalf("pbkdf2 single-iteration vector mismatch: %s", hexEncode(derived))
	}

	// Multi-iteration vector from the same corpus (iterations = 2).
	derived = pbkdf2SHA256([]byte("password"), []byte("salt"), 2, 32)
	expected = "ae4d0c95af6b46d32d0adff928f06dd02a303f8ef3c251dfd6e2d85a95474c43"
	if hexEncode(derived) != expected {
		t.Fatalf("pbkdf2 two-iteration vector mismatch: %s", hexEncode(derived))
	}
}

func hexEncode(value []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, len(value)*2)
	for _, b := range value {
		out = append(out, digits[b>>4], digits[b&0x0f])
	}
	return string(out)
}
