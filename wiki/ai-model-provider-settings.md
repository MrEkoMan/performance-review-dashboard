# AI Model Provider Settings

The application can store configuration for five hosted AI providers and a
local Ollama server. This feature stores configuration only; it does not yet
send performance data to any model.

## Recommended hosted providers

- OpenAI
- Anthropic
- Google Gemini
- Azure OpenAI
- OpenRouter

OpenRouter is included as a multi-model gateway with an OpenAI-compatible API.
AWS Bedrock is deferred because supporting it properly requires IAM access-key,
secret-key, session-token, region, and credential-chain decisions rather than
a single API key.

## Local provider

Ollama configuration supports loopback addresses such as:

```text
http://localhost:11434
```

The backend accepts HTTP for Ollama only when the hostname or IP address is
loopback. Remote Ollama hosts are rejected so a local configuration cannot
silently become an unencrypted network connection.

## Configuration fields

Each provider supports:

- Configuration label
- Base URL
- Model identifier, model slug, or Azure deployment name
- Optional API version
- Enabled status
- Encrypted API key for hosted providers

Model names are intentionally not seeded because provider catalogs and aliases
change frequently. Users enter the model identifier supported by their account
and provider.

Hosted provider base URLs must use HTTPS. API keys are encrypted using the same
AES-256-GCM key used for integration credentials and are never returned to the
browser. Saving an existing hosted configuration with a blank API-key field
preserves the encrypted key.

## APIs

- `GET /api/ai-providers`
- `PUT /api/ai-providers/{provider}`
- `DELETE /api/ai-providers/{provider}`

Supported provider identifiers are `openai`, `anthropic`, `gemini`,
`azure_openai`, `openrouter`, and `ollama`.

## Current boundary

This feature does not:

- test provider connections;
- submit prompts;
- generate summaries or review drafts;
- select models automatically; or
- send employee data outside the application.

Those actions require a separate privacy, prompt, consent, and data-handling
design before implementation.
