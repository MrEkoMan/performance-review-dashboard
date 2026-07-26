import { useMemo, useState } from "react";
import { Dialog } from "@mui/material";
import { Award, Pencil, Plus, Trash2, X } from "lucide-react";

const sourceTypes = [
  ["manager", "Manager"],
  ["peer", "Peer"],
  ["product", "Product"],
  ["customer", "Customer"],
  ["leadership", "Leadership"],
  ["cross_functional", "Cross-functional stakeholder"],
  ["external_partner", "External partner"],
];

const categories = [
  ["business_impact", "Business impact"],
  ["technical_excellence", "Technical excellence"],
  ["operational_excellence", "Operational excellence"],
  ["mentoring", "Mentoring"],
  ["collaboration", "Collaboration"],
  ["leadership", "Leadership"],
  ["innovation", "Innovation"],
  ["customer_focus", "Customer focus"],
];

function today() {
  return new Date().toISOString().slice(0, 10);
}

function emptyRecognition(reviewCycle = "") {
  return {
    recognitionDate: today(),
    source: "",
    sourceType: "manager",
    category: "business_impact",
    summary: "",
    details: "",
    relatedWork: "",
    reviewCycle,
  };
}

function labelFor(options, value) {
  return options.find(([key]) => key === value)?.[1] || value;
}

function RecognitionPanel({
  recognitions = [],
  reviewCycle = "",
  onCreate,
  onUpdate,
  onDelete,
}) {
  const [form, setForm] = useState(() => emptyRecognition(reviewCycle));
  const [editing, setEditing] = useState(null);
  const [showForm, setShowForm] = useState(false);
  const [categoryFilter, setCategoryFilter] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const visibleItems = useMemo(
    () => recognitions.filter((item) =>
      !categoryFilter || item.category === categoryFilter),
    [recognitions, categoryFilter],
  );
  const ninetyDaysAgo = new Date();
  ninetyDaysAgo.setDate(ninetyDaysAgo.getDate() - 90);
  const recentCount = recognitions.filter(
    (item) => new Date(`${item.recognitionDate}T00:00:00`) >= ninetyDaysAgo,
  ).length;
  const sourceCount = new Set(
    recognitions.map((item) => item.source.trim().toLowerCase()).filter(Boolean),
  ).size;

  function updateField(event) {
    const { name, value } = event.target;
    setForm((current) => ({ ...current, [name]: value }));
  }

  function startCreate() {
    setEditing(null);
    setForm(emptyRecognition(reviewCycle));
    setError("");
    setShowForm(true);
  }

  function startEdit(item) {
    setEditing(item);
    setForm({ ...emptyRecognition(reviewCycle), ...item });
    setError("");
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setEditing(null);
    setForm(emptyRecognition(reviewCycle));
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
    if (!window.confirm(`Delete the recognition "${item.summary}"?`)) {
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
    <section className="recognition-section">
      <div className="recognition-heading">
        <div>
          <p className="profile-eyebrow">Positive impact</p>
          <h2>Recognition</h2>
          <p>Preserve praise and examples of influence for balanced reviews.</p>
        </div>
        <button type="button" onClick={startCreate}>
          <Plus size={16} /> Add recognition
        </button>
      </div>

      <div className="recognition-metrics">
        <div><strong>{recognitions.length}</strong><span>Total</span></div>
        <div><strong>{recentCount}</strong><span>Last 90 days</span></div>
        <div><strong>{sourceCount}</strong><span>Distinct sources</span></div>
      </div>

      <label className="recognition-filter">
        Category
        <select value={categoryFilter} onChange={(event) => setCategoryFilter(event.target.value)}>
          <option value="">All recognition</option>
          {categories.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </select>
      </label>

      {error && <div className="error">Error: {error}</div>}

      <Dialog
        open={showForm}
        onClose={saving ? undefined : closeForm}
        fullWidth
        maxWidth="sm"
        aria-label={editing ? "Edit recognition" : "New recognition"}
      >
        <form className="recognition-form" onSubmit={submit}>
          <div className="recognition-form-heading">
            <h3>{editing ? "Edit recognition" : "New recognition"}</h3>
            <button type="button" className="icon-button" onClick={closeForm} aria-label="Close recognition form">
              <X size={17} />
            </button>
          </div>
          <label>
            Recognition date
            <input type="date" name="recognitionDate" value={form.recognitionDate} onChange={updateField} required />
          </label>
          <label>
            Review cycle
            <input name="reviewCycle" value={form.reviewCycle} onChange={updateField} />
          </label>
          <label>
            Source type
            <select name="sourceType" value={form.sourceType} onChange={updateField}>
              {sourceTypes.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
            </select>
          </label>
          <label>
            Source
            <input name="source" value={form.source} onChange={updateField} placeholder="Name, team, or organization" required />
          </label>
          <label className="recognition-field-wide">
            Category
            <select name="category" value={form.category} onChange={updateField}>
              {categories.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
            </select>
          </label>
          <label className="recognition-field-wide">
            Summary
            <input name="summary" value={form.summary} onChange={updateField} required />
          </label>
          <label className="recognition-field-wide">
            Details
            <textarea name="details" value={form.details} onChange={updateField} rows="4" />
          </label>
          <label className="recognition-field-wide">
            Related work
            <input name="relatedWork" value={form.relatedWork} onChange={updateField} placeholder="Project, incident, initiative, or evidence reference" />
          </label>
          <div className="form-actions recognition-field-wide">
            <button type="submit" disabled={saving}>
              {saving ? "Saving..." : editing ? "Save changes" : "Create recognition"}
            </button>
            <button type="button" className="secondary-button" onClick={closeForm}>Cancel</button>
          </div>
        </form>
      </Dialog>

      <div className="recognition-list">
        {visibleItems.length === 0 ? (
          <p className="empty-state">
            {recognitions.length ? "No recognition matches this filter." : "No recognition recorded yet."}
          </p>
        ) : visibleItems.map((item) => (
          <article className="recognition-card" key={item.id}>
            <div className="recognition-card-heading">
              <div className="recognition-title">
                <Award size={20} />
                <div>
                  <span className="recognition-category">{labelFor(categories, item.category)}</span>
                  <h3>{item.summary}</h3>
                </div>
              </div>
              <div className="table-actions">
                <button type="button" className="icon-button" onClick={() => startEdit(item)} aria-label={`Edit ${item.summary}`}>
                  <Pencil size={15} />
                </button>
                <button type="button" className="icon-button danger" onClick={() => remove(item)} aria-label={`Delete ${item.summary}`}>
                  <Trash2 size={15} />
                </button>
              </div>
            </div>
            <div className="recognition-meta">
              <span>{item.recognitionDate}</span>
              <span>{labelFor(sourceTypes, item.sourceType)}: <strong>{item.source}</strong></span>
              {item.reviewCycle && <span>{item.reviewCycle}</span>}
            </div>
            {item.details && <p>{item.details}</p>}
            {item.relatedWork && <p className="recognition-related"><strong>Related work:</strong> {item.relatedWork}</p>}
          </article>
        ))}
      </div>
    </section>
  );
}

export default RecognitionPanel;
