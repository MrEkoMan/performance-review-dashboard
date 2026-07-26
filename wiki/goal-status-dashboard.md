# Goal Status Dashboard

The Goal Status dashboard provides a team-wide view of active development and
delivery goals. It extends the per-engineer Goals tab with consistent,
deterministic health signals that help a manager decide where discussion or
support is needed.

## Goal health

Active goals are assigned one derived health state:

- `blocked`: the goal status is explicitly Blocked.
- `overdue`: the target date has passed.
- `at_risk`: progress trails the percentage of elapsed time between the start
  and target dates by at least 20 percentage points.
- `on_track`: none of the conditions above apply.

Blocked takes precedence over overdue. An active goal without both a start and
target date is not inferred to be at risk. These signals organize work; they
are not employee ratings.

## Dashboard functionality

- Shows counts for all active, blocked, overdue, at-risk, and on-track goals.
- Filters by health, engineer, priority, and review cycle.
- Shows actual progress and schedule-expected progress.
- Links directly to the engineer's Goals tab.
- Sorts goals requiring attention before on-track goals.

## API

`GET /api/dashboard/goals`

Supported query parameters:

- `includeClosed`: defaults to `false`; set to `true` to include completed and
  cancelled goals.
- `health`: `blocked`, `overdue`, `at_risk`, or `on_track`.
- `status`: any supported goal status.
- `priority`: `low`, `medium`, or `high`.
- `engineerId`: a positive engineer ID.
- `reviewCycle`: an exact review-cycle value.

Each response includes `health`, `daysToTarget`, and `expectedProgress` as
derived fields. No derived value is persisted to the goal record.
