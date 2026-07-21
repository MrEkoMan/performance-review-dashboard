import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

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
        baseUrlLabel: "API or Enterprise URL",
    },
    {
        id: "gitlab",
        name: "GitLab",
        baseUrlLabel: "API or Enterprise URL",
    },
    {
        id: "jira",
        name: "Jira",
        baseUrlLabel: "Jira Base URL",
    },
    {
        id: "slack",
        name: "Slack",
        baseUrlLabel: "Workspace URL",
    },
    {
        id: "teams",
        name: "Microsoft Teams",
        baseUrlLabel: "Webhook or Tenant URL",
    },
];

function createEmptyIntegration() {
    return {
        accountLabel: "",
        baseUrl: "",
        secret: "",
        enabled: true,
        hasSecret: false,
    };
}

function SettingsPage() {
    const [theme, setTheme] = useState("light");
    const [integrations, setIntegrations] = useState({});
    const [savingProvider, setSavingProvider] = useState("");
    const [error, setError] = useState("");

    async function loadSettings() {
        try {
            setError("");

            const [settingsData, integrationData] = await Promise.all([
                getSettings(),
                getIntegrations(),
            ]);

            const loadedTheme = settingsData?.theme || "light";
            setTheme(loadedTheme);

            document.documentElement.dataset.theme = loadedTheme;

            const integrationMap = {};

            providers.forEach((provider) => {
                integrationMap[provider.id] = createEmptyIntegration();
            });

            (Array.isArray(integrationData)
                ? integrationData
                : []
            ).forEach((integration) => {
                integrationMap[integration.provider] = {
                    ...createEmptyIntegration(),
                    ...integration,
                    secret: "",
                };
            });

            setIntegrations(integrationMap);
        } catch (err) {
            setError(err.message);
        }
    }

    useEffect(() => {
        loadSettings();
    }, []);

    async function handleThemeChange(event) {
        const nextTheme = event.target.value;

        setTheme(nextTheme);
        document.documentElement.dataset.theme = nextTheme;

        try {
            await updateSetting("theme", nextTheme);
        } catch (err) {
            setError(err.message);
        }
    }

    function updateIntegrationField(provider, field, value) {
        setIntegrations((current) => ({
            ...current,
            [provider]: {
                ...current[provider],
                [field]: value,
            },
        }));
    }

    async function handleSaveIntegration(provider) {
        const integration = integrations[provider];

        if (!integration.secret) {
            setError(
                `Enter a credential before saving ${provider}`
            );
            return;
        }

        try {
            setSavingProvider(provider);
            setError("");

            await saveIntegration(provider, integration);
            await loadSettings();
        } catch (err) {
            setError(err.message);
        } finally {
            setSavingProvider("");
        }
    }

    async function handleDeleteIntergration(provider) {
        const confirmed = window.confirm(
            `Remove the save ${provider} credential?`
        );

        if (!confirmed) {
            return;
        }

        try {
            setError("");
            await deleteIntegration(provider);
            await loadSettings();
        } catch (err) {
            setError(err.message);
        }
    }

    return (
        <main>
            <section className="settings-page">
                <div>
                    <Link to="/" className="back-link">
                        Back to Dashboard
                    </Link>

                    <h1>Settings</h1>
                    <p className="settings-description">
                        Manage appearance and locally stored integration credentials.
                    </p>
                </div>

                {error && <div className="error">{error}</div>}

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

                <div className="integration-grid">
                    {providers.map((provider) => {
                        const integration = 
                            integrations[provider.id] ||
                            createEmptyIntegration();

                        return (
                            <article
                                className="settings-card"
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
                                            : "Not Configured"
                                        }
                                    </span>
                                </div>

                                <label>Account or workspace label</label>
                                <input 
                                    value={integration.accountLabel}
                                    onChange={(event) => 
                                        updateIntegrationField(
                                            provider.id,
                                            "accountLabel",
                                            event.target.value
                                        )
                                    }
                                />

                                <label>{provider.baseUrlLabel}</label>
                                <input 
                                    value={integration.baseUrl}
                                    onChange={(event) => 
                                        updateIntegrationField(
                                            provider.id,
                                            "baseUrl",
                                            event.target.value
                                        )
                                    }
                                />

                                <label>Credentials</label>
                                <input 
                                    type="password"
                                    value={integration.secret}
                                    placeholder={
                                        integration.hasSecret
                                            ? "Enter a new value to replace"
                                            : "Enter credential"   
                                    }
                                    onChange={(event) => 
                                        updateIntegrationField(
                                            provider.id,
                                            "secret",
                                            event.target.value
                                        )
                                    }
                                />

                                <label className="checkbox-row">
                                    <input 
                                        type="checkbox"
                                        checked={integration.enabled}
                                        onChange={(event) => (
                                            provider.id,
                                            "enabled",
                                            event.target.checked
                                            )
                                        }
                                    />
                                    Enabled
                                </label>

                                <div className="form-actions">
                                    <button
                                        type="button"
                                        onClick={() => 
                                            handleSaveIntegration(provider.id)
                                        }
                                        disabled={
                                            savingProvider === provider.id
                                        }
                                    >
                                        {savingProvider === provider.id
                                            ? "Saving..."
                                            : "Save"
                                        }
                                    </button>

                                    {integration.hasSecret && (
                                        <button
                                            type="button"
                                            className="danger-button"
                                            onClick={() =>
                                                handleDeleteIntergration(provider.id)
                                            }
                                        >
                                            Remove
                                        </button>
                                    )}
                                </div>
                            </article>
                        );
                    })}
                </div>
            </section>
        </main>
    );
}

export default SettingsPage;
