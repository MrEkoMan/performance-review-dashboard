package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const maxAttachmentSize = 10 << 20

var allowedAttachmentTypes = map[string]string{
	"image/png": ".png", "image/jpeg": ".jpg", "image/webp": ".webp",
}

type storedAttachment struct {
	originalName string
	relativePath string
	absolutePath string
	mimeType     string
	size         int64
	hash         string
}

func generateStoredFilename(extension string) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(randomBytes) + extension, nil
}

func attachmentDirectoryForEngineer(engineerID int64, engineerName string) (string, error) {
	root, err := attachmentStorageRoot()
	if err != nil {
		return "", err
	}
	safeName := sanitizeFolderName(engineerName)
	if safeName == "" {
		safeName = "engineer"
	}
	now := time.Now()
	directory := filepath.Join(root, "engineers",
		fmt.Sprintf("%d-%s", engineerID, safeName),
		strconv.Itoa(now.Year()), now.Format("01"))
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	return directory, nil
}

func attachmentStorageRoot() (string, error) {
	root, err := getSettingValue("attachment_storage_root")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(root) == "" {
		return "", errors.New("attachment storage root is not configured")
	}
	return filepath.Abs(root)
}

func validateImageType(header []byte, originalFilename string) (string, string, error) {
	detectedType := http.DetectContentType(header)
	extension, allowed := allowedAttachmentTypes[detectedType]
	if !allowed {
		return "", "", errors.New("only PNG, JPEG, and WebP images are supported")
	}
	if detectedType == "image/jpeg" &&
		strings.EqualFold(filepath.Ext(originalFilename), ".jpeg") {
		extension = ".jpeg"
	}
	return detectedType, extension, nil
}

func parseAttachmentID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func resolveAttachmentPath(storedPath string) (string, error) {
	root, err := attachmentStorageRoot()
	if err != nil {
		return "", err
	}
	resolved := filepath.Join(root, filepath.Clean(storedPath))
	relative, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", errors.New("attachment path is outside the configured root")
	}
	return resolved, nil
}

func storeAttachmentFile(
	file multipart.File,
	header *multipart.FileHeader,
	engineerID int64,
	engineerName string,
) (*storedAttachment, error) {
	sniff := make([]byte, 512)
	count, err := io.ReadFull(file, sniff)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("read attachment header: %w", err)
	}
	mimeType, extension, err := validateImageType(sniff[:count], header.Filename)
	if err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind attachment: %w", err)
	}
	directory, err := attachmentDirectoryForEngineer(engineerID, engineerName)
	if err != nil {
		return nil, err
	}
	filename, err := generateStoredFilename(extension)
	if err != nil {
		return nil, err
	}
	absolutePath := filepath.Join(directory, filename)
	output, err := os.OpenFile(absolutePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("create attachment: %w", err)
	}
	hasher := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(output, hasher), io.LimitReader(file, maxAttachmentSize+1))
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil || size == 0 || size > maxAttachmentSize {
		_ = os.Remove(absolutePath)
		if copyErr != nil {
			return nil, fmt.Errorf("copy attachment: %w", copyErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close attachment: %w", closeErr)
		}
		if size == 0 {
			return nil, errors.New("attachment file is empty")
		}
		return nil, errors.New("attachment exceeds the 10 MB limit")
	}
	root, err := attachmentStorageRoot()
	if err != nil {
		_ = os.Remove(absolutePath)
		return nil, err
	}
	relativePath, err := filepath.Rel(root, absolutePath)
	if err != nil || relativePath == ".." ||
		strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) {
		_ = os.Remove(absolutePath)
		return nil, errors.New("attachment storage path is invalid")
	}
	return &storedAttachment{
		originalName: filepath.Base(header.Filename),
		relativePath: relativePath,
		absolutePath: absolutePath,
		mimeType:     mimeType,
		size:         size,
		hash:         hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}
