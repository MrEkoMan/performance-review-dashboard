package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// createRecognitionForTest inserts a recognition via the JSON API and returns its id.
func createRecognitionForTest(t *testing.T, router http.Handler, engineerID int64) int64 {
	t.Helper()
	body := []byte(`{"recognitionDate":"2026-08-25","source":"Peer shoutout","sourceType":"peer","category":"collaboration","summary":"Great pairing","reviewCycle":"2026-H2"}`)
	got := request(t, router, http.MethodPost,
		"/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/recognitions", body)
	if got.Code != http.StatusCreated {
		t.Fatalf("create recognition = %d %s", got.Code, got.Body.String())
	}
	var rec Recognition
	if err := json.Unmarshal(got.Body.Bytes(), &rec); err != nil {
		t.Fatal(err)
	}
	return rec.ID
}

func TestRecognitionAttachmentLifecycle(t *testing.T) {
	setupTestDatabase(t)
	root := t.TempDir()
	if _, err := db.Exec(`INSERT INTO application_settings(setting_key,setting_value)
		VALUES('attachment_storage_root',?)`, root); err != nil {
		t.Fatal(err)
	}
	engineerID := insertEngineer(t)
	router := newRouter()
	recognitionID := createRecognitionForTest(t, router, engineerID)
	recognitionIDString := strconv.FormatInt(recognitionID, 10)

	// Invalid recognition id and missing recognition return 400/404.
	if got := request(t, router, http.MethodGet, "/api/recognitions/nope/attachments", nil); got.Code != 400 {
		t.Fatalf("invalid recognition attachments: %d", got.Code)
	}
	if got := serveRequest(router, multipartRequest(t, http.MethodPost, "/api/recognitions/999/attachments", "x.png", pngFixture, nil)); got.Code != 404 {
		t.Fatalf("missing recognition upload: %d", got.Code)
	}

	// Empty list before any upload.
	if got := request(t, router, http.MethodGet, "/api/recognitions/"+recognitionIDString+"/attachments", nil); got.Code != 200 || got.Body.String() != "[]\n" {
		t.Fatalf("empty attachments: %d %s", got.Code, got.Body.String())
	}

	// Missing file / invalid file type.
	if got := serveRequest(router, multipartRequest(t, http.MethodPost, "/api/recognitions/"+recognitionIDString+"/attachments", "", nil, nil)); got.Code != 400 {
		t.Fatalf("missing file: %d %s", got.Code, got.Body.String())
	}
	if got := serveRequest(router, multipartRequest(t, http.MethodPost, "/api/recognitions/"+recognitionIDString+"/attachments", "x.txt", []byte("text"), nil)); got.Code != 400 {
		t.Fatalf("invalid file: %d", got.Code)
	}

	// Upload a screenshot.
	upload := multipartRequest(t, http.MethodPost, "/api/recognitions/"+recognitionIDString+"/attachments", "shot.png", pngFixture,
		map[string]string{"sourceSystem": "Microsoft Teams", "sourceAuthor": "Ada", "caption": "Praise"})
	created := serveRequest(router, upload)
	if created.Code != 201 || !strings.Contains(created.Body.String(), `"sourceSystem":"Microsoft Teams"`) {
		t.Fatalf("upload: %d %s", created.Code, created.Body.String())
	}
	var attachment Attachment
	if err := json.Unmarshal(created.Body.Bytes(), &attachment); err != nil {
		t.Fatal(err)
	}

	// List now contains it; content serves the bytes; delete removes it.
	listed := request(t, router, http.MethodGet, "/api/recognitions/"+recognitionIDString+"/attachments", nil)
	if listed.Code != 200 || !strings.Contains(listed.Body.String(), `"shot.png"`) {
		t.Fatalf("list: %d %s", listed.Code, listed.Body.String())
	}
	content := request(t, router, http.MethodGet, attachment.ContentURL, nil)
	if content.Code != 200 || content.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("content: %d %s", content.Code, content.Header().Get("Content-Type"))
	}
	if got := request(t, router, http.MethodDelete, "/api/attachments/"+strconv.FormatInt(attachment.ID, 10), nil); got.Code != 204 {
		t.Fatalf("delete: %d %s", got.Code, got.Body.String())
	}
	listed = request(t, router, http.MethodGet, "/api/recognitions/"+recognitionIDString+"/attachments", nil)
	if listed.Body.String() != "[]\n" {
		t.Fatalf("attachment should be removed after delete: %s", listed.Body.String())
	}
}

