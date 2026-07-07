function EngineerFilter({ engineers, selectedEngineer, onEngineerChange }) {
    return (
        <div className="toolbar">
            <label htmlFor="engineer-filter">Engineer</label>

            <select
                id="engineer-filter"
                value={selectedEngineer}
                onChange={(e) => onEngineerChange(e.target.value)}
            >
                <option value="">All Engineers</option>

                {engineers.map((engineer) => (
                    <option key={engineer.id} value={engineer.id}>
                        {engineer.name}
                    </option>
                ))}
            </select>
        </div>
    );
}

export default EngineerFilter;