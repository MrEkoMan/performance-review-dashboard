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
  return request(`/notes/${id}`, { method: "DELETE" });
}

export function getSettings() {
  return request("/settings");
}

export function updateSetting(key, value) {
  return request(`/settings/${key}`, {
    method: "PUT",
    body: JSON.stringify({ key, value }),
  });
}

export function getIntegrations() {
  return request("/integrations");
}

export function saveIntegration(provider, integration) {
  return request(`/integrations/${provider}`, {
    method: "PUT",
    body: JSON.stringify({ ...integration, provider }),
  });
}

export function deleteIntegration(provider) {
  return request(`/integrations/${provider}`, { method: "DELETE" });
}

export function getNoteAttachments(noteId) {
  return request(`/notes/${noteId}/attachments`);
}

export async function uploadNoteAttachment(noteId, formData) {
  const response = await fetch(
    `http://localhost:8080/api/notes/${noteId}/attachments`,
    { method: "POST", body: formData },
  );
  if (!response.ok) {
    const message = await response.text();
    const error = new Error(
      message || `Request failed with status ${response.status}`,
    );
    error.status = response.status;
    throw error;
  }
  return response.json();
}

export function deleteAttachment(attachmentId) {
  return request(`/attachments/${attachmentId}`, { method: "DELETE" });
}

export async function createNoteWithAttachment(formData) {
  const response = await fetch(
    "http://localhost:8080/api/notes-with-attachment",
    { method: "POST", body: formData },
  );
  if (!response.ok) {
    const message = await response.text();
    const error = new Error(
      message || `Request failed with status ${response.status}`,
    );
    error.status = response.status;
    throw error;
  }
  return response.json();
}

export function getGoals(engineerId, filters = {}) {
  const parameters = new URLSearchParams();
  if (filters.status) {
    parameters.set("status", filters.status);
  }
  if (filters.reviewCycle) {
    parameters.set("reviewCycle", filters.reviewCycle);
  }
  const query = parameters.toString();
  return request(
    `/engineers/${engineerId}/goals${query ? `?${query}` : ""}`,
  );
}

export function createGoal(engineerId, goal) {
  return request(`/engineers/${engineerId}/goals`, {
    method: "POST",
    body: JSON.stringify(goal),
  });
}

export function updateGoal(goalId, goal) {
  return request(`/goals/${goalId}`, {
    method: "PUT",
    body: JSON.stringify(goal),
  });
}

export function deleteGoal(goalId) {
  return request(`/goals/${goalId}`, { method: "DELETE" });
}

export function getOneOnOnes(engineerId, status = "") {
  const query = status ? `?status=${encodeURIComponent(status)}` : "";
  return request(`/engineers/${engineerId}/one-on-ones${query}`);
}

export function createOneOnOne(engineerId, meeting) {
  return request(`/engineers/${engineerId}/one-on-ones`, {
    method: "POST",
    body: JSON.stringify(meeting),
  });
}

export function updateOneOnOne(meetingId, meeting) {
  return request(`/one-on-ones/${meetingId}`, {
    method: "PUT",
    body: JSON.stringify(meeting),
  });
}

export function deleteOneOnOne(meetingId) {
  return request(`/one-on-ones/${meetingId}`, { method: "DELETE" });
}

export function getFollowUps(engineerId, status = "") {
  const query = status ? `?status=${encodeURIComponent(status)}` : "";
  return request(`/engineers/${engineerId}/follow-ups${query}`);
}

export function createFollowUp(engineerId, followUp) {
  return request(`/engineers/${engineerId}/follow-ups`, {
    method: "POST",
    body: JSON.stringify(followUp),
  });
}

export function updateFollowUp(followUpId, followUp) {
  return request(`/follow-ups/${followUpId}`, {
    method: "PUT",
    body: JSON.stringify(followUp),
  });
}

export function deleteFollowUp(followUpId) {
  return request(`/follow-ups/${followUpId}`, { method: "DELETE" });
}

export function getRecognitions(engineerId, filters = {}) {
  const parameters = new URLSearchParams();
  if (filters.category) {
    parameters.set("category", filters.category);
  }
  if (filters.reviewCycle) {
    parameters.set("reviewCycle", filters.reviewCycle);
  }
  const query = parameters.toString();
  return request(
    `/engineers/${engineerId}/recognitions${query ? `?${query}` : ""}`,
  );
}

export function createRecognition(engineerId, recognition) {
  return request(`/engineers/${engineerId}/recognitions`, {
    method: "POST",
    body: JSON.stringify(recognition),
  });
}

export function updateRecognition(recognitionId, recognition) {
  return request(`/recognitions/${recognitionId}`, {
    method: "PUT",
    body: JSON.stringify(recognition),
  });
}

export function deleteRecognition(recognitionId) {
  return request(`/recognitions/${recognitionId}`, { method: "DELETE" });
}

export function getTimeline(engineerId, filters = {}) {
  const parameters = new URLSearchParams();
  if (filters.eventType) {
    parameters.set("eventType", filters.eventType);
  }
  if (filters.reviewCycle) {
    parameters.set("reviewCycle", filters.reviewCycle);
  }
  if (filters.from) {
    parameters.set("from", filters.from);
  }
  if (filters.to) {
    parameters.set("to", filters.to);
  }
  const query = parameters.toString();
  return request(
    `/engineers/${engineerId}/timeline${query ? `?${query}` : ""}`,
  );
}

export function getDashboardAttention(filters = {}) {
  const parameters = new URLSearchParams();
  if (filters.type) {
    parameters.set("type", filters.type);
  }
  if (filters.severity) {
    parameters.set("severity", filters.severity);
  }
  const query = parameters.toString();
  return request(`/dashboard/attention${query ? `?${query}` : ""}`);
}

export function getUpcomingOneOnOnes(days = 14) {
  return request(`/dashboard/upcoming-one-on-ones?days=${days}`);
}

export function getDashboardFollowUps(filters = {}) {
  const parameters = new URLSearchParams();
  parameters.set("overdue", String(filters.overdue ?? true));
  if (filters.status) {
    parameters.set("status", filters.status);
  }
  if (filters.priority) {
    parameters.set("priority", filters.priority);
  }
  if (filters.engineerId) {
    parameters.set("engineerId", filters.engineerId);
  }
  if (filters.owner) {
    parameters.set("owner", filters.owner);
  }
  return request(`/dashboard/follow-ups?${parameters.toString()}`);
}

export function getDashboardGoals(filters = {}) {
  const parameters = new URLSearchParams();
  if (filters.includeClosed) {
    parameters.set("includeClosed", "true");
  }
  if (filters.health) {
    parameters.set("health", filters.health);
  }
  if (filters.status) {
    parameters.set("status", filters.status);
  }
  if (filters.priority) {
    parameters.set("priority", filters.priority);
  }
  if (filters.engineerId) {
    parameters.set("engineerId", filters.engineerId);
  }
  if (filters.reviewCycle) {
    parameters.set("reviewCycle", filters.reviewCycle);
  }
  const query = parameters.toString();
  return request(`/dashboard/goals${query ? `?${query}` : ""}`);
}
