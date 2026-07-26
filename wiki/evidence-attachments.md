# Evidence Attachments

## Purpose

Attachments add visual source evidence to performance notes while keeping
binary files outside SQLite.

## Supported files

- PNG
- JPEG
- WebP
- Maximum file size: 10 MB

The backend detects the file type from its content rather than trusting only
the filename extension.

## Configuration

Set the local attachment root from **Settings > Local Attachment Storage**.
The backend creates this structure beneath the selected root:

```text
engineers/
  {engineer-id}-{sanitized-name}/
    {year}/
      {month}/
        {generated-filename}
```

SQLite stores relative paths and metadata. Path resolution rejects attempts to
escape the configured storage root.

## Metadata

- Original filename
- MIME type
- File size
- SHA-256 hash
- Source system
- Source author
- Source date
- Caption
- Created date

## Workflows

- Add an attachment to an existing note.
- Create a new note and attachment in one atomic database transaction.
- List attachments associated with a note.
- View attachment content inline.
- Delete attachment metadata and its local file.

If database persistence fails after a new file is written, the backend removes
the file to avoid leaving orphaned content.

## APIs

- `GET /api/notes/{id}/attachments`
- `POST /api/notes/{id}/attachments`
- `GET /api/attachments/{id}/content`
- `DELETE /api/attachments/{id}`
- `POST /api/notes-with-attachment`
