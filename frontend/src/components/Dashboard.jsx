import { useEffect, useState } from "react";
import { getEngineers, getNotes, deleteNote, updateNote } from "../api/performanceAPI";

import EngineerFilter from "./EngineerFilter";
import EngineerProfile from "./EngineerProfile";
import Metrics from "./Metrics";
import AddNoteForm from "./AddNoteForm";
import AddEngineerForm from "./AddEngineerForm";
import NotesTable from "./NotesTables";

function Dashboard() {
    const [engineers, setEngineers] = useState([]);
    const [notes, setNotes] = useState([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");
    const [noteToEdit, setNoteToEdit] = useState(null);
    const [searchTerm, setSearchTerm] = useState("");
    const [profileEngineerId, setProfileEngineerId] = useState(null); 
    const [currentView, setCurrentView] = useState("dashbaord");
    const [selectedEngineer, setSelectedEngineer] = useState(null);

    const profileEngineer = engineers.find(
        (engineer) => 
            String(engineer.id) === String(profileEngineerId)
    );

    const safeNotes = Array.isArray(notes) ? notes : [];

    const profileNotes = safeNotes.filter(
        (note) =>
            String(note.engineerId) === String(profileEngineerId)
    );

    function handleViewProfile() {
        if (!selectedEngineer) {
            return;
        }

        setProfileEngineerId(selectedEngineer);
    }

    function handleCloseProfile() {
        setProfileEngineerId(null);
    }

    async function loadEngineers() {
        try {
            const data = await getEngineers();
            setEngineers(data);
        } catch (err) {
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
        <div className="dashboard">
            <h1>Engineer Manager Dashboard</h1>

            {error && <div className="error">Error: {error}</div>}
            <EngineerFilter
                engineers={engineers}
                selectedEngineer={selectedEngineer}
                onEngineerChange={setSelectedEngineer}
                onViewProfile={handleViewProfile}
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
