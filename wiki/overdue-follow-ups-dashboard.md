# Overdue Follow-Ups Dashboard

The Overdue Follow-Ups dashboard gives an engineering manager a portfolio-level
view of commitments that need intervention. It complements the per-engineer
Follow-Ups tab by surfacing aging work across the entire team.

## What appears by default

The dashboard includes follow-ups that:

- have a due date before today;
- have an `open` or `in_progress` status; and
- belong to an existing engineer.

Completed and cancelled follow-ups, along with work that is not yet due, are
excluded from the default view.

Each item shows the engineer, owner, priority, status, description, due date,
days overdue, notes, and originating feature when available. Selecting an
engineer opens their profile directly on the Follow-Ups tab.

## Aging and filters

The dashboard provides actionable aging thresholds:

- all overdue;
- 7 or more days overdue;
- 30 or more days overdue; and
- 60 or more days overdue.

Managers can further narrow the list by engineer, owner, and priority. Aging
thresholds use the manager's local calendar date.

## API

`GET /api/dashboard/follow-ups`

Supported query parameters:

- `overdue`: defaults to `true`; use `false` to query all due dates.
- `status`: `open`, `in_progress`, `completed`, or `cancelled`.
- `priority`: `low`, `medium`, `high`, or `critical`.
- `engineerId`: a positive engineer ID.
- `owner`: case-insensitive exact owner name.

The response is ordered by priority, status, due date, and engineer name. Each
record includes a calculated `daysOverdue` value.

## Interpretation boundary

An overdue item is a prompt for coordination, not a performance rating.
Managers should review context with the engineer, confirm that the owner and
due date remain accurate, and update or close stale follow-ups rather than
treating age alone as a negative signal.
