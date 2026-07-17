import { Pencil, Trash, Trash2 } from "lucide-react";

function NotesTable({ 
    notes = [], 
    loading, 
    onEdit, 
    onDelete,
}) {
    const safeNotes = Array.isArray(notes) ? notes : [];

    if (loading) {
        return <p>Loading notes...</p>
    }

    return (
        <section>
            <h2>Performance Evidence</h2>

            <table className="notes-table">
                <thead>
                    <tr>
                        <th>Date</th>
                        <th>Engineer</th>
                        <th>Category</th>
                        <th>Summary</th>
                        <th>Impact</th>
                        <th>Follow-Up</th>
                        <th>Actions</th>
                    </tr>
                </thead>

                <tbody>
                    {safeNotes.length === 0 ? (
                        <tr>
                            <td colSpan="7">No notes found.</td>
                        </tr>
                    ) : (
                        safeNotes.map((note) => (
                            <tr key={note.id}>
                                <td>{note.noteDate}</td>
                                <td>{note.engineerName}</td>
                                <td>{note.category}</td>
                                <td>{note.summary}</td>
                                <td>{note.impact || "-"}</td>
                                <td>{note.followUpNeeded ? "Yes" : "No"}</td>

                                <td className="actions-cell">
                                    <div className="table-actions">
                                        <button 
                                            type="button"
                                            className="icon-button"
                                            onClick={() => onEdit?.(note)}
                                            title="Edit Note"
                                            aria-label="Edit note">
                                            <Pencil size={12} />
                                        </button>
                                        <button 
                                            type="button"
                                            className="icon-button danger" 
                                            onClick={() => onDelete?.(note.id)}
                                            title="Delete note"
                                            aria-label="Delete note"> 
                                            <Trash2 size={12} />
                                        </button>
                                    </div>
                                </td>
                            </tr>
                        ))
                    )}
                </tbody>
            </table>
        </section>
    );
}

export default NotesTable;