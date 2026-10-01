import { useMemo, useState } from "react";
import { Pencil, Plus, Trash2, Upload, X } from "lucide-react";
import { Dialog } from "@mui/material";
import { parseDevelopmentPlanFile } from "../api/performanceApi";

// The structured sections of a parsed plan, kept here as data so the read and
// edit views stay in sync with the backend DevelopmentPlanFields JSON shape.
const EMPTY_HEADER = {
  developer: "",
  currentRole: "",
  targetRole: "",
  planPeriod: "",
  manager: "",
};

function emptyFields() {
  return {
    header: { ...EMPTY_HEADER },
    strengths: [],
    growthAreas: [],
    nextRole: [],
    focusAreas: [],
    goals: [],
    accomplishments: [],
  };
}

function emptyForm(reviewCycle = "") {
  return {
    title: "Personal Development Plan",
    planDate: new Date().toISOString().slice(0, 10),
    reviewCycle,
    rawMarkdown: "",
    fields: emptyFields(),
    linkedGoalId: null,
  };
}

function today() {
  return new Date().toISOString().slice(0, 10);
}

function DevelopmentPlanPanel({
  engineerId,
  plans = [],
  goals = [],
  reviewCycle = "",
  engineers = [],
  onCreate,
  onUpdate,
  onDelete,
  onCreateGoal,
  readOnly = false,
}) {
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState(null);
  const [form, setForm] = useState(() => emptyForm(reviewCycle));
  const [saving, setSaving] = useState(false);
  const [parsing, setParsing] = useState(false);
  const [error, setError] = useState("");
  const [parseNotice, setParseNotice] = useState("");

  const engineerOptions = useMemo(
    () =>
      [{ id: Number(engineerId), label: "This engineer" }]
        .concat(
          engineers
            .filter((e) => String(e.id) !== String(engineerId))
            .map((e) => ({ id: e.id, label: e.name })),
        ),
    [engineers, engineerId],
  );
  const [targetEngineerId, setTargetEngineerId] = useState(Number(engineerId));

  function startCreate() {
    setEditing(null);
    setForm(emptyForm(reviewCycle));
    setTargetEngineerId(Number(engineerId));
    setError("");
    setParseNotice("");
    setShowForm(true);
  }

  function startEdit(plan) {
    setEditing(plan);
    setForm({
      title: plan.title || "Personal Development Plan",
      planDate: plan.planDate || today(),
      reviewCycle: plan.reviewCycle || reviewCycle,
      rawMarkdown: plan.rawMarkdown || "",
      fields: mergeFields(emptyFields(), plan.fields),
      linkedGoalId: plan.linkedGoalId ?? null,
    });
    setTargetEngineerId(Number(engineerId));
    setError("");
    setParseNotice("");
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setEditing(null);
    setError("");
    setParseNotice("");
  }

  async function handleParse(event) {
    const file = event.target.files?.[0];
    if (!file) return;
    setError("");
    setParseNotice("");
    setParsing(true);
    try {
      const parsed = await parseDevelopmentPlanFile(file);
      setForm((current) => ({
        ...current,
        rawMarkdown: current.rawMarkdown || "",
        fields: mergeFields(emptyFields(), current.fields, parsed),
      }));
      setParseNotice(
        `Imported "${file.name}". Review the fields below and save when ready.`,
      );
    } catch (err) {
      setError(err.message);
    } finally {
      setParsing(false);
      event.target.value = "";
    }
  }

  function updateHeader(field, value) {
    setForm((current) => ({
      ...current,
      fields: {
        ...current.fields,
        header: { ...current.fields.header, [field]: value },
      },
    }));
  }

  function updateListField(field, idx, value) {
    setForm((current) => {
      const next = [...current.fields[field]];
      next[idx] = value;
      return { ...current, fields: { ...current.fields, [field]: next } };
    });
  }

  function addListItem(field) {
    setForm((current) => ({
      ...current,
      fields: { ...current.fields, [field]: [...current.fields[field], ""] },
    }));
  }

  function removeListItem(field, idx) {
    setForm((current) => {
      const next = current.fields[field].filter((_, i) => i !== idx);
      return { ...current, fields: { ...current.fields, [field]: next } };
    });
  }

  function updateNextRole(idx, field, value) {
    setForm((current) => {
      const next = [...current.fields.nextRole];
      next[idx] = { ...next[idx], [field]: value };
      return { ...current, fields: { ...current.fields, nextRole: next } };
    });
  }

  function updateGoal(idx, field, value) {
    setForm((current) => {
      const goals = [...current.fields.goals];
      goals[idx] = { ...goals[idx], [field]: value };
      return { ...current, fields: { ...current.fields, goals } };
    });
  }

  function updateGoalMonth(goalIdx, monthIdx, field, value) {
    setForm((current) => {
      const goals = [...current.fields.goals];
      const months = [...(goals[goalIdx].monthlyProgress || [])];
      months[monthIdx] = { ...months[monthIdx], [field]: value };
      goals[goalIdx] = { ...goals[goalIdx], monthlyProgress: months };
      return { ...current, fields: { ...current.fields, goals } };
    });
  }

  function addGoal() {
    setForm((current) => ({
      ...current,
      fields: {
        ...current.fields,
        goals: [
          ...current.fields.goals,
          {
            title: "",
            goal: "",
            why: "",
            successLooksLike: "",
            targetDate: "",
            monthlyProgress: [],
          },
        ],
      },
    }));
  }

  function removeGoal(idx) {
    setForm((current) => ({
      ...current,
      fields: {
        ...current.fields,
        goals: current.fields.goals.filter((_, i) => i !== idx),
      },
    }));
  }

  function updateAccomplishment(idx, field, value) {
    setForm((current) => {
      const accs = [...current.fields.accomplishments];
      accs[idx] = { ...accs[idx], [field]: value };
      return { ...current, fields: { ...current.fields, accomplishments: accs } };
    });
  }

  async function createGoalFromPlan(goalIdx) {
    const g = form.fields.goals[goalIdx];
    if (!g.goal?.trim() && !g.title?.trim()) {
      setError("This plan goal has no title or goal text to create a Goal from.");
      return;
    }
    try {
      setError("");
      const created = await onCreateGoal(targetEngineerId, {
        title: g.title?.trim() || g.goal.trim(),
        description: g.why?.trim() || "",
        goalType: "career_development",
        status: "not_started",
        priority: "medium",
        startDate: today(),
        targetDate: "",
        completionDate: "",
        progressPercent: 0,
        successCriteria: g.successLooksLike?.trim() || "",
        managerNotes: "",
        engineerNotes: "",
        reviewCycle: form.reviewCycle || reviewCycle,
      });
      const newGoalId = created?.id;
      if (newGoalId) {
        setForm((current) => ({
          ...current,
          linkedGoalId: newGoalId,
        }));
        setParseNotice(
          `Created Goal "${created.title}" from this plan and linked it.`,
        );
      }
    } catch (err) {
      setError(err.message);
    }
  }

  async function submit(event) {
    event.preventDefault();
    try {
      setSaving(true);
      setError("");
      const payload = {
        title: form.title,
        planDate: form.planDate,
        reviewCycle: form.reviewCycle,
        rawMarkdown: form.rawMarkdown,
        fields: form.fields,
        linkedGoalId: form.linkedGoalId,
      };
      if (editing) {
        await onUpdate(editing.id, payload);
      } else {
        // Upload supports selecting which engineer the plan corresponds to.
        await onCreate(targetEngineerId, payload);
      }
      closeForm();
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  }

  async function remove(plan) {
    if (!window.confirm(`Delete "${plan.title}"?`)) return;
    try {
      setError("");
      await onDelete(plan.id);
    } catch (err) {
      setError(err.message);
    }
  }

  const goalsById = useMemo(() => {
    const map = new Map();
    goals.forEach((g) => map.set(String(g.id), g));
    return map;
  }, [goals]);

  return (
    <section className="devplan-section">
      <div className="devplan-heading">
        <div>
          <p className="profile-eyebrow">Forward-looking growth</p>
          <h2>Development Plans</h2>
          <p>
            Upload a personal development plan (markdown) to parse its goals,
            focus areas, and accomplishments, then link or create Goals from it.
          </p>
        </div>
        {!readOnly && (
          <button type="button" onClick={startCreate}>
            <Plus size={16} /> Upload / create
          </button>
        )}
      </div>

      {error && <div className="error">Error: {error}</div>}

      <div className="devplan-grid">
        {plans.length === 0 ? (
          <p className="empty-state">
            No development plans yet. Click <strong>Upload / create</strong> to
            upload a markdown plan and parse it, or create one manually.
          </p>
        ) : (
          plans.map((plan) => {
            const linked = plan.linkedGoalId
              ? goalsById.get(String(plan.linkedGoalId))
              : null;
            return (
              <article className="devplan-card" key={plan.id}>
                <div className="devplan-card-heading">
                  <div>
                    <h3>{plan.title}</h3>
                    <p className="devplan-meta">
                      <span>Plan date: {plan.planDate || "—"}</span>
                      {plan.reviewCycle && <span>{plan.reviewCycle}</span>}
                    </p>
                  </div>
                  {!readOnly && (
                    <div className="table-actions">
                      <button
                        type="button"
                        className="icon-button"
                        onClick={() => startEdit(plan)}
                        aria-label={`Edit ${plan.title}`}
                      >
                        <Pencil size={15} />
                      </button>
                      <button
                        type="button"
                        className="icon-button danger"
                        onClick={() => remove(plan)}
                        aria-label={`Delete ${plan.title}`}
                      >
                        <Trash2 size={15} />
                      </button>
                    </div>
                  )}
                </div>

                {plan.fields?.header && hasHeader(plan.fields.header) && (
                  <dl className="devplan-header-grid">
                    {plan.fields.header.developer && (
                      <div><dt>Developer</dt><dd>{plan.fields.header.developer}</dd></div>
                    )}
                    {plan.fields.header.currentRole && (
                      <div><dt>Current role</dt><dd>{plan.fields.header.currentRole}</dd></div>
                    )}
                    {plan.fields.header.targetRole && (
                      <div><dt>Target role</dt><dd>{plan.fields.header.targetRole}</dd></div>
                    )}
                    {plan.fields.header.planPeriod && (
                      <div><dt>Plan period</dt><dd>{plan.fields.header.planPeriod}</dd></div>
                    )}
                    {plan.fields.header.manager && (
                      <div><dt>Manager</dt><dd>{plan.fields.header.manager}</dd></div>
                    )}
                  </dl>
                )}

                {plan.fields?.focusAreas?.length > 0 && (
                  <div className="devplan-tags">
                    {plan.fields.focusAreas.map((f, i) => (
                      <span key={i}>{f}</span>
                    ))}
                  </div>
                )}

                {plan.fields?.goals?.length > 0 && (
                  <ul className="devplan-goal-list">
                    {plan.fields.goals.map((g, i) => (
                      <li key={i}>
                        <strong>{g.title || "Untitled goal"}</strong>
                        {g.targetDate && <span> · {g.targetDate}</span>}
                        {g.goal && <p>{g.goal}</p>}
                      </li>
                    ))}
                  </ul>
                )}

                {linked && (
                  <p className="devplan-linked">
                    Linked to goal: <strong>{linked.title}</strong>
                  </p>
                )}
                {!linked && plan.linkedGoalId && (
                  <p className="devplan-linked devplan-linked-missing">
                    Linked goal {plan.linkedGoalId} no longer exists.
                  </p>
                )}
              </article>
            );
          })
        )}
      </div>

      <Dialog
        open={showForm}
        onClose={saving || parsing ? undefined : closeForm}
        fullWidth
        maxWidth="md"
        aria-label={editing ? "Edit development plan" : "New development plan"}
      >
        <form className="devplan-form" onSubmit={submit}>
          <div className="devplan-form-heading">
            <h3>{editing ? "Edit development plan" : "New development plan"}</h3>
            <button
              type="button"
              className="icon-button"
              onClick={closeForm}
              aria-label="Close development plan form"
            >
              <X size={17} />
            </button>
          </div>

          <div className="devplan-upload devplan-field-wide">
            <Upload size={16} />
            <label className="devplan-upload-label">
              {parsing ? "Parsing…" : "Upload development plan (.md / .txt) to auto-fill"}
              <input
                type="file"
                accept=".md,.markdown,.txt,text/plain"
                onChange={handleParse}
                disabled={parsing || saving}
              />
            </label>
          </div>
          {parseNotice && (
            <p className="devplan-parse-notice devplan-field-wide">{parseNotice}</p>
          )}

          {!editing && (
            <label className="devplan-field-wide">
              Applies to which engineer?
              <select
                value={targetEngineerId}
                onChange={(event) => setTargetEngineerId(Number(event.target.value))}
                disabled={saving || parsing}
              >
                {engineerOptions.map((opt) => (
                  <option key={opt.id} value={opt.id}>{opt.label}</option>
                ))}
              </select>
            </label>
          )}

          <label className="devplan-field-wide">
            Plan title
            <input
              name="title"
              value={form.title}
              onChange={(event) =>
                setForm((current) => ({ ...current, title: event.target.value }))
              }
            />
          </label>

          <div className="devplan-form-row">
            <label>
              Plan date
              <input
                type="date"
                name="planDate"
                value={form.planDate}
                onChange={(event) =>
                  setForm((current) => ({ ...current, planDate: event.target.value }))
                }
              />
            </label>
            <label>
              Review cycle
              <input
                name="reviewCycle"
                value={form.reviewCycle}
                onChange={(event) =>
                  setForm((current) => ({ ...current, reviewCycle: event.target.value }))
                }
              />
            </label>
          </div>

          <div className="devplan-form-group devplan-field-wide">
            <h4>Header</h4>
            <div className="devplan-form-row">
              <label>
                Developer
                <input
                  value={form.fields.header.developer}
                  onChange={(event) => updateHeader("developer", event.target.value)}
                />
              </label>
              <label>
                Current role
                <input
                  value={form.fields.header.currentRole}
                  onChange={(event) => updateHeader("currentRole", event.target.value)}
                />
              </label>
            </div>
            <div className="devplan-form-row">
              <label>
                Target role
                <input
                  value={form.fields.header.targetRole}
                  onChange={(event) => updateHeader("targetRole", event.target.value)}
                />
              </label>
              <label>
                Plan period
                <input
                  value={form.fields.header.planPeriod}
                  onChange={(event) => updateHeader("planPeriod", event.target.value)}
                />
              </label>
            </div>
            <label>
              Manager
              <input
                value={form.fields.header.manager}
                onChange={(event) => updateHeader("manager", event.target.value)}
              />
            </label>
          </div>

          <ListEditor
            title="Focus areas"
            items={form.fields.focusAreas}
            onAdd={() => addListItem("focusAreas")}
            onChange={(idx, value) => updateListField("focusAreas", idx, value)}
            onRemove={(idx) => removeListItem("focusAreas", idx)}
          />

          <div className="devplan-form-group devplan-field-wide">
            <h4>Next-role gaps</h4>
            {form.fields.nextRole.map((gap, idx) => (
              <div className="devplan-form-row" key={idx}>
                <label>
                  What the next role looks like
                  <textarea
                    rows="2"
                    value={gap.nextRoleLooksLike}
                    onChange={(event) =>
                      updateNextRole(idx, "nextRoleLooksLike", event.target.value)
                    }
                  />
                </label>
                <label>
                  Gap from where I am today
                  <textarea
                    rows="2"
                    value={gap.gap}
                    onChange={(event) => updateGap(idx, "gap", event.target.value)}
                  />
                </label>
                <button
                  type="button"
                  className="secondary-button"
                  onClick={() => removeNextRole(idx)}
                >
                  Remove
                </button>
              </div>
            ))}
            <button
              type="button"
              className="secondary-button"
              onClick={addNextRole}
            >
              Add gap row
            </button>
          </div>

          <div className="devplan-form-group devplan-field-wide">
            <div className="devplan-form-group-heading">
              <h4>Goals</h4>
              <button type="button" className="secondary-button" onClick={addGoal}>
                Add goal
              </button>
            </div>
            {form.fields.goals.length === 0 && (
              <p className="empty-state">No goals parsed or added yet.</p>
            )}
            {form.fields.goals.map((goal, idx) => (
              <div className="devplan-goal-block" key={idx}>
                <div className="devplan-goal-block-heading">
                  <strong>Goal {idx + 1}</strong>
                  <button
                    type="button"
                    className="secondary-button"
                    onClick={() => removeGoal(idx)}
                  >
                    Remove
                  </button>
                </div>
                <label>
                  Title
                  <input
                    value={goal.title}
                    onChange={(event) => updateGoal(idx, "title", event.target.value)}
                  />
                </label>
                <label className="devplan-field-wide">
                  Goal
                  <textarea
                    rows="2"
                    value={goal.goal}
                    onChange={(event) => updateGoal(idx, "goal", event.target.value)}
                  />
                </label>
                <label className="devplan-field-wide">
                  Why
                  <textarea
                    rows="2"
                    value={goal.why}
                    onChange={(event) => updateGoal(idx, "why", event.target.value)}
                  />
                </label>
                <div className="devplan-form-row">
                  <label className="devplan-field-wide">
                    Success looks like
                    <textarea
                      rows="2"
                      value={goal.successLooksLike}
                      onChange={(event) =>
                        updateGoal(idx, "successLooksLike", event.target.value)
                      }
                    />
                  </label>
                </div>
                <label>
                  Target date
                  <input
                    value={goal.targetDate}
                    onChange={(event) => updateGoal(idx, "targetDate", event.target.value)}
                  />
                </label>

                {goal.monthlyProgress?.length > 0 && (
                  <div className="devplan-monthly">
                    <p className="devplan-monthly-label">Monthly progress</p>
                    {goal.monthlyProgress.map((m, mi) => (
                      <div className="devplan-form-row" key={mi}>
                        <input
                          value={m.month}
                          placeholder="Month"
                          onChange={(event) =>
                            updateGoalMonth(idx, mi, "month", event.target.value)
                          }
                        />
                        <input
                          value={m.status}
                          placeholder="Status"
                          onChange={(event) =>
                            updateGoalMonth(idx, mi, "status", event.target.value)
                          }
                        />
                        <input
                          value={m.update}
                          placeholder="Update"
                          onChange={(event) =>
                            updateGoalMonth(idx, mi, "update", event.target.value)
                          }
                        />
                      </div>
                    ))}
                  </div>
                )}

                <button
                  type="button"
                  className="secondary-button"
                  onClick={() => createGoalFromPlan(idx)}
                  disabled={saving}
                >
                  Create a Goal from this plan goal
                </button>
              </div>
            ))}
          </div>

          <div className="devplan-form-group devplan-field-wide">
            <div className="devplan-form-group-heading">
              <h4>Accomplishments</h4>
              <button
                type="button"
                className="secondary-button"
                onClick={() =>
                  setForm((current) => ({
                    ...current,
                    fields: {
                      ...current.fields,
                      accomplishments: [
                        ...current.fields.accomplishments,
                        { label: "", accomplishment: "", problem: "", valueDelivered: "" },
                      ],
                    },
                  }))
                }
              >
                Add accomplishment
              </button>
            </div>
            {form.fields.accomplishments.length === 0 && (
              <p className="empty-state">No accomplishments recorded.</p>
            )}
            {form.fields.accomplishments.map((acc, idx) => (
              <div className="devplan-acc-block" key={idx}>
                <label className="devplan-field-wide">
                  Accomplishment
                  <textarea
                    rows="2"
                    value={acc.accomplishment}
                    onChange={(event) =>
                      updateAccomplishment(idx, "accomplishment", event.target.value)
                    }
                  />
                </label>
                <label className="devplan-field-wide">
                  Problem
                  <textarea
                    rows="2"
                    value={acc.problem}
                    onChange={(event) =>
                      updateAccomplishment(idx, "problem", event.target.value)
                    }
                  />
                </label>
                <label className="devplan-field-wide">
                  Value delivered
                  <textarea
                    rows="2"
                    value={acc.valueDelivered}
                    onChange={(event) =>
                      updateAccomplishment(idx, "valueDelivered", event.target.value)
                    }
                  />
                </label>
                <button
                  type="button"
                  className="secondary-button"
                  onClick={() =>
                    setForm((current) => ({
                      ...current,
                      fields: {
                        ...current.fields,
                        accomplishments: current.fields.accomplishments.filter(
                          (_, i) => i !== idx,
                        ),
                      },
                    }))
                  }
                >
                  Remove
                </button>
              </div>
            ))}
          </div>

          <label className="devplan-field-wide devplan-link-goal">
            Link to existing goal
            <select
              value={form.linkedGoalId ?? ""}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  linkedGoalId: event.target.value
                    ? Number(event.target.value)
                    : null,
                }))
              }
              disabled={saving}
            >
              <option value="">None</option>
              {goals.map((g) => (
                <option key={g.id} value={g.id}>{g.title}</option>
              ))}
            </select>
          </label>

          <div className="form-actions devplan-field-wide">
            <button type="submit" disabled={saving || parsing}>
              {saving ? "Saving..." : editing ? "Save changes" : "Save development plan"}
            </button>
            <button type="button" className="secondary-button" onClick={closeForm}>
              Cancel
            </button>
          </div>
        </form>
      </Dialog>
    </section>
  );

  function addNextRole() {
    setForm((current) => ({
      ...current,
      fields: {
        ...current.fields,
        nextRole: [
          ...current.fields.nextRole,
          { nextRoleLooksLike: "", gap: "" },
        ],
      },
    }));
  }

  function removeNextRole(idx) {
    setForm((current) => ({
      ...current,
      fields: {
        ...current.fields,
        nextRole: current.fields.nextRole.filter((_, i) => i !== idx),
      },
    }));
  }

  function updateGap(idx, field, value) {
    updateNextRole(idx, field, value);
  }
}

