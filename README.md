# Performance Review Dashboard

A local-first Engineering Manager workspace for tracking engineers,
performance evidence, screenshots, goals, 1:1 history, structured follow-ups,
recognition, a unified engineer timeline, review-cycle context, and encrypted
integration configuration.

## Prerequisites

- Go 1.25.7 or a compatible newer Go toolchain
- Node.js 22 or newer
- npm

## Local setup

Clone the repository and open PowerShell in its root directory.

### 1. Prepare the backend

The backend expects its data directory to exist:

```powershell
cd backend
New-Item -ItemType Directory -Force data
go mod download
```

### 2. Configure credential encryption and start the backend

The encryption key is required only when saving integration credentials.
Generate a Base64-encoded 32-byte key in the same PowerShell session used to
run the backend:

```powershell
$keyBytes = New-Object byte[] 32
$generator = [Security.Cryptography.RandomNumberGenerator]::Create()
$generator.GetBytes($keyBytes)
$generator.Dispose()
$env:MANAGER_DASHBOARD_ENCRYPTION_KEY = [Convert]::ToBase64String($keyBytes)
go run .
```

If integration credentials are not needed, `go run .` can be executed without
setting the environment variable.

The API starts at `http://localhost:8080`. SQLite creates
`backend/data/performance.db` automatically on first launch.

Save the key in a secure local secret manager if encrypted credentials must
remain usable across backend restarts. Do not commit the key or an `.env`
file.

### 3. Start the frontend

Open a second PowerShell window:

```powershell
cd frontend
npm install
npm run dev
```

Open `http://localhost:5173`.

### 4. Configure attachment storage

In the application:

1. Open **Settings**.
2. Enter a writable local directory under **Local Attachment Storage**.
3. Save the storage location.

Screenshots are stored beneath that directory; metadata and relative paths are
stored in SQLite.

## Verification

Backend:

```powershell
cd backend
go test -count=1 -cover ./...
go vet ./...
```

Frontend:

```powershell
cd frontend
npm run lint
npm run build
```

## Documentation

See the [wiki index](wiki/README.md) for separate guides covering every
implemented feature:

- Dashboard and metrics
- Engineer profiles
- Performance evidence
- Evidence attachments
- Goals and development plans
- 1:1 management
- Structured follow-up management
- Recognition
- Engineer timeline
- Settings and themes
- Integrations and credential security
- Local data and API behavior

The long-term direction is documented in the
[Engineering Manager OS roadmap](wiki/engineering_manager_os_roadmap.md).

## Local data safety

- SQLite databases and WAL files are ignored by Git.
- Integration keys must remain outside the repository.
- Saved integration secrets are encrypted before being written to SQLite.
- The application is designed to run locally by default.
