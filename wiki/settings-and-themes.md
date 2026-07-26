# Settings and Themes

## Purpose

The settings page at `/settings` manages application appearance, local
attachment storage, and integration credentials.

## Appearance

The application supports:

- Light theme
- Dark theme

The selected theme is stored in SQLite and applied when the application loads.

## Attachment storage

The local attachment root:

- Must not be empty.
- Is converted to an absolute path.
- Is created by the backend when saved if it does not exist.
- Must be writable by the backend process.

See [Evidence attachments](evidence-attachments.md) for the storage layout and
security behavior.

## APIs

- `GET /api/settings`
- `PUT /api/settings/theme`
- `PUT /api/settings/attachment_storage_root`

Only the `theme` and `attachment_storage_root` setting keys are accepted.
