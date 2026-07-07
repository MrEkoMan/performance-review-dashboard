function MetricCard({ title, value }) {
    return (
        <div className="metric-card">
            <div className="metric-title">{title}</div>
            <div className="metric-value">{value}</div>
        </div>
    );
}

function Metrics({ notes }) {
    const totalNotes = notes.length;

    const growthAreas = notes.filter(
        (note) => note.category === "Growth Area"
    ).length;

    const followUps = notes.filter(
        (note) => note.followUpNeeded
    ).length;

    const businessImpact = notes.filter(
        (note) => note.category === "Business Impact"
    ).length;

    const technicalExcellence = notes.filter(
        (note) => note.category === "Technical Excellence"
    ).length;

    return (
        <div className="metrics">
            <MetricCard
                title="Total Notes"
                value={totalNotes}
            />

            <MetricCard
                title="Business Impact"
                value={businessImpact}
            />

            <MetricCard
                title="Technical Excellence"
                value={technicalExcellence}
            />

            <MetricCard
                title="Growth Areas"
                value={growthAreas}
            />

            <MetricCard
                title="Follow-Ups"
                value={followUps}
            />
        </div>
    );
}

export default Metrics;