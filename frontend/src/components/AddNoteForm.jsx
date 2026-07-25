import { useEffect, useMemo, useState, useRef } from "react";
import { createNote, createNoteWithAttachment } from "../api/performanceApi";
import { NotepadText, ImagePlus } from "lucide-react";
import NoteAttachments from "./NoteAttachments.jsx";

const categories = [
    "Business Impact",
    "Technical Excellence",
    "Operational Excellence",
    "Team Contribution",
    "Growth Area",
    "Career Development",
    "Feedback Received",
];

function getReviewCycles() {
    const currentYear = new Date().getFullYear();
    const cycles = [];

    for (let year = currentYear - 1; year <= currentYear + 2; year++) {
        cycles.push(`${year} H1`);
        cycles.push(`${year} H2`);
    }

    return cycles;
}

function getToday() {
    return new Date().toISOString().slice(0, 10);
}

function AddNoteForm({ 
    engineers, 
    onNoteCreated ,
    noteToEdit,
    onNoteUpdated,
    onEditComplete,
    onCancelEdit,
}) {
    const reviewCycles = useMemo(() => getReviewCycles(), []);

    const createEmptyForm = () => ({
        engineerId: "",
        noteDate: getToday(),
        category: "Business Impact",
        summary: "",
        details: "",
        impact: "",
        followUpNeeded: false,
        reviewCycle: reviewCycles[0],
    });

    const [form, setForm] = useState(createEmptyForm);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState("");

    const [attachmentFile, setAttachmentFile] = useState(null);
    const [attachmentSource, setAttachmentSource] = useState("Microsoft Teams");
    const [attachmentAuthor, setAttachmentAuthor] = useState("");
    const [attachmentDate, setAttachmentDate] = useState("");
    const [attachmentCaption, setAttachmentCaption] = useState("");
    const [storageSetupRequired, setStorageSetupRequired] = useState(false);

    const attachmentInputRef = useRef(null);

    const isEditing = Boolean(noteToEdit);

    useEffect(() => {
        if (noteToEdit) {
            setForm({
                engineerId: String(noteToEdit.engineerId ?? ""),
                noteDate: noteToEdit.noteDate ?? getToday(),
                category: noteToEdit.category ?? "Business Impact",
                summary: noteToEdit.summary ?? "",
                details: noteToEdit.details ?? "",
                impact: noteToEdit.impact ?? "",
                followUpNeeded: Boolean(noteToEdit.followUpNeeded),
                reviewCycle: noteToEdit.reviewCycle ?? reviewCycles[0],
            });
        } else {
            setForm(createEmptyForm());
        }

        setError("");
    }, [noteToEdit]);

    function handleChange(event) {
        const { name, value, type, checked } = event.target;

        setForm((current) => ({
            ...current,
            [name]: type === "checkbox" ? checked : value,
        }));
    }

    function resetAttachmentForm() {
        setAttachmentFile(null);
        setAttachmentSource("Microsoft Teams");
        setAttachmentAuthor("");
        setAttachmentDate("");
        setAttachmentCaption("");
        setStorageSetupRequired(false);

        if (attachmentInputRef.current) {
            attachmentInputRef.current.value = "";
        }
    }

    async function handleNoteCreated(createdNote) {
        await loadNotes();
    }

    async function handleSubmit(event) {
        event.preventDefault();

        setError("");
        setSaving(true);
        setStorageSetupRequired(false);

        const notePayload = {
            ...form,
            engineerId: Number(form.engineerId),
        };

        try {
            if (isEditing) {
                await onNoteUpdated({
                    ...notePayload,
                    id: noteToEdit.id,
                });
            } else if (attachmentFile) {
                const formData = new FormData();

                formData.append(
                    "engineerId",
                    String(notePayload.engineerId)
                );
                formData.append("noteDate", notePayload.noteDate);
                formData.append("category", notePayload.category);
                formData.append("summary", notePayload.summary);
                formData.append("details", notePayload.details || "");
                formData.append("impact", notePayload.impact || "");
                formData.append(
                    "followUpNeeded",
                    String(notePayload.followUpNeeded)
                );
                formData.append(
                    "reviewCycle",
                    notePayload.reviewCycle || ""
                );

                formData.append("file", attachmentFile);
                formData.append("sourceSystem", attachmentSource);
                formData.append("sourceAuthor", attachmentAuthor);
                formData.append("sourceDate", attachmentDate);
                formData.append("caption", attachmentCaption);

                const result = await createNoteWithAttachment(formData);

                setForm(createEmptyForm());
                resetAttachmentForm();

                if (onNoteCreated) {
                    await onNoteCreated(result.note);
                } else {
                    const createdNote = await createNote(notePayload);

                    setForm(createEmptyForm());
                    resetAttachmentForm();

                    if (onNoteCreated) {
                        await onNoteCreated(createdNote);
                    }
                }
            }
        } catch (err) {
            setError(err.message);
        } finally {
            setSaving(false);
        }
    }

    function handleCancel() {
        setForm(createEmptyForm());

        if (onCancelEdit) {
            onCancelEdit();
        }
    }

    return (
        <form className="form-card" onSubmit={handleSubmit}>
            <h2>{isEditing ? "Edit Performance Note" : "Add Performance Note"}</h2>

            {error && <div className="error">Error: {error}</div>}

            <label htmlFor="note-engineer">Engineer</label>
            <select
                id="note-engineer"
                name="engineerId"
                value={form.engineerId}
                onChange={handleChange}
                required
            >
                <option value="">Select engineer</option>
                {engineers.map((engineer) => (
                    <option key={engineer.id} value={engineer.id}>
                        {engineer.name}
                    </option>
                ))}
            </select>

            <label htmlFor="note-date">Date</label>
            <input
                id="note-date"
                type="date"
                name="noteDate"
                value={form.noteDate}
                onChange={handleChange}
                required
            />

            <label htmlFor="note-category">Category</label>
            <select
                id="note-category"
                name="category"
                value={form.category}
                onChange={handleChange}
            >
                {categories.map((category) => (
                    <option key={category} value={category}>
                        {category}
                    </option>
                ))}
            </select>

            <label htmlFor="note-summary">Summary</label>
            <input
                id="note-summary"
                name="summary"
                value={form.summary}
                onChange={handleChange}
                required
            />

            <label htmlFor="note-details">Details</label>
            <textarea
                id="note-details"
                name="details"
                value={form.details}
                onChange={handleChange}
            />

            <label htmlFor="note-impact">Impact</label>
            <textarea
                id="note-impact"
                name="impact"
                value={form.impact}
                onChange={handleChange}
            />

            <label htmlFor="note-review-cycle">Review Cycle</label>
            <select
                id="note-review-cycle"
                name="reviewCycle"
                value={form.reviewCycle}
                onChange={handleChange}
            >
                {reviewCycles.map((cycle) => (
                    <option key={cycle} value={cycle}>
                        {cycle}
                    </option>
                ))}
            </select>

            <label className="checkbox-row">
                <input
                    type="checkbox"
                    name="followUpNeeded"
                    checked={form.followUpNeeded}
                    onChange={handleChange}
                />
                Follow-up Needed
            </label>

            {!isEditing && (
                <section className="attachment-fields">
                    <div className="attachment-heading">
                        <ImagePlus size={18} />

                        <div>
                            <h3>Supporting Screenshot</h3>
                            <p>
                            Optionally attach recognition or feedback from Teams,
                            Slack, email, GitHub, Jira, or another source.
                            </p>
                        </div>
                    </div>

                    <label htmlFor="note-attachment">
                        Screenshot
                    </label>

                    <input
                        ref={attachmentInputRef}
                        id="note-attachment"
                        type="file"
                        accept="image/png,image/jpeg,image/webp"
                        onChange={(event) =>
                            setAttachmentFile(
                            event.target.files?.[0] || null
                            )
                        }
                    />

                    {attachmentFile && (
                        <>
                            <label htmlFor="attachment-source">
                                Source
                            </label>

                            <select
                                id="attachment-source"
                                value={attachmentSource}
                                onChange={(event) =>
                                    setAttachmentSource(event.target.value)
                                }
                            >
                                <option value="Microsoft Teams">
                                    Microsoft Teams
                                </option>
                                <option value="Slack">Slack</option>
                                <option value="Email">Email</option>
                                <option value="GitHub">GitHub</option>
                                <option value="Jira">Jira</option>
                                <option value="Other">Other</option>
                            </select>

                            <label htmlFor="attachment-author">
                                From
                            </label>

                            <input
                                id="attachment-author"
                                value={attachmentAuthor}
                                onChange={(event) =>
                                    setAttachmentAuthor(event.target.value)
                                }
                                placeholder="Person or role that provided the feedback"
                            />

                            <label htmlFor="attachment-date">
                                Source date
                            </label>

                            <input
                                id="attachment-date"
                                type="date"
                                value={attachmentDate}
                                onChange={(event) =>
                                    setAttachmentDate(event.target.value)
                                }
                            />

                            <label htmlFor="attachment-caption">
                                Caption
                            </label>

                            <textarea
                                id="attachment-caption"
                                value={attachmentCaption}
                                onChange={(event) =>
                                    setAttachmentCaption(event.target.value)
                                }
                                placeholder="Explain why this screenshot is relevant."
                            />
                        </>
                    )}

                    {storageSetupRequired && (
                    <div className="error">
                        Local attachment storage is not configured. Open
                        Settings and configure a root storage folder.
                    </div>
                    )}
                </section>
            )}

            <div className="form-actions">
                <button type="submit" disabled={saving || engineers.length === 0}>
                    {saving
                        ? attachmentFile
                            ? "Saving Note and Screenshots..."
                            : "Saving..."
                        : isEditing
                            ? "Save Changes"
                            : "Add Note"}
                </button>

                {isEditing && (
                    <button
                        type="button"
                        className="secondary-button"
                        onClick={handleCancel}
                        disabled={saving}
                    >
                        Cancel
                    </button>
                )}
            </div>

            {isEditing && noteToEdit?.id && (
                <NoteAttachments noteId={noteToEdit.id} />
            )}
        </form>
    );
}

export default AddNoteForm;