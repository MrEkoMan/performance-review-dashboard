import { useMemo, useState } from "react";
import { CalendarDays, Pencil, Plus, Trash2, X } from "lucide-react";
import { Dialog } from "@mui/material";

const statuses = [
  ["scheduled", "Scheduled"],
  ["completed", "Completed"],
  ["cancelled", "Cancelled"],
];

function today() {
  return new Date().toISOString().slice(0, 10);
}

function emptyMeeting() {
  return {
    meetingDate: today(),
    wins: "",
    challenges: "",
    careerDiscussion: "",
    feedback: "",
    managerTopics: "",
    engineerTopics: "",
    privateManagerNotes: "",
    sharedNotes: "",
    followUpDate: "",
    status: "scheduled",
  };
}

function statusLabel(value) {
  return statuses.find(([status]) => status === value)?.[1] || value;
}

function daysSince(date) {
  if (!date) {
    return null;
  }
  const elapsed = Date.now() - new Date(`${date}T00:00:00`).getTime();
  return Math.max(0, Math.floor(elapsed / 86400000));
}

function OneOnOnesPanel({ meetings = [], onCreate, onUpdate, onDelete }) {
  const [form, setForm] = useState(emptyMeeting);
  const [editingMeeting, setEditingMeeting] = useState(null);
  const [showForm, setShowForm] = useState(false);
  const [statusFilter, setStatusFilter] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const visibleMeetings = useMemo(
    () =>
      meetings.filter(
        (meeting) => !statusFilter || meeting.status === statusFilter,
      ),
    [meetings, statusFilter],
  );
  const nextMeeting = [...meetings]
    .filter(
      (meeting) =>
        meeting.status === "scheduled" && meeting.meetingDate >= today(),
    )
    .sort((first, second) =>
      first.meetingDate.localeCompare(second.meetingDate),
    )[0];
  const lastCompleted = meetings.find(
    (meeting) => meeting.status === "completed",
  );
  const sinceLastMeeting = daysSince(lastCompleted?.meetingDate);

  function updateField(event) {
    const { name, value } = event.target;
    setForm((current) => ({ ...current, [name]: value }));
  }

  function startCreate() {
    setEditingMeeting(null);
    setForm(emptyMeeting());
    setError("");
    setShowForm(true);
  }

  function startEdit(meeting) {
    setEditingMeeting(meeting);
    setForm({ ...emptyMeeting(), ...meeting });
    setError("");
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setEditingMeeting(null);
    setForm(emptyMeeting());
    setError("");
  }

  async function submit(event) {
    event.preventDefault();
    try {
      setSaving(true);
      setError("");
      if (editingMeeting) {
        await onUpdate(editingMeeting.id, form);
      } else {
        await onCreate(form);
      }
      closeForm();
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  }

  async function remove(meeting) {
    if (!window.confirm(`Delete the 1:1 from ${meeting.meetingDate}?`)) {
      return;
    }
    try {
      setError("");
      await onDelete(meeting.id);
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <section className="one-on-ones-section">
      <div className="one-on-ones-heading">
        <div>
          <p className="profile-eyebrow">Conversation continuity</p>
          <h2>1:1 History</h2>
          <p>Prepare conversations and preserve coaching context over time.</p>
        </div>
        <button type="button" onClick={startCreate}>
          <Plus size={16} /> Add 1:1
        </button>
      </div>

      <div className="one-on-one-metrics">
        <div>
          <CalendarDays size={19} />
          <span>Next 1:1</span>
          <strong>{nextMeeting?.meetingDate || "Not scheduled"}</strong>
        </div>
        <div>
          <span>Last completed</span>
          <strong>{lastCompleted?.meetingDate || "No history"}</strong>
        </div>
        <div>
          <span>Days since last 1:1</span>
          <strong>{sinceLastMeeting ?? "—"}</strong>
        </div>
      </div>

      <label className="one-on-one-filter">
        Status
        <select
          value={statusFilter}
          onChange={(event) => setStatusFilter(event.target.value)}
        >
          <option value="">All meetings</option>
          {statuses.map(([value, label]) => (
            <option key={value} value={value}>{label}</option>
          ))}
        </select>
      </label>

      {error && <div className="error">Error: {error}</div>}

      <Dialog
        open={showForm}
        onClose={saving ? undefined : closeForm}
        fullWidth
        maxWidth="md"
        aria-label={editingMeeting ? "Edit 1:1" : "New 1:1"}
      >
        <form className="one-on-one-form" onSubmit={submit}>
          <div className="one-on-one-form-heading">
            <h3>{editingMeeting ? "Edit 1:1" : "New 1:1"}</h3>
            <button
              type="button"
              className="icon-button"
              onClick={closeForm}
              aria-label="Close 1:1 form"
            >
              <X size={17} />
            </button>
          </div>
          <label>
            Meeting date
            <input
              type="date"
              name="meetingDate"
              value={form.meetingDate}
              onChange={updateField}
              required
            />
          </label>
          <label>
            Status
            <select name="status" value={form.status} onChange={updateField}>
              {statuses.map(([value, label]) => (
                <option key={value} value={value}>{label}</option>
              ))}
            </select>
          </label>
          <label>
            Follow-up date
            <input
              type="date"
              name="followUpDate"
              value={form.followUpDate}
              onChange={updateField}
            />
          </label>
          <label className="one-on-one-field-wide">
            Wins
            <textarea name="wins" value={form.wins} onChange={updateField} rows="3" />
          </label>
          <label className="one-on-one-field-wide">
            Challenges
            <textarea name="challenges" value={form.challenges} onChange={updateField} rows="3" />
          </label>
          <label className="one-on-one-field-wide">
            Career discussion
            <textarea name="careerDiscussion" value={form.careerDiscussion} onChange={updateField} rows="3" />
          </label>
          <label className="one-on-one-field-wide">
            Feedback
            <textarea name="feedback" value={form.feedback} onChange={updateField} rows="3" />
          </label>
          <label className="one-on-one-field-wide">
            Manager topics
            <textarea name="managerTopics" value={form.managerTopics} onChange={updateField} rows="3" />
          </label>
          <label className="one-on-one-field-wide">
            Engineer topics
            <textarea name="engineerTopics" value={form.engineerTopics} onChange={updateField} rows="3" />
          </label>
          <label className="one-on-one-field-wide shared-notes-field">
            Shared notes
            <textarea name="sharedNotes" value={form.sharedNotes} onChange={updateField} rows="4" />
            <small>Content intended to be shared with the engineer.</small>
          </label>
          <label className="one-on-one-field-wide private-notes-field">
            Private manager notes
            <textarea
              name="privateManagerNotes"
              value={form.privateManagerNotes}
              onChange={updateField}
              rows="4"
            />
            <small>Manager-only context. This remains in the local application.</small>
          </label>
          <div className="form-actions one-on-one-field-wide">
            <button type="submit" disabled={saving}>
              {saving ? "Saving..." : editingMeeting ? "Save changes" : "Create 1:1"}
            </button>
            <button type="button" className="secondary-button" onClick={closeForm}>
              Cancel
            </button>
          </div>
        </form>
      </Dialog>

      <div className="one-on-one-history">
        {visibleMeetings.length === 0 ? (
          <p className="empty-state">
            {meetings.length
              ? "No 1:1 records match this filter."
              : "No 1:1 history recorded yet."}
          </p>
        ) : visibleMeetings.map((meeting) => (
          <article className="one-on-one-card" key={meeting.id}>
            <div className="one-on-one-card-heading">
              <div>
                <span className={`meeting-status meeting-status-${meeting.status}`}>
                  {statusLabel(meeting.status)}
                </span>
                <h3>{meeting.meetingDate}</h3>
                {meeting.followUpDate && (
                  <p>Follow up by {meeting.followUpDate}</p>
                )}
              </div>
              <div className="table-actions">
                <button type="button" className="icon-button" onClick={() => startEdit(meeting)} aria-label={`Edit 1:1 from ${meeting.meetingDate}`}>
                  <Pencil size={15} />
                </button>
                <button type="button" className="icon-button danger" onClick={() => remove(meeting)} aria-label={`Delete 1:1 from ${meeting.meetingDate}`}>
                  <Trash2 size={15} />
                </button>
              </div>
            </div>
            <div className="one-on-one-content">
              {meeting.wins && <div><h4>Wins</h4><p>{meeting.wins}</p></div>}
              {meeting.challenges && <div><h4>Challenges</h4><p>{meeting.challenges}</p></div>}
              {meeting.careerDiscussion && <div><h4>Career discussion</h4><p>{meeting.careerDiscussion}</p></div>}
              {meeting.feedback && <div><h4>Feedback</h4><p>{meeting.feedback}</p></div>}
              {meeting.managerTopics && <div><h4>Manager topics</h4><p>{meeting.managerTopics}</p></div>}
              {meeting.engineerTopics && <div><h4>Engineer topics</h4><p>{meeting.engineerTopics}</p></div>}
              {meeting.sharedNotes && <div className="shared-note"><h4>Shared notes</h4><p>{meeting.sharedNotes}</p></div>}
              {meeting.privateManagerNotes && <div className="private-note"><h4>Private manager notes</h4><p>{meeting.privateManagerNotes}</p></div>}
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}

export default OneOnOnesPanel;
