import { useMemo, useState } from "react";
import { Dialog } from "@mui/material";
import { CheckCircle2, Pencil, Plus, Trash2, X } from "lucide-react";

const statuses = [
  ["open", "Open"],
  ["in_progress", "In progress"],
  ["completed", "Completed"],
  ["cancelled", "Cancelled"],
];

const sourceTypes = [
  ["manual", "Manual"],
  ["note", "Performance evidence"],
  ["goal", "Goal"],
  ["one_on_one", "1:1"],
];

function today() {
  return new Date().toISOString().slice(0, 10);
}

function emptyFollowUp() {
  return {
    sourceType: "manual",
    sourceId: null,
    description: "",
    owner: "Manager",
    dueDate: "",
    status: "open",
    priority: "medium",
    completionDate: "",
    notes: "",
  };
}

function labelFor(options, value) {
  return options.find(([key]) => key === value)?.[1] || value;
}

function isOverdue(item) {
  return item.dueDate && !["completed", "cancelled"].includes(item.status)
    && item.dueDate < today();
}

function sourceOptions(type, notes, goals, meetings) {
  if (type === "note") {
    return notes.map((note) => [note.id, `${note.noteDate}: ${note.summary}`]);
  }
  if (type === "goal") {
    return goals.map((goal) => [goal.id, goal.title]);
  }
  if (type === "one_on_one") {
    return meetings.map((meeting) => [meeting.id, `1:1 on ${meeting.meetingDate}`]);
  }
  return [];
}

