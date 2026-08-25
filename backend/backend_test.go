package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func setupTestDatabase(t *testing.T) {
	t.Helper()
	database, err := openDatabase("file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db = database
	t.Cleanup(func() { database.Close() })
}

func request(t *testing.T, handler http.Handler, method, target string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func insertEngineer(t *testing.T) int64 {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO engineers (name, role, level, team, career_goal, review_cycle)
		VALUES ('Ada', 'Engineer', 'Senior', 'Platform', 'Staff', '2026-H1')`)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	return id
}

func insertNote(t *testing.T, engineerID int64) int64 {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO performance_notes
			(engineer_id, note_date, category, summary, details, impact, follow_up_needed, review_cycle)
		VALUES (?, '2026-07-25', 'Delivery', 'Shipped', 'Details', 'High', 1, '2026-H1')`,
		engineerID)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	return id
}

func TestDatabaseInitialization(t *testing.T) {
	setupTestDatabase(t)
	var theme string
	if err := db.QueryRow(`SELECT setting_value FROM application_settings WHERE setting_key='theme'`).Scan(&theme); err != nil {
		t.Fatal(err)
	}
	if theme != "light" {
		t.Fatalf("theme = %q", theme)
	}
	if _, err := db.Exec(`INSERT INTO performance_notes
		(engineer_id,note_date,category,summary) VALUES (999,'x','x','x')`); err == nil {
		t.Fatal("foreign keys were not enabled")
	}
	// A path whose parent is an existing file (not a directory) is genuinely
	// unwritable: MkdirAll cannot turn a file into a directory, so openDatabase
	// must surface the error rather than silently creating it.
	blockerDir := t.TempDir()
	blocker := filepath.Join(blockerDir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := openDatabase(filepath.Join(blocker, "db.sqlite")); err == nil {
		t.Fatal("expected openDatabase to fail when the parent path is a file")
	}
}

func TestOpenDatabaseCreatesMissingDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "data", "performance.db")
	database, err := openDatabase(target)
	if err != nil {
		t.Fatalf("openDatabase = %v, want nil (it should create the missing folder)", err)
	}
	t.Cleanup(func() { database.Close() })
	if info, statErr := os.Stat(filepath.Join(dir, "data")); statErr != nil || !info.IsDir() {
		t.Fatalf("data dir was not created: stat err = %v", statErr)
	}
}

func TestRunBuildsServer(t *testing.T) {
	original := listenAndServe
	t.Cleanup(func() { listenAndServe = original })
	called := false
	listenAndServe = func(address string, handler http.Handler) error {
		called = address == ":0" && handler != nil
		return errors.New("stop")
	}
	err := run(filepath.Join(t.TempDir(), "app.sqlite"), ":0")
	if err == nil || !called {
		t.Fatalf("run = %v, called = %v", err, called)
	}
	if err := run(filepath.Join(t.TempDir(), "missing", "app.sqlite"), ":0"); err == nil {
		t.Fatal("invalid database path should fail")
	}
}

