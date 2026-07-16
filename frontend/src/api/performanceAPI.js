const BASE_URL = "http://localhost:8080/api";

async function request(path, options = {}) {
    const response = await fetch(`${BASE_URL}${path}`, {
        headers: { 
            "Content-Type": "application/json",
            ...(options.headers || {}),
        },
        ...options,
    });

    if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Request failed with status ${response.status}`);
    }

    const text = await response.text();
    
    if (!text) {
        return null;
    }

    return JSON.parse(text);
}

export function getEngineers() {
    return request("/engineers");
}

export function createEngineer(engineer) {
    return request("/engineers", {
        method: "POST",
        body: JSON.stringify(engineer),
    });
}

export function getNotes(engineerId = "") {
    const query = engineerId ? `?engineerId=${engineerId}` : "";
    return request(`/notes${query}`);
}

export function createNote(note) {
    return request("/notes", {
        method: "POST",
        body: JSON.stringify(note),
    });
}

export function updateNote(id, note) {
    return request(`/notes/${id}`, {
        method: "PUT",
        body: JSON.stringify(note),
    });
}

export function deleteNote(id) {
    return request(`/notes/${id}`, {
        method: "DELETE",
    });
}