import { useEffect, useState } from "react";
import { ArrowLeft, Trash2 } from "lucide-react";
import { Link } from "react-router-dom";
import { useThemeMode } from "../theme.jsx";

import {
  deleteIntegration,
  getIntegrations,
  getSettings,
  saveIntegration,
  updateSetting,
} from "../api/performanceApi.js";

const providers = [
  {
    id: "github",
    name: "GitHub",
    baseUrlLabel: "API or GitHub Enterprise URL",
    baseUrlPlaceholder: "https://api.github.com",
    secretLabel: "Personal access token",
  },
  {
    id: "jira",
    name: "Jira",
    baseUrlLabel: "Jira base URL",
    baseUrlPlaceholder: "https://your-company.atlassian.net",
    secretLabel: "API token",
  },
  {
    id: "slack",
    name: "Slack",
    baseUrlLabel: "Workspace URL",
    baseUrlPlaceholder: "https://your-workspace.slack.com",
    secretLabel: "Bot or user token",
  },
  {
    id: "teams",
    name: "Microsoft Teams",
    baseUrlLabel: "Webhook or tenant URL",
    baseUrlPlaceholder: "https://...",
    secretLabel: "Credential or webhook secret",
  },
];

function createEmptyIntegration() {
  return {
    accountLabel: "",
    baseUrl: "",
    secret: "",
    enabled: true,
    hasSecret: false,
    updatedAt: "",
  };
}

