package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

var allowedAIProviders = map[string]bool{
	"openai": true, "anthropic": true, "gemini": true,
	"azure_openai": true, "openrouter": true, "ollama": true,
}

func getAIProviderConfigurations(w http.ResponseWriter, _ *http.Request) {
	rows, err := db.Query(`
		SELECT provider, COALESCE(display_name, ''), base_url, model,
			COALESCE(api_version, ''), COALESCE(encrypted_api_key, '') <> '',
			enabled, updated_at
		FROM ai_provider_configurations ORDER BY provider`)
	if err != nil {
		http.Error(w, "Failed to retrieve AI provider settings", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	items := make([]AIProviderConfigurationResponse, 0)
	for rows.Next() {
		var item AIProviderConfigurationResponse
		if err := rows.Scan(
			&item.Provider, &item.DisplayName, &item.BaseURL, &item.Model,
			&item.APIVersion, &item.HasAPIKey, &item.Enabled, &item.UpdatedAt,
		); err != nil {
			http.Error(w, "Failed to read AI provider settings", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading AI provider settings", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func saveAIProviderConfiguration(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if !allowedAIProviders[provider] {
		http.Error(w, "Unsupported AI provider", http.StatusBadRequest)
		return
	}
	var input AIProviderConfigurationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.BaseURL = strings.TrimSpace(input.BaseURL)
	input.Model = strings.TrimSpace(input.Model)
	input.APIVersion = strings.TrimSpace(input.APIVersion)
	input.APIKey = strings.TrimSpace(input.APIKey)
	if input.BaseURL == "" || input.Model == "" {
		http.Error(w, "Base URL and model are required", http.StatusBadRequest)
		return
	}
	if err := validateAIProviderURL(provider, input.BaseURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	encryptedAPIKey := ""
	if input.APIKey != "" {
		var err error
		encryptedAPIKey, err = encryptSecret(input.APIKey)
		if err != nil {
			http.Error(w, "API key encryption is not configured", http.StatusInternalServerError)
			return
		}
	} else if provider != "ollama" {
		err := db.QueryRow(`
			SELECT COALESCE(encrypted_api_key, '')
			FROM ai_provider_configurations WHERE provider = ?`, provider,
		).Scan(&encryptedAPIKey)
		if errors.Is(err, sql.ErrNoRows) || encryptedAPIKey == "" {
			http.Error(w, "API key is required", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "Failed to retrieve existing AI provider", http.StatusInternalServerError)
			return
		}
	}
	_, err := db.Exec(`
		INSERT INTO ai_provider_configurations
			(provider, display_name, base_url, model, api_version,
			 encrypted_api_key, enabled, updated_at)
		VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, CURRENT_TIMESTAMP)
		ON CONFLICT(provider) DO UPDATE SET
			display_name = excluded.display_name,
			base_url = excluded.base_url,
			model = excluded.model,
			api_version = excluded.api_version,
			encrypted_api_key = COALESCE(
				excluded.encrypted_api_key,
				ai_provider_configurations.encrypted_api_key
			),
			enabled = excluded.enabled,
			updated_at = CURRENT_TIMESTAMP`,
		provider, input.DisplayName, input.BaseURL, input.Model,
		input.APIVersion, encryptedAPIKey, input.Enabled,
	)
	if err != nil {
		http.Error(w, "Failed to save AI provider settings", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func deleteAIProviderConfiguration(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if !allowedAIProviders[provider] {
		http.Error(w, "Unsupported AI provider", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`
		DELETE FROM ai_provider_configurations WHERE provider = ?`, provider)
	if err != nil {
		http.Error(w, "Failed to remove AI provider settings", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm AI provider removal", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "AI provider is not configured", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateAIProviderURL(provider, value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return errors.New("Base URL is invalid")
	}
	if provider != "ollama" {
		if parsed.Scheme != "https" {
			return errors.New("Hosted AI provider URLs must use HTTPS")
		}
		return nil
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("Ollama URL must use HTTP or HTTPS")
	}
	hostname := strings.ToLower(parsed.Hostname())
	if hostname != "localhost" && hostname != "::1" &&
		net.ParseIP(hostname) == nil {
		return errors.New("Ollama must use a loopback address")
	}
	ip := net.ParseIP(hostname)
	if ip != nil && !ip.IsLoopback() {
		return errors.New("Ollama must use a loopback address")
	}
	return nil
}
