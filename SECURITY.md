# Security Policy

## Supported versions

Only the latest release published on the
[Releases page](https://github.com/freepik-company/jev-mcp/releases) receives
security fixes. Please upgrade before reporting.

## Reporting a vulnerability

Please **do not** open a public issue for security problems.

Report them privately through
[GitHub private vulnerability reporting](https://github.com/freepik-company/jev-mcp/security/advisories/new).
Include the version, the provider (`JEV_PROVIDER`), a minimal reproduction and
the impact you see. Never include real API keys in a report.

We aim to acknowledge reports within 5 working days and to publish a fix and
an advisory as soon as one is available. Reporters are credited in the advisory
unless they prefer otherwise.

## What this server does to protect you

- Credentials are read only from the server environment. They are never
  accepted as tool arguments, never echoed in error messages and never
  forwarded on HTTP redirects.
- `BASE_URL` must be HTTPS; plain HTTP is allowed on loopback only, for local
  proxies and tests.
- Request and response sizes are bounded (1 MiB in, 2 MiB out) and every call
  has a 30-second timeout. Paid inference calls are never retried automatically.
- Provider answers are validated against the question you asked before they are
  returned. Answers outside the offered options or with inconsistent
  probabilities fail the call.
- Release binaries and images are built by CI from a tagged commit. Archives are
  published with SHA-256 checksums; images carry OCI source labels.

Data you send to a tool is forwarded to the configured provider (OpenRouter or
TypeSafe). Review their terms before sending sensitive content.
