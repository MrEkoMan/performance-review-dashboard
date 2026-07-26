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

- Engineers: `/api/engineers`
- Performance notes: `/api/notes`
- Goals: `/api/engineers/{engineerId}/goals` and `/api/goals/{id}`
- 1:1 records: `/api/engineers/{engineerId}/one-on-ones` and
  `/api/one-on-ones/{id}`
- Attachments: `/api/notes/{id}/attachments` and `/api/attachments/{id}`
- Settings: `/api/settings`
- Integrations: `/api/integrations`

See the individual feature pages in the [wiki index](README.md) for endpoint
details.

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
