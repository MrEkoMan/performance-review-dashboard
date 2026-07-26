import { useState } from "react";
import { Link } from "react-router-dom";
import { ChevronRight, FileClock } from "lucide-react";

const recencyLabels = {
  never: "No evidence",
  critical: "91+ days",
  stale: "61–90 days",
  aging: "31–60 days",
  recent: "Last 30 days",
};

function EvidenceRecencyPanel({ engineers = [] }) {
  const [recency, setRecency] = useState("");
  const [engineerId, setEngineerId] = useState("");
  const [team, setTeam] = useState("");
  const [reviewCycle, setReviewCycle] = useState("");

  const teams = [...new Set(engineers.map((item) => item.team).filter(Boolean))].sort();
  const reviewCycles = [...new Set(
    engineers.map((item) => item.reviewCycle).filter(Boolean),
  )].sort();
  const visibleItems = engineers.filter((item) =>
    (!recency || item.recency === recency)
    && (!engineerId || String(item.engineerId) === engineerId)
    && (!team || item.team === team)
    && (!reviewCycle || item.reviewCycle === reviewCycle));
  const countRecency = (value) =>
    engineers.filter((item) => item.recency === value).length;

  return (
    <section className="evidence-recency-section">
      <div className="evidence-recency-heading">
        <div>
          <p className="profile-eyebrow">Evidence coverage</p>
          <h2>Evidence Recency</h2>
          <p>See where current, balanced performance evidence may need attention.</p>
        </div>
        <span className="evidence-recency-total">{visibleItems.length} engineers</span>
      </div>

      <div className="evidence-recency-summary">
        {["", "never", "critical", "stale", "aging", "recent"].map((value) => (
          <button
            type="button"
            className={recency === value ? "selected" : ""}
            onClick={() => setRecency(value)}
            key={value || "all"}
          >
            <strong>{value ? countRecency(value) : engineers.length}</strong>
            <span>{value ? recencyLabels[value] : "All engineers"}</span>
          </button>
        ))}
      </div>

      <div className="evidence-recency-filters">
        <label>
          Engineer
          <select value={engineerId} onChange={(event) => setEngineerId(event.target.value)}>
            <option value="">All engineers</option>
            {engineers.map((item) => (
              <option value={item.engineerId} key={item.engineerId}>{item.engineerName}</option>
            ))}
          </select>
        </label>
        <label>
          Team
          <select value={team} onChange={(event) => setTeam(event.target.value)}>
            <option value="">All teams</option>
            {teams.map((value) => <option value={value} key={value}>{value}</option>)}
          </select>
        </label>
        <label>
          Review cycle
          <select value={reviewCycle} onChange={(event) => setReviewCycle(event.target.value)}>
            <option value="">All cycles</option>
            {reviewCycles.map((value) => <option value={value} key={value}>{value}</option>)}
          </select>
        </label>
      </div>

      <div className="evidence-recency-list">
        {visibleItems.length === 0 ? (
          <p className="evidence-recency-empty">No engineers match these filters.</p>
        ) : visibleItems.map((item) => (
          <Link
            className={`evidence-recency-item evidence-recency-${item.recency}`}
            to={`/engineers/${item.engineerId}?tab=evidence`}
            key={item.engineerId}
          >
            <div className="evidence-recency-icon"><FileClock size={19} /></div>
            <div className="evidence-recency-content">
              <div className="evidence-recency-meta">
                <span className={`evidence-recency-label evidence-recency-label-${item.recency}`}>
                  {recencyLabels[item.recency]}
                </span>
                <span>{item.team}</span>
                <span>{item.reviewCycle}</span>
              </div>
              <h3>{item.engineerName}</h3>
              <p>
                {item.lastEvidenceDate
                  ? `Last evidence ${item.lastEvidenceDate} · ${item.daysSinceEvidence} days ago`
                  : "No performance evidence has been recorded"}
              </p>
              <div className="evidence-recency-counts">
                <span><strong>{item.evidenceLast30Days}</strong> last 30 days</span>
                <span><strong>{item.currentCycleEvidence}</strong> current cycle</span>
                <span><strong>{item.totalEvidence}</strong> total</span>
              </div>
            </div>
            <ChevronRight className="evidence-recency-chevron" size={20} />
          </Link>
        ))}
      </div>
      <p className="evidence-recency-note">
        Recency identifies documentation gaps; it does not measure engineer performance.
      </p>
    </section>
  );
}

export default EvidenceRecencyPanel;
