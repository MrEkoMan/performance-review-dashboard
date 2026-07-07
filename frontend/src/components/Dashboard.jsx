import { useEffect, useState } from "react";
import { getEngineers, getNotes } from "../api/performanceAPI";

import EngineerFilter from "./EngineerFilter";
import Metrics from "./Metrics";
import AddNoteForm from "./AddNoteForm";
import AddEngineerForm from "./AddEngineerForm";
import NotesTable from "./NotesTables";

function Dashboard() {
    const [engineers, setEngineers] = useState([]);
    const [notes, setNotes] = useState([]);
    const [selectedEngineer, setSelectedEngineer] = useState("");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");

    async function loadEngineers() {
        try {
            const data = await getEngineers();
            setEngineers(data);
        } catch (err) {
            /*setError(err.message);*/
            console.error("loadEngineers failed:", err);
            setError(err.messgae);
        }
    }

    async function loadNotes() {
        try {
            setLoading(true);

            const data = await getNotes(selectedEngineer);
            setNotes(data);
        } catch (err) {
            console.error("loadNotes failed: err");
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }

    useEffect(() => {
        loadEngineers();
    }, []);

    useEffect(() => {
        loadNotes();
    }, [selectedEngineer]);
    

    return (
        <div className="dashboard">
            <h1>Engineer Manager Dashboard</h1>

            {error && <div className="error">Error: {error}</div>}
            <EngineerFilter
                engineers={engineers}
                selectedEngineer={selectedEngineer}
                onEngineerChange={setSelectedEngineer}
            />

            <Metrics notes={notes} />

            <div className="forms-container">
                <AddEngineerForm onEngineerCreated={loadEngineers} />
                <AddNoteForm engineers={engineers} onNoteCreated={loadNotes} />
            </div>

            <h2>Performance Evidence</h2>

            <NotesTable notes={notes} loading={loading} />
        </div>
    );
    /*return <h1>Dashboard is rendering</h1>*/
}

export default Dashboard;
