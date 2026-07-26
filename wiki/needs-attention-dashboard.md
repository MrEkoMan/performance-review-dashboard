# Needs Attention Dashboard

## Purpose

The Needs Attention panel turns stored manager records into a deterministic
queue of work across engineers. It does not use AI or infer employee
performance.

## Attention rules

### High severity

- Structured follow-up is open or in progress and its due date has passed.
- Goal is blocked.
- Goal is not started or in progress and its target date has passed.

### Medium severity

- No performance evidence has been recorded in the last 30 days.
- A scheduled 1:1 occurs within the next 7 days.
- A performance-evidence record still has its legacy follow-up flag set.

### Low severity

- No recognition has been recorded in the last 90 days.

All date windows are calculated using the backend's local current date.

## Dashboard functionality

- Shows high, medium, and total signal counts.
- Sorts high-severity items first.
- Filters visible items by severity.
- Displays the engineer, reason, relevant date, and signal type.
- Links directly to the relevant engineer profile tab.

## API

`GET /api/dashboard/attention`

Optional query parameters:

- `type`
- `severity`

## Interpretation boundary

Attention items are workflow reminders, not performance ratings. Evidence and
recognition recency rules indicate missing documentation only; they do not
claim that work or recognition-worthy contributions did not occur.
