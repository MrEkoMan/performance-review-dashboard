import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Sparkles, Trash2 } from "lucide-react";

import {
  createAIAnalysis,
  deleteAIAnalysis,
  getAIAnalyses,
  getAIProviders,
} from "../api/performanceApi.js";

const PROVIDER_LABELS = {
  openai: "OpenAI",
  anthropic: "Anthropic",
  gemini: "Google Gemini",
  azure_openai: "Azure OpenAI",
  openrouter: "OpenRouter",
  ollama: "Ollama",
};

function AIInsightsPanel({ engineerId, engineer }) {
  const [providers, setProviders] = useState([]);
  const [providerId, setProviderId] = useState("");
  const [reviewCycle, setReviewCycle] = useState("");
  const [analyses, setAnalyses] = useState([]);
  const [selectedId, setSelectedId] = useState(null);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState("");

  async function loadPanel() {
    try {
      setError("");
      const [providerData, analysisData] = await Promise.all([
        getAIProviders(),
        getAIAnalyses(engineerId),
      ]);
      const enabled = (Array.isArray(providerData) ? providerData : []).filter(
        (item) => item.enabled,
      );
      setProviders(enabled);
      setProviderId((current) => current || (enabled[0]?.provider ?? ""));
      setAnalyses(Array.isArray(analysisData) ? analysisData : []);
      setSelectedId((current) => current ?? analysisData[0]?.id ?? null);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadPanel();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [engineerId]);

  async function handleGenerate() {
    try {
      setGenerating(true);
      setError("");
      const created = await createAIAnalysis(engineerId, {
        provider: providerId,
        reviewCycle,
      });
      setAnalyses((current) => [created, ...current]);
      setSelectedId(created.id);
    } catch (err) {
      setError(err.message);
    } finally {
      setGenerating(false);
    }
  }

  async function handleDelete(analysisId) {
    if (!window.confirm("Remove this saved analysis?")) {
      return;
    }
    try {
      setError("");
      await deleteAIAnalysis(analysisId);
      setAnalyses((current) => current.filter((item) => item.id !== analysisId));
      setSelectedId((current) =>
        current === analysisId
          ? (analyses.find((item) => item.id !== analysisId)?.id ?? null)
          : current,
      );
    } catch (err) {
      setError(err.message);
    }
  }

  if (loading) {
    return <p>Loading AI insights...</p>;
  }

  const selected = analyses.find((item) => item.id === selectedId) || null;
  const cycles = [
    ...new Set(
      analyses
        .map((item) => item.reviewCycle)
        .filter(Boolean)
        .concat(engineer?.reviewCycle ? [engineer.reviewCycle] : []),
    ),
  ];

  return (
    <section className="panel-section">
      <div className="settings-section-heading">
        <h2>
          <Sparkles size={18} /> AI Insights
        </h2>
        <p className="settings-description">
          Have a configured AI provider analyze this engineer's notes, goals,
          1:1s, follow-ups, and recognition as review preparation. Results are
          saved below so you can compare runs across time.
        </p>
      </div>

      {providers.length === 0 ? (
        <div className="panel-empty-state">
          No AI provider is enabled.{" "}
          <Link to="/settings">Configure one in Settings</Link> to generate
          insights.
        </div>
      ) : (
        <div className="ai-insights-controls">
          <label>
            Provider
            <select
              value={providerId}
              onChange={(event) => setProviderId(event.target.value)}
              disabled={generating}
            >
              {providers.map((provider) => (
                <option key={provider.provider} value={provider.provider}>
                  {PROVIDER_LABELS[provider.provider] || provider.provider}
                  {provider.model ? ` — ${provider.model}` : ""}
                </option>
              ))}
            </select>
          </label>
          <label>
            Review cycle
            <select
              value={reviewCycle}
              onChange={(event) => setReviewCycle(event.target.value)}
              disabled={generating}
            >
              <option value="">All cycles</option>
              {cycles.map((cycle) => (
                <option key={cycle} value={cycle}>
                  {cycle}
                </option>
              ))}
            </select>
          </label>
          <button
            type="button"
            onClick={handleGenerate}
            disabled={generating || !providerId}
          >
            {generating ? "Analyzing engineer data — this can take up to two minutes..." : "Generate analysis"}
          </button>
        </div>
      )}

      {error && <div className="error">Error: {error}</div>}

      {analyses.length > 0 && (
        <div className="ai-insights-history">
          {analyses.map((analysis) => (
            <button
              key={analysis.id}
              type="button"
              className={
                "ai-insights-history-item" +
                (analysis.id === selectedId ? " selected" : "")
              }
              onClick={() => setSelectedId(analysis.id)}
            >
              <span className="ai-insights-history-title">
                {PROVIDER_LABELS[analysis.provider] || analysis.provider}
                {analysis.model ? ` · ${analysis.model}` : ""}
              </span>
              <span className="ai-insights-history-meta">
                {analysis.reviewCycle || "All cycles"} ·{" "}
                {new Date(analysis.createdAt + "Z").toLocaleString()}
              </span>
              <span
                role="button"
                tabIndex={0}
                className="icon-button danger"
                aria-label="Delete analysis"
                onClick={(event) => {
                  event.stopPropagation();
                  handleDelete(analysis.id);
                }}
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    event.stopPropagation();
                    handleDelete(analysis.id);
                  }
                }}
              >
                <Trash2 size={15} />
              </span>
            </button>
          ))}
        </div>
      )}

      {selected && (
        <article className="ai-insights-output">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>
            {selected.outputMarkdown}
          </ReactMarkdown>
        </article>
      )}
    </section>
  );
}

export default AIInsightsPanel;
