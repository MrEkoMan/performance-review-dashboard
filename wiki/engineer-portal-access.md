# Engineer Portal Access

Engineers can be granted a login to the dashboard. A signed-in engineer sees
only their own records and can contribute self-context that the manager
reviews as evidence. All access rules are enforced server-side.

## Granting access

The manager opens an engineer's profile and uses the **Portal Access** card on
the Overview tab:

- **Grant access** — assign an email and an initial password (minimum 8
  characters; use **Generate** for a random one). The password is shown once
  and must be shared out-of-band (chat, in person). It is never retrievable
  from the app afterwards.
- **Reset password** — sets a new password and immediately signs out the
  engineer's existing sessions.
- **Revoke access** — removes the login entirely; the engineer is signed out
  and can no longer reach the app.

## Manager bootstrap

There are two ways to create the manager (administrator) account:

1. **Register through the UI (recommended).** While no accounts exist, the
   login page shows a "Create manager account" form. The first registered
   account becomes the administrator and is signed in immediately. If engineer
   portal access was granted first (engineer-only accounts exist without a
   manager), the claiming form is deliberately withheld so a former engineer
   cannot promote themselves; use option 2 to recover.
2. **Environment variables.** Set `MANAGER_EMAIL` and `MANAGER_PASSWORD` the
   first time the backend starts:

```powershell
$env:MANAGER_EMAIL = "you@example.com"
$env:MANAGER_PASSWORD = "your-password"
cd backend
go run .
```

Seeding is idempotent — the variables can be removed after the first start.
Once a manager exists, a signed-in manager can register additional accounts
(engineer accounts must be bound to an engineer record) from
`POST /api/auth/register`.

### Open mode

If **no accounts exist at all**, the app behaves exactly as it did before
authentication: every request is treated as the manager and no login is
required. The moment one account exists (a registered or seeded manager, or
any granted engineer login), sessions are required for everything except the
login endpoints — and the login page offers to create the manager account
while no manager exists.

## What an engineer can do

- See only their own profile: evidence notes, goals, development plans,
  follow-ups, recognition, timeline, and their onboarding profile (read-only).
- Add **self-context** entries on the My Context tab: what they have been
  working on, wins, and challenges. These land in the evidence list flagged
  **Engineer-provided** and flow into review readiness and AI analysis like
  manager-recorded evidence.
- Edit or delete only their own self-context entries.

## What an engineer cannot do

- See or touch any other engineer's data (server-side 403).
- Edit manager-recorded evidence, goals, 1:1s, follow-ups, recognition,
  development plans, or their engineer profile.
- See **private manager notes** on 1:1s — those fields are stripped by the
  API before the response is written.
- Use the AI Insights tab. Stored AI analyses can reference private manager
  notes (the analysis prompt includes them), so analyses are manager-only
  until the prompt source is filtered.
- Access dashboards, settings, integrations, AI provider configuration,
  Jira evidence harvesting, or review-period management.

## Technical notes

- Passwords are hashed with PBKDF2-HMAC-SHA256 (600,000 iterations, per-entry
  salt) using only the Go standard library; no plaintext is stored.
- Sessions are opaque random tokens in an `HttpOnly`, `SameSite=Lax` cookie;
  only the SHA-256 of the token is stored, so the database cannot be used to
  resurrect live sessions. Sessions expire after 7 days.
- The API rejects unauthenticated requests with 401 once any account exists,
  and cross-engineer access with 403.
