import { useState, useSyncExternalStore } from "react";
import { createEngineer } from "../api/performanceAPI";

function AddEngineerForm({ onEngineerCreated }) {
const reviewCycles = getReviewCycles();

    const [form, setForm] = useState({
        name: "",
        role: "",
        level: "",
        team: "",
        careerGoal: "",
        reviewCycle: reviewCycles[0],
    });

    const [saving, setSaving] = useState(false);
    const [error, setError] = useState("");

    function handleChange(event) {
        const { name, value } = event.target;

        setForm((current) => ({
            ...current,
            [name]: value,
        }));
    }

    async function handleSubmit(event) {
        event.preventDefault();
        setError("");
        setSaving(true);

        try {
            await createEngineer(form);

            setForm({
                name: "",
                role: "",
                level: "",
                team: "",
                careerGoal: "",
                reviewCycle: reviewCycles[0],
            });

            if (onEngineerCreated) {
                onEngineerCreated();
            }
        } catch (err) {
            setError(err.message);
        } finally {
            setSaving(false);
        }
    }

    return (
        <form className="form-card" onSubmit={handleSubmit}>
            <h2>Add Engineer</h2>

            {error && <div className="error">Error: {error}</div>}

            <label>Name</label>
            <input 
                name="name"
                value={form.name}
                onChange={handleChange}
                required
            />

            <label>Role</label>
            <input 
                name="role"
                value={form.role}
                onChange={handleChange}
            />

            <label>Level</label>
            <input
                name="level"
                value={form.level}
                onChange={handleChange}
            />

            <label>Team</label>
            <input
                name="team"
                value={form.team}
                onChange={handleChange}
            />

            <label>Career Goal</label>
            <input
                name="careerGoal"
                value={form.careerGoal}
                onChange={handleChange}
            />

            <label>Review Cycle</label>
            <input
                name="reviewCycle"
                value={form.reviewCycle}
                onChange={handleChange}
            />

            <button type="submit" disabled={saving}>
                {saving ? "Saving..." : "Add Engineer"}
            </button>
        </form>
    );
}

function getReviewCycles() {
    const currentYear = new Date().getFullYear();

    const cycles = []

    for (let year = currentYear - 1; year <= currentYear + 2; year++) {
        cycles.push(`${year} H1`);
        cycles.push(`${year} H2`);
    }

    return cycles;
}

export default AddEngineerForm;