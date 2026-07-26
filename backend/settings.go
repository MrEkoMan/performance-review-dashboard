package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

var allowedSettings = map[string]bool{"theme": true, "attachment_storage_root": true}

func getApplicationSettings(w http.ResponseWriter, _ *http.Request) {
	rows, err := db.Query(`SELECT setting_key, setting_value FROM application_settings`)
	if err != nil {
		http.Error(w, "Failed to retrieve settings.", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			http.Error(w, "Failed to read settings.", http.StatusInternalServerError)
			return
		}
		settings[key] = value
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading settings.", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func updateApplicationSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if !allowedSettings[key] {
		http.Error(w, "Unsupported setting", http.StatusBadRequest)
		return
	}
	var setting ApplicationSetting
	if err := json.NewDecoder(r.Body).Decode(&setting); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	setting.Value = strings.TrimSpace(setting.Value)
	if key == "theme" && setting.Value != "light" && setting.Value != "dark" {
		http.Error(w, "Theme must be light or dark", http.StatusBadRequest)
		return
	}
	if key == "attachment_storage_root" {
		if setting.Value == "" {
			http.Error(w, "Attachment storage root cannot be empty", http.StatusBadRequest)
			return
		}
		absolutePath, err := filepath.Abs(setting.Value)
		if err != nil {
			http.Error(w, "Attachment storage root is invalid", http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(absolutePath, 0700); err != nil {
			http.Error(w, "Attachment storage root could not be created", http.StatusBadRequest)
			return
		}
		setting.Value = absolutePath
	}
	_, err := db.Exec(`
		INSERT INTO application_settings (setting_key, setting_value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(setting_key) DO UPDATE SET
			setting_value = excluded.setting_value,
			updated_at = CURRENT_TIMESTAMP`, key, setting.Value)
	if err != nil {
		http.Error(w, "Failed to update setting", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func getSettingValue(key string) (string, error) {
	var value string
	err := db.QueryRow(`
		SELECT setting_value FROM application_settings WHERE setting_key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

func sanitizeFolderName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	previousDash := false
	for _, character := range value {
		valid := character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9'
		if valid {
			builder.WriteRune(character)
			previousDash = false
		} else if !previousDash {
			builder.WriteRune('-')
			previousDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
