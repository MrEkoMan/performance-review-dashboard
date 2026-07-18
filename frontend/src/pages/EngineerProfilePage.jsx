import { Link, useParams } from "react-router-dom";
import { useEffect, useState } from "react";
import { ArrowLeft } from "lucide-react";

import {
    deleteNote,
    getEngineers,
    getNotes,
    updateNote,
} from "../api/performanceApi.js";

import AddNoteForm from "../components/AddNoteForm.jsx";
import Metrics from "../components/Metrics.jsx";
import NotesTable from "../components/NotesTables.jsx";

function EngineerProfilePage() {
    const { engineerId } = useParams();

    const [engineers, setEngineers] = useState([]);
    const [engineer, setEngineer] = useState(null);
    const [notes, setNotes] = useState([]);
    const [noteToEdit, setNoteToEdit] = useState(null);

    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    async function loadProfile() {
        try {
            setLoading(true);
            setError("");

            const [engineerData, noteData] = await Promise.all([
                getEngineers(),
                getNotes(engineerId),
            ]);

            const safeEngineers = Array.isArray(engineerData) ? engineerData : [];
            const safeNotes = Array.isArray(noteData) ? noteData : [];
            const matchingEngineer = safeEngineers.find((item) => String(item.id) === String(engineerId));

            setEngineers(safeEngineers);
            setEngineer(matchingEngineer ?? null);
            setNotes(safeNotes);
        } catch (err) {
            console.error("Failed to load engineer profile:", err);
            setError(err.message);
        } finally {
            console.log("Setting loading to false");
            setLoading(false);
        }
    }

    useEffect(() => {
        loadProfile();
    }, [engineerId]);

    console.log("EngineerProfile render", {
        loading,
        engineer,
        notesCount: notes.length,
    });

    function handleEditNote(note) {
        setNoteToEdit(note);

        window.scrollTo({
            top: 0,
            behavior: "smooth",
        });
    }

    function handleCancelEdit() {
        setNoteToEdit(null);
    }

    async function handleUpdateNote(note) {
        try {
            setError("");

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
            await loadProfile();
        } catch (err) {
            console.error("Failed to update note:", err);
            setError(err.message);
            throw err;
        }
    }

    async function handleDeleteNote(noteId) {
        const confirmed = window.confirm("Are you sure you want to delete this note?");

        if (!confirmed) {
            return;
        }

        try {
            setError("");

            await deleteNote(noteId);
            if (noteToEdit?.id === noteId) {
                setNoteToEdit(null);
            }

            await loadProfile();
        } catch (err) {
            console.error("Failed to delete note:", err);
            setError(err.message);
        }
    }

    if (loading) {
        return (
            <main>
                <p>Loading engineer profiles...</p>
            </main>
        );
    }

    if (!engineer) {
        return (
            <main>
                <Link to="/" className="back-link">
                    <ArrowLeft size={18} />
                    Back to Dashboard
                </Link>

                <div className="error">
                    Engineer with ID {engineerId} was not found.
                </div>
            </main>
        );
    }

    const safeNotes = Array.isArray(notes) ? notes : [];
    const followUpCount = safeNotes.filter((note) => note.followUpNeeded).length;

    const latestNote = [...safeNotes]
        .filter((note) => note.notedate)
        .sort(
            (first, second) =>
                new Date(second.noteDate) - new Date(first.noteDate)
        )[0];

    return (
        <main>
            <section className="engineer-profile">
                <Link to="/" className="back-link">
                    <ArrowLeft size={18} />
                    Back to Dashboard
                </Link>

                {error && (
                    <div className="error">
                        Error: {error}
                    </div>
                )}

                <header className="profile-header">
                    <div>
                        <p className="profile-eyebrow">
                            Engineer Profile
                        </p>

                        <h1>{engineer.name}</h1>

                        <p className="profile-subtitle">
                            {[
                                engineer.role,
                                engineer.level,
                                engineer.team,
                            ]
                                .filter(Boolean)
                                .join(" . ") || "Engineer details not recorded"}
                        </p>
                    </div>

                    <div className="profile-review-cycle">
                        <span>Review Cycle</span>
                        <strong>
                            {engineer.reviewCycle || "Not assigned"}
                        </strong>
                    </div>
                </header>

                <div className="profile-details-grid">
                    <article className="profile-card">
                        <h2>Career Goal</h2>

                        <p>
                            {engineer.careerGoal || "No career goal has been recorded."}
                        </p>
                    </article>

                    <article className="profile-card">
                        <h2>Profile Summary</h2>

                        <dl className="profile-summary-list">
                            <div>
                                <dt>Total Evidence</dt>
                                <dd>{safeNotes.length}</dd>
                            </div>

                            <div>
                                <dt>Open Follow-Ups</dt>
                                <dd>{followUpCount}</dd>
                            </div>

                            <div>
                                <dt>Most Recent Note</dt>
                                <dd>
                                    {latestNote?.noteDate || "No notes yet"}
                                </dd>
                            </div>
                        </dl>
                    </article>
                </div>

                <Metrics notes={safeNotes} />

                <div className="profile-content-grid">
                    <AddNoteForm
                        engineers={engineers}
                        noteToEdit={noteToEdit}
                        onNoteCreated={loadProfile}
                        onNoteUpdated={handleUpdateNote}
                        onCancelEdit={handleCancelEdit} />

                    <article className="profile-card-profile-guidance">
                        <h2>Manager Focus</h2>

                        <p>
                            Use this profile to capture accomplishments,
                            coaching themes, operational contributions,
                            recognition, and career-development evidence
                            throughout the review cycle.
                        </p>

                        {followUpCount > 0 ? (
                            <p className="profile-attention">
                                {followUpCount} open follow-up
                                {followUpCount === 1 ? "" : "s"} require attentionn.
                            </p>
                        ) : (
                            <p>No open follow-ups.</p>
                        )}
                    </article>
                </div>

                <NotesTable 
                    notes={safeNotes}
                    loading={loading}
                    onEdit={handleEditNote}
                    onDelete={handleDeleteNote}
                />
            </section>
        </main>
    );
}

export default EngineerProfilePage;