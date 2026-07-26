import { useMemo, useState } from "react";
import {
  Award,
  CalendarDays,
  CheckSquare,
  FileText,
  Target,
} from "lucide-react";

const eventTypes = [
  ["evidence", "Evidence"],
  ["goal", "Goal"],
  ["one_on_one", "1:1"],
  ["follow_up", "Follow-up"],
  ["recognition", "Recognition"],
];

const tabByType = {
  evidence: "evidence",
  goal: "goals",
  one_on_one: "one-on-ones",
  follow_up: "follow-ups",
  recognition: "recognition",
};

const iconByType = {
  evidence: FileText,
  goal: Target,
  one_on_one: CalendarDays,
  follow_up: CheckSquare,
  recognition: Award,
};

function labelFor(value) {
  return eventTypes.find(([key]) => key === value)?.[1] || value;
}

function TimelinePanel({ events = [], reviewCycle = "", onNavigate }) {
  const [eventType, setEventType] = useState("");
  const [cycle, setCycle] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  const visibleEvents = useMemo(
    () => events.filter((event) =>
      (!eventType || event.eventType === eventType)
      && (!cycle || event.reviewCycle === cycle)
      && (!from || event.eventDate >= from)
      && (!to || event.eventDate <= to)),
    [events, eventType, cycle, from, to],
  );
  const cycles = [...new Set(events.map((event) => event.reviewCycle).filter(Boolean))];
  if (reviewCycle && !cycles.includes(reviewCycle)) {
    cycles.unshift(reviewCycle);
  }

  return (
    <section className="timeline-section">
      <div className="timeline-heading">
        <div>
          <p className="profile-eyebrow">Chronological context</p>
          <h2>Engineer Timeline</h2>
          <p>Review evidence, conversations, commitments, goals, and recognition together.</p>
        </div>
        <span className="timeline-count">{visibleEvents.length} events</span>
      </div>

      <div className="timeline-filters">
        <label>
          Event type
          <select value={eventType} onChange={(event) => setEventType(event.target.value)}>
            <option value="">All events</option>
            {eventTypes.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>
        </label>
        <label>
          Review cycle
          <select value={cycle} onChange={(event) => setCycle(event.target.value)}>
            <option value="">All cycles</option>
            {cycles.map((value) => <option key={value} value={value}>{value}</option>)}
          </select>
        </label>
        <label>
          From
          <input type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
        </label>
        <label>
          To
          <input type="date" value={to} onChange={(event) => setTo(event.target.value)} />
        </label>
      </div>

      <div className="timeline-list">
        {visibleEvents.length === 0 ? (
          <p className="empty-state">
            {events.length ? "No timeline events match these filters." : "No timeline events recorded yet."}
          </p>
        ) : visibleEvents.map((event) => {
          const Icon = iconByType[event.eventType] || FileText;
          return (
            <article className={`timeline-event timeline-event-${event.eventType}`} key={`${event.eventType}-${event.sourceId}`}>
              <div className="timeline-marker"><Icon size={18} /></div>
              <div className="timeline-event-content">
                <div className="timeline-event-heading">
                  <div>
                    <span className="timeline-type">{labelFor(event.eventType)}</span>
                    <time>{event.eventDate}</time>
                    <h3>{event.title}</h3>
                  </div>
                  <button type="button" className="timeline-source-link" onClick={() => onNavigate(tabByType[event.eventType])}>
                    View source
                  </button>
                </div>
                {event.summary && <p>{event.summary}</p>}
                <div className="timeline-event-meta">
                  {event.status && <span>{event.status.replaceAll("_", " ")}</span>}
                  {event.reviewCycle && <span>{event.reviewCycle}</span>}
                </div>
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}

export default TimelinePanel;