function SettingsPage() {
  const { setMode } = useThemeMode();
  const [theme, setTheme] = useState("light");

  const [storageRoot, setStorageRoot] = useState("");
  const [storageConfigured, setStorageConfigured] = useState(false);
  const [savingStorage, setSavingStorage] = useState(false);

  const [integrations, setIntegrations] = useState({});
  const [savingProvider, setSavingProvider] = useState("");
  const [deletingProvider, setDeletingProvider] = useState("");

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [successMessage, setSuccessMessage] = useState("");

  async function loadSettings() {
    try {
      setLoading(true);
      setError("");

      const [settingsData, integrationData] = await Promise.all([
        getSettings(),
        getIntegrations(),
      ]);

      const loadedTheme = settingsData?.theme || "light";
      const loadedStorageRoot =
        settingsData?.attachment_storage_root || "";

      setTheme(loadedTheme);
      setMode(loadedTheme);
      setStorageRoot(loadedStorageRoot);
      setStorageConfigured(Boolean(loadedStorageRoot));

      document.documentElement.dataset.theme = loadedTheme;

      const integrationMap = {};

      providers.forEach((provider) => {
        integrationMap[provider.id] = createEmptyIntegration();
      });

      const safeIntegrations = Array.isArray(integrationData)
        ? integrationData
        : [];

      safeIntegrations.forEach((integration) => {
        integrationMap[integration.provider] = {
          ...createEmptyIntegration(),
          ...integration,
          secret: "",
        };
      });

      setIntegrations(integrationMap);
    } catch (err) {
      console.error("Failed to load settings:", err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    // Initial API synchronization.
    loadSettings();
    // loadSettings is intentionally run once for initial API synchronization.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function clearMessages() {
    setError("");
    setSuccessMessage("");
  }

  async function handleThemeChange(event) {
    const previousTheme = theme;
    const nextTheme = event.target.value;

    clearMessages();

    setTheme(nextTheme);
    setMode(nextTheme);
    document.documentElement.dataset.theme = nextTheme;

    try {
      await updateSetting("theme", nextTheme);
      setSuccessMessage("Theme preference saved.");
    } catch (err) {
      setTheme(previousTheme);
      setMode(previousTheme);
      document.documentElement.dataset.theme = previousTheme;
      setError(err.message);
    }
  }

  async function handleSaveStorageRoot() {
    const normalizedStorageRoot = storageRoot.trim();

    if (!normalizedStorageRoot) {
      setError("Enter a local attachment storage folder.");
      return;
    }

    try {
      clearMessages();
      setSavingStorage(true);

      await updateSetting(
        "attachment_storage_root",
        normalizedStorageRoot
      );

      setStorageRoot(normalizedStorageRoot);
      setStorageConfigured(true);
      setSuccessMessage("Local attachment storage saved.");

      await loadSettings();
    } catch (err) {
      setError(err.message);
    } finally {
      setSavingStorage(false);
    }
  }

  function updateIntegrationField(provider, field, value) {
    clearMessages();

    setIntegrations((current) => ({
      ...current,
      [provider]: {
        ...createEmptyIntegration(),
        ...current[provider],
        [field]: value,
      },
    }));
  }

  async function handleSaveIntegration(provider) {
    const integration =
      integrations[provider] || createEmptyIntegration();

    if (!integration.secret.trim()) {
      setError(
        `Enter a credential before saving the ${provider} integration.`
      );
      return;
    }

    try {
      clearMessages();
      setSavingProvider(provider);

      await saveIntegration(provider, {
        accountLabel: integration.accountLabel.trim(),
        baseUrl: integration.baseUrl.trim(),
        secret: integration.secret,
        enabled: Boolean(integration.enabled),
      });

      setSuccessMessage(
        `${providers.find((item) => item.id === provider)?.name || provider} integration saved.`
      );

      await loadSettings();
    } catch (err) {
      setError(err.message);
    } finally {
      setSavingProvider("");
    }
  }

  async function handleDeleteIntegration(provider) {
    const providerName =
      providers.find((item) => item.id === provider)?.name ||
      provider;

    const confirmed = window.confirm(
      `Remove the saved ${providerName} credential?`
    );

    if (!confirmed) {
      return;
    }

    try {
      clearMessages();
      setDeletingProvider(provider);

      await deleteIntegration(provider);

      setSuccessMessage(
        `${providerName} integration removed.`
      );

      await loadSettings();
    } catch (err) {
      setError(err.message);
    } finally {
      setDeletingProvider("");
    }
  }

  if (loading) {
    return (
      <main>
        <p>Loading settings...</p>
      </main>
    );
  }

  return (
    <main>
      <section className="settings-page">
        <header className="settings-header">
          <div>
            <Link to="/" className="back-link">
              <ArrowLeft size={18} />
              Back to Dashboard
            </Link>

            <h1>Settings</h1>

            <p className="settings-description">
              Manage appearance, local evidence storage, and
              integration credentials.
            </p>
          </div>
        </header>

        {error && (
          <div className="error">
            Error: {error}
          </div>
        )}

        {successMessage && (
          <div className="success-message">
            {successMessage}
          </div>
        )}

        <article className="settings-card">
          <h2>Appearance</h2>

          <label htmlFor="theme-setting">Theme</label>

          <select
            id="theme-setting"
            value={theme}
            onChange={handleThemeChange}
          >
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </article>

        <article className="settings-card">
          <div className="settings-card-header">
            <div>
              <h2>Local Attachment Storage</h2>

              <p className="settings-description">
                Screenshots and supporting evidence will be
                stored beneath this root folder in
                engineer-specific directories.
              </p>
            </div>

            <span
              className={
                storageConfigured
                  ? "credential-status configured"
                  : "credential-status"
              }
            >
              {storageConfigured
                ? "Configured"
                : "Setup required"}
            </span>
          </div>

          <label htmlFor="attachment-storage-root">
            Root folder
          </label>

          <input
            id="attachment-storage-root"
            value={storageRoot}
            onChange={(event) => {
              clearMessages();
              setStorageRoot(event.target.value);
            }}
            placeholder="C:\ManagerDashboardData"
          />

          <p className="field-help">
            The Go backend must have permission to create and
            write files in this location.
          </p>

          <div className="form-actions">
            <button
              type="button"
              onClick={handleSaveStorageRoot}
              disabled={
                savingStorage || !storageRoot.trim()
              }
            >
              {savingStorage
                ? "Saving..."
                : "Save Storage Location"}
            </button>
          </div>
        </article>

        <section>
          <div className="settings-section-heading">
            <h2>Integrations</h2>

            <p className="settings-description">
              Credentials are encrypted before being stored in
              SQLite. Saved secrets are never returned to the
              browser.
            </p>
          </div>

          <div className="integration-grid">
            {providers.map((provider) => {
              const integration =
                integrations[provider.id] ||
                createEmptyIntegration();

              const isSaving =
                savingProvider === provider.id;

              const isDeleting =
                deletingProvider === provider.id;

              return (
                <article
                  className="settings-card integration-card"
                  key={provider.id}
                >
                  <div className="integration-header">
                    <h2>{provider.name}</h2>

                    <span
                      className={
                        integration.hasSecret
                          ? "credential-status configured"
                          : "credential-status"
                      }
                    >
                      {integration.hasSecret
                        ? "Configured"
                        : "Not configured"}
                    </span>
                  </div>

                  <label
                    htmlFor={`${provider.id}-account-label`}
                  >
                    Account or workspace label
                  </label>

                  <input
                    id={`${provider.id}-account-label`}
                    value={integration.accountLabel}
                    onChange={(event) =>
                      updateIntegrationField(
                        provider.id,
                        "accountLabel",
                        event.target.value
                      )
                    }
                    placeholder={`${provider.name} account`}
                  />

                  <label
                    htmlFor={`${provider.id}-base-url`}
                  >
                    {provider.baseUrlLabel}
                  </label>

                  <input
                    id={`${provider.id}-base-url`}
                    value={integration.baseUrl}
                    onChange={(event) =>
                      updateIntegrationField(
                        provider.id,
                        "baseUrl",
                        event.target.value
                      )
                    }
                    placeholder={provider.baseUrlPlaceholder}
                  />

                  <label
                    htmlFor={`${provider.id}-secret`}
                  >
                    {provider.secretLabel}
                  </label>

                  <input
                    id={`${provider.id}-secret`}
                    type="password"
                    value={integration.secret}
                    onChange={(event) =>
                      updateIntegrationField(
                        provider.id,
                        "secret",
                        event.target.value
                      )
                    }
                    placeholder={
                      integration.hasSecret
                        ? "Enter a new value to replace the saved credential"
                        : "Enter credential"
                    }
                    autoComplete="new-password"
                  />

                  <label className="checkbox-row">
                    <input
                      type="checkbox"
                      checked={Boolean(integration.enabled)}
                      onChange={(event) =>
                        updateIntegrationField(
                          provider.id,
                          "enabled",
                          event.target.checked
                        )
                      }
                    />
                    Enabled
                  </label>

                  {integration.updatedAt && (
                    <p className="field-help">
                      Last updated: {integration.updatedAt}
                    </p>
                  )}

                  <div className="form-actions">
                    <button
                      type="button"
                      onClick={() =>
                        handleSaveIntegration(provider.id)
                      }
                      disabled={isSaving || isDeleting}
                    >
                      {isSaving ? "Saving..." : "Save"}
                    </button>

                    {integration.hasSecret && (
                      <button
                        type="button"
                        className="icon-button danger"
                        title={`Remove ${provider.name} credential`}
                        aria-label={`Remove ${provider.name} credential`}
                        onClick={() =>
                          handleDeleteIntegration(provider.id)
                        }
                        disabled={isSaving || isDeleting}
                      >
                        <Trash2 size={17} />
                      </button>
                    )}
                  </div>
                </article>
              );
            })}
          </div>
        </section>
      </section>
    </main>
  );
}

export default SettingsPage;
