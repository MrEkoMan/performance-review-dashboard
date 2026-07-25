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

export function getSettings() {
    return request("/settings");
}

export function updateSetting(key, value) {
    return request(`/settings/${key}`, {
        method: "PUT",
        body: JSON.stringify({
            key,
            value,
        }),
    });
}

export function getIntegrations() {
    return request("/integrations");
}

export function saveIntegration(provider, integration) {
    return request(`/integrations/${provider}`, {
        method: "PUT",
        body: JSON.stringify({
            ...integration,
            provider,
        }),
    });
}

export function deleteIntegration(provider) {
    return request(`/integrations/${provider}`, {
        method: "DELETE",
    });
}

export function getNoteAttachments(noteId) {
  return request(`/notes/${noteId}/attachments`);
}

export async function uploadNoteAttachment(noteId, formData) {
  const response = await fetch(
    `http://localhost:8080/api/notes/${noteId}/attachments`,
    {
      method: "POST",
      body: formData,
    }
  );

  if (!response.ok) {
    const message = await response.text();

    const error = new Error(
      message || `Request failed with status ${response.status}`
    );

    error.status = response.status;
    throw error;
  }

  return response.json();
}

export function deleteAttachment(attachmentId) {
  return request(`/attachments/${attachmentId}`, {
    method: "DELETE",
  });
}

export async function createNoteWithAttachment(formData) {
  const response = await fetch(
    "http://localhost:8080/api/notes-with-attachment",
    {
      method: "POST",
      body: formData,
    }
  );

  if (!response.ok) {
    const message = await response.text();

    const error = new Error(
      message ||
        `Request failed with status ${response.status}`
    );

    error.status = response.status;

    throw error;
  }

  return response.json();
}