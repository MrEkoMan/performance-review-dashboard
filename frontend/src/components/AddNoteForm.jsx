import { useEffect, useMemo, useState } from "react";
import { createNote } from "../api/performanceAPI";
import { NotepadText } from "lucide-react";

const categories = [
    "Business Impact",
    "Technical Excellence",
    "Operational Excellence",
    "Team Contribution",
    "Growth Area",
    "Career Development",
    "Feedback Received",
];

function getReviewCycles() {
    const currentYear = new Date().getFullYear();
    const cycles = [];

    for (let year = currentYear - 1; year <= currentYear + 2; year++) {
        cycles.push(`${year} H1`);
        cycles.push(`${year} H2`);
    }

    return cycles;
}

function getToday() {
    return new Date().toISOString().slice(0, 10);
}

function AddNoteForm({ 
    engineers, 
    onNoteCreated ,
    noteToEdit,
    onNoteUpdated,
    onEditComplete,
    onCancelEdit,
}) {
    const reviewCycles = useMemo(() => getReviewCycles(), []);

    const createEmptyForm = () => ({
        engineerId: "",
        noteDate: getToday(),
        category: "Business Impact",
        summary: "",
        details: "",
        impact: "",
        followUpNeeded: false,
        reviewCycle: reviewCycles[0],
    });

    const [form, setForm] = useState(createEmptyForm);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState("");

    const isEditing = Boolean(noteToEdit);

    useEffect(() => {
        if (noteToEdit) {
            setForm({
                engineerId: String(noteToEdit.engineerId ?? ""),
                noteDate: noteToEdit.noteDate ?? getToday(),
                category: noteToEdit.category ?? "Business Impact",
                summary: noteToEdit.summary ?? "",
                details: noteToEdit.details ?? "",
                impact: noteToEdit.impact ?? "",
                followUpNeeded: Boolean(noteToEdit.followUpNeeded),
                reviewCycle: noteToEdit.reviewCycle ?? reviewCycles[0],
            });
        } else {
            setForm(createEmptyForm());
        }

        setError("");
    }, [noteToEdit]);

    function handleChange(event) {
        const { name, value, type, checked } = event.target;

        setForm((current) => ({
            ...current,
            [name]: type === "checkbox" ? checked : value,
        }));
    }

    async function handleSubmit(event) {
        event.preventDefault();
        setError("")
        setSaving(true);

        const notePayload = {
            ...form,
            engineerId: Number(form.engineerId),
        };

        console.log("Saving note payload:", notePayload);
        
        try {
            if (isEditing) {
                await onNoteUpdated({
                    ...notePayload,
                    id: noteToEdit.id,
                });
            } else {
                await createNote(notePayload);

                setForm(createEmptyForm());

                if (onNoteCreated) {
                    await onNoteCreated();
                }
            }
        } catch (err) {
            setError(err.message);
        } finally {
            setSaving(false);
        }
    }

    function handleCancel() {
        setForm(createEmptyForm());

        if (onCancelEdit) {
            onCancelEdit();
        }
    }

    return (
        <form className="form-card" onSubmit={handleSubmit}>
            <h2>{isEditing ? "Edit Performance Note" : "Add Performance Note"}</h2>

            {error && <div className="error">Error: {error}</div>}

            <label htmlFor="note-engineer">Engineer</label>
            <select
                id="note-engineer"
                name="engineerId"
                value={form.engineerId}
                onChange={handleChange}
                required
            >
                <option value="">Select engineer</option>
                {engineers.map((engineer) => (
                    <option key={engineer.id} value={engineer.id}>
                        {engineer.name}
                    </option>
                ))}
            </select>

            <label htmlFor="note-date">Date</label>
            <input
                id="note-date"
                type="date"
                name="noteDate"
                value={form.noteDate}
                onChange={handleChange}
                required
            />

            <label htmlFor="note-category">Category</label>
            <select
                id="note-category"
                name="category"
                value={form.category}
                onChange={handleChange}
            >
                {categories.map((category) => (
                    <option key={category} value={category}>
                        {category}
                    </option>
                ))}
            </select>

            <label htmlFor="note-summary">Summary</label>
            <input
                id="note-summary"
                name="summary"
                value={form.summary}
                onChange={handleChange}
                required
            />

            <label htmlFor="note-details">Details</label>
            <textarea
                id="note-details"
                name="details"
                value={form.details}
                onChange={handleChange}
            />

            <label htmlFor="note-impact">Impact</label>
            <textarea
                id="note-impact"
                name="impact"
                value={form.impact}
                onChange={handleChange}
            />

            <label htmlFor="note-review-cycle">Review Cycle</label>
            <select
                id="note-review-cycle"
                name="reviewCycle"
                value={form.reviewCycle}
                onChange={handleChange}
            >
                {reviewCycles.map((cycle) => (
                    <option key={cycle} value={cycle}>
                        {cycle}
                    </option>
                ))}
            </select>

            <label className="checkbox-row">
                <input
                    type="checkbox"
                    name="followUpNeeded"
                    checked={form.followUpNeeded}
                    onChange={handleChange}
                />
                Follow-up Needed
            </label>

            <div className="form-actions">
                <button type="submit" disabled={saving || engineers.length === 0}>
                    {saving
                        ? "Saving..."
                        : isEditing
                            ? "Save Changes"
                            : "Add Note"}
                </button>

                {isEditing && (
                    <button
                        type="button"
                        className="secondary-button"
                        onClick={handleCancel}
                        disabled={saving}
                    >
                        Cancel
                    </button>
                )}
            </div>
        </form>
    );
}

export default AddNoteForm;