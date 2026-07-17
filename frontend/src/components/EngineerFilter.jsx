import { useNavigate } from "react-router-dom";

function EngineerFilter({ 
    engineers, 
    selectedEngineer, 
    onEngineerChange, 
}) {
    const navigate = useNavigate();

    function handleViewProfile() {
        if (!selectedEngineer) {
            return;
        }

        navigate(`/engineers/${selectedEngineer}`);
    }

    return (
        <div className="toolbar">
            <div className="filter-field">
                <label htmlFor="engineer-filter">Engineer</label>

                <select
                    id="engineer-filter"
                    value={selectedEngineer ?? ""}
                    onChange={(event) => onEngineerChange(event.target.value)}
                >
                    <option value="">All Engineers</option>

                    {engineers.map((engineer) => (
                        <option key={engineer.id} value={engineer.id}>
                            {engineer.name}
                        </option>
                    ))}
                </select>
            </div>

            <button
                type="button"
                className="secondary-button view-profile-button"
                onClick={handleViewProfile}
                disabled={!selectedEngineer}>
                    View Profile
            </button>
        </div>
    );
}

export default EngineerFilter;