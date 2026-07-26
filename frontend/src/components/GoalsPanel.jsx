import { useMemo, useState } from "react";
import { Pencil, Plus, Trash2, X } from "lucide-react";
import { Dialog } from "@mui/material";

const goalTypes = [
  ["delivery", "Delivery"],
  ["technical_growth", "Technical growth"],
  ["leadership", "Leadership"],
  ["communication", "Communication"],
  ["operational_excellence", "Operational excellence"],
  ["mentoring", "Mentoring"],
  ["career_development", "Career development"],
  ["stretch_assignment", "Stretch assignment"],
];

const statuses = [
  ["not_started", "Not started"],
  ["in_progress", "In progress"],
  ["blocked", "Blocked"],
  ["completed", "Completed"],
  ["cancelled", "Cancelled"],
];

function emptyGoal(reviewCycle = "") {
  return {
    title: "",
    description: "",
    goalType: "career_development",
    status: "not_started",
    priority: "medium",
    startDate: "",
    targetDate: "",
    completionDate: "",
    progressPercent: 0,
    successCriteria: "",
    managerNotes: "",
    engineerNotes: "",
    reviewCycle,
  };
}

function labelFor(options, value) {
  return options.find(([key]) => key === value)?.[1] || value;
}

function isOverdue(goal) {
  if (!goal.targetDate || ["completed", "cancelled"].includes(goal.status)) {
    return false;
  }
  return goal.targetDate < new Date().toISOString().slice(0, 10);
}

