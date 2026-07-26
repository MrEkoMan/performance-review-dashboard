# Performance Evidence

## Purpose

Performance evidence captures concrete observations throughout a review cycle
so reviews and coaching discussions do not depend on memory.

## Evidence fields

- Engineer
- Evidence date
- Category
- Summary
- Details
- Impact
- Follow-up required
- Review cycle

## Categories

- Business Impact
- Technical Excellence
- Operational Excellence
- Team Contribution
- Growth Area
- Career Development
- Feedback Received

## Functionality

- Create evidence from the dashboard or engineer profile.
- Edit existing evidence.
- Delete evidence after confirmation.
- Filter evidence by engineer.
- Search evidence from the dashboard.
- Optionally create a note and screenshot attachment atomically.
- View and manage attachments while editing an existing note.

Deleting a note also removes its attachment relationships through SQLite
foreign-key cascading. Attachment files are managed by the attachment
workflow described in [Evidence attachments](evidence-attachments.md).

## APIs

- `GET /api/notes`
- `GET /api/notes?engineerId={id}`
- `POST /api/notes`
- `PUT /api/notes/{id}`
- `DELETE /api/notes/{id}`
- `POST /api/notes-with-attachment`
