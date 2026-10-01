import { useEffect, useState } from "react";
import { CheckCircle2, Download, RefreshCw } from "lucide-react";

import {
  acceptJiraEvidence,
  getJiraEvidence,
  getTimeline,
} from "../api/performanceApi.js";

// Default harvest window: the last 90 days, so the button "just works".
function defaultRange() {
  const to = new Date();
  const from = new Date();
  from.setDate(from.getDate() - 90);
  return {
    from: from.toISOString().slice(0, 10),
    to: to.toISOString().slice(0, 10),
  };
}

// Maps a harvested issue to the accept-payload shape. Suggested values from
// the backend are used unless the user overrides them in the UI.
function toEvidenceItem(item) {
  return {
    issueKey: item.issueKey,
    summary: item.suggestedNote?.summary || `[${item.issueKey}] ${item.summary}`,
    resolvedDate: (item.resolvedDate || "").slice(0, 10),
    category: item.suggestedNote?.category || "Technical Excellence",
    details: item.suggestedNote?.details || "",
    impact: item.suggestedNote?.impact || "",
    browseUrl: item.browseUrl,
  };
}

function JiraEvidencePanel({ engineerId, engineer, onNotesChanged }) {
  const [range, setRange] = useState(defaultRange);
  const [evidence, setEvidence] = useState([]);
  const [selected, setSelected] = useState(new Set());
  const [categories, setCategories] = useState({});
  const [loading, setLoading] = useState(false);
  const [accepting, setAccepting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [hasMore, setHasMore] = useState(false);
  const [nextPage, setNextPage] = useState(0);
  const [page, setPage] = useState(1);

  async function loadEvidence(nextPageNumber) {
    try {
      setError("");
      setSuccess("");
      setLoading(true);
      const result = await getJiraEvidence(engineerId, {
        from: range.from,
        to: range.to,
        page: nextPageNumber || 1,
      });
      const items = Array.isArray(result?.evidence) ? result.evidence : [];
      if (nextPageNumber && nextPageNumber > 1) {
        setEvidence((current) => [...current, ...items]);
      } else {
        setEvidence(items);
      }
      setHasMore(Boolean(result?.hasMore));
      setNextPage(result?.nextPage || 0);
      setPage(nextPageNumber || 1);
      // Preselect nothing: the user chooses what counts as evidence.
      setSelected(new Set());
      // Initialize category selections from the suggested note.
      const initialCategories = {};
      for (const item of items) {
        initialCategories[item.issueKey] =
          item.suggestedNote?.category || "Technical Excellence";
      }
      setCategories((current) => ({ ...initialCategories, ...current }));
    } catch (err) {
      setEvidence([]);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  // Nothing loads on mount: the fetch is on-demand only, so opening the panel
  // never fires a Jira request the user did not ask for.
  useEffect(() => {
    setEvidence([]);
    setSelected(new Set());
    setRange(defaultRange());
    setError("");
    setSuccess("");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [engineerId]);

  function toggle(issueKey) {
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(issueKey)) {
        next.delete(issueKey);
      } else {
        next.add(issueKey);
      }
      return next;
    });
  }

  async function handleAccept() {
    if (selected.size === 0) {
      setError("Select at least one item to accept.");
      return;
    }
    try {
      setAccepting(true);
      setError("");
      const items = evidence
        .filter((item) => selected.has(item.issueKey))
        .map((item) => ({
          ...toEvidenceItem(item),
          category: categories[item.issueKey] || item.suggestedNote?.category,
        }));
      await acceptJiraEvidence(engineerId, items);
      setSuccess(
        `Accepted ${items.length} evidence item${items.length === 1 ? "" : "s"} into notes.`,
      );
      setEvidence((current) =>
        current.filter((item) => !selected.has(item.issueKey)),
      );
      setSelected(new Set());
      if (onNotesChanged) {
        await onNotesChanged();
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setAccepting(false);
    }
  }

  const selectedCount = selected.size;

  return (
    <section className="jira-evidence">
      <div className="jira-evidence-header">
        <div className="jira-evidence-title">
          <h3>Jira Evidence</h3>
          <p className="field-help">
            Harvest {engineer?.name ? `${engineer.name}'s` : "this engineer's"}{" "}
            resolved Jira issues as suggested evidence notes. Select the cards
            you want, then accept them as notes.
          </p>
        </div>
        <div className="jira-evidence-controls">
          <label className="jira-evidence-date">
            <span>From</span>
            <input
              type="date"
              value={range.from}
              onChange={(event) =>
                setRange((current) => ({ ...current, from: event.target.value }))
              }
              disabled={loading || accepting}
            />
          </label>
          <label className="jira-evidence-date">
            <span>To</span>
            <input
              type="date"
              value={range.to}
              onChange={(event) =>
                setRange((current) => ({ ...current, to: event.target.value }))
              }
              disabled={loading || accepting}
            />
          </label>
          <button
            type="button"
            onClick={() => loadEvidence(1)}
            disabled={loading || accepting}
          >
            {loading ? <RefreshCw size={14} className="spin" /> : <Download size={14} />}
            {loading ? "Fetching..." : "Fetch"}
          </button>
        </div>
      </div>

      {error && <div className="error">Error: {error}</div>}
      {success && <div className="success-message">{success}</div>}

      {evidence.length > 0 && (
        <>
          <div className="jira-evidence-actions">
            <span>
              {selectedCount} of {evidence.length} selected
            </span>
            <button
              type="button"
              onClick={handleAccept}
              disabled={accepting || selectedCount === 0}
            >
              {accepting ? "Accepting..." : `Accept ${selectedCount || ""} as notes`}
            </button>
          </div>
          <div className="jira-evidence-grid">
            {evidence.map((item) => {
              const isSelected = selected.has(item.issueKey);
              // Card-level click toggles selection; interactive children
              // opt out so their own behavior wins.
              return (
                <article
                  className={
                    "jira-evidence-card" +
                    (isSelected ? " selected" : "")
                  }
                  key={item.issueKey}
                  onClick={(event) => {
                    if (event.target.closest("a, input, select")) return;
                    toggle(item.issueKey);
                  }}
                  role="checkbox"
                  aria-checked={isSelected}
                  tabIndex={0}
                  onKeyDown={(event) => {
                    if (event.key === " " || event.key === "Enter") {
                      event.preventDefault();
                      toggle(item.issueKey);
                    }
                  }}
                >
                  <div className="jira-evidence-card-head">
                    <input
                      type="checkbox"
                      checked={isSelected}
                      onChange={() => toggle(item.issueKey)}
                      onClick={(event) => event.stopPropagation()}
                      disabled={accepting}
                      aria-label={`Use ${item.issueKey} as evidence`}
                    />
                    <span className="jira-evidence-key">{item.issueKey}</span>
                    {isSelected && (
                      <CheckCircle2
                        size={15}
                        className="jira-evidence-check"
                        aria-hidden="true"
                      />
                    )}
                    <span className="jira-evidence-meta">
                      {(item.resolvedDate || "").slice(0, 10)}
                    </span>
                  </div>
                  <a
                    href={item.browseUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="jira-evidence-summary"
                    title={item.summary}
                    onClick={(event) => event.stopPropagation()}
                  >
                    {item.summary}
                  </a>
                  <label className="jira-evidence-category">
                    <select
                      value={categories[item.issueKey] || "Technical Excellence"}
                      onChange={(event) =>
                        setCategories((current) => ({
                          ...current,
                          [item.issueKey]: event.target.value,
                        }))
                      }
                      onClick={(event) => event.stopPropagation()}
                      disabled={accepting || !isSelected}
                      size={1}
                    >
                      <option value="Business Impact">Business impact</option>
                      <option value="Technical Excellence">Technical excellence</option>
                      <option value="Operational Excellence">Operational excellence</option>
                      <option value="Team Contribution">Team contribution</option>
                      <option value="Growth Area">Growth area</option>
                      <option value="Career Development">Career development</option>
                      <option value="Feedback Received">Feedback received</option>
                    </select>
                  </label>
                </article>
              );
            })}
          </div>
          {hasMore && (
            <button
              type="button"
              className="secondary-button"
              onClick={() => loadEvidence(nextPage)}
              disabled={loading || accepting}
            >
              {loading ? "Loading..." : `Load more (page ${nextPage})`}
            </button>
          )}
        </>
      )}

      {evidence.length === 0 && !loading && !error && (
        <p className="field-help">
          No evidence loaded yet. Pick a date range and fetch from Jira.
        </p>
      )}
    </section>
  );
}

export default JiraEvidencePanel;
