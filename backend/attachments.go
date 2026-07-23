package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const maxAttachmentSize = 10 << 20 //10MB

var allowedAttachmentTypes = map[string]string{
	"image/png": ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
}

func generateStoredFilename(extension, string) (string, error) {
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes) + extension, nil
}

func generateStoredFilename(extension string) (string, error) {
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes) + extension, nil
}

func attachmentDirectory() string {
	now := time.Now()

	return filepath.Join(
		"data",
		"attachments",
		strconv.Itoa(now.Year()),
		now.Format("01"),
	)
}

func validateImageType(
	header []byte,
	originalFilename string,
) (string, string, error) {
	detectedType := http.DetectContentType(header)

	extension, allowed := allowedAttachmentTypes[detectedType]
	if !allowed {
		return "", "", errors.New(
			"only PNG, JPEG and WebP images are supported",
		)
	}

	fileExtension := strings.ToLower(
		filepath.Ext(originalFilename),
	)

	if detectedType == "image/jpeg" && fileExtension == ".jpeg" {
		extension = ".jpeg"
	}

	return detectedType, extension, nil
}

func parseAttachmentID(r *http.Request) (int64, error) {
	return strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)
}