import { Link } from "react-router-dom";
import { CalendarDays, ChevronRight } from "lucide-react";

function timingLabel(daysUntil) {
  if (daysUntil < 0) {
    const days = Math.abs(daysUntil);
    return `${days} day${days === 1 ? "" : "s"} overdue`;
  }
  if (daysUntil === 0) {
    return "Today";
  }
  if (daysUntil === 1) {
    return "Tomorrow";
  }
  return `In ${daysUntil} days`;
}

function UpcomingOneOnOnesPanel({
  meetings = [],
  windowDays = 14,
  onWindowChange,
}) {
  return (
    <section className="upcoming-section">
      <div className="upcoming-heading">
        <div>
          <p className="profile-eyebrow">Conversation preparation</p>
          <h2>Upcoming 1:1s</h2>
          <p>Review commitments and recent context before each conversation.</p>
        </div>
        <label className="upcoming-window">
          Window
          <select value={windowDays} onChange={(event) => onWindowChange(Number(event.target.value))}>
            <option value="7">7 days</option>
            <option value="14">14 days</option>
            <option value="30">30 days</option>
          </select>
        </label>
      </div>

      <div className="upcoming-grid">
        {meetings.length === 0 ? (
          <p className="upcoming-empty">No scheduled 1:1s in this window.</p>
        ) : meetings.map((meeting) => (
          <article className={`upcoming-card ${meeting.daysUntil < 0 ? "upcoming-card-overdue" : ""}`} key={meeting.meetingId}>
            <div className="upcoming-card-heading">
              <div className="upcoming-date-icon"><CalendarDays size={20} /></div>
              <div>
                <span className={`upcoming-timing ${meeting.daysUntil < 0 ? "upcoming-timing-overdue" : ""}`}>
                  {timingLabel(meeting.daysUntil)}
                </span>
                <h3>{meeting.engineerName}</h3>
                <time>{meeting.meetingDate}</time>
              </div>
            </div>

            <dl className="upcoming-context">
              <div><dt>Last completed</dt><dd>{meeting.lastCompletedDate || "No history"}</dd></div>
              <div><dt>Open follow-ups</dt><dd>{meeting.openFollowUps}</dd></div>
              <div><dt>Blocked goals</dt><dd>{meeting.blockedGoals}</dd></div>
              <div><dt>Overdue goals</dt><dd>{meeting.overdueGoals}</dd></div>
              <div><dt>Recent evidence</dt><dd>{meeting.recentEvidenceCount}</dd></div>
              <div><dt>Recent recognition</dt><dd>{meeting.recentRecognitionCount}</dd></div>
            </dl>

            <Link className="upcoming-prepare" to={`/engineers/${meeting.engineerId}?tab=one-on-ones`}>
              Prepare for 1:1 <ChevronRight size={18} />
            </Link>
          </article>
        ))}
      </div>
    </section>
  );
}

export default UpcomingOneOnOnesPanel;
