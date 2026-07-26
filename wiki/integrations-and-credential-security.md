# Integrations and Credential Security

## Current functionality

The settings page can store configuration for:

- GitHub
- Jira
- Slack
- Microsoft Teams

Each configuration includes:

- Account or workspace label
- Base URL
- Secret or token
- Enabled status
- Last-updated timestamp

Saved configurations can be replaced or deleted. API responses indicate
whether a secret exists but never return the secret itself.

Each configured and enabled provider can be tested from Settings. Connection
tests run only in the Go backend, decrypt the saved credential in memory, apply
an eight-second timeout, and never return the credential to the browser.

Provider checks use:

- GitHub: `GET /user`.
- Jira Cloud: `GET /rest/api/3/myself` using account email and API token.
- Slack: the non-mutating `auth.test` method.
- Microsoft Teams: Microsoft Graph `GET /v1.0/me` using a delegated access
  token.

Incoming Teams webhooks are not tested because doing so would send a message.
Connection results distinguish configuration, authentication, authorization,
rate-limit, provider, network, timeout, and invalid-response failures.

## Encryption

Secrets are encrypted with AES-256-GCM before they are written to SQLite. The
encryption key is supplied through:

```text
MANAGER_DASHBOARD_ENCRYPTION_KEY
```

The value must be a Base64-encoded 32-byte key. The key is not stored in the
database and must remain stable between runs if previously saved credentials
need to be decrypted later.

## Current boundary

The application stores integration configuration and supports read-only
connection tests. It does not yet:

- Synchronize data
- Import GitHub or Jira activity
- Import Slack or Teams messages
- Track synchronization history

Those capabilities are described in the
[Engineering Manager OS roadmap](engineering_manager_os_roadmap.md).

## APIs

- `GET /api/integrations`
- `PUT /api/integrations/{provider}`
- `DELETE /api/integrations/{provider}`
- `POST /api/integrations/{provider}/test`
