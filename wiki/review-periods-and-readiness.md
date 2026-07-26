# Review Periods and Review-Cycle Readiness

Structured review periods add calendar dates to the review-cycle labels already
used by engineers, evidence, goals, and recognition. The Review-Cycle Readiness
dashboard uses those dates and existing records to provide a preparation
checklist for every engineer.

## Review-period management

Review periods are managed from Settings. Each period has:

- a unique label;
- a start date; and
- an end date.

The label must exactly match the review-cycle value assigned to an engineer.
On first initialization, the database is seeded with H1 and H2 periods from the
previous year through two years ahead. This seed runs only once, so users can
rename unassigned periods, change dates, add organization-specific cycles, or
delete defaults without them returning after a restart.

Existing free-form labels remain valid and appear as unconfigured until a
matching structured period is created.

Configured labels are loaded into the performance-evidence editor on engineer
profiles. Generated H1/H2 labels and a record's historical label remain
available as fallbacks so existing evidence can always be edited safely.

Periods are classified from their dates as planned, active, or closed. Dates
can be edited. A label cannot be renamed while an engineer is assigned to it.
Deleting a period never rewrites historical records; engineers retaining that
label appear as Period Not Configured until a matching period is created.

### Review-period API

- `GET /api/review-periods`
- `POST /api/review-periods`
- `PUT /api/review-periods/{id}`
- `DELETE /api/review-periods/{id}`

## Readiness checklist

An engineer is Ready when the current cycle has:

- at least 3 performance-evidence notes;
- evidence covering at least 2 categories;
- at least 1 linked goal;
- at least 1 recognition record;
- at least 1 completed 1:1 within the period dates; and
- no overdue open or in-progress follow-ups.

If any check is missing, readiness is Needs Attention and the response lists
the individual missing items. If the engineer's cycle has no matching
structured period, readiness is Period Not Configured.

These thresholds are transparent workflow defaults. Readiness is a preparation
checklist, not a performance score, rating, or recommendation.

## Dashboard functionality

- Shows counts for Ready, Needs Attention, and Period Not Configured.
- Filters by readiness, engineer, team, review cycle, and cycles ending within
  30, 60, or 90 days.
- Shows review-period phase and days remaining.
- Shows counts for evidence, evidence categories, goals, recognition, completed
  1:1s, and overdue actions.
- Lists every missing checklist item.
- Links to the engineer profile for preparation work.

### Readiness API

`GET /api/dashboard/review-readiness`

Supported query parameters:

- `readiness`: `ready`, `needs_attention`, or `unconfigured`.
- `engineerId`: a positive engineer ID.
- `team`: an exact team value.
- `reviewCycle`: an exact review-cycle label.
- `endingWithinDays`: an integer from 1 through 365.
