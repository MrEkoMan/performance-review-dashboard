import { useEffect, useState } from "react";
import { Trash2 } from "lucide-react";
import {
  deleteAIProvider,
  getAIProviders,
  saveAIProvider,
} from "../api/performanceApi.js";

const providers = [
  {
    id: "openai",
    name: "OpenAI",
    baseURL: "https://api.openai.com/v1",
    modelLabel: "Model ID",
  },
  {
    id: "anthropic",
    name: "Anthropic",
    baseURL: "https://api.anthropic.com",
    modelLabel: "Model ID",
  },
  {
    id: "gemini",
    name: "Google Gemini",
    baseURL: "https://generativelanguage.googleapis.com/v1beta",
    modelLabel: "Model ID",
  },
  {
    id: "azure_openai",
    name: "Azure OpenAI",
    baseURL: "",
    baseURLPlaceholder: "https://your-resource.openai.azure.com/openai/v1",
    modelLabel: "Deployment or model ID",
    supportsAPIVersion: true,
  },
  {
    id: "openrouter",
    name: "OpenRouter",
    baseURL: "https://openrouter.ai/api/v1",
    modelLabel: "Model slug",
  },
  {
    id: "ollama",
    name: "Ollama",
    baseURL: "http://localhost:11434",
    modelLabel: "Local model name",
    local: true,
  },
];

function emptyConfiguration(provider) {
  return {
    displayName: "",
    baseUrl: provider.baseURL,
    model: "",
    apiVersion: "",
    apiKey: "",
    hasApiKey: false,
    enabled: true,
    updatedAt: "",
  };
}

function AIProvidersSettings() {
  const [configurations, setConfigurations] = useState({});
  const [savingProvider, setSavingProvider] = useState("");
  const [deletingProvider, setDeletingProvider] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  async function loadConfigurations() {
    try {
      setLoading(true);
      const saved = await getAIProviders();
      const configurationMap = {};
      providers.forEach((provider) => {
        configurationMap[provider.id] = emptyConfiguration(provider);
      });
      (Array.isArray(saved) ? saved : []).forEach((item) => {
        const provider = providers.find((candidate) => candidate.id === item.provider);
        if (provider) {
          configurationMap[item.provider] = {
            ...emptyConfiguration(provider),
            ...item,
            apiKey: "",
          };
        }
      });
      setConfigurations(configurationMap);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadConfigurations();
  }, []);

  function updateField(provider, field, value) {
    setError("");
    setSuccess("");
    setConfigurations((current) => ({
      ...current,
      [provider]: {
        ...current[provider],
        [field]: value,
      },
    }));
  }

  async function save(provider) {
    try {
      setError("");
      setSuccess("");
      setSavingProvider(provider.id);
      await saveAIProvider(provider.id, configurations[provider.id]);
      setSuccess(`${provider.name} configuration saved.`);
      await loadConfigurations();
    } catch (err) {
      setError(err.message);
    } finally {
      setSavingProvider("");
    }
  }

  async function remove(provider) {
    if (!window.confirm(`Remove the ${provider.name} AI configuration?`)) {
      return;
    }
    try {
      setError("");
      setSuccess("");
      setDeletingProvider(provider.id);
      await deleteAIProvider(provider.id);
      setSuccess(`${provider.name} configuration removed.`);
      await loadConfigurations();
    } catch (err) {
      setError(err.message);
    } finally {
      setDeletingProvider("");
    }
  }

  if (loading) {
    return <p>Loading AI provider settings...</p>;
  }

  return (
    <section>
      <div className="settings-section-heading">
        <h2>AI Model Providers</h2>
        <p className="settings-description">
          Configure hosted model APIs or a local Ollama server. Saving settings
          does not send any application data to a model.
        </p>
      </div>

      {error && <div className="error">Error: {error}</div>}
      {success && <div className="success-message">{success}</div>}

      <div className="ai-provider-grid">
        {providers.map((provider) => {
          const configuration =
            configurations[provider.id] || emptyConfiguration(provider);
          const configured = configuration.updatedAt || configuration.hasApiKey;
          const busy =
            savingProvider === provider.id || deletingProvider === provider.id;
          return (
            <article className="settings-card ai-provider-card" key={provider.id}>
              <div className="integration-header">
                <div>
                  <h2>{provider.name}</h2>
                  {provider.local && <span className="field-help">Local inference</span>}
                </div>
                <span className={configured ? "credential-status configured" : "credential-status"}>
                  {configured ? "Configured" : "Not configured"}
                </span>
              </div>

              <label>
                Configuration label
                <input
                  value={configuration.displayName}
                  onChange={(event) => updateField(provider.id, "displayName", event.target.value)}
                  placeholder={`${provider.name} default`}
                />
              </label>
              <label>
                Base URL
                <input
                  value={configuration.baseUrl}
                  onChange={(event) => updateField(provider.id, "baseUrl", event.target.value)}
                  placeholder={provider.baseURLPlaceholder || provider.baseURL}
                  required
                />
              </label>
              <label>
                {provider.modelLabel}
                <input
                  value={configuration.model}
                  onChange={(event) => updateField(provider.id, "model", event.target.value)}
                  placeholder={provider.local ? "llama3.2" : "Enter provider model identifier"}
                  required
                />
              </label>
              {provider.supportsAPIVersion && (
                <label>
                  API version
                  <input
                    value={configuration.apiVersion}
                    onChange={(event) => updateField(provider.id, "apiVersion", event.target.value)}
                    placeholder="v1 or provider API version"
                  />
                </label>
              )}
              {!provider.local && (
                <label>
                  API key
                  <input
                    type="password"
                    value={configuration.apiKey}
                    onChange={(event) => updateField(provider.id, "apiKey", event.target.value)}
                    placeholder={
                      configuration.hasApiKey
                        ? "Enter a new key to replace the saved key"
                        : "Enter API key"
                    }
                    autoComplete="new-password"
                  />
                </label>
              )}
              <label className="checkbox-row">
                <input
                  type="checkbox"
                  checked={configuration.enabled}
                  onChange={(event) => updateField(provider.id, "enabled", event.target.checked)}
                />
                Enabled
              </label>
              {configuration.updatedAt && (
                <p className="field-help">Last updated: {configuration.updatedAt}</p>
              )}

              <div className="form-actions">
                <button type="button" disabled={busy} onClick={() => save(provider)}>
                  {savingProvider === provider.id ? "Saving..." : "Save"}
                </button>
                {configured && (
                  <button
                    type="button"
                    className="icon-button danger"
                    disabled={busy}
                    onClick={() => remove(provider)}
                    aria-label={`Remove ${provider.name} AI configuration`}
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
  );
}

export default AIProvidersSettings;
