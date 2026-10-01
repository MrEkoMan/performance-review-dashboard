# Local Data and API

## Local-first storage

The backend stores application records in:

```text
backend/data/performance.db
```

SQLite tables are initialized automatically when the backend starts. Database
files, WAL files, environment files, and coverage artifacts are ignored by
Git.

Attachments are stored separately under the root selected in Settings.

## Runtime services

- Backend API: `http://localhost:8080`
- Frontend development server: `http://localhost:5173`

The frontend calls the backend under `/api`. CORS currently allows the local
Vite development origin.

## API groups

- Authentication: `/api/auth/login`, `/api/auth/logout`, `/api/auth/me`
- Engineers: `/api/engineers`
- Performance notes: `/api/notes`
- Goals: `/api/engineers/{engineerId}/goals` and `/api/goals/{id}`
- 1:1 records: `/api/engineers/{engineerId}/one-on-ones` and
  `/api/one-on-ones/{id}`
- Attachments: `/api/notes/{id}/attachments` and `/api/attachments/{id}`
- Settings: `/api/settings`
- Integrations: `/api/integrations`
- Engineer portal access (manager): `/api/engineers/{engineerId}/portal-access`

See the individual feature pages in the [wiki index](README.md) for endpoint
details.

## Authentication

Once any account exists (a manager seeded via `MANAGER_EMAIL` /
`MANAGER_PASSWORD`, or an engineer granted portal access), every API call
except the auth endpoints requires a session. Sessions are carried by an
`HttpOnly` cookie (`md_session`, 7-day expiry); the frontend sends
`credentials: "include"` on every request and CORS allows credentials from
the local Vite origin.

Engineer sessions are scoped server-side: engineer-scoped routes accept only
the engineer's own `engineerId`, `/api/notes` is filtered to their records,
`private_manager_notes` on 1:1s is redacted in responses, and manager-only
surfaces (dashboards, settings, integrations, AI analyses, Jira evidence)
return 403. See [Engineer portal access](engineer-portal-access.md).

## Testing

Backend tests use isolated in-memory SQLite databases and temporary attachment
directories. The suite covers HTTP routes, validation, database constraints,
encryption, storage safety, and rollback paths.

Run:

```powershell
cd backend
go test -count=1 -cover ./...
go vet ./...
```
