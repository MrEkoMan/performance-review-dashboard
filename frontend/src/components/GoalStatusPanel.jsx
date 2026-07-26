import { useState } from "react";
import { Link } from "react-router-dom";
import { ChevronRight, Target } from "lucide-react";

const healthLabels = {
  blocked: "Blocked",
  overdue: "Overdue",
  at_risk: "At risk",
  on_track: "On track",
};

function GoalStatusPanel({ goals = [] }) {
  const [health, setHealth] = useState("");
  const [engineerId, setEngineerId] = useState("");
  const [priority, setPriority] = useState("");
  const [reviewCycle, setReviewCycle] = useState("");

  const engineers = [...new Map(
    goals.map((goal) => [String(goal.engineerId), goal.engineerName]),
  )];
  const reviewCycles = [...new Set(
    goals.map((goal) => goal.reviewCycle).filter(Boolean),
  )].sort();
  const visibleGoals = goals.filter((goal) =>
    (!health || goal.health === health)
    && (!engineerId || String(goal.engineerId) === engineerId)
    && (!priority || goal.priority === priority)
    && (!reviewCycle || goal.reviewCycle === reviewCycle));
  const countHealth = (value) => goals.filter((goal) => goal.health === value).length;

  return (
    <section className="goal-status-section">
      <div className="goal-status-heading">
        <div>
          <p className="profile-eyebrow">Development portfolio</p>
          <h2>Goal Status</h2>
          <p>Track blocked, overdue, and progress-at-risk goals across the team.</p>
        </div>
        <span className="goal-status-total">{visibleGoals.length} visible</span>
      </div>

      <div className="goal-health-summary">
        {["", "blocked", "overdue", "at_risk", "on_track"].map((value) => (
          <button
            type="button"
            className={health === value ? "selected" : ""}
            onClick={() => setHealth(value)}
            key={value || "all"}
          >
            <strong>{value ? countHealth(value) : goals.length}</strong>
            <span>{value ? healthLabels[value] : "All active"}</span>
          </button>
        ))}
      </div>

      <div className="goal-status-filters">
        <label>
          Engineer
          <select value={engineerId} onChange={(event) => setEngineerId(event.target.value)}>
            <option value="">All engineers</option>
            {engineers.map(([id, name]) => <option value={id} key={id}>{name}</option>)}
          </select>
        </label>
        <label>
          Priority
          <select value={priority} onChange={(event) => setPriority(event.target.value)}>
            <option value="">All priorities</option>
            <option value="high">High</option>
            <option value="medium">Medium</option>
            <option value="low">Low</option>
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

      <div className="goal-status-list">
        {visibleGoals.length === 0 ? (
          <p className="goal-status-empty">No active goals match these filters.</p>
        ) : visibleGoals.map((goal) => (
          <Link
            className={`goal-status-item goal-health-${goal.health}`}
            to={`/engineers/${goal.engineerId}?tab=goals`}
            key={goal.id}
          >
            <div className="goal-status-icon"><Target size={19} /></div>
            <div className="goal-status-content">
              <div className="goal-status-meta">
                <span className={`goal-health-label goal-health-label-${goal.health}`}>
                  {healthLabels[goal.health]}
                </span>
                <span>{goal.engineerName}</span>
                <span>{goal.priority} priority</span>
                {goal.reviewCycle && <span>{goal.reviewCycle}</span>}
              </div>
              <h3>{goal.title}</h3>
              <div className="goal-status-progress" aria-label={`${goal.progressPercent}% complete`}>
                <span style={{ width: `${goal.progressPercent}%` }} />
              </div>
              <p>
                {goal.progressPercent}% complete
                {goal.expectedProgress > 0 && ` · ${goal.expectedProgress}% expected`}
                {goal.targetDate && ` · Target ${goal.targetDate}`}
              </p>
            </div>
            <ChevronRight className="goal-status-chevron" size={20} />
          </Link>
        ))}
      </div>
      <p className="goal-risk-note">
        At risk means progress trails elapsed schedule by at least 20 percentage points.
      </p>
    </section>
  );
}

export default GoalStatusPanel;
