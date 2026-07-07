const BASE_URL = "http://localhost:8080/api";

async function request(path, options = {}) {
    const url = `${BASE_URL}${path}`;
    console.log("Requesting:", url);

    const response = await fetch(url, {
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
    
    if (response.status === 204) {
        return null;
    }

    return response.json();
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