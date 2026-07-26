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

The application currently stores and manages integration configuration only.
It does not yet:

- Test connections
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
