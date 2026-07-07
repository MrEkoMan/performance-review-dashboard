function NotesTable({ notes, loading }) {
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
                    </tr>
                </thead>

                <tbody>
                    {notes.length === 0 ? (
                        <tr>
                            <td colSpan="6">No notes found.</td>
                        </tr>
                    ) : (
                        notes.map((note) => (
                            <tr key={note.id}>
                                <td>{note.noteDate}</td>
                                <td>{note.engineerName}</td>
                                <td>{note.category}</td>
                                <td>{note.summary}</td>
                                <td>{note.impact}</td>
                                <td>{followUpNeeded ? "Yes" : "No"}</td>
                            </tr>
                        ))
                    )}
                </tbody>
            </table>
        </section>
    );
}

export default NotesTable;