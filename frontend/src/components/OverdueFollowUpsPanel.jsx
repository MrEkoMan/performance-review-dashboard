import { useState } from "react";
import { Link } from "react-router-dom";
import { CheckSquare, ChevronRight } from "lucide-react";

function OverdueFollowUpsPanel({ followUps = [] }) {
  const [engineerId, setEngineerId] = useState("");
  const [owner, setOwner] = useState("");
  const [priority, setPriority] = useState("");
  const [minimumAge, setMinimumAge] = useState(0);

  const engineers = [...new Map(
    followUps.map((item) => [String(item.engineerId), item.engineerName]),
  )];
  const owners = [...new Set(followUps.map((item) => item.owner))].sort();
  const visibleItems = followUps.filter((item) =>
    (!engineerId || String(item.engineerId) === engineerId)
    && (!owner || item.owner === owner)
    && (!priority || item.priority === priority)
    && item.daysOverdue >= minimumAge);
  const sevenPlus = followUps.filter((item) => item.daysOverdue >= 7).length;
  const thirtyPlus = followUps.filter((item) => item.daysOverdue >= 30).length;
  const sixtyPlus = followUps.filter((item) => item.daysOverdue >= 60).length;

  return (
    <section className="overdue-section">
      <div className="overdue-heading">
        <div>
          <p className="profile-eyebrow">Commitment portfolio</p>
          <h2>Overdue Follow-Ups</h2>
          <p>Review outstanding commitments across engineers, owners, and aging groups.</p>
        </div>
        <span className="overdue-total">{visibleItems.length} visible</span>
      </div>

      <div className="overdue-aging">
        <button type="button" className={minimumAge === 0 ? "selected" : ""} onClick={() => setMinimumAge(0)}>
          <strong>{followUps.length}</strong><span>All overdue</span>
        </button>
        <button type="button" className={minimumAge === 7 ? "selected" : ""} onClick={() => setMinimumAge(7)}>
          <strong>{sevenPlus}</strong><span>7+ days</span>
        </button>
        <button type="button" className={minimumAge === 30 ? "selected" : ""} onClick={() => setMinimumAge(30)}>
          <strong>{thirtyPlus}</strong><span>30+ days</span>
        </button>
        <button type="button" className={minimumAge === 60 ? "selected" : ""} onClick={() => setMinimumAge(60)}>
          <strong>{sixtyPlus}</strong><span>60+ days</span>
        </button>
      </div>

      <div className="overdue-filters">
        <label>
          Engineer
          <select value={engineerId} onChange={(event) => setEngineerId(event.target.value)}>
            <option value="">All engineers</option>
            {engineers.map(([id, name]) => <option key={id} value={id}>{name}</option>)}
          </select>
        </label>
        <label>
          Owner
          <select value={owner} onChange={(event) => setOwner(event.target.value)}>
            <option value="">All owners</option>
            {owners.map((value) => <option key={value} value={value}>{value}</option>)}
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
      </div>

      <div className="overdue-list">
        {visibleItems.length === 0 ? (
          <p className="overdue-empty">No overdue follow-ups match these filters.</p>
        ) : visibleItems.map((item) => (
          <Link
            className={`overdue-item overdue-item-${item.priority}`}
            to={`/engineers/${item.engineerId}?tab=follow-ups`}
            key={item.id}
          >
            <div className="overdue-icon"><CheckSquare size={19} /></div>
            <div className="overdue-content">
              <div className="overdue-meta">
                <span className={`overdue-priority overdue-priority-${item.priority}`}>{item.priority}</span>
                <span>{item.engineerName}</span>
                <span>{item.owner}</span>
                <span>{item.status.replaceAll("_", " ")}</span>
              </div>
              <h3>{item.description}</h3>
              <p>
                Due {item.dueDate} · {item.daysOverdue} day
                {item.daysOverdue === 1 ? "" : "s"} overdue
              </p>
              {item.notes && <p className="overdue-notes">{item.notes}</p>}
              {item.sourceType !== "manual" && (
                <span className="overdue-source">Source: {item.sourceType.replaceAll("_", " ")} #{item.sourceId}</span>
              )}
            </div>
            <ChevronRight className="overdue-chevron" size={20} />
          </Link>
        ))}
      </div>
    </section>
  );
}

export default OverdueFollowUpsPanel;
