import { ArrowLeft } from "lucide-react";
import Metrics from "./Metrics";
import NotesTable from "./NotesTables";

function EngineerProfile ({
    engineer,
    notes = [],
    loading,
    onBack,
    onEditNote,
    onDeleteNote,
}) {
    const safeNotes = Array.isArray(notes) ?  notes : [];

    if (!engineer) {
        return null;
    }

    const followUps = safeNotes.filter(
        (note) => note.followUpNeeded
    ).length;

    const latestNote = [...safeNotes]
        .filter((note) => note.noteDate)
        .sort(
            (first, second) =>
                new Date(second.noteDate) - new Date(first.noteDate)
        )[0];

    return (
        <section className="engineer-profile">
            <button
                type="button"
                className="back-button"
                onClick={onBack}
            >
                <ArrowLeft size={18} />
                Back to Dashboard
            </button>

            <div className="profile-header">
                <div>
                    <p className="profile-eyebrow">Engineer Profile</p>
                    <h1>{engineer.name}</h1>

                    <p className="profile-subtitle">
                        {[engineer.role, engineer.level, engineer.team]
                            .filter(Boolean)
                            .join(" . ")}
                    </p>
                </div>

                <div className="profile-review-cycle">
                    <span>Review Cycle</span>
                    <strong>{engineer.reviewCycle || "Not assigned"}</strong>
                </div>
            </div>

            <div className="profile-details-grid">
                <article className="profile-card">
                    <h2>Career Goal</h2>
                    <p>{engineer.careerGoal || "No career goal recorded."}</p>
                </article>

                <article className="profile-card">
                    <h2>Current Status</h2>

                    <dl className="profile-summary-list">
                        <div>
                            <dt>Total Evidence</dt>
                            <dd>{safeNotes.length}</dd>
                        </div>

                        <div>
                            <dt>Open Follow-ups</dt>
                            <dd>{followUps}</dd>
                        </div>

                        <div>
                            <dt>Most Recent Note</dt>
                            <dd>{latestNote?.noteDate || "No notes yet"}</dd>
                        </div>
                    </dl>
                </article>
            </div>

            <Metrics notes={safeNotes} />

            <NotesTable
                notes={safeNotes}     
                loading={loading}
                onEdit={onEditNote}
                onDelete={onDeleteNote}
            />
        </section>
    );
}

export default EngineerProfile;