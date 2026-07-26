# Engineer Profiles

## Purpose

Each engineer has a dedicated workspace at `/engineers/{engineerId}` that
combines identity, career context, evidence, goals, and summary metrics.

## Engineer fields

- Name
- Role
- Level
- Team
- Career goal
- Review cycle

Name, role, level, team, and review cycle are required when creating an
engineer.

## Profile functionality

- Displays role, level, team, career goal, and review cycle.
- Shows total evidence and open follow-up counts.
- Shows the date of the most recent evidence record.
- Displays category-based evidence metrics.
- Loads the engineer's evidence and goals.
- Loads the engineer's 1:1 history and next-meeting context.
- Allows evidence creation, editing, and deletion.
- Allows goal creation, editing, filtering, and deletion.
- Allows 1:1 creation, editing, status filtering, and deletion.

## APIs

- `GET /api/engineers`
- `POST /api/engineers`

The current implementation supports engineer creation and listing. Editing and
deleting an engineer are not yet implemented.
