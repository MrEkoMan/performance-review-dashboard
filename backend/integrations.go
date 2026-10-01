package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

var allowedProviders = map[string]bool{
	"github": true, "gitlab": true, "jira": true, "slack": true, "teams": true,
}

// allowedDeploymentTypes constrains the Jira connection-test behavior. Empty
// means auto-detect; "cloud" and "server" force the Jira Cloud or internally
// hosted (Server / Data Center) request shape.
var allowedDeploymentTypes = map[string]bool{
	"": true, "cloud": true, "server": true,
}

func getIntegrationCredentials(w http.ResponseWriter, _ *http.Request) {
	rows, err := db.Query(`
		SELECT provider, COALESCE(account_label, ''), COALESCE(base_url, ''),
			encrypted_secret <> '', enabled, COALESCE(deployment_type, ''),
			updated_at
		FROM integration_credentials ORDER BY provider`)
	if err != nil {
		http.Error(w, "Failed to retrieve integrations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	integrations := make([]IntegrationCredentialResponse, 0)
	for rows.Next() {
		var integration IntegrationCredentialResponse
		if err := rows.Scan(&integration.Provider, &integration.AccountLabel,
			&integration.BaseURL, &integration.HasSecret, &integration.Enabled,
			&integration.DeploymentType, &integration.UpdatedAt); err != nil {
			http.Error(w, "Failed to read integration", http.StatusInternalServerError)
			return
		}
		integrations = append(integrations, integration)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading integrations", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, integrations)
}

func saveIntegrationCredential(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if !allowedProviders[provider] {
		http.Error(w, "Unsupported integration provider.", http.StatusBadRequest)
		return
	}
	var input IntegrationCredentialInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body.", http.StatusBadRequest)
		return
	}
	if input.Secret == "" {
		http.Error(w, "Credential is required.", http.StatusBadRequest)
		return
	}
	input.DeploymentType = strings.TrimSpace(input.DeploymentType)
	if !allowedDeploymentTypes[input.DeploymentType] {
		http.Error(w, "Unsupported Jira deployment type.", http.StatusBadRequest)
		return
	}
	encryptedSecret, err := encryptSecret(input.Secret)
	if err != nil {
		http.Error(w, "Credential encryption is not configured.", http.StatusInternalServerError)
		return
	}
	_, err = db.Exec(`
		INSERT INTO integration_credentials
			(provider, account_label, base_url, encrypted_secret, enabled,
			 deployment_type, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(provider) DO UPDATE SET
			account_label = excluded.account_label,
			base_url = excluded.base_url,
			encrypted_secret = excluded.encrypted_secret,
			enabled = excluded.enabled,
			deployment_type = excluded.deployment_type,
			updated_at = CURRENT_TIMESTAMP`,
		provider, input.AccountLabel, input.BaseURL, encryptedSecret,
		input.Enabled, input.DeploymentType)
	if err != nil {
		http.Error(w, "Failed to save integration.", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func deleteIntegrationCredential(w http.ResponseWriter, r *http.Request) {
	result, err := db.Exec(`DELETE FROM integration_credentials WHERE provider = ?`,
		chi.URLParam(r, "provider"))
	if err != nil {
		http.Error(w, "Failed to remove integration.", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm integration removal.", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Integration not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
