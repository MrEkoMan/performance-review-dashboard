# Feature Backlog

This page collects the candidate features identified for expanding the
usefulness of the dashboard. They are ordered roughly by usefulness-per-effort.
Nothing on this page is implemented yet; the implemented feature set is
documented in the [wiki index](README.md).

## 1. Integration-driven evidence (highest value)

**Goal:** stop depending on manual note entry by harvesting evidence from the
systems already configured in Integrations.

Pull completed issues, merged merge requests / pull requests, and code reviews
for an engineer over a review period from Jira, GitLab, and GitHub. Surface
them as **suggested evidence notes** on the engineer's profile (for example:
"Jira PROJ-1234 closed on 2026-09-15 — promote to a note?"), which the manager
accepts or edits before they are persisted.

Why it matters: evidence recency and review readiness currently depend entirely
on the manager recording what engineers did. This is the only proposed feature
that reduces manual data-entry burden rather than just displaying existing data
better. The plumbing (credential storage, connection testing, decryption) and
the GitLab/GitHub/Jira provider connections already exist.

Sub-features worth considering:

- One-click "promote to note" with pre-filled category, summary, and date.
- A review-period date-range picker on the harvesting UI.
- Deduplication so the same Jira/GitLab item is never suggested twice.

## 2. Review document generation

**Goal:** produce the final review artifact, not just the inputs.

Assemble notes, goals, recognition, and 1:1 themes into a structured review
document per engineer, scoped to a review period, and export it to Markdown or
PDF. Optionally AI-assisted using the existing AI provider configuration.

The data model is already shaped for this: review periods, readiness scoring,
per-engineer evidence, and the AI provider adapters all exist.

## 3. Additional AI modes

**Goal:** reuse the existing AI provider adapter layer (`ai_clients.go`) for
more purpose-specific prompts. Each mode is a cheap addition because the
provider configuration, analysis persistence, and UI pattern (the AI Insights
panel) already exist.

- **1:1 agenda generator** — given the gap since the last 1:1 and open
  follow-ups, draft talking points before the meeting.
- **Development plan draft** — feed career goal, goal history, and AI analysis
  growth areas into a first-draft development plan.
- **Goal health diagnosis** — for blocked or overdue goals, suggest unblocking
  actions.

## 4. Proactive nudges (Slack / Teams)

**Goal:** turn the dashboard from something the manager must remember to open
into something that reaches them.

A lightweight scheduler sends the manager a daily or per-event message over
Slack or Microsoft Teams: upcoming 1:1s today, overdue follow-ups, and goals
turning red. The Teams/Slack credentials are already stored and testable; the
Teams connection test deliberately avoids messaging, so notification sending
would use the user's own token / bot token explicitly.

## 5. Engineer self-service (lighter weight)

**Goal:** let engineers contribute wins the manager never sees, without
changing who owns the record.

A read-only link per engineer where they can view their own profile and
**propose** note or recognition entries for manager approval. The manager
accepts or edits before anything is persisted. Requires thinking about access
control, which does not exist today (the API has no authentication layer).

## 6. Smaller hygiene wins

- **CSV / Excel export** of any table (notes, follow-ups, review readiness) for
  HR processes.
- **Team-level roll-up page** — evidence recency and review readiness already
  aggregate; a per-team heat-map view would help prioritize who needs attention
  this week.
- **Search across all evidence** — a single query box over notes, goals, and
  1:1 text.

## Comparison of the top candidates

| Feature | Effort | Ongoing value | Depends on |
| --- | --- | --- | --- |
| Integration-driven evidence | Medium–High | Reduces manual data entry every week | Existing integration credentials |
| Review document generation | Medium | High at review time only | AI providers (optional) |
| Additional AI modes | Low | Recurring, cheap per use | AI providers |
| Proactive nudges | Medium | Turns reactive into proactive | Slack/Teams credentials |
| Engineer self-service | Medium–High | Captures invisible wins | Needs auth layer |
| Export / roll-up / search | Low–Medium | Occasional but broad | None |
