import {
  getNoteAttachments,
  uploadNoteAttachment,
} from "../api/performanceApi.js";
import AttachmentsPanel from "./AttachmentsPanel.jsx";

// NoteAttachments is a thin wrapper over the generic AttachmentsPanel, preserving
// the note-scoped prop interface (<NoteAttachments noteId={...} />) used by AddNoteForm.
function NoteAttachments({ noteId }) {
  return (
    <AttachmentsPanel
      parentId={noteId}
      getAttachments={getNoteAttachments}
      uploadAttachment={uploadNoteAttachment}
      fileInputId={`attachment-file-${noteId}`}
    />
  );
}

export default NoteAttachments;
