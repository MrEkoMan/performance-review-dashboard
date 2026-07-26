import { Link, useParams } from "react-router-dom";
import { useEffect, useState } from "react";
import { ArrowLeft } from "lucide-react";
import { Box, Tab, Tabs } from "@mui/material";

import {
    deleteNote,
    createGoal,
    createOneOnOne,
    deleteGoal,
    deleteOneOnOne,
    createFollowUp,
    createRecognition,
    deleteFollowUp,
    deleteRecognition,
    getEngineers,
    getGoals,
    getNotes,
    getOneOnOnes,
    getFollowUps,
    getRecognitions,
    getTimeline,
    updateGoal,
    updateNote,
    updateOneOnOne,
    updateFollowUp,
    updateRecognition,
} from "../api/performanceApi.js";

import AddNoteForm from "../components/AddNoteForm.jsx";
import Metrics from "../components/Metrics.jsx";
import NotesTable from "../components/NotesTables.jsx";
import GoalsPanel from "../components/GoalsPanel.jsx";
import OneOnOnesPanel from "../components/OneOnOnesPanel.jsx";
import FollowUpsPanel from "../components/FollowUpsPanel.jsx";
import RecognitionPanel from "../components/RecognitionPanel.jsx";
import TimelinePanel from "../components/TimelinePanel.jsx";

