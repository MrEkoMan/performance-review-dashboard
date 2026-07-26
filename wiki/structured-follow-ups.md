# Structured Follow-Up Management

## Purpose

Structured follow-ups turn coaching actions and commitments into trackable
records instead of relying only on a boolean flag or meeting date.

## Fields

- Engineer
- Source type and optional source record
- Description
- Owner
- Due date
- Status
- Priority
- Completion date
- Notes

## Sources

A follow-up can be created manually or linked to performance evidence, a goal,
or a 1:1 record. The backend verifies that a linked record belongs to the same
engineer as the follow-up.

## Statuses

- Open
- In progress
- Completed
- Cancelled

Completed follow-ups require a completion date. Non-completed follow-ups
cannot retain one.

## Engineer profile functionality

- Create, edit, and delete follow-ups.
- Filter follow-ups by status.
- Display open, overdue, and completed totals.
- Highlight overdue actions.
- Show open and overdue action totals on the profile overview.
- Link actions back to the type and identifier of their source record.

## APIs

- `GET /api/engineers/{engineerId}/follow-ups`
- `GET /api/engineers/{engineerId}/follow-ups?status={status}`
- `POST /api/engineers/{engineerId}/follow-ups`
- `GET /api/follow-ups/{id}`
- `PUT /api/follow-ups/{id}`
- `DELETE /api/follow-ups/{id}`

## Current boundary

Follow-ups are currently managed within an engineer profile. A cross-team
Needs Attention dashboard is the next dashboard-level roadmap capability.
