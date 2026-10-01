package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newRequest builds an http.Request for the shared test helpers.
func newRequest(method, target string, body []byte) *http.Request {
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

// newRecorder creates a response recorder for requestAs.
func newRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

// loginAsErr performs a login and returns an error instead of failing the
// test, for cases where the caller wants to assert on failure modes itself.
func loginAsErr(t *testing.T, router http.Handler, email, password string) (string, error) {
	t.Helper()
	got := request(t, router, http.MethodPost, "/api/auth/login",
		[]byte(`{"email":"`+email+`","password":"`+password+`"}`))
	if got.Code != http.StatusOK {
		return "", fmt.Errorf("login = %d %s", got.Code, got.Body.String())
	}
	for _, cookie := range got.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("no session cookie set")
}

// readBody reads a recorder response body as a string (the requestAs variant
// wraps an http.Response, so this takes the recorder directly).
func readBody(resp *http.Response) string {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return ""
	}
	return buf.String()
}
