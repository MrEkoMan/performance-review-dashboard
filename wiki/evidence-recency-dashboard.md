# Evidence Recency Dashboard

The Evidence Recency dashboard gives managers a portfolio view of how recently
performance evidence has been documented for every engineer. It expands the
Needs Attention stale-evidence warning into a complete view that includes
engineers with current evidence.

## Recency bands

Each engineer is assigned one deterministic recency state from their latest
performance-note date:

- `recent`: evidence recorded within the last 30 days.
- `aging`: the latest evidence is 31 to 60 days old.
- `stale`: the latest evidence is 61 to 90 days old.
- `critical`: the latest evidence is more than 90 days old.
- `never`: no performance evidence has been recorded.

Future-dated notes are treated as zero days old. Engineers without evidence are
kept distinct from dated aging bands.

## Dashboard functionality

- Shows counts for all engineers and each recency band.
- Filters by recency, engineer, team, and current review cycle.
- Shows the most recent evidence date and its age.
- Shows evidence counts from the last 30 days, the engineer's current review
  cycle, and all time.
- Links directly to the engineer's Evidence tab.

Engineers with no evidence or the stalest evidence are listed first.

## API

`GET /api/dashboard/evidence-recency`

Supported query parameters:

- `recency`: `recent`, `aging`, `stale`, `critical`, or `never`.
- `engineerId`: a positive engineer ID.
- `team`: an exact team value.
- `reviewCycle`: an exact current engineer review-cycle value.

## Interpretation boundary

Evidence recency measures documentation coverage, not engineer performance.
Managers should use it to prompt balanced note-taking and check whether useful
context has been missed. A lack of recent evidence must not be interpreted as
poor performance.
