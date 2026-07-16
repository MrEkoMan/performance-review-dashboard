import { useEffect, useState } from "react";
import { getEngineers, getNotes, deleteNote, updateNote } from "../api/performanceAPI";

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
    const [noteToEdit, setNoteToEdit] = useState(null);

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

            setNotes(Array.isArray(data) ? data : []);
        } catch (err) {
            console.error("loadNotes failed: err");
            setError(err.message);
            setNotes([]);
        } finally {
            setLoading(false);
        }
    }

    async function handleDeleteNote(id) {
        const confirmed = window.confirm(
            "Are you sure you want to delete this note?"
        );

        if (!confirmed) {
            return;
        }

        try {
            await deleteNote(id);
            await loadNotes();
        } catch (err) {
            console.error("deleteNote failed:", err);
            setError(err.message);
        }
    }

    function handleEditNote(note) {
        setNoteToEdit(note);
    }

    async function handleUpdateNote(note) {
        try {
            await updateNote(note.id, {
                engineerId: Number(note.engineerId),
                noteDate: note.noteDate,
                category: note.category,
                summary: note.summary,
                details: note.details,
                impact: note.impact,
                followUpNeeded: note.followUpNeeded,
                reviewCycle: note.reviewCycle,
            });

            setNoteToEdit(null);
            await loadNotes();
        } catch (err) {
            console.error("updateNote failed:", err);
            setError(err.message);
        }
    }

    function handleCancelEdit() {
        setNoteToEdit(null);
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
                <AddNoteForm 
                    engineers={engineers} 
                    noteToEdit={noteToEdit}
                    onNoteCreated={loadNotes}
                    onNoteUpdated={handleUpdateNote}
                    onCancelEdit={handleCancelEdit} 
                />
            </div>

            <h2>Performance Evidence</h2>

            <NotesTable 
                notes={notes} 
                loading={loading} 
                onDelete={handleDeleteNote}
                onEdit={handleEditNote}
            />
        </div>
    );
    /*return <h1>Dashboard is rendering</h1>*/
}

export default Dashboard;