function EngineerProfilePage() {
    const { engineerId } = useParams();

    const [engineers, setEngineers] = useState([]);
    const [engineer, setEngineer] = useState(null);
    const [notes, setNotes] = useState([]);
    const [goals, setGoals] = useState([]);
    const [oneOnOnes, setOneOnOnes] = useState([]);
    const [followUps, setFollowUps] = useState([]);
    const [recognitions, setRecognitions] = useState([]);
    const [timeline, setTimeline] = useState([]);
    const [noteToEdit, setNoteToEdit] = useState(null);
    const [activeTab, setActiveTab] = useState("overview");

    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    async function loadProfile() {
        try {
            setLoading(true);
            setError("");

            const [engineerData, noteData, goalData, oneOnOneData, followUpData, recognitionData, timelineData] = await Promise.all([
                getEngineers(),
                getNotes(engineerId),
                getGoals(engineerId),
                getOneOnOnes(engineerId),
                getFollowUps(engineerId),
                getRecognitions(engineerId),
                getTimeline(engineerId),
            ]);

            const safeEngineers = Array.isArray(engineerData) ? engineerData : [];
            const safeNotes = Array.isArray(noteData) ? noteData : [];
            const matchingEngineer = safeEngineers.find((item) => String(item.id) === String(engineerId));

            setEngineers(safeEngineers);
            setEngineer(matchingEngineer ?? null);
            setNotes(safeNotes);
            setGoals(Array.isArray(goalData) ? goalData : []);
            setOneOnOnes(Array.isArray(oneOnOneData) ? oneOnOneData : []);
            setFollowUps(Array.isArray(followUpData) ? followUpData : []);
            setRecognitions(Array.isArray(recognitionData) ? recognitionData : []);
            setTimeline(Array.isArray(timelineData) ? timelineData : []);
        } catch (err) {
            console.error("Failed to load engineer profile:", err);
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }

    useEffect(() => {
        // Synchronize the route parameter with its API-backed profile.
        loadProfile();
        // loadProfile intentionally closes over engineerId.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [engineerId]);

    function handleEditNote(note) {
        setNoteToEdit(note);

        window.scrollTo({
            top: 0,
            behavior: "smooth",
        });
    }

    async function handleNoteCreated() {
        await loadProfile();
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

    async function handleCreateGoal(goal) {
        await createGoal(engineerId, goal);
        await loadProfile();
    }

    async function handleUpdateGoal(goalId, goal) {
        await updateGoal(goalId, goal);
        await loadProfile();
    }

    async function handleDeleteGoal(goalId) {
        await deleteGoal(goalId);
        await loadProfile();
    }

    async function handleCreateOneOnOne(meeting) {
        await createOneOnOne(engineerId, meeting);
        await loadProfile();
    }

    async function handleUpdateOneOnOne(meetingId, meeting) {
        await updateOneOnOne(meetingId, meeting);
        await loadProfile();
    }

    async function handleDeleteOneOnOne(meetingId) {
        await deleteOneOnOne(meetingId);
        await loadProfile();
    }

    async function handleCreateFollowUp(followUp) {
        await createFollowUp(engineerId, followUp);
        await loadProfile();
    }

    async function handleUpdateFollowUp(followUpId, followUp) {
        await updateFollowUp(followUpId, followUp);
        await loadProfile();
    }

    async function handleDeleteFollowUp(followUpId) {
        await deleteFollowUp(followUpId);
        await loadProfile();
    }

    async function handleCreateRecognition(recognition) {
        await createRecognition(engineerId, recognition);
        await loadProfile();
    }

    async function handleUpdateRecognition(recognitionId, recognition) {
        await updateRecognition(recognitionId, recognition);
        await loadProfile();
    }

    async function handleDeleteRecognition(recognitionId) {
        await deleteRecognition(recognitionId);
        await loadProfile();
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
    const openStructuredFollowUps = followUps.filter((item) =>
        ["open", "in_progress"].includes(item.status)
    );
    const overdueStructuredFollowUps = openStructuredFollowUps.filter(
        (item) => item.dueDate && item.dueDate < new Date().toISOString().slice(0, 10)
    );

    const latestNote = [...safeNotes]
        .filter((note) => note.noteDate)
        .sort(
            (first, second) =>
                new Date(second.noteDate) - new Date(first.noteDate)
        )[0];
    const latestRecognition = recognitions[0];

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

                <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
                    <Tabs
                        value={activeTab}
                        onChange={(_, value) => setActiveTab(value)}
                        variant="scrollable"
                        scrollButtons="auto"
                        aria-label="Engineer profile sections"
                    >
                        <Tab value="overview" label="Overview" />
                        <Tab value="evidence" label={`Evidence (${safeNotes.length})`} />
                        <Tab value="goals" label={`Goals (${goals.length})`} />
                        <Tab value="one-on-ones" label={`1:1s (${oneOnOnes.length})`} />
                        <Tab value="follow-ups" label={`Follow-ups (${followUps.length})`} />
                        <Tab value="recognition" label={`Recognition (${recognitions.length})`} />
                        <Tab value="timeline" label="Timeline" />
                    </Tabs>
                </Box>

                {activeTab === "overview" && (
                    <>
                        <div className="profile-details-grid">
                            <article className="profile-card">
                                <h2>Career Goal</h2>
                                <p>{engineer.careerGoal || "No career goal has been recorded."}</p>
                            </article>
                            <article className="profile-card">
                                <h2>Profile Summary</h2>
                                <dl className="profile-summary-list">
                                    <div><dt>Total Evidence</dt><dd>{safeNotes.length}</dd></div>
                                    <div><dt>Open Follow-Ups</dt><dd>{followUpCount}</dd></div>
                                    <div><dt>Open Action Items</dt><dd>{openStructuredFollowUps.length}</dd></div>
                                    <div><dt>Overdue Actions</dt><dd>{overdueStructuredFollowUps.length}</dd></div>
                                    <div><dt>Recognition</dt><dd>{recognitions.length}</dd></div>
                                    <div><dt>Most Recent Recognition</dt><dd>{latestRecognition?.recognitionDate || "None yet"}</dd></div>
                                    <div><dt>Most Recent Note</dt><dd>{latestNote?.noteDate || "No notes yet"}</dd></div>
                                </dl>
                            </article>
                        </div>
                        <Metrics notes={safeNotes} />
                        <article className="profile-card">
                            <h2>Manager Focus</h2>
                            <p>Capture accomplishments, coaching themes, operational contributions, recognition, and career-development evidence throughout the review cycle.</p>
                            {overdueStructuredFollowUps.length > 0 ? (
                                <p className="profile-attention">
                                    {overdueStructuredFollowUps.length} structured action
                                    {overdueStructuredFollowUps.length === 1 ? "" : "s"} overdue.
                                </p>
                            ) : followUpCount > 0 ? (
                                <p className="profile-attention">
                                    {followUpCount} open follow-up{followUpCount === 1 ? "" : "s"} require attention.
                                </p>
                            ) : <p>No open follow-ups.</p>}
                        </article>
                    </>
                )}

                {activeTab === "evidence" && (
                    <>
                        <AddNoteForm
                            engineers={engineers}
                            noteToEdit={noteToEdit}
                            onNoteCreated={handleNoteCreated}
                            onNoteUpdated={handleUpdateNote}
                            onCancelEdit={handleCancelEdit}
                        />
                        <NotesTable
                            notes={safeNotes}
                            loading={loading}
                            onEdit={handleEditNote}
                            onDelete={handleDeleteNote}
                        />
                    </>
                )}

                {activeTab === "goals" && (
                    <GoalsPanel
                        goals={goals}
                        reviewCycle={engineer.reviewCycle}
                        onCreate={handleCreateGoal}
                        onUpdate={handleUpdateGoal}
                        onDelete={handleDeleteGoal}
                    />
                )}

                {activeTab === "one-on-ones" && (
                    <OneOnOnesPanel
                        meetings={oneOnOnes}
                        onCreate={handleCreateOneOnOne}
                        onUpdate={handleUpdateOneOnOne}
                        onDelete={handleDeleteOneOnOne}
                    />
                )}

                {activeTab === "follow-ups" && (
                    <FollowUpsPanel
                        followUps={followUps}
                        notes={safeNotes}
                        goals={goals}
                        meetings={oneOnOnes}
                        onCreate={handleCreateFollowUp}
                        onUpdate={handleUpdateFollowUp}
                        onDelete={handleDeleteFollowUp}
                    />
                )}

                {activeTab === "recognition" && (
                    <RecognitionPanel
                        recognitions={recognitions}
                        reviewCycle={engineer.reviewCycle}
                        onCreate={handleCreateRecognition}
                        onUpdate={handleUpdateRecognition}
                        onDelete={handleDeleteRecognition}
                    />
                )}

                {activeTab === "timeline" && (
                    <TimelinePanel
                        events={timeline}
                        reviewCycle={engineer.reviewCycle}
                        onNavigate={setActiveTab}
                    />
                )}
            </section>
        </main>
    );
}

export default EngineerProfilePage;
