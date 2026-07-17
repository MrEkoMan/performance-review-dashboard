import { Link, useParams } from "react-router-dom";

function EngineerProfilePage() {
    const { engineerId } = useParams();

    return (
        <main>
            <Link to="/">Back to Dashboard</Link>

            <h1>Engineer Profile</h1>

            <p>Engineer ID: {engineerId}</p>
        </main>
    );
}

export default EngineerProfilePage;