func TestRecognitionAttachmentUnconfiguredStorage(t *testing.T) {
	setupTestDatabase(t)
	engineerID := insertEngineer(t)
	router := newRouter()
	recognitionID := createRecognitionForTest(t, router, engineerID)

	// Upload with no storage root configured returns 428 (graceful).
	upload := multipartRequest(t, http.MethodPost, "/api/recognitions/"+strconv.FormatInt(recognitionID, 10)+"/attachments", "x.png", pngFixture, nil)
	if got := serveRequest(router, upload); got.Code != http.StatusPreconditionRequired {
		t.Fatalf("unconfigured upload: %d %s", got.Code, got.Body.String())
	}

	// Combined create-with-attachment also returns 428 when storage is unset.
	create := multipartRequest(t, http.MethodPost, "/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/recognitions-with-attachment", "x.png", pngFixture,
		map[string]string{"recognitionDate": "2026-08-25", "source": "Peer", "sourceType": "peer", "category": "collaboration", "summary": "s"})
	if got := serveRequest(router, create); got.Code != http.StatusPreconditionRequired {
		t.Fatalf("unconfigured atomic create: %d %s", got.Code, got.Body.String())
	}
}

func TestCreateRecognitionWithAttachment(t *testing.T) {
	setupTestDatabase(t)
	root := t.TempDir()
	if _, err := db.Exec(`INSERT INTO application_settings(setting_key,setting_value)
		VALUES('attachment_storage_root',?)`, root); err != nil {
		t.Fatal(err)
	}
	engineerID := insertEngineer(t)
	router := newRouter()

	// Invalid engineer id.
	if got := serveRequest(router, multipartRequest(t, http.MethodPost, "/api/engineers/nope/recognitions-with-attachment", "x.png", pngFixture,
		map[string]string{"recognitionDate": "2026-08-25", "source": "Peer", "sourceType": "peer", "category": "collaboration", "summary": "s"})); got.Code != 400 {
		t.Fatalf("invalid engineer create-with-attachment: %d", got.Code)
	}

	// Missing required fields.
	if got := serveRequest(router, multipartRequest(t, http.MethodPost, "/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/recognitions-with-attachment", "x.png", pngFixture,
		map[string]string{"recognitionDate": "", "source": "Peer", "sourceType": "peer", "category": "collaboration", "summary": "s"})); got.Code != 400 {
		t.Fatalf("missing date create-with-attachment: %d %s", got.Code, got.Body.String())
	}

	// Successful combined create returns both recognition and attachment.
	create := multipartRequest(t, http.MethodPost, "/api/engineers/"+strconv.FormatInt(engineerID, 10)+"/recognitions-with-attachment", "evidence.png", pngFixture,
		map[string]string{"recognitionDate": "2026-08-25", "source": "Peer shoutout", "sourceType": "peer", "category": "collaboration", "summary": "Great pairing", "reviewCycle": "2026-H2"})
	created := serveRequest(router, create)
	if created.Code != 201 {
		t.Fatalf("create-with-attachment: %d %s", created.Code, created.Body.String())
	}
	var result struct {
		Recognition Recognition `json:"recognition"`
		Attachment  Attachment  `json:"attachment"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Recognition.ID == 0 || result.Recognition.Summary != "Great pairing" {
		t.Fatalf("recognition = %+v", result.Recognition)
	}
	if result.Attachment.ID == 0 || result.Attachment.OriginalFilename != "evidence.png" {
		t.Fatalf("attachment = %+v", result.Attachment)
	}

	// The attachment is linked to the created recognition.
	listed := request(t, router, http.MethodGet, "/api/recognitions/"+strconv.FormatInt(result.Recognition.ID, 10)+"/attachments", nil)
	if listed.Code != 200 || !strings.Contains(listed.Body.String(), "evidence.png") {
		t.Fatalf("linked attachment list: %d %s", listed.Code, listed.Body.String())
	}
}
