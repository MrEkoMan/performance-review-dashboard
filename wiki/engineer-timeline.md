# Engineer Timeline

## Purpose

The engineer timeline combines records from separate manager workflows into a
single chronological view without copying them into a second event store.

## Included event types

- Performance evidence
- Goals
- 1:1 meetings
- Structured follow-ups
- Recognition

## Event normalization

Every timeline event contains its type, source record ID, event date, title,
summary, status or category, and review cycle when supported by the source.

Evidence and recognition use their recorded dates. Meetings use the meeting
date. Goals use completion date, target date, start date, or creation date in
that order. Follow-ups use completion date, due date, or creation date in that
order.

## Engineer profile functionality

- Display all events newest-first.
- Filter by event type, review cycle, and from/to dates.
- Navigate from an event to its source feature tab.

## API

`GET /api/engineers/{engineerId}/timeline`

Optional query parameters are `eventType`, `reviewCycle`, `from`, and `to`.
Dates use `YYYY-MM-DD`.

## Current boundary

The timeline is read-only and aggregated at query time. Role changes,
promotions, incidents, project milestones, and review outcomes can be added
when those source entities are introduced.
