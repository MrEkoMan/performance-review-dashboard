import { useEffect, useState } from "react";
import { ArrowLeft, Trash2 } from "lucide-react";
import { Link } from "react-router-dom";
import { useThemeMode } from "../theme.jsx";
import AIProvidersSettings from "../components/AIProvidersSettings.jsx";

import {
  deleteIntegration,
  createReviewPeriod,
  deleteReviewPeriod,
  getIntegrations,
  getReviewPeriods,
  getSettings,
  saveIntegration,
  testIntegration,
  updateReviewPeriod,
  updateSetting,
} from "../api/performanceApi.js";

const providers = [
  {
    id: "github",
    name: "GitHub",
    accountLabel: "Account label",
    baseUrlLabel: "API or GitHub Enterprise URL",
    baseUrlPlaceholder: "https://api.github.com",
    secretLabel: "Personal access token",
  },
  {
    id: "jira",
    name: "Jira",
    accountLabel: "Atlassian account email",
    baseUrlLabel: "Jira base URL",
    baseUrlPlaceholder: "https://your-company.atlassian.net",
    secretLabel: "API token",
  },
  {
    id: "slack",
    name: "Slack",
    accountLabel: "Workspace label",
    baseUrlLabel: "Slack API base URL",
    baseUrlPlaceholder: "https://slack.com",
    secretLabel: "Bot or user token",
  },
  {
    id: "teams",
    name: "Microsoft Teams",
    accountLabel: "Tenant or account label",
    baseUrlLabel: "Microsoft Graph base URL",
    baseUrlPlaceholder: "https://graph.microsoft.com/v1.0",
    secretLabel: "Delegated Microsoft Graph access token",
    help: "Connection tests use the read-only /me endpoint. Incoming webhooks are not tested because that would send a message.",
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
  const [testingProvider, setTestingProvider] = useState("");
  const [connectionResults, setConnectionResults] = useState({});
  const [reviewPeriods, setReviewPeriods] = useState([]);
  const [reviewPeriodForm, setReviewPeriodForm] = useState({
    label: "",
    startDate: "",
    endDate: "",
  });
  const [editingReviewPeriodId, setEditingReviewPeriodId] = useState(null);
  const [savingReviewPeriod, setSavingReviewPeriod] = useState(false);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [successMessage, setSuccessMessage] = useState("");

  async function loadSettings() {
    try {
      setLoading(true);
      setError("");

      const [settingsData, integrationData, reviewPeriodData] = await Promise.all([
        getSettings(),
        getIntegrations(),
        getReviewPeriods(),
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
      setReviewPeriods(Array.isArray(reviewPeriodData) ? reviewPeriodData : []);
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

  async function handleTestIntegration(provider) {
    try {
      clearMessages();
      setTestingProvider(provider);
      const result = await testIntegration(provider);
      setConnectionResults((current) => ({
        ...current,
        [provider]: result,
      }));
    } catch (err) {
      setConnectionResults((current) => ({
        ...current,
        [provider]: {
          success: false,
          category: "configuration",
          message: err.message,
        },
      }));
    } finally {
      setTestingProvider("");
    }
  }

  function resetReviewPeriodForm() {
    setEditingReviewPeriodId(null);
    setReviewPeriodForm({ label: "", startDate: "", endDate: "" });
  }

  function editReviewPeriod(period) {
    clearMessages();
    setEditingReviewPeriodId(period.id);
    setReviewPeriodForm({
      label: period.label,
      startDate: period.startDate,
      endDate: period.endDate,
    });
  }

  async function handleSaveReviewPeriod(event) {
    event.preventDefault();
    try {
      clearMessages();
      setSavingReviewPeriod(true);
      if (editingReviewPeriodId) {
        await updateReviewPeriod(editingReviewPeriodId, reviewPeriodForm);
      } else {
        await createReviewPeriod(reviewPeriodForm);
      }
      setSuccessMessage(
        editingReviewPeriodId ? "Review period updated." : "Review period created.",
      );
      resetReviewPeriodForm();
      await loadSettings();
    } catch (err) {
      setError(err.message);
    } finally {
      setSavingReviewPeriod(false);
    }
  }

  async function handleDeleteReviewPeriod(period) {
    if (!window.confirm(`Delete review period "${period.label}"?`)) {
      return;
    }
    try {
      clearMessages();
      await deleteReviewPeriod(period.id);
      setSuccessMessage("Review period deleted.");
      if (editingReviewPeriodId === period.id) {
        resetReviewPeriodForm();
      }
      await loadSettings();
    } catch (err) {
      setError(err.message);
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

        <article className="settings-card">
          <div className="settings-card-header">
            <div>
              <h2>Review Periods</h2>
              <p className="settings-description">
                Define dates for review-cycle labels used by engineers and evidence.
              </p>
            </div>
          </div>

          <form className="review-period-form" onSubmit={handleSaveReviewPeriod}>
            <label>
              Cycle label
              <input
                value={reviewPeriodForm.label}
                onChange={(event) => setReviewPeriodForm((current) => ({
                  ...current,
                  label: event.target.value,
                }))}
                placeholder="2026 H2"
                required
              />
            </label>
            <label>
              Start date
              <input
                type="date"
                value={reviewPeriodForm.startDate}
                onChange={(event) => setReviewPeriodForm((current) => ({
                  ...current,
                  startDate: event.target.value,
                }))}
                required
              />
            </label>
            <label>
              End date
              <input
                type="date"
                value={reviewPeriodForm.endDate}
                onChange={(event) => setReviewPeriodForm((current) => ({
                  ...current,
                  endDate: event.target.value,
                }))}
                required
              />
            </label>
            <div className="form-actions">
              <button type="submit" disabled={savingReviewPeriod}>
                {savingReviewPeriod
                  ? "Saving..."
                  : editingReviewPeriodId ? "Save period" : "Add period"}
              </button>
              {editingReviewPeriodId && (
                <button type="button" className="secondary-button" onClick={resetReviewPeriodForm}>
                  Cancel
                </button>
              )}
            </div>
          </form>

          <div className="review-period-list">
            {reviewPeriods.length === 0 ? (
              <p className="field-help">No structured review periods configured.</p>
            ) : reviewPeriods.map((period) => (
              <div className="review-period-item" key={period.id}>
                <div>
                  <strong>{period.label}</strong>
                  <span>{period.startDate} to {period.endDate}</span>
                </div>
                <span className={`credential-status ${period.phase === "active" ? "configured" : ""}`}>
                  {period.phase}
                </span>
                <div className="form-actions">
                  <button type="button" className="secondary-button" onClick={() => editReviewPeriod(period)}>
                    Edit
                  </button>
                  <button
                    type="button"
                    className="icon-button danger"
                    aria-label={`Delete ${period.label}`}
                    onClick={() => handleDeleteReviewPeriod(period)}
                  >
                    <Trash2 size={17} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        </article>

        <AIProvidersSettings />

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
              const isTesting =
                testingProvider === provider.id;
              const connectionResult =
                connectionResults[provider.id];

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
                    {provider.accountLabel}
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

                  {provider.help && (
                    <p className="field-help">{provider.help}</p>
                  )}

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

                  {connectionResult && (
                    <div
                      className={
                        connectionResult.success
                          ? "connection-result success"
                          : "connection-result failure"
                      }
                      role="status"
                    >
                      <strong>
                        {connectionResult.success
                          ? "Connection successful"
                          : "Connection failed"}
                      </strong>
                      <span>{connectionResult.message}</span>
                      {connectionResult.identity && (
                        <span>Authenticated as {connectionResult.identity}</span>
                      )}
                      {!connectionResult.success && connectionResult.category && (
                        <span>Category: {connectionResult.category.replaceAll("_", " ")}</span>
                      )}
                    </div>
                  )}

                  <div className="form-actions">
                    <button
                      type="button"
                      onClick={() =>
                        handleSaveIntegration(provider.id)
                      }
                      disabled={isSaving || isDeleting || isTesting}
                    >
                      {isSaving ? "Saving..." : "Save"}
                    </button>

                    {integration.hasSecret && (
                      <button
                        type="button"
                        className="secondary-button"
                        onClick={() => handleTestIntegration(provider.id)}
                        disabled={isSaving || isDeleting || isTesting || !integration.enabled}
                      >
                        {isTesting ? "Testing..." : "Test connection"}
                      </button>
                    )}

                    {integration.hasSecret && (
                      <button
                        type="button"
                        className="icon-button danger"
                        title={`Remove ${provider.name} credential`}
                        aria-label={`Remove ${provider.name} credential`}
                        onClick={() =>
                          handleDeleteIntegration(provider.id)
                        }
                        disabled={isSaving || isDeleting || isTesting}
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
