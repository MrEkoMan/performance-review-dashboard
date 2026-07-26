import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { AlertTriangle, CalendarClock, ChevronRight } from "lucide-react";

const severities = [
  ["high", "High"],
  ["medium", "Medium"],
  ["low", "Low"],
];

const typeLabels = {
  overdue_follow_up: "Overdue follow-up",
  blocked_goal: "Blocked goal",
  overdue_goal: "Overdue goal",
  stale_evidence: "Evidence recency",
  upcoming_one_on_one: "Upcoming 1:1",
  legacy_follow_up: "Evidence follow-up",
  stale_recognition: "Recognition recency",
};

function NeedsAttentionPanel({ items = [] }) {
  const [severity, setSeverity] = useState("");
  const visibleItems = useMemo(
    () => items.filter((item) => !severity || item.severity === severity),
    [items, severity],
  );
  const highCount = items.filter((item) => item.severity === "high").length;
  const mediumCount = items.filter((item) => item.severity === "medium").length;

  return (
    <section className="attention-section">
      <div className="attention-heading">
        <div>
          <p className="profile-eyebrow">Manager priorities</p>
          <h2>Needs Attention</h2>
          <p>Deterministic signals from commitments, goals, conversations, and evidence recency.</p>
        </div>
        <label className="attention-filter">
          Severity
          <select value={severity} onChange={(event) => setSeverity(event.target.value)}>
            <option value="">All</option>
            {severities.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>
        </label>
      </div>

      <div className="attention-summary">
        <span><strong>{highCount}</strong> high priority</span>
        <span><strong>{mediumCount}</strong> medium priority</span>
        <span><strong>{items.length}</strong> total signals</span>
      </div>

      <div className="attention-list">
        {visibleItems.length === 0 ? (
          <p className="attention-clear">No attention items match this filter.</p>
        ) : visibleItems.map((item) => (
          <Link
            className={`attention-item attention-item-${item.severity}`}
            to={`/engineers/${item.engineerId}?tab=${item.targetTab}`}
            key={`${item.itemType}-${item.engineerId}-${item.sourceId}`}
          >
            <div className="attention-icon">
              {item.itemType === "upcoming_one_on_one"
                ? <CalendarClock size={20} />
                : <AlertTriangle size={20} />}
            </div>
            <div className="attention-content">
              <div className="attention-meta">
                <span className={`attention-severity attention-severity-${item.severity}`}>{item.severity}</span>
                <span>{typeLabels[item.itemType] || item.itemType}</span>
                {item.dueDate && <time>{item.dueDate}</time>}
              </div>
              <h3>{item.engineerName}: {item.title}</h3>
              <p>{item.reason}</p>
            </div>
            <ChevronRight className="attention-chevron" size={20} />
          </Link>
        ))}
      </div>
    </section>
  );
}

export default NeedsAttentionPanel;
