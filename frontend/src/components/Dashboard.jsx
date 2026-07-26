import { useEffect, useState } from "react";
import { Settings } from "lucide-react";
import { Link } from "react-router-dom";
import {
    getDashboardAttention,
    getEngineers,
    getNotes,
    deleteNote,
    updateNote,
} from "../api/performanceApi";

import EngineerFilter from "./EngineerFilter";
import Metrics from "./Metrics";
import AddNoteForm from "./AddNoteForm";
import AddEngineerForm from "./AddEngineerForm";
import NotesTable from "./NotesTables";
import NeedsAttentionPanel from "./NeedsAttentionPanel";

function Dashboard() {
    const [engineers, setEngineers] = useState([]);
    const [notes, setNotes] = useState([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");
    const [noteToEdit, setNoteToEdit] = useState(null);
    const [searchTerm, setSearchTerm] = useState("");
    const [selectedEngineer, setSelectedEngineer] = useState(null);
    const [attentionItems, setAttentionItems] = useState([]);

    async function loadEngineers() {
        try {
            const data = await getEngineers();
            setEngineers(data);
        } catch (err) {
            console.error("loadEngineers failed:", err);
            setError(err.message);
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

    async function loadAttention() {
        try {
            const data = await getDashboardAttention();
            setAttentionItems(Array.isArray(data) ? data : []);
        } catch (err) {
            console.error("loadAttention failed:", err);
            setError(err.message);
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
            await Promise.all([loadNotes(), loadAttention()]);
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
            await Promise.all([loadNotes(), loadAttention()]);
        } catch (err) {
            console.error("updateNote failed:", err);
            setError(err.message);
        }
    }

    async function handleNoteCreated() {
        await Promise.all([loadNotes(), loadAttention()]);
    }

    async function handleEngineerCreated() {
        await Promise.all([loadEngineers(), loadAttention()]);
    }

    function handleCancelEdit() {
        setNoteToEdit(null);
    }

    useEffect(() => {
        // Initial API synchronization.
        loadEngineers();
        loadAttention();
    }, []);

    useEffect(() => {
        // Refresh the evidence list when the active engineer changes.
        loadNotes();
        // loadNotes intentionally uses the current selectedEngineer value.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [selectedEngineer]);
    
    const filteredNotes = notes.filter((note) => {
        const searchValue = searchTerm.trim().toLowerCase();

        if (!searchValue) {
            return true;
        }

        return [
            note.engineerName,
            note.category,
            note.summary,
            note.details,
            note.impact,
            note.reviewCycle,
        ].some((value) => 
            String(value ?? "")
                .toLowerCase()
                .includes(searchValue)
        );
    });

    return (
        <div className="dashboard-header">
            <div>
                <h1>Engineer Manager Dashboard</h1>
                <p>Track performance evidence through the review cycle.</p>
            </div>

            <Link
                to="/settings"
                className="icon-button"
                title="Settings"
                aria-label="Settings"
            >
                <Settings size={18} />
            </Link>

            {error && <div className="error">Error: {error}</div>}
            <NeedsAttentionPanel items={attentionItems} />
            <EngineerFilter
                engineers={engineers}
                selectedEngineer={selectedEngineer}
                onEngineerChange={setSelectedEngineer}
            />

            <div className="search-container">
                <label htmlFor="notes-search">Search Notes</label>

                <input 
                    id="note-search"
                    type="search"
                    value={searchTerm}
                    onChange={(event) => setSearchTerm(event.target.value)}
                    placeholder="Search summary, impact, category..."
                />
            </div>

            <Metrics notes={filteredNotes} />

            <div className="forms-container">
                <AddEngineerForm onEngineerCreated={handleEngineerCreated} />
                <AddNoteForm 
                    engineers={engineers} 
                    noteToEdit={noteToEdit}
                    onNoteCreated={handleNoteCreated}
                    onNoteUpdated={handleUpdateNote}
                    onCancelEdit={handleCancelEdit} 
                />
            </div>

            <h2>Performance Evidence</h2>

            <p className="results-count">
                Showing {filteredNotes.length} of {notes.length} notes
            </p>
            <NotesTable 
                notes={filteredNotes} 
                loading={loading} 
                onDelete={handleDeleteNote}
                onEdit={handleEditNote}
            />
        </div>
    );
}

export default Dashboard;
