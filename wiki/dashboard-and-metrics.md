# Dashboard and Metrics

The dashboard begins with a
[Needs Attention panel](needs-attention-dashboard.md) that surfaces
deterministic workflow reminders across all engineers.
The [Upcoming 1:1s widget](upcoming-one-on-ones-dashboard.md) adds preparation
context for scheduled conversations.
The [Overdue Follow-Ups widget](overdue-follow-ups-dashboard.md) surfaces
aging open commitments with engineer, owner, and priority filters.
The [Goal Status widget](goal-status-dashboard.md) organizes active goals into
blocked, overdue, at-risk, and on-track health states.
The [Evidence Recency widget](evidence-recency-dashboard.md) shows documentation
coverage across 30-, 60-, and 90-day aging bands.

## Purpose

The dashboard provides a portfolio view of recorded performance evidence. It
is the default route at `/`.

## Functionality

- Lists all performance notes or filters them to one engineer.
- Searches across engineer name, category, summary, details, impact, and review
  cycle.
- Displays the number of visible results relative to all loaded notes.
- Supports editing and deleting evidence.
- Links to an individual engineer profile.
- Provides forms for adding engineers and performance notes.

## Metrics

Metrics respond to the current engineer filter and text search:

- Total notes
- Business Impact notes
- Technical Excellence notes
- Growth Area notes
- Notes requiring follow-up

These metrics are counts of stored evidence, not employee scores or performance
ratings.

## Routes and APIs

- UI: `/`
- `GET /api/engineers`
- `GET /api/notes`
- `GET /api/notes?engineerId={id}`