function GoalsPanel({
  goals = [],
  reviewCycle = "",
  onCreate,
  onUpdate,
  onDelete,
}) {
  const [editingGoal, setEditingGoal] = useState(null);
  const [form, setForm] = useState(() => emptyGoal(reviewCycle));
  const [showForm, setShowForm] = useState(false);
  const [statusFilter, setStatusFilter] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const visibleGoals = useMemo(
    () => goals.filter((goal) => !statusFilter || goal.status === statusFilter),
    [goals, statusFilter],
  );
  const activeCount = goals.filter((goal) =>
    ["not_started", "in_progress", "blocked"].includes(goal.status),
  ).length;
  const blockedCount = goals.filter((goal) => goal.status === "blocked").length;
  const overdueCount = goals.filter(isOverdue).length;

  function updateField(event) {
    const { name, value } = event.target;
    setForm((current) => ({
      ...current,
      [name]: name === "progressPercent" ? Number(value) : value,
      ...(name === "status" && value !== "completed"
        ? { completionDate: "" }
        : {}),
      ...(name === "status" && value === "completed"
        ? {
            progressPercent: 100,
            completionDate:
              current.completionDate || new Date().toISOString().slice(0, 10),
          }
        : {}),
    }));
  }

  function startCreate() {
    setEditingGoal(null);
    setForm(emptyGoal(reviewCycle));
    setError("");
    setShowForm(true);
  }

  function startEdit(goal) {
    setEditingGoal(goal);
    setForm({ ...emptyGoal(reviewCycle), ...goal });
    setError("");
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setEditingGoal(null);
    setForm(emptyGoal(reviewCycle));
    setError("");
  }

  async function submit(event) {
    event.preventDefault();
    try {
      setSaving(true);
      setError("");
      if (editingGoal) {
        await onUpdate(editingGoal.id, form);
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

  async function remove(goal) {
    if (!window.confirm(`Delete the goal "${goal.title}"?`)) {
      return;
    }
    try {
      setError("");
      await onDelete(goal.id);
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <section className="goals-section">
      <div className="goals-heading">
        <div>
          <p className="profile-eyebrow">Development planning</p>
          <h2>Goals</h2>
          <p>Track forward-looking commitments, progress, and outcomes.</p>
        </div>
        <button type="button" onClick={startCreate}>
          <Plus size={16} /> Add goal
        </button>
      </div>

      <div className="goal-metrics">
        <div><strong>{activeCount}</strong><span>Active</span></div>
        <div><strong>{blockedCount}</strong><span>Blocked</span></div>
        <div><strong>{overdueCount}</strong><span>Overdue</span></div>
      </div>

      <label className="goal-filter">
        Status
        <select
          value={statusFilter}
          onChange={(event) => setStatusFilter(event.target.value)}
        >
          <option value="">All goals</option>
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
        aria-label={editingGoal ? "Edit goal" : "New goal"}
      >
        <form className="goal-form" onSubmit={submit}>
          <div className="goal-form-heading">
            <h3>{editingGoal ? "Edit goal" : "New goal"}</h3>
            <button
              type="button"
              className="icon-button"
              onClick={closeForm}
              aria-label="Close goal form"
            >
              <X size={17} />
            </button>
          </div>

          <label className="goal-field-wide">
            Title
            <input
              name="title"
              value={form.title}
              onChange={updateField}
              required
            />
          </label>
          <label className="goal-field-wide">
            Description
            <textarea
              name="description"
              value={form.description}
              onChange={updateField}
              rows="3"
            />
          </label>
          <label>
            Type
            <select name="goalType" value={form.goalType} onChange={updateField}>
              {goalTypes.map(([value, label]) => (
                <option key={value} value={value}>{label}</option>
              ))}
            </select>
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
            Priority
            <select name="priority" value={form.priority} onChange={updateField}>
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
            </select>
          </label>
          <label>
            Progress ({form.progressPercent}%)
            <input
              type="range"
              name="progressPercent"
              min="0"
              max="100"
              value={form.progressPercent}
              onChange={updateField}
            />
          </label>
          <label>
            Start date
            <input type="date" name="startDate" value={form.startDate} onChange={updateField} />
          </label>
          <label>
            Target date
            <input type="date" name="targetDate" value={form.targetDate} onChange={updateField} />
          </label>
          {form.status === "completed" && (
            <label>
              Completion date
              <input
                type="date"
                name="completionDate"
                value={form.completionDate}
                onChange={updateField}
                required
              />
            </label>
          )}
          <label>
            Review cycle
            <input name="reviewCycle" value={form.reviewCycle} onChange={updateField} />
          </label>
          <label className="goal-field-wide">
            Success criteria
            <textarea name="successCriteria" value={form.successCriteria} onChange={updateField} rows="2" />
          </label>
          <label className="goal-field-wide">
            Manager notes
            <textarea name="managerNotes" value={form.managerNotes} onChange={updateField} rows="2" />
          </label>
          <label className="goal-field-wide">
            Engineer notes
            <textarea name="engineerNotes" value={form.engineerNotes} onChange={updateField} rows="2" />
          </label>
          <div className="form-actions goal-field-wide">
            <button type="submit" disabled={saving}>
              {saving ? "Saving..." : editingGoal ? "Save changes" : "Create goal"}
            </button>
            <button type="button" className="secondary-button" onClick={closeForm}>
              Cancel
            </button>
          </div>
        </form>
      </Dialog>

      <div className="goals-grid">
        {visibleGoals.length === 0 ? (
          <p className="empty-state">
            {goals.length ? "No goals match this filter." : "No goals recorded yet."}
          </p>
        ) : visibleGoals.map((goal) => (
          <article
            className={`goal-card ${isOverdue(goal) ? "goal-overdue" : ""}`}
            key={goal.id}
          >
            <div className="goal-card-heading">
              <div>
                <span className={`goal-status goal-status-${goal.status}`}>
                  {labelFor(statuses, goal.status)}
                </span>
                {isOverdue(goal) && <span className="goal-status goal-status-overdue">Overdue</span>}
                <h3>{goal.title}</h3>
              </div>
              <div className="table-actions">
                <button type="button" className="icon-button" onClick={() => startEdit(goal)} aria-label={`Edit ${goal.title}`}>
                  <Pencil size={15} />
                </button>
                <button type="button" className="icon-button danger" onClick={() => remove(goal)} aria-label={`Delete ${goal.title}`}>
                  <Trash2 size={15} />
                </button>
              </div>
            </div>
            <p>{goal.description || "No description provided."}</p>
            <div className="goal-progress" aria-label={`${goal.progressPercent}% complete`}>
              <span style={{ width: `${goal.progressPercent}%` }} />
            </div>
            <div className="goal-meta">
              <span><strong>{goal.progressPercent}%</strong> complete</span>
              <span>{labelFor(goalTypes, goal.goalType)}</span>
              <span>{goal.priority} priority</span>
              <span>Target: {goal.targetDate || "Not set"}</span>
            </div>
            {goal.successCriteria && (
              <p className="goal-criteria"><strong>Success:</strong> {goal.successCriteria}</p>
            )}
          </article>
        ))}
      </div>
    </section>
  );
}

export default GoalsPanel;
