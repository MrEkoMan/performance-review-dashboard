import { useState } from "react";
import { Link } from "react-router-dom";
import { CalendarCheck, ChevronRight } from "lucide-react";

const readinessLabels = {
  ready: "Ready",
  needs_attention: "Needs attention",
  unconfigured: "Period not configured",
};

function ReviewReadinessPanel({ items = [] }) {
  const [readiness, setReadiness] = useState("");
  const [engineerId, setEngineerId] = useState("");
  const [team, setTeam] = useState("");
  const [reviewCycle, setReviewCycle] = useState("");
  const [endingWindow, setEndingWindow] = useState(0);

  const teams = [...new Set(items.map((item) => item.team).filter(Boolean))].sort();
  const cycles = [...new Set(items.map((item) => item.reviewCycle).filter(Boolean))].sort();
  const visibleItems = items.filter((item) =>
    (!readiness || item.readiness === readiness)
    && (!engineerId || String(item.engineerId) === engineerId)
    && (!team || item.team === team)
    && (!reviewCycle || item.reviewCycle === reviewCycle)
    && (!endingWindow || (
      item.periodEnd
      && item.daysUntilEnd >= 0
      && item.daysUntilEnd <= endingWindow
    )));
  const countReadiness = (value) =>
    items.filter((item) => item.readiness === value).length;

  return (
    <section className="review-readiness-section">
      <div className="review-readiness-heading">
        <div>
          <p className="profile-eyebrow">Review preparation</p>
          <h2>Review-Cycle Readiness</h2>
          <p>Check whether each engineer has the records needed for an evidence-backed review.</p>
        </div>
        <span className="review-readiness-total">{visibleItems.length} engineers</span>
      </div>

      <div className="review-readiness-summary">
        {["", "unconfigured", "needs_attention", "ready"].map((value) => (
          <button
            type="button"
            className={readiness === value ? "selected" : ""}
            onClick={() => setReadiness(value)}
            key={value || "all"}
          >
            <strong>{value ? countReadiness(value) : items.length}</strong>
            <span>{value ? readinessLabels[value] : "All engineers"}</span>
          </button>
        ))}
      </div>

      <div className="review-readiness-filters">
        <label>
          Engineer
          <select value={engineerId} onChange={(event) => setEngineerId(event.target.value)}>
            <option value="">All engineers</option>
            {items.map((item) => (
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
            {cycles.map((value) => <option value={value} key={value}>{value}</option>)}
          </select>
        </label>
        <label>
          Cycle ending
          <select value={endingWindow} onChange={(event) => setEndingWindow(Number(event.target.value))}>
            <option value={0}>Any time</option>
            <option value={30}>Within 30 days</option>
            <option value={60}>Within 60 days</option>
            <option value={90}>Within 90 days</option>
          </select>
        </label>
      </div>

      <div className="review-readiness-list">
        {visibleItems.length === 0 ? (
          <p className="review-readiness-empty">No engineers match these filters.</p>
        ) : visibleItems.map((item) => (
          <Link
            className={`review-readiness-item review-readiness-${item.readiness}`}
            to={`/engineers/${item.engineerId}`}
            key={item.engineerId}
          >
            <div className="review-readiness-icon"><CalendarCheck size={19} /></div>
            <div className="review-readiness-content">
              <div className="review-readiness-meta">
                <span className={`review-readiness-label review-readiness-label-${item.readiness}`}>
                  {readinessLabels[item.readiness]}
                </span>
                <span>{item.team}</span>
                <span>{item.reviewCycle || "No cycle assigned"}</span>
                {item.periodPhase !== "unconfigured" && <span>{item.periodPhase}</span>}
              </div>
              <h3>{item.engineerName}</h3>
              {item.periodEnd && (
                <p>
                  Period {item.periodStart} to {item.periodEnd}
                  {item.daysUntilEnd >= 0
                    ? ` · ${item.daysUntilEnd} days remaining`
                    : ` · ended ${Math.abs(item.daysUntilEnd)} days ago`}
                </p>
              )}
              <div className="review-readiness-counts">
                <span><strong>{item.evidenceCount}</strong> evidence</span>
                <span><strong>{item.evidenceCategoryCount}</strong> categories</span>
                <span><strong>{item.goalCount}</strong> goals</span>
                <span><strong>{item.recognitionCount}</strong> recognition</span>
                <span><strong>{item.completedOneOnOnes}</strong> completed 1:1s</span>
                <span><strong>{item.overdueFollowUps}</strong> overdue actions</span>
              </div>
              {item.missingItems.length > 0 && (
                <ul className="review-readiness-missing">
                  {item.missingItems.map((missing) => <li key={missing}>{missing}</li>)}
                </ul>
              )}
            </div>
            <ChevronRight className="review-readiness-chevron" size={20} />
          </Link>
        ))}
      </div>
      <p className="review-readiness-note">
        Readiness is a preparation checklist, not a performance score or rating.
      </p>
    </section>
  );
}

export default ReviewReadinessPanel;