// ListEditor renders an editable list of single-line text items (used for
// strengths, growth areas, focus areas).
function ListEditor({ title, items, onAdd, onChange, onRemove }) {
  return (
    <div className="devplan-form-group devplan-field-wide">
      <div className="devplan-form-group-heading">
        <h4>{title}</h4>
        <button type="button" className="secondary-button" onClick={onAdd}>
          Add
        </button>
      </div>
      {items.length === 0 && <p className="empty-state">None recorded.</p>}
      {items.map((item, idx) => (
        <div className="devplan-list-row" key={idx}>
          <input
            value={item}
            onChange={(event) => onChange(idx, event.target.value)}
          />
          <button
            type="button"
            className="icon-button"
            onClick={() => onRemove(idx)}
            aria-label={`Remove ${title} item ${idx + 1}`}
          >
            <X size={15} />
          </button>
        </div>
      ))}
    </div>
  );
}

// mergeFields layers parsed/edited values over the defaults so a fresh upload
// fills empty fields while a re-parse preserves manager edits where the parser
// returned nothing.
function mergeFields(base, ...overrides) {
  const merged = JSON.parse(JSON.stringify(base));
  for (const override of overrides) {
    if (!override) continue;
    if (override.header) {
      merged.header = {
        ...merged.header,
        ...stripEmpty(override.header),
      };
    }
    if (Array.isArray(override.strengths) && override.strengths.length) {
      merged.strengths = override.strengths;
    }
    if (Array.isArray(override.growthAreas) && override.growthAreas.length) {
      merged.growthAreas = override.growthAreas;
    }
    if (Array.isArray(override.nextRole) && override.nextRole.length) {
      merged.nextRole = override.nextRole;
    }
    if (Array.isArray(override.focusAreas) && override.focusAreas.length) {
      merged.focusAreas = override.focusAreas;
    }
    if (Array.isArray(override.goals) && override.goals.length) {
      merged.goals = override.goals;
    }
    if (Array.isArray(override.accomplishments) && override.accomplishments.length) {
      merged.accomplishments = override.accomplishments;
    }
  }
  return merged;
}

function stripEmpty(obj) {
  const out = {};
  for (const [k, v] of Object.entries(obj)) {
    if (v !== "" && v !== null && v !== undefined) out[k] = v;
  }
  return out;
}

function hasHeader(header) {
  return Object.values(header || {}).some((v) => v && String(v).trim());
}

export default DevelopmentPlanPanel;
