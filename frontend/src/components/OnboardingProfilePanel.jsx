import { useMemo, useState } from "react";
import { Pencil, Plus, Upload, X } from "lucide-react";
import { Dialog } from "@mui/material";
import { parseOnboardingFile } from "../api/performanceApi";

// The four sections and their questions, keyed to the backend OnboardingAnswers
// JSON shape. Keeping this as data keeps the read/edit/summary views in sync and
// makes it cheap to reorder or reword a question later without touching markup.
const SECTIONS = [
  {
    key: "careerMotivation",
    label: "Career & Motivation",
    questions: [
      ["enjoyMost", "What do you enjoy most about your current role?"],
      ["energyGivers", "What type of work gives you energy?"],
      ["energyDrainers", "What type of work drains your energy?"],
      ["skillsThisYear", "What skills are you hoping to develop this year?"],
      ["careerNext2to3", "Where do you want your career to go over the next 2-3 years?"],
    ],
  },
  {
    key: "teamOrg",
    label: "Team & Organization",
    questions: [
      ["teamDoesWell", "What does this team do really well?"],
      ["frustrations", "What frustrates you about how the team operates?"],
      ["emForADay", "If you were Engineering Manager for a day, what would you change?"],
      ["slowingUsDown", "What is slowing us down?"],
      ["techDebtRisk", "Where do you see technical debt creating risk?"],
    ],
  },
  {
    key: "workingStyle",
    label: "Individual Working Style",
    questions: [
      ["preferredFeedback", "How do you prefer feedback?"],
      ["coachingVsAutonomy", "How often do you want coaching vs autonomy?"],
      ["greatManager", "What does a great manager look like to you?"],
      ["workedWellWithPrev", "What has worked well with previous managers?"],
      ["hasntWorked", "What hasn't worked?"],
    ],
  },
  {
    key: "currentWork",
    label: "Current Work",
    questions: [
      ["proudOf", "What projects are you most proud of?"],
      ["workingOnNow", "What are you working on now?"],
      ["roadmapConcerns", "What concerns you most about our roadmap?"],
      ["underutilized", "Where do you feel underutilized?"],
    ],
  },
];

const ONE_THING_KEY = "oneThingToKnow";

function today() {
  return new Date().toISOString().slice(0, 10);
}

function emptyAnswers() {
  const answers = { [ONE_THING_KEY]: "" };
  SECTIONS.forEach((section) => {
    answers[section.key] = {};
    section.questions.forEach(([field]) => {
      answers[section.key][field] = "";
    });
  });
  return answers;
}

function emptyForm() {
  return {
    meetingDate: today(),
    answers: emptyAnswers(),
  };
}

function countAnswered(answers) {
  if (!answers) return 0;
  let count = 0;
  SECTIONS.forEach((section) => {
    section.questions.forEach(([field]) => {
      if (answers?.[section.key]?.[field]?.trim()) count += 1;
    });
  });
  if (answers?.[ONE_THING_KEY]?.trim()) count += 1;
  return count;
}

