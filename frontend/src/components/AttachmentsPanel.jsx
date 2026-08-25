import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ImagePlus, Trash2 } from "lucide-react";

import { deleteAttachment } from "../api/performanceApi.js";

const API_ROOT = "http://localhost:8080";

const SOURCE_OPTIONS = [
  "Microsoft Teams",
  "Slack",
  "Email",
  "GitHub",
  "Jira",
  "Other",
];

// AttachmentsPanel renders the upload form + thumbnail grid for any parent that
// owns attachments (performance notes, recognitions). It is parameterized by the
// parent id and the two API calls (list + upload) that differ per parent; delete
// and content-serving are parent-agnostic and reused directly.
function AttachmentsPanel({
  parentId,
  getAttachments,
  uploadAttachment,
  title = "Supporting Evidence",
  description = "Attach screenshots from Teams, Slack, email, or other sources.",
  fileInputId,
}) {
  const [attachments, setAttachments] = useState([]);
  const [file, setFile] = useState(null);

  const [sourceSystem, setSourceSystem] = useState("Microsoft Teams");
  const [sourceAuthor, setSourceAuthor] = useState("");
  const [sourceDate, setSourceDate] = useState("");
  const [caption, setCaption] = useState("");

  const [uploading, setUploading] = useState(false);
  const [storageSetupRequired, setStorageSetupRequired] = useState(false);
  const [error, setError] = useState("");

  async function loadAttachments() {
    if (!parentId) {
      setAttachments([]);
      return;
    }
    try {
      setError("");
      const data = await getAttachments(parentId);
      setAttachments(Array.isArray(data) ? data : []);
    } catch (err) {
      setError(err.message);
    }
  }

  useEffect(() => {
    loadAttachments();
    // loadAttachments intentionally closes over parentId.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [parentId]);

  async function handleUpload(event) {
    event.preventDefault();
    if (!file || !parentId) {
      return;
    }
    const formData = new FormData();
    formData.append("file", file);
    formData.append("sourceSystem", sourceSystem);
    formData.append("sourceAuthor", sourceAuthor);
    formData.append("sourceDate", sourceDate);
    formData.append("caption", caption);

    try {
      setUploading(true);
      setError("");
      setStorageSetupRequired(false);
      await uploadAttachment(parentId, formData);
      setFile(null);
      setSourceAuthor("");
      setSourceDate("");
      setCaption("");
      const input = document.getElementById(fileInputId);
      if (input) {
        input.value = "";
      }
      await loadAttachments();
    } catch (err) {
      if (err.status === 428) {
        setStorageSetupRequired(true);
      }
      setError(err.message);
    } finally {
      setUploading(false);
    }
  }

  async function handleDelete(attachmentId) {
    if (!window.confirm("Delete this screenshot?")) {
      return;
    }
    try {
      setError("");
      await deleteAttachment(attachmentId);
      await loadAttachments();
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <section className="note-attachments">
      <div className="attachment-heading">
        <div>
          <h3>{title}</h3>
          <p>{description}</p>
        </div>
      </div>

      {error && <div className="error">{error}</div>}

      {storageSetupRequired && (
        <Link to="/settings" className="secondary-button storage-setup-link">
          Configure Local Storage
        </Link>
      )}

      <form className="attachment-form" onSubmit={handleUpload}>
        <label htmlFor={fileInputId}>Screenshot</label>
        <input
          id={fileInputId}
          type="file"
          accept="image/png,image/jpeg,image/webp"
          onChange={(event) => setFile(event.target.files?.[0] || null)}
          required
        />

        <label>Source</label>
        <select
          value={sourceSystem}
          onChange={(event) => setSourceSystem(event.target.value)}
        >
          {SOURCE_OPTIONS.map((option) => (
            <option key={option} value={option}>
              {option}
            </option>
          ))}
        </select>

        <label>From</label>
        <input
          value={sourceAuthor}
          onChange={(event) => setSourceAuthor(event.target.value)}
          placeholder="Name or role"
        />

        <label>Source date</label>
        <input
          type="date"
          value={sourceDate}
          onChange={(event) => setSourceDate(event.target.value)}
        />

        <label>Caption</label>
        <textarea
          value={caption}
          onChange={(event) => setCaption(event.target.value)}
          placeholder="Why this screenshot matters"
        />

        <button type="submit" disabled={uploading || !file}>
          <ImagePlus size={17} />
          {uploading ? "Uploading..." : "Add Screenshot"}
        </button>
      </form>

      {attachments.length > 0 && (
        <div className="attachment-grid">
          {attachments.map((attachment) => (
            <article className="attachment-card" key={attachment.id}>
              <a
                href={`${API_ROOT}${attachment.contentUrl}`}
                target="_blank"
                rel="noreferrer"
              >
                <img
                  src={`${API_ROOT}${attachment.contentUrl}`}
                  alt={attachment.caption || attachment.originalFilename}
                />
              </a>
              <div className="attachment-meta">
                <strong>{attachment.caption || attachment.originalFilename}</strong>
                <span>{attachment.sourceSystem || "Unknown source"}</span>
                {attachment.sourceAuthor && (
                  <span>From: {attachment.sourceAuthor}</span>
                )}
                {attachment.sourceDate && (
                  <span>Date: {attachment.sourceDate}</span>
                )}
              </div>
              <button
                type="button"
                className="icon-button danger"
                title="Delete screenshot"
                aria-label="Delete screenshot"
                onClick={() => handleDelete(attachment.id)}
              >
                <Trash2 size={16} />
              </button>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

export default AttachmentsPanel;
