import { useEffect, useState } from "react";
import { createEngineer, getReviewPeriods } from "../api/performanceApi";

function AddEngineerForm({ onEngineerCreated }) {
    const [reviewCycles, setReviewCycles] = useState([]);
    const [form, setForm] = useState({
        name: "",
        role: "",
        level: "",
        team: "",
        careerGoal: "",
        reviewCycle: "",
    });

    const [saving, setSaving] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        let cancelled = false;
        async function loadReviewPeriods() {
            try {
                const periods = await getReviewPeriods();
                if (!cancelled && Array.isArray(periods)) {
                    const labels = periods.map((period) => period.label);
                    setReviewCycles(labels);
                    setForm((current) => ({
                        ...current,
                        reviewCycle: current.reviewCycle || labels[0] || "",
                    }));
                }
            } catch (err) {
                setError(err.message);
            }
        }
        loadReviewPeriods();
        return () => {
            cancelled = true;
        };
    }, []);

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
                reviewCycle: reviewCycles[0] || "",
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
            <select
                name="reviewCycle"
                value={form.reviewCycle}
                onChange={handleChange}
                required
            >
                <option value="">Select review cycle</option>
                {reviewCycles.map((cycle) => (
                    <option value={cycle} key={cycle}>{cycle}</option>
                ))}
            </select>

            <button type="submit" disabled={saving}>
                {saving ? "Saving..." : "Add Engineer"}
            </button>
        </form>
    );
}

export default AddEngineerForm;