function OnboardingProfilePanel({ profile, onSave }) {
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const [saving, setSaving] = useState(false);
  const [parsing, setParsing] = useState(false);
  const [error, setError] = useState("");
  const [parseNotice, setParseNotice] = useState("");

  const totalQuestions = useMemo(() => {
    let total = 1; // one thing
    SECTIONS.forEach((section) => {
      total += section.questions.length;
    });
    return total;
  }, []);

  const answered = countAnswered(profile?.answers);
  const hasProfile = Boolean(profile?.id);

  function startEdit() {
    setError("");
    setParseNotice("");
    setForm({
      meetingDate: profile?.meetingDate || today(),
      answers: { ...emptyAnswers(), ...(profile?.answers || {}) },
    });
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setError("");
    setParseNotice("");
  }

  function updateAnswer(sectionKey, field, value) {
    setForm((current) => ({
      ...current,
      answers: {
        ...current.answers,
        [sectionKey]: { ...current.answers[sectionKey], [field]: value },
      },
    }));
  }

  function updateOneThing(value) {
    setForm((current) => ({
      ...current,
      answers: { ...current.answers, [ONE_THING_KEY]: value },
    }));
  }

  async function handleParse(event) {
    const file = event.target.files?.[0];
    if (!file) return;
    setError("");
    setParseNotice("");
    setParsing(true);
    try {
      const parsed = await parseOnboardingFile(file);
      // Merge parsed values over the current form so the manager can review and
      // fix anything the tolerant parser mis-attributed before saving.
      setForm((current) => ({
        ...current,
        answers: { ...emptyAnswers(), ...current.answers, ...parsed },
      }));
      setParseNotice(
        `Imported "${file.name}". Review the fields below and save when ready.`,
      );
    } catch (err) {
      setError(err.message);
    } finally {
      setParsing(false);
      // Reset the input so the same file can be re-selected after an error.
      event.target.value = "";
    }
  }

  async function submit(event) {
    event.preventDefault();
    try {
      setSaving(true);
      setError("");
      await onSave({
        meetingDate: form.meetingDate,
        answers: form.answers,
      });
      closeForm();
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="onboarding-section">
      <div className="onboarding-heading">
        <div>
          <p className="profile-eyebrow">Initial 1:1 baseline</p>
          <h2>Onboarding Profile</h2>
          <p>
            Structured answers from the initial 1:1 — career direction, working
            style, and team perspective.
          </p>
        </div>
        <button type="button" onClick={startEdit}>
          {hasProfile ? <Pencil size={16} /> : <Plus size={16} />}
          {hasProfile ? "Edit / upload" : "Create / upload"}
        </button>
      </div>

      {!hasProfile ? (
        <p className="empty-state">
          No onboarding profile yet. Click <strong>Create / upload</strong> to
          fill it in manually, or upload your initial 1:1 notes (markdown/text)
          to parse and review the answers.
        </p>
      ) : (
        <div className="onboarding-summary">
          <div className="onboarding-meta">
            <span>Initial 1:1 on {profile.meetingDate || "—"}</span>
            <span>{answered}/{totalQuestions} answers captured</span>
          </div>
          {SECTIONS.map((section) => {
            const sectionAnswers = section.questions
              .map(([field, prompt]) => ({
                prompt,
                value: profile.answers?.[section.key]?.[field],
              }))
              .filter((entry) => entry.value?.trim());
            if (sectionAnswers.length === 0) return null;
            return (
              <div className="onboarding-group" key={section.key}>
                <h3>{section.label}</h3>
                {sectionAnswers.map(({ prompt, value }) => (
                  <div className="onboarding-qa" key={prompt}>
                    <p className="onboarding-question">{prompt}</p>
                    <p className="onboarding-answer">{value}</p>
                  </div>
                ))}
              </div>
            );
          })}
          {profile.answers?.[ONE_THING_KEY]?.trim() && (
            <div className="onboarding-group onboarding-one-thing">
              <h3>The one thing I should know</h3>
              <p className="onboarding-answer">{profile.answers[ONE_THING_KEY]}</p>
            </div>
          )}
        </div>
      )}

      <Dialog
        open={showForm}
        onClose={saving || parsing ? undefined : closeForm}
        fullWidth
        maxWidth="md"
        aria-label="Edit onboarding profile"
      >
        <form className="onboarding-form" onSubmit={submit}>
          <div className="onboarding-form-heading">
            <h3>{hasProfile ? "Edit onboarding profile" : "New onboarding profile"}</h3>
            <button
              type="button"
              className="icon-button"
              onClick={closeForm}
              aria-label="Close onboarding form"
            >
              <X size={17} />
            </button>
          </div>

          <div className="onboarding-upload onboarding-field-wide">
            <Upload size={16} />
            <label className="onboarding-upload-label">
              {parsing ? "Parsing…" : "Upload initial 1:1 notes (.md / .txt) to auto-fill"}
              <input
                type="file"
                accept=".md,.markdown,.txt,text/plain"
                onChange={handleParse}
                disabled={parsing || saving}
              />
            </label>
          </div>
          {parseNotice && <p className="onboarding-parse-notice onboarding-field-wide">{parseNotice}</p>}

          <label className="onboarding-field-wide">
            Initial 1:1 date
            <input
              type="date"
              name="meetingDate"
              value={form.meetingDate}
              onChange={(event) =>
                setForm((current) => ({ ...current, meetingDate: event.target.value }))
              }
            />
          </label>

          {SECTIONS.map((section) => (
            <div className="onboarding-form-group onboarding-field-wide" key={section.key}>
              <h4>{section.label}</h4>
              {section.questions.map(([field, prompt]) => (
                <label key={field}>
                  {prompt}
                  <textarea
                    rows="2"
                    value={form.answers[section.key]?.[field] || ""}
                    onChange={(event) =>
                      updateAnswer(section.key, field, event.target.value)
                    }
                  />
                </label>
              ))}
            </div>
          ))}

          <div className="onboarding-form-group onboarding-field-wide">
            <h4>One thing I should know</h4>
            <label>
              What is the one thing I should know about you that will help me be a better manager?
              <textarea
                rows="3"
                value={form.answers[ONE_THING_KEY] || ""}
                onChange={(event) => updateOneThing(event.target.value)}
              />
            </label>
          </div>

          {error && <div className="error onboarding-field-wide">Error: {error}</div>}

          <div className="form-actions onboarding-field-wide">
            <button type="submit" disabled={saving || parsing}>
              {saving ? "Saving..." : hasProfile ? "Save changes" : "Save onboarding profile"}
            </button>
            <button type="button" className="secondary-button" onClick={closeForm}>
              Cancel
            </button>
          </div>
        </form>
      </Dialog>
    </section>
  );
}

export default OnboardingProfilePanel;
