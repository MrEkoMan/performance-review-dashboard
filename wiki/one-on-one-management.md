# 1:1 Management

## Purpose

1:1 records preserve conversation continuity for each engineer and make it
easier to prepare for the next meeting without relying on memory.

## Fields

- Meeting date
- Status
- Wins
- Challenges
- Career discussion
- Feedback
- Manager topics
- Engineer topics
- Shared notes
- Private manager notes
- Follow-up date

## Statuses

- Scheduled
- Completed
- Cancelled

## Engineer profile functionality

- Create, edit, and delete 1:1 records.
- Display history newest-first.
- Filter history by status.
- Show the next scheduled 1:1.
- Show the most recent completed 1:1.
- Show the number of days since the last completed meeting.
- Visually distinguish shared notes from private manager notes.

## Privacy boundary

Private manager notes are visually identified and remain in the local
application. The current application does not implement user accounts or
role-based authorization, so "private" distinguishes intended use rather than
enforcing access between multiple authenticated users. Anyone with access to
the local API or SQLite database can access the data.

## Follow-ups

The follow-up date provides meeting context only. Reusable action items,
owners, priorities, completion status, and source-record links are managed
through [structured follow-ups](structured-follow-ups.md).

## APIs

- `GET /api/engineers/{engineerId}/one-on-ones`
- `GET /api/engineers/{engineerId}/one-on-ones?status={status}`
- `POST /api/engineers/{engineerId}/one-on-ones`
- `GET /api/one-on-ones/{id}`
- `PUT /api/one-on-ones/{id}`
- `DELETE /api/one-on-ones/{id}`