function FollowUpsPanel({
  followUps = [],
  notes = [],
  goals = [],
  meetings = [],
  onCreate,
  onUpdate,
  onDelete,
}) {
  const [form, setForm] = useState(emptyFollowUp);
  const [editing, setEditing] = useState(null);
  const [showForm, setShowForm] = useState(false);
  const [statusFilter, setStatusFilter] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const visibleItems = useMemo(
    () => followUps.filter((item) => !statusFilter || item.status === statusFilter),
    [followUps, statusFilter],
  );
  const linkedSources = sourceOptions(form.sourceType, notes, goals, meetings);
  const openCount = followUps.filter((item) =>
    ["open", "in_progress"].includes(item.status)).length;
  const overdueCount = followUps.filter(isOverdue).length;
  const completedCount = followUps.filter((item) => item.status === "completed").length;

  function updateField(event) {
    const { name, value } = event.target;
    setForm((current) => ({
      ...current,
      [name]: name === "sourceId" ? (value ? Number(value) : null) : value,
      ...(name === "sourceType" ? { sourceId: null } : {}),
      ...(name === "status" && value === "completed"
        ? { completionDate: current.completionDate || today() }
        : {}),
      ...(name === "status" && value !== "completed" ? { completionDate: "" } : {}),
    }));
  }

  function startCreate() {
    setEditing(null);
    setForm(emptyFollowUp());
    setError("");
    setShowForm(true);
  }

  function startEdit(item) {
    setEditing(item);
    setForm({ ...emptyFollowUp(), ...item });
    setError("");
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setEditing(null);
    setForm(emptyFollowUp());
    setError("");
  }

  async function submit(event) {
    event.preventDefault();
    try {
      setSaving(true);
      setError("");
      if (editing) {
        await onUpdate(editing.id, form);
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

  async function remove(item) {
    if (!window.confirm(`Delete the follow-up "${item.description}"?`)) {
      return;
    }
    try {
      setError("");
      await onDelete(item.id);
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <section className="follow-ups-section">
      <div className="follow-ups-heading">
        <div>
          <p className="profile-eyebrow">Commitment tracking</p>
          <h2>Follow-ups</h2>
          <p>Keep coaching actions and shared commitments visible through completion.</p>
        </div>
        <button type="button" onClick={startCreate}>
          <Plus size={16} /> Add follow-up
        </button>
      </div>

      <div className="follow-up-metrics">
        <div><strong>{openCount}</strong><span>Open</span></div>
        <div><strong>{overdueCount}</strong><span>Overdue</span></div>
        <div><strong>{completedCount}</strong><span>Completed</span></div>
      </div>

      <label className="follow-up-filter">
        Status
        <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
          <option value="">All follow-ups</option>
          {statuses.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </select>
      </label>

      {error && <div className="error">Error: {error}</div>}

      <Dialog
        open={showForm}
        onClose={saving ? undefined : closeForm}
        fullWidth
        maxWidth="sm"
        aria-label={editing ? "Edit follow-up" : "New follow-up"}
      >
        <form className="follow-up-form" onSubmit={submit}>
          <div className="follow-up-form-heading">
            <h3>{editing ? "Edit follow-up" : "New follow-up"}</h3>
            <button type="button" className="icon-button" onClick={closeForm} aria-label="Close follow-up form">
              <X size={17} />
            </button>
          </div>
          <label className="follow-up-field-wide">
            Description
            <textarea name="description" value={form.description} onChange={updateField} rows="3" required />
          </label>
          <label>
            Owner
            <input name="owner" value={form.owner} onChange={updateField} required />
          </label>
          <label>
            Priority
            <select name="priority" value={form.priority} onChange={updateField}>
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
            </select>
          </label>
          <label>
            Status
            <select name="status" value={form.status} onChange={updateField}>
              {statuses.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
            </select>
          </label>
          <label>
            Due date
            <input type="date" name="dueDate" value={form.dueDate} onChange={updateField} />
          </label>
          {form.status === "completed" && (
            <label>
              Completion date
              <input type="date" name="completionDate" value={form.completionDate} onChange={updateField} required />
            </label>
          )}
          <label>
            Source
            <select name="sourceType" value={form.sourceType} onChange={updateField}>
              {sourceTypes.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
            </select>
          </label>
          {form.sourceType !== "manual" && (
            <label className="follow-up-field-wide">
              Source record
              <select name="sourceId" value={form.sourceId || ""} onChange={updateField} required>
                <option value="">Select a record</option>
                {linkedSources.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
              </select>
            </label>
          )}
          <label className="follow-up-field-wide">
            Notes
            <textarea name="notes" value={form.notes} onChange={updateField} rows="3" />
          </label>
          <div className="form-actions follow-up-field-wide">
            <button type="submit" disabled={saving}>
              {saving ? "Saving..." : editing ? "Save changes" : "Create follow-up"}
            </button>
            <button type="button" className="secondary-button" onClick={closeForm}>Cancel</button>
          </div>
        </form>
      </Dialog>

      <div className="follow-up-list">
        {visibleItems.length === 0 ? (
          <p className="empty-state">
            {followUps.length ? "No follow-ups match this filter." : "No follow-ups recorded yet."}
          </p>
        ) : visibleItems.map((item) => (
          <article className={`follow-up-card ${isOverdue(item) ? "follow-up-overdue" : ""}`} key={item.id}>
            <div className="follow-up-card-heading">
              <div>
                <span className={`follow-up-status follow-up-status-${item.status}`}>
                  {labelFor(statuses, item.status)}
                </span>
                {isOverdue(item) && <span className="follow-up-status follow-up-status-overdue">Overdue</span>}
                <h3>{item.description}</h3>
              </div>
              <div className="table-actions">
                <button type="button" className="icon-button" onClick={() => startEdit(item)} aria-label={`Edit ${item.description}`}>
                  <Pencil size={15} />
                </button>
                <button type="button" className="icon-button danger" onClick={() => remove(item)} aria-label={`Delete ${item.description}`}>
                  <Trash2 size={15} />
                </button>
              </div>
            </div>
            <div className="follow-up-meta">
              <span><strong>Owner:</strong> {item.owner}</span>
              <span><strong>Priority:</strong> {item.priority}</span>
              <span><strong>Due:</strong> {item.dueDate || "Not set"}</span>
              {item.sourceType !== "manual" && <span><strong>Source:</strong> {labelFor(sourceTypes, item.sourceType)} #{item.sourceId}</span>}
            </div>
            {item.notes && <p>{item.notes}</p>}
            {item.status === "completed" && (
              <p className="follow-up-completed"><CheckCircle2 size={16} /> Completed {item.completionDate}</p>
            )}
          </article>
        ))}
      </div>
    </section>
  );
}

export default FollowUpsPanel;
