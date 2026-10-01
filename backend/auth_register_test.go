package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestRegisterClaimManagerFlow covers the lockout-recovery path: with no
// accounts in the database, anyone reaching the local app may register the
// manager account; the account is signed in immediately and a second claim
// is refused.
func TestRegisterClaimManagerFlow(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()

	// The status endpoint reports claiming as available.
	status := request(t, router, http.MethodGet, "/api/auth/registration-status", nil)
	if status.Code != http.StatusOK {
		t.Fatalf("registration status = %d", status.Code)
	}
	var state map[string]any
	if err := json.Unmarshal(status.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state["canClaimManager"] != true || state["openMode"] != true {
		t.Errorf("expected open claiming state, got %v", state)
	}

	// Validation: missing email, short password.
	if got := request(t, router, http.MethodPost, "/api/auth/register",
		[]byte(`{"email":"","password":"longenough1"}`)); got.Code != http.StatusBadRequest {
		t.Fatalf("missing email = %d, want 400", got.Code)
	}
	if got := request(t, router, http.MethodPost, "/api/auth/register",
		[]byte(`{"email":"admin@example.com","password":"short"}`)); got.Code != http.StatusBadRequest {
		t.Fatalf("short password = %d, want 400", got.Code)
	}

	// Successful claim creates a manager and signs the caller in.
	got := request(t, router, http.MethodPost, "/api/auth/register",
		[]byte(`{"email":"admin@example.com","password":"sup3r-secret"}`))
	if got.Code != http.StatusCreated {
		t.Fatalf("register manager = %d %s", got.Code, got.Body.String())
	}
	var created SessionUserResponse
	if err := json.Unmarshal(got.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Role != roleManager || created.Email != "admin@example.com" {
		t.Errorf("created = %+v", created)
	}
	var cookie *http.Cookie
	for _, c := range got.Result().Cookies() {
		if c.Name == sessionCookieName {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("registration did not set the session cookie")
	}

	// A second anonymous claim is refused now that a manager exists (only
	// authenticated managers may register further accounts).
	if got2 := request(t, router, http.MethodPost, "/api/auth/register",
		[]byte(`{"email":"second@example.com","password":"sup3r-secret"}`)); got2.Code != http.StatusForbidden {
		t.Fatalf("second claim = %d, want 403", got2.Code)
	}

	// The claimed session works.
	me := requestAs(t, router, cookie.Value, http.MethodGet, "/api/auth/me", nil)
	if me.StatusCode != http.StatusOK {
		t.Fatalf("me after register = %d", me.StatusCode)
	}
}

// TestRegisterByManager covers manager-delegated registration: an
// authenticated manager may create engineer or additional manager accounts,
// while engineers and anonymous callers cannot.
func TestRegisterByManager(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	seedManagerAccount(t, "manager@example.com", "sup3r-secret")
	engineerID := insertEngineer(t)
	managerCookie := loginAs(t, router, "manager@example.com", "sup3r-secret")

	// Anonymous registration is refused once a manager exists.
	if got := request(t, router, http.MethodPost, "/api/auth/register",
		[]byte(`{"email":"anon@example.com","password":"sup3r-secret"}`)); got.Code != http.StatusForbidden {
		t.Fatalf("anonymous register with manager present = %d, want 403", got.Code)
	}

	// Manager registers an engineer account bound to an engineer record.
	got := requestAs(t, router, managerCookie, http.MethodPost, "/api/auth/register",
		[]byte(`{"email":"ada@example.com","password":"ada-password","role":"engineer","engineerId":`+
			int64String(engineerID)+`}`))
	if got.StatusCode != http.StatusCreated {
		t.Fatalf("register engineer = %d %s", got.StatusCode, readBody(got))
	}
	var created SessionUserResponse
	if err := json.NewDecoder(got.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Role != roleEngineer || created.EngineerID == nil || *created.EngineerID != engineerID {
		t.Errorf("created = %+v", created)
	}

	// The new engineer can log in.
	if _, err := loginAsErr(t, router, "ada@example.com", "ada-password"); err != nil {
		t.Fatalf("engineer login: %v", err)
	}

	// Engineer accounts must reference an engineer record.
	if code := responseCode(requestAs(t, router, managerCookie, http.MethodPost, "/api/auth/register",
		[]byte(`{"email":"loose@example.com","password":"ada-password","role":"engineer"}`))); code != http.StatusBadRequest {
		t.Fatalf("engineer account without engineerId = %d, want 400", code)
	}

	// Duplicate email is rejected regardless of role.
	if code := responseCode(requestAs(t, router, managerCookie, http.MethodPost, "/api/auth/register",
		[]byte(`{"email":"ada@example.com","password":"another-pass"}`))); code != http.StatusConflict {
		t.Fatalf("duplicate email = %d, want 409", code)
	}

	// An engineer session cannot register anyone.
	adaCookie := loginAs(t, router, "ada@example.com", "ada-password")
	if code := responseCode(requestAs(t, router, adaCookie, http.MethodPost, "/api/auth/register",
		[]byte(`{"email":"more@example.com","password":"sup3r-secret"}`))); code != http.StatusForbidden {
		t.Fatalf("engineer register = %d, want 403", code)
	}
}

// TestRegisterEngineerOnlyLockout documents the deliberate boundary: if
// engineer-only accounts exist without a manager, anonymous claiming stays
// closed so a former engineer cannot promote themselves.
func TestRegisterEngineerOnlyLockout(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	engineerID := insertEngineer(t)

	// Grant portal access directly so no manager exists.
	createEngineerUser(t, engineerID, "ada@example.com", "ada-password")

	status := request(t, router, http.MethodGet, "/api/auth/registration-status", nil)
	if status.Code != http.StatusOK {
		t.Fatalf("registration status = %d", status.Code)
	}
	var state map[string]any
	if err := json.Unmarshal(status.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state["canClaimManager"] != false {
		t.Errorf("claiming must be closed when engineer-only accounts exist: %v", state)
	}
	if state["openMode"] != false {
		t.Errorf("open mode must be off with engineer-only accounts: %v", state)
	}

	// The register endpoint refuses the anonymous claim as well.
	if got := request(t, router, http.MethodPost, "/api/auth/register",
		[]byte(`{"email":"evil@example.com","password":"sup3r-secret"}`)); got.Code != http.StatusForbidden {
		t.Fatalf("anonymous register in engineer-only state = %d, want 403", got.Code)
	}
}

func int64String(value int64) string {
	digits := "0123456789"
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	out := make([]byte, 0, 20)
	for value > 0 {
		out = append([]byte{digits[value%10]}, out...)
		value /= 10
	}
	if negative {
		return "-" + string(out)
	}
	return string(out)
}
