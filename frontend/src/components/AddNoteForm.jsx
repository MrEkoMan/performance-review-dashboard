import { useState } from "react";
import { createNote } from "../api/performanceAPI";

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

function AddNoteForm({ engineers, onNoteCreated }) {
    const reviewCycles = getReviewCycles();

    const [form, setForm] = useState({
        engineerId: "",
        noteDate: new Date().toISOString().slice(0, 10),
        category: "Business Impact",
        summary: "",
        details: "",
        impact: "",
        followUpNeeded: false,
        reviewCycle: reviewCycles[0],
    });

    const [saving, setSaving] = useState(false);
    const [error, setError] = useState("");

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

        try {
            await createNote({
                ...form,
                engineerId: Number(form.engineerId),
            });

            setForm({
                engineerId: "",
                noteDate: new Date().toISOString().slice(0, 10),
                category: "Business Impact",
                summary: "",
                details: "",
                impact: "",
                followUpNeeded: false,
                reviewCycle: reviewCycles[0],
            });

            if (onNoteCreated) {
                onNoteCreated();
            }
        } catch (err) {
            setError(err.message);
        } finally {
            setSaving(false);
        }
    }

    return (
        <form className="form-card" onSubmit={handleSubmit}>
            <h2>Add Performance Note</h2>

            {error && <div className="error">Error: {error}</div>}

            <label>Engineer</label>
            <select
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

            <label>Date</label>
            <input
                type="date"
                name="noteDate"
                value={form.noteDate}
                onChange={handleChange}
                required
            />

            <label>Category</label>
            <select
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

            <label>Summary</label>
            <input
                name="summary"
                value={form.summary}
                onChange={handleChange}
                required
            />

            <label>Details</label>
            <textarea
                name="details"
                value={form.details}
                onChange={handleChange}
            />

            <label>Impact</label>
            <textarea
                name="impact"
                value={form.impact}
                onChange={handleChange}
            />

            <label>Review Cycle</label>
            <select
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

            <button type="submit" disabled={saving || engineers.length === 0}>
                {saving ? "Saving..." : "Add Note"}
            </button>
        </form>
    );
}

export default AddNoteForm;