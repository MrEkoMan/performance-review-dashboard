import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ImagePlus, Trash2 } from "lucide-react";

import {
  deleteAttachment,
  getNoteAttachments,
  uploadNoteAttachment,
} from "../api/performanceApi.js";

const API_ROOT = "http://localhost:8080";

function NoteAttachments({ noteId }) {
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
    if (!noteId) {
      setAttachments([]);
      return;
    }

    try {
      setError("");

      const data = await getNoteAttachments(noteId);

      setAttachments(Array.isArray(data) ? data : []);
    } catch (err) {
      setError(err.message);
    }
  }

  useEffect(() => {
    loadAttachments();
  }, [noteId]);

  async function handleUpload(event) {
    event.preventDefault();

    if (!file || !noteId) {
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

      await uploadNoteAttachment(noteId, formData);

      setFile(null);
      setSourceAuthor("");
      setSourceDate("");
      setCaption("");

      const input = document.getElementById(
        `attachment-file-${noteId}`
      );

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
    const confirmed = window.confirm(
      "Delete this screenshot?"
    );

    if (!confirmed) {
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
          <h3>Supporting Evidence</h3>
          <p>
            Attach screenshots from Teams, Slack, email, or
            other sources.
          </p>
        </div>
      </div>

      {error && <div className="error">{error}</div>}

      {storageSetupRequired && (
        <Link
          to="/settings"
          className="secondary-button storage-setup-link"
        >
          Configure Local Storage
        </Link>
      )}

      <form
        className="attachment-form"
        onSubmit={handleUpload}
      >
        <label htmlFor={`attachment-file-${noteId}`}>
          Screenshot
        </label>

        <input
          id={`attachment-file-${noteId}`}
          type="file"
          accept="image/png,image/jpeg,image/webp"
          onChange={(event) =>
            setFile(event.target.files?.[0] || null)
          }
          required
        />

        <label>Source</label>

        <select
          value={sourceSystem}
          onChange={(event) =>
            setSourceSystem(event.target.value)
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

        <label>From</label>

        <input
          value={sourceAuthor}
          onChange={(event) =>
            setSourceAuthor(event.target.value)
          }
          placeholder="Name or role"
        />

        <label>Source date</label>

        <input
          type="date"
          value={sourceDate}
          onChange={(event) =>
            setSourceDate(event.target.value)
          }
        />

        <label>Caption</label>

        <textarea
          value={caption}
          onChange={(event) =>
            setCaption(event.target.value)
          }
          placeholder="Why this screenshot matters"
        />

        <button
          type="submit"
          disabled={uploading || !file}
        >
          <ImagePlus size={17} />
          {uploading ? "Uploading..." : "Add Screenshot"}
        </button>
      </form>

      {attachments.length > 0 && (
        <div className="attachment-grid">
          {attachments.map((attachment) => (
            <article
              className="attachment-card"
              key={attachment.id}
            >
              <a
                href={`${API_ROOT}${attachment.contentUrl}`}
                target="_blank"
                rel="noreferrer"
              >
                <img
                  src={`${API_ROOT}${attachment.contentUrl}`}
                  alt={
                    attachment.caption ||
                    attachment.originalFilename
                  }
                />
              </a>

              <div className="attachment-meta">
                <strong>
                  {attachment.caption ||
                    attachment.originalFilename}
                </strong>

                <span>
                  {attachment.sourceSystem || "Unknown source"}
                </span>

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
                onClick={() =>
                  handleDelete(attachment.id)
                }
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

export default NoteAttachments;