func TestEngineerAndNoteRoutes(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()

	if got := request(t, router, http.MethodGet, "/api/engineers", nil); got.Code != http.StatusOK || got.Body.String() != "[]\n" {
		t.Fatalf("empty engineers: %d %s", got.Code, got.Body.String())
	}
	for _, body := range [][]byte{[]byte("{"), []byte(`{"name":"Ada"}`)} {
		if got := request(t, router, http.MethodPost, "/api/engineers", body); got.Code != http.StatusBadRequest {
			t.Fatalf("invalid engineer: %d", got.Code)
		}
	}
	engineerJSON := []byte(`{"name":"Ada","role":"Engineer","level":"Senior","team":"Platform","careerGoal":"Staff","reviewCycle":"2026-H1"}`)
	if got := request(t, router, http.MethodPost, "/api/engineers", engineerJSON); got.Code != http.StatusCreated {
		t.Fatalf("create engineer: %d %s", got.Code, got.Body.String())
	}
	if got := request(t, router, http.MethodGet, "/api/engineers", nil); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"Ada"`) {
		t.Fatalf("get engineers: %d %s", got.Code, got.Body.String())
	}

	for _, body := range [][]byte{[]byte("{"), []byte(`{"engineerId":1}`)} {
		if got := request(t, router, http.MethodPost, "/api/notes", body); got.Code != http.StatusBadRequest {
			t.Fatalf("invalid note: %d", got.Code)
		}
	}
	noteJSON := []byte(`{"engineerId":1,"noteDate":"2026-07-25","category":"Delivery","summary":"Shipped","details":"D","impact":"High","followUpNeeded":true,"reviewCycle":"2026-H1"}`)
	created := request(t, router, http.MethodPost, "/api/notes", noteJSON)
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), `"id":1`) {
		t.Fatalf("create note: %d %s", created.Code, created.Body.String())
	}
	for _, target := range []string{"/api/notes", "/api/notes?engineerId=1"} {
		if got := request(t, router, http.MethodGet, target, nil); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"Shipped"`) {
			t.Fatalf("get notes: %d %s", got.Code, got.Body.String())
		}
	}
	if got := request(t, router, http.MethodPut, "/api/notes/nope", noteJSON); got.Code != http.StatusBadRequest {
		t.Fatalf("invalid update id: %d", got.Code)
	}
	if got := request(t, router, http.MethodPut, "/api/notes/1", []byte("{")); got.Code != http.StatusBadRequest {
		t.Fatalf("invalid update body: %d", got.Code)
	}
	if got := request(t, router, http.MethodPut, "/api/notes/999", noteJSON); got.Code != http.StatusNotFound {
		t.Fatalf("missing update: %d", got.Code)
	}
	if got := request(t, router, http.MethodPut, "/api/notes/1", noteJSON); got.Code != http.StatusOK {
		t.Fatalf("update note: %d %s", got.Code, got.Body.String())
	}
	if got := request(t, router, http.MethodDelete, "/api/notes/nope", nil); got.Code != http.StatusBadRequest {
		t.Fatalf("invalid delete id: %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/notes/999", nil); got.Code != http.StatusNotFound {
		t.Fatalf("missing delete: %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/notes/1", nil); got.Code != http.StatusNoContent {
		t.Fatalf("delete note: %d", got.Code)
	}
}

func TestSettingsRoutesAndHelpers(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	if got := request(t, router, http.MethodGet, "/api/settings", nil); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"theme":"light"`) {
		t.Fatalf("settings: %d %s", got.Code, got.Body.String())
	}
	tests := []struct {
		target string
		body   string
		code   int
	}{
		{"/api/settings/nope", `{"value":"x"}`, 400},
		{"/api/settings/theme", `{`, 400},
		{"/api/settings/theme", `{"value":"blue"}`, 400},
		{"/api/settings/attachment_storage_root", `{"value":" "}`, 400},
		{"/api/settings/theme", `{"value":" dark "}`, 204},
	}
	for _, test := range tests {
		if got := request(t, router, http.MethodPut, test.target, []byte(test.body)); got.Code != test.code {
			t.Errorf("%s: got %d want %d (%s)", test.target, got.Code, test.code, got.Body.String())
		}
	}
	root := filepath.Join(t.TempDir(), "attachments")
	body, _ := json.Marshal(ApplicationSetting{Value: root})
	if got := request(t, router, http.MethodPut, "/api/settings/attachment_storage_root", body); got.Code != http.StatusNoContent {
		t.Fatalf("storage root: %d %s", got.Code, got.Body.String())
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Fatalf("storage directory not created: %v", err)
	}
	value, err := getSettingValue("missing")
	if err != nil || value != "" {
		t.Fatalf("missing setting = %q, %v", value, err)
	}
	cases := map[string]string{
		" Ada Lovelace! ": "ada-lovelace",
		"---":             "",
		"A__B  C":         "a-b-c",
		"abc123":          "abc123",
	}
	for input, expected := range cases {
		if actual := sanitizeFolderName(input); actual != expected {
			t.Errorf("sanitize %q = %q, want %q", input, actual, expected)
		}
	}
}

func TestEncryptionAndIntegrationRoutes(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	t.Setenv("MANAGER_DASHBOARD_ENCRYPTION_KEY", "")
	if _, err := loadEncryptionKey(); err == nil {
		t.Fatal("missing key should fail")
	}
	t.Setenv("MANAGER_DASHBOARD_ENCRYPTION_KEY", "not-base64")
	if _, err := loadEncryptionKey(); err == nil {
		t.Fatal("invalid key should fail")
	}
	t.Setenv("MANAGER_DASHBOARD_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString([]byte("short")))
	if _, err := loadEncryptionKey(); err == nil {
		t.Fatal("short key should fail")
	}
	key := bytes.Repeat([]byte{7}, 32)
	t.Setenv("MANAGER_DASHBOARD_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(key))
	encrypted, err := encryptSecret("secret")
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := decryptSecret(encrypted)
	if err != nil || decrypted != "secret" {
		t.Fatalf("decrypt = %q, %v", decrypted, err)
	}
	for _, invalid := range []string{"!", base64.StdEncoding.EncodeToString([]byte("tiny"))} {
		if _, err := decryptSecret(invalid); err == nil {
			t.Fatalf("decrypt %q should fail", invalid)
		}
	}
	t.Setenv("MANAGER_DASHBOARD_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{8}, 32)))
	if _, err := decryptSecret(encrypted); err == nil {
		t.Fatal("wrong key should fail")
	}
	t.Setenv("MANAGER_DASHBOARD_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(key))

	tests := []struct {
		target string
		body   string
		code   int
	}{
		{"/api/integrations/dropbox", `{}`, 400},
		{"/api/integrations/github", `{`, 400},
		{"/api/integrations/github", `{"secret":""}`, 400},
		{"/api/integrations/github", `{"accountLabel":"Work","baseUrl":"https://github.com","secret":"token","enabled":true}`, 204},
	}
	for _, test := range tests {
		if got := request(t, router, http.MethodPut, test.target, []byte(test.body)); got.Code != test.code {
			t.Errorf("%s: got %d want %d (%s)", test.target, got.Code, test.code, got.Body.String())
		}
	}
	if got := request(t, router, http.MethodGet, "/api/integrations", nil); got.Code != 200 || strings.Contains(got.Body.String(), "token") || !strings.Contains(got.Body.String(), `"hasSecret":true`) {
		t.Fatalf("integrations: %d %s", got.Code, got.Body.String())
	}
	if got := request(t, router, http.MethodDelete, "/api/integrations/gitlab", nil); got.Code != 404 {
		t.Fatalf("missing integration delete: %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/integrations/github", nil); got.Code != 204 {
		t.Fatalf("integration delete: %d", got.Code)
	}
}

func multipartRequest(t *testing.T, method, target, filename string, content []byte, fields map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if filename != "" {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func serveRequest(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

var pngFixture = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
}

func TestAttachmentHelpers(t *testing.T) {
	setupTestDatabase(t)
	if name, err := generateStoredFilename(".png"); err != nil || len(name) != 36 || !strings.HasSuffix(name, ".png") {
		t.Fatalf("filename = %q, %v", name, err)
	}
	if mimeType, extension, err := validateImageType(pngFixture, "x.png"); err != nil || mimeType != "image/png" || extension != ".png" {
		t.Fatalf("PNG validation = %q %q %v", mimeType, extension, err)
	}
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46}
	if _, extension, err := validateImageType(jpeg, "x.jpeg"); err != nil || extension != ".jpeg" {
		t.Fatalf("JPEG validation = %q %v", extension, err)
	}
	if _, _, err := validateImageType([]byte("text"), "x.txt"); err == nil {
		t.Fatal("text should be rejected")
	}
	if _, err := attachmentDirectoryForEngineer(1, "Ada"); err == nil {
		t.Fatal("unconfigured storage should fail")
	}
	root := t.TempDir()
	if _, err := db.Exec(`INSERT INTO application_settings(setting_key,setting_value)
		VALUES('attachment_storage_root',?)`, root); err != nil {
		t.Fatal(err)
	}
	directory, err := attachmentDirectoryForEngineer(7, "Ada Lovelace")
	if err != nil || !strings.Contains(directory, "7-ada-lovelace") {
		t.Fatalf("directory = %q, %v", directory, err)
	}
	resolved, err := resolveAttachmentPath(filepath.Join("engineers", "file.png"))
	if err != nil || !strings.HasPrefix(resolved, root) {
		t.Fatalf("resolved = %q, %v", resolved, err)
	}
	if _, err := resolveAttachmentPath(filepath.Join("..", "outside")); err == nil {
		t.Fatal("path traversal should fail")
	}
	directory, err = attachmentDirectoryForEngineer(8, "---")
	if err != nil || !strings.Contains(directory, "8-engineer") {
		t.Fatalf("fallback directory = %q, %v", directory, err)
	}

	store := func(name string, contents []byte) (*storedAttachment, error) {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, contents, 0600); err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		return storeAttachmentFile(file, &multipart.FileHeader{Filename: name}, 1, "Ada")
	}
	if _, err := store("empty.png", nil); err == nil {
		t.Fatal("empty file should fail")
	}
	if _, err := store("text.txt", []byte("text")); err == nil {
		t.Fatal("text file should fail")
	}
	large := append(append([]byte{}, pngFixture...), bytes.Repeat([]byte{0}, maxAttachmentSize)...)
	if _, err := store("large.png", large); err == nil {
		t.Fatal("oversized file should fail")
	}
}

func TestAttachmentLifecycle(t *testing.T) {
	setupTestDatabase(t)
	root := t.TempDir()
	if _, err := db.Exec(`INSERT INTO application_settings(setting_key,setting_value)
		VALUES('attachment_storage_root',?)`, root); err != nil {
		t.Fatal(err)
	}
	engineerID := insertEngineer(t)
	noteID := insertNote(t, engineerID)
	router := newRouter()

	if got := request(t, router, http.MethodPost, "/api/notes/nope/attachments", nil); got.Code != 400 {
		t.Fatalf("invalid upload note: %d", got.Code)
	}
	malformed := httptest.NewRequest(http.MethodPost,
		"/api/notes/"+strconv.FormatInt(noteID, 10)+"/attachments",
		strings.NewReader("not multipart"))
	malformed.Header.Set("Content-Type", "multipart/form-data; boundary=broken")
	if got := serveRequest(router, malformed); got.Code != 400 {
		t.Fatalf("malformed upload: %d", got.Code)
	}
	if got := serveRequest(router, multipartRequest(t, http.MethodPost, "/api/notes/999/attachments", "x.png", pngFixture, nil)); got.Code != 404 {
		t.Fatalf("missing upload note: %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, "/api/notes/nope/attachments", nil); got.Code != 400 {
		t.Fatalf("invalid note attachments: %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, "/api/notes/999/attachments", nil); got.Code != 200 || got.Body.String() != "[]\n" {
		t.Fatalf("empty attachments: %d %s", got.Code, got.Body.String())
	}
	noteIDString := strconv.FormatInt(noteID, 10)
	missing := multipartRequest(t, http.MethodPost, "/api/notes/"+noteIDString+"/attachments", "", nil, nil)
	if got := serveRequest(router, missing); got.Code != 400 {
		t.Fatalf("missing file: %d %s", got.Code, got.Body.String())
	}
	invalid := multipartRequest(t, http.MethodPost, "/api/notes/"+noteIDString+"/attachments", "x.txt", []byte("text"), nil)
	if got := serveRequest(router, invalid); got.Code != 400 {
		t.Fatalf("invalid file: %d %s", got.Code, got.Body.String())
	}
	upload := multipartRequest(t, http.MethodPost, "/api/notes/"+noteIDString+"/attachments", "shot.png", pngFixture,
		map[string]string{"sourceSystem": "Jira", "sourceAuthor": "Ada", "caption": "Proof"})
	created := serveRequest(router, upload)
	if created.Code != 201 || !strings.Contains(created.Body.String(), `"sourceSystem":"Jira"`) {
		t.Fatalf("upload: %d %s", created.Code, created.Body.String())
	}
	var attachment Attachment
	if err := json.Unmarshal(created.Body.Bytes(), &attachment); err != nil {
		t.Fatal(err)
	}
	listed := request(t, router, http.MethodGet, "/api/notes/"+noteIDString+"/attachments", nil)
	if listed.Code != 200 || !strings.Contains(listed.Body.String(), `"shot.png"`) {
		t.Fatalf("list: %d %s", listed.Code, listed.Body.String())
	}
	if got := request(t, router, http.MethodGet, "/api/attachments/nope/content", nil); got.Code != 400 {
		t.Fatalf("invalid content id: %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, "/api/attachments/999/content", nil); got.Code != 404 {
		t.Fatalf("missing content: %d", got.Code)
	}
	content := request(t, router, http.MethodGet, attachment.ContentURL, nil)
	if content.Code != 200 || content.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("content: %d %s", content.Code, content.Header().Get("Content-Type"))
	}
	if got := request(t, router, http.MethodDelete, "/api/attachments/nope", nil); got.Code != 400 {
		t.Fatalf("invalid delete: %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/attachments/999", nil); got.Code != 404 {
		t.Fatalf("missing delete: %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/attachments/"+strconv.FormatInt(attachment.ID, 10), nil); got.Code != 204 {
		t.Fatalf("delete: %d %s", got.Code, got.Body.String())
	}
}

func TestAttachmentStorageAndPersistenceFailures(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	noteID := insertNote(t, engineerID)
	router := newRouter()
	upload := multipartRequest(t, http.MethodPost, "/api/notes/"+strconv.FormatInt(noteID, 10)+"/attachments", "x.png", pngFixture, nil)
	if got := serveRequest(router, upload); got.Code != http.StatusPreconditionRequired {
		t.Fatalf("unconfigured upload: %d %s", got.Code, got.Body.String())
	}
	create := multipartRequest(t, http.MethodPost, "/api/notes-with-attachment", "x.png", pngFixture,
		map[string]string{"engineerId": strconv.FormatInt(engineerID, 10), "noteDate": "d", "category": "c", "summary": "s"})
	if got := serveRequest(router, create); got.Code != http.StatusPreconditionRequired {
		t.Fatalf("unconfigured atomic create: %d %s", got.Code, got.Body.String())
	}
	if _, err := persistAttachment(parentNote, 999, &storedAttachment{
		originalName: "orphan.png", relativePath: "orphan.png",
		mimeType: "image/png", size: 1, hash: "hash",
	}, "", "", "", ""); err == nil {
		t.Fatal("invalid note link should roll back")
	}
	if _, _, err := persistNoteWithAttachment(
		CreateNoteWithAttachmentInput{EngineerID: 999, NoteDate: "d", Category: "c", Summary: "s"},
		"Missing", &storedAttachment{}); err == nil {
		t.Fatal("invalid engineer should roll back")
	}

	root := t.TempDir()
	if _, err := db.Exec(`INSERT INTO application_settings(setting_key,setting_value)
		VALUES('attachment_storage_root',?)`, root); err != nil {
		t.Fatal(err)
	}
	missingPath := filepath.Join("engineers", "missing.png")
	result, err := db.Exec(`INSERT INTO attachments
		(original_filename,stored_filename,mime_type,file_size,sha256_hash)
		VALUES('missing.png',?,'image/png',1,'hash')`, missingPath)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	if got := request(t, router, http.MethodGet, "/api/attachments/"+strconv.FormatInt(id, 10)+"/content", nil); got.Code != 404 {
		t.Fatalf("missing file content: %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/attachments/"+strconv.FormatInt(id, 10), nil); got.Code != 204 {
		t.Fatalf("delete missing file: %d", got.Code)
	}
	escapeResult, err := db.Exec(`INSERT INTO attachments
		(original_filename,stored_filename,mime_type,file_size,sha256_hash)
		VALUES('escape.png','../escape.png','image/png',1,'escape')`)
	if err != nil {
		t.Fatal(err)
	}
	escapeID, _ := escapeResult.LastInsertId()
	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/attachments/" + strconv.FormatInt(escapeID, 10) + "/content"},
		{http.MethodDelete, "/api/attachments/" + strconv.FormatInt(escapeID, 10)},
	} {
		if got := request(t, router, route.method, route.path, nil); got.Code != 500 {
			t.Errorf("escaped attachment %s = %d", route.method, got.Code)
		}
	}
}

func TestCreateNoteWithAttachment(t *testing.T) {
	setupTestDatabase(t)
	root := t.TempDir()
	if _, err := db.Exec(`INSERT INTO application_settings(setting_key,setting_value)
		VALUES('attachment_storage_root',?)`, root); err != nil {
		t.Fatal(err)
	}
	engineerID := insertEngineer(t)
	router := newRouter()
	malformed := httptest.NewRequest(http.MethodPost, "/api/notes-with-attachment",
		strings.NewReader("not multipart"))
	malformed.Header.Set("Content-Type", "multipart/form-data; boundary=broken")
	if got := serveRequest(router, malformed); got.Code != 400 {
		t.Fatalf("malformed atomic request: %d", got.Code)
	}

	tests := []struct {
		fields map[string]string
		file   []byte
		name   string
		code   int
	}{
		{map[string]string{"engineerId": "bad"}, pngFixture, "x.png", 400},
		{map[string]string{"engineerId": strconv.FormatInt(engineerID, 10)}, pngFixture, "x.png", 400},
		{map[string]string{"engineerId": "999", "noteDate": "2026-07-25", "category": "Delivery", "summary": "S"}, pngFixture, "x.png", 404},
		{map[string]string{"engineerId": strconv.FormatInt(engineerID, 10), "noteDate": "2026-07-25", "category": "Delivery", "summary": "S"}, nil, "", 400},
		{map[string]string{"engineerId": strconv.FormatInt(engineerID, 10), "noteDate": "2026-07-25", "category": "Delivery", "summary": "S"}, []byte("bad"), "x.txt", 400},
	}
	for _, test := range tests {
		got := serveRequest(router, multipartRequest(t, http.MethodPost, "/api/notes-with-attachment", test.name, test.file, test.fields))
		if got.Code != test.code {
			t.Errorf("fields %#v: got %d want %d (%s)", test.fields, got.Code, test.code, got.Body.String())
		}
	}
	fields := map[string]string{
		"engineerId": strconv.FormatInt(engineerID, 10), "noteDate": "2026-07-25",
		"category": "Delivery", "summary": "Atomic", "details": "D", "impact": "High",
		"followUpNeeded": "true", "reviewCycle": "2026-H1", "sourceSystem": "Jira",
	}
	got := serveRequest(router, multipartRequest(t, http.MethodPost, "/api/notes-with-attachment", "atomic.png", pngFixture, fields))
	if got.Code != 201 || !strings.Contains(got.Body.String(), `"summary":"Atomic"`) {
		t.Fatalf("atomic create: %d %s", got.Code, got.Body.String())
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM performance_notes`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("note count = %d, %v", count, err)
	}
}

func TestClosedDatabaseErrors(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	db.Close()
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	t.Setenv("MANAGER_DASHBOARD_ENCRYPTION_KEY", key)
	cases := []struct {
		method string
		target string
		body   []byte
	}{
		{"GET", "/api/engineers", nil},
		{"POST", "/api/engineers", []byte(`{"name":"A","role":"R","level":"L","team":"T","reviewCycle":"C"}`)},
		{"GET", "/api/notes", nil},
		{"POST", "/api/notes", []byte(`{"engineerId":1,"noteDate":"d","category":"c","summary":"s"}`)},
		{"PUT", "/api/notes/1", []byte(`{"engineerId":1}`)},
		{"DELETE", "/api/notes/1", nil},
		{"GET", "/api/settings", nil},
		{"PUT", "/api/settings/theme", []byte(`{"value":"dark"}`)},
		{"GET", "/api/integrations", nil},
		{"PUT", "/api/integrations/github", []byte(`{"secret":"x"}`)},
		{"DELETE", "/api/integrations/github", nil},
		{"GET", "/api/notes/1/attachments", nil},
		{"GET", "/api/attachments/1/content", nil},
		{"DELETE", "/api/attachments/1", nil},
	}
	for _, test := range cases {
		got := request(t, router, test.method, test.target, test.body)
		if got.Code != http.StatusInternalServerError {
			t.Errorf("%s %s = %d, want 500", test.method, test.target, got.Code)
		}
	}
	if _, err := getSettingValue("theme"); err == nil {
		t.Fatal("closed database setting lookup should fail")
	}
	atomic := multipartRequest(t, http.MethodPost, "/api/notes-with-attachment", "x.png", pngFixture,
		map[string]string{"engineerId": "1", "noteDate": "d", "category": "c", "summary": "s"})
	if got := serveRequest(router, atomic); got.Code != 500 {
		t.Fatalf("closed database atomic create = %d", got.Code)
	}
	if _, err := persistAttachment(parentNote, 1, &storedAttachment{}, "", "", "", ""); err == nil {
		t.Fatal("closed database attachment persistence should fail")
	}
	if _, _, err := persistNoteWithAttachment(
		CreateNoteWithAttachmentInput{EngineerID: 1}, "A", &storedAttachment{}); err == nil {
		t.Fatal("closed database atomic persistence should fail")
	}
}

type failingMultipartFile struct{}

func (failingMultipartFile) Read([]byte) (int, error)          { return 0, errors.New("read failed") }
func (failingMultipartFile) ReadAt([]byte, int64) (int, error) { return 0, errors.New("read failed") }
func (failingMultipartFile) Seek(int64, int) (int64, error)    { return 0, errors.New("seek failed") }
func (failingMultipartFile) Close() error                      { return nil }

var _ multipart.File = failingMultipartFile{}
var _ io.Reader = failingMultipartFile{}

func TestStoreAttachmentReadFailure(t *testing.T) {
	setupTestDatabase(t)
	if _, err := db.Exec(`INSERT INTO application_settings(setting_key,setting_value)
		VALUES('attachment_storage_root',?)`, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := storeAttachmentFile(failingMultipartFile{},
		&multipart.FileHeader{Filename: "x.png"}, 1, "Ada"); err == nil {
		t.Fatal("read failure should propagate")
	}
}

func TestInitializeDatabaseFailure(t *testing.T) {
	closed, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()
	if err := initializeDatabase(closed); err == nil {
		t.Fatal("closed database should fail initialization")
	}
	broken, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer broken.Close()
	if _, err := broken.Exec(`CREATE TABLE application_settings (wrong TEXT)`); err != nil {
		t.Fatal(err)
	}
	if err := initializeDatabase(broken); err == nil {
		t.Fatal("incompatible schema should fail initialization")
	}
}

func TestGoalRoutes(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	router := newRouter()
	base := "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/goals"

	if got := request(t, router, http.MethodGet, "/api/engineers/nope/goals", nil); got.Code != 400 {
		t.Fatalf("invalid engineer goals = %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, base+"?status=unknown", nil); got.Code != 400 {
		t.Fatalf("invalid status filter = %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, base, nil); got.Code != 200 || got.Body.String() != "[]\n" {
		t.Fatalf("empty goals = %d %s", got.Code, got.Body.String())
	}

	valid := Goal{
		Title: "Lead architecture", Description: "Own the design",
		GoalType: "leadership", Status: "in_progress", Priority: "high",
		StartDate: "2026-07-01", TargetDate: "2026-12-31",
		ProgressPercent: 25, SuccessCriteria: "Approved design",
		ManagerNotes: "Provide sponsorship", EngineerNotes: "Drafting RFC",
		ReviewCycle: "2026-H2",
	}
	invalidBodies := [][]byte{
		[]byte("{"),
		[]byte(`{"goalType":"leadership","status":"in_progress","priority":"high"}`),
		[]byte(`{"title":"X","goalType":"bad","status":"in_progress","priority":"high"}`),
		[]byte(`{"title":"X","goalType":"leadership","status":"bad","priority":"high"}`),
		[]byte(`{"title":"X","goalType":"leadership","status":"in_progress","priority":"bad"}`),
		[]byte(`{"title":"X","goalType":"leadership","status":"in_progress","priority":"high","progressPercent":101}`),
		[]byte(`{"title":"X","goalType":"leadership","status":"in_progress","priority":"high","startDate":"07/01/2026"}`),
		[]byte(`{"title":"X","goalType":"leadership","status":"in_progress","priority":"high","startDate":"2026-08-01","targetDate":"2026-07-01"}`),
		[]byte(`{"title":"X","goalType":"leadership","status":"completed","priority":"high","progressPercent":90,"completionDate":"2026-07-01"}`),
		[]byte(`{"title":"X","goalType":"leadership","status":"completed","priority":"high","progressPercent":100}`),
	}
	for index, body := range invalidBodies {
		if got := request(t, router, http.MethodPost, base, body); got.Code != 400 {
			t.Errorf("invalid goal %d = %d %s", index, got.Code, got.Body.String())
		}
	}
	validJSON, _ := json.Marshal(valid)
	if got := request(t, router, http.MethodPost, "/api/engineers/999/goals", validJSON); got.Code != 404 {
		t.Fatalf("missing engineer goal = %d %s", got.Code, got.Body.String())
	}
	created := request(t, router, http.MethodPost, base, validJSON)
	if created.Code != 201 {
		t.Fatalf("create goal = %d %s", created.Code, created.Body.String())
	}
	var goal Goal
	if err := json.Unmarshal(created.Body.Bytes(), &goal); err != nil {
		t.Fatal(err)
	}
	if goal.ID <= 0 || goal.EngineerID != engineerID || goal.ProgressPercent != 25 {
		t.Fatalf("created goal = %#v", goal)
	}

	for _, target := range []string{
		base,
		base + "?status=in_progress",
		base + "?reviewCycle=2026-H2",
		"/api/goals/" + strconv.FormatInt(goal.ID, 10),
	} {
		if got := request(t, router, http.MethodGet, target, nil); got.Code != 200 ||
			!strings.Contains(got.Body.String(), "Lead architecture") {
			t.Errorf("get %s = %d %s", target, got.Code, got.Body.String())
		}
	}
	if got := request(t, router, http.MethodGet, "/api/goals/nope", nil); got.Code != 400 {
		t.Fatalf("invalid goal get = %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, "/api/goals/999", nil); got.Code != 404 {
		t.Fatalf("missing goal get = %d", got.Code)
	}
	if got := request(t, router, http.MethodPut, "/api/goals/nope", validJSON); got.Code != 400 {
		t.Fatalf("invalid goal update = %d", got.Code)
	}
	if got := request(t, router, http.MethodPut, "/api/goals/999", validJSON); got.Code != 404 {
		t.Fatalf("missing goal update = %d", got.Code)
	}
	valid.Status = "completed"
	valid.ProgressPercent = 100
	valid.CompletionDate = "2026-07-25"
	updatedJSON, _ := json.Marshal(valid)
	updated := request(t, router, http.MethodPut,
		"/api/goals/"+strconv.FormatInt(goal.ID, 10), updatedJSON)
	if updated.Code != 200 || !strings.Contains(updated.Body.String(), `"status":"completed"`) {
		t.Fatalf("update goal = %d %s", updated.Code, updated.Body.String())
	}
	if got := request(t, router, http.MethodDelete, "/api/goals/nope", nil); got.Code != 400 {
		t.Fatalf("invalid goal delete = %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/goals/999", nil); got.Code != 404 {
		t.Fatalf("missing goal delete = %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete,
		"/api/goals/"+strconv.FormatInt(goal.ID, 10), nil); got.Code != 204 {
		t.Fatalf("delete goal = %d %s", got.Code, got.Body.String())
	}
}

func TestGoalDatabaseErrors(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	db.Close()
	valid := []byte(`{
		"title":"X","goalType":"delivery","status":"not_started",
		"priority":"medium","progressPercent":0
	}`)
	cases := []struct {
		method string
		target string
		body   []byte
	}{
		{http.MethodGet, "/api/engineers/1/goals", nil},
		{http.MethodPost, "/api/engineers/1/goals", valid},
		{http.MethodGet, "/api/goals/1", nil},
		{http.MethodPut, "/api/goals/1", valid},
		{http.MethodDelete, "/api/goals/1", nil},
	}
	for _, test := range cases {
		if got := request(t, router, test.method, test.target, test.body); got.Code != 500 {
			t.Errorf("%s %s = %d, want 500", test.method, test.target, got.Code)
		}
	}
}

func TestOneOnOneRoutes(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	router := newRouter()
	base := "/api/engineers/" + strconv.FormatInt(engineerID, 10) + "/one-on-ones"

	if got := request(t, router, http.MethodGet, "/api/engineers/nope/one-on-ones", nil); got.Code != 400 {
		t.Fatalf("invalid engineer meetings = %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, base+"?status=unknown", nil); got.Code != 400 {
		t.Fatalf("invalid meeting filter = %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, base, nil); got.Code != 200 || got.Body.String() != "[]\n" {
		t.Fatalf("empty meetings = %d %s", got.Code, got.Body.String())
	}

	invalidBodies := [][]byte{
		[]byte("{"),
		[]byte(`{"status":"scheduled"}`),
		[]byte(`{"meetingDate":"07/25/2026","status":"scheduled"}`),
		[]byte(`{"meetingDate":"2026-07-25","followUpDate":"tomorrow","status":"scheduled"}`),
		[]byte(`{"meetingDate":"2026-07-25","status":"unknown"}`),
	}
	for index, body := range invalidBodies {
		if got := request(t, router, http.MethodPost, base, body); got.Code != 400 {
			t.Errorf("invalid meeting %d = %d %s", index, got.Code, got.Body.String())
		}
	}
	meeting := OneOnOne{
		MeetingDate: "2026-07-25", Wins: "Shipped the release",
		Challenges: "Cross-team dependency", CareerDiscussion: "Staff path",
		Feedback: "Increase design visibility", ManagerTopics: "Roadmap",
		EngineerTopics:      "Architecture ownership",
		PrivateManagerNotes: "Watch support load", SharedNotes: "Draft RFC",
		FollowUpDate: "2026-08-08", Status: "scheduled",
	}
	body, _ := json.Marshal(meeting)
	if got := request(t, router, http.MethodPost,
		"/api/engineers/999/one-on-ones", body); got.Code != 404 {
		t.Fatalf("missing engineer meeting = %d %s", got.Code, got.Body.String())
	}
	created := request(t, router, http.MethodPost, base, body)
	if created.Code != 201 {
		t.Fatalf("create meeting = %d %s", created.Code, created.Body.String())
	}
	var saved OneOnOne
	if err := json.Unmarshal(created.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.ID <= 0 || saved.EngineerID != engineerID ||
		saved.PrivateManagerNotes != "Watch support load" {
		t.Fatalf("created meeting = %#v", saved)
	}
	id := strconv.FormatInt(saved.ID, 10)
	for _, target := range []string{
		base, base + "?status=scheduled", "/api/one-on-ones/" + id,
	} {
		if got := request(t, router, http.MethodGet, target, nil); got.Code != 200 ||
			!strings.Contains(got.Body.String(), "Shipped the release") {
			t.Errorf("get %s = %d %s", target, got.Code, got.Body.String())
		}
	}
	if got := request(t, router, http.MethodGet, "/api/one-on-ones/nope", nil); got.Code != 400 {
		t.Fatalf("invalid meeting get = %d", got.Code)
	}
	if got := request(t, router, http.MethodGet, "/api/one-on-ones/999", nil); got.Code != 404 {
		t.Fatalf("missing meeting get = %d", got.Code)
	}
	if got := request(t, router, http.MethodPut, "/api/one-on-ones/nope", body); got.Code != 400 {
		t.Fatalf("invalid meeting update = %d", got.Code)
	}
	if got := request(t, router, http.MethodPut, "/api/one-on-ones/999", body); got.Code != 404 {
		t.Fatalf("missing meeting update = %d", got.Code)
	}
	meeting.Status = "completed"
	meeting.SharedNotes = "RFC published"
	updatedBody, _ := json.Marshal(meeting)
	if got := request(t, router, http.MethodPut, "/api/one-on-ones/"+id, updatedBody); got.Code != 200 ||
		!strings.Contains(got.Body.String(), `"status":"completed"`) {
		t.Fatalf("update meeting = %d %s", got.Code, got.Body.String())
	}
	if got := request(t, router, http.MethodDelete, "/api/one-on-ones/nope", nil); got.Code != 400 {
		t.Fatalf("invalid meeting delete = %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/one-on-ones/999", nil); got.Code != 404 {
		t.Fatalf("missing meeting delete = %d", got.Code)
	}
	if got := request(t, router, http.MethodDelete, "/api/one-on-ones/"+id, nil); got.Code != 204 {
		t.Fatalf("delete meeting = %d %s", got.Code, got.Body.String())
	}
}

func TestOneOnOneDatabaseErrors(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	db.Close()
	valid := []byte(`{"meetingDate":"2026-07-25","status":"scheduled"}`)
	cases := []struct {
		method string
		target string
		body   []byte
	}{
		{http.MethodGet, "/api/engineers/1/one-on-ones", nil},
		{http.MethodPost, "/api/engineers/1/one-on-ones", valid},
		{http.MethodGet, "/api/one-on-ones/1", nil},
		{http.MethodPut, "/api/one-on-ones/1", valid},
		{http.MethodDelete, "/api/one-on-ones/1", nil},
	}
	for _, test := range cases {
		if got := request(t, router, test.method, test.target, test.body); got.Code != 500 {
			t.Errorf("%s %s = %d, want 500", test.method, test.target, got.Code)
		}
	}
}
