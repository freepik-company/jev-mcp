# Jev MCP

[![CI](https://github.com/freepik-company/jev-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/freepik-company/jev-mcp/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/freepik-company/jev-mcp)](https://github.com/freepik-company/jev-mcp/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/freepik-company/jev-mcp.svg)](https://pkg.go.dev/github.com/freepik-company/jev-mcp)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

A small, dependency-light **MCP server** that gives any MCP client typed,
probabilistic decisions from [Jev](https://docs.typesafe.ai/concepts/system-one),
TypeSafe's System One model, through **OpenRouter** or **TypeSafe** directly.

Jev does not write text. You send it data plus typed questions and it returns
one calibrated answer per question: a choice among your options, a probability
for a yes/no, or a position on a scale you define, each with a confidence and
the full probability distribution. It answers in a single parallel pass, fast
and at a fraction of the cost of a chat model, and it cannot return an option
you did not offer.

This server exposes that as five MCP tools, validates every answer the provider
returns, never leaks your credential, and never retries a paid call.

## Why use it

- **Let your agent decide instead of guess.** Routing, triage, classification,
  claim checking and ranking become explicit, typed calls with probabilities
  your workflow can act on or escalate.
- **Cheap and fast.** One paid inference call per tool invocation, regardless
  of how many items or questions it carries.
- **Provider agnostic.** Use an OpenRouter key you already have, or a TypeSafe
  key. Switch with one environment variable.
- **Honest outputs.** Answers are validated against the question you asked;
  inconsistent or incomplete answers fail the call instead of being fixed up.
  The unchanged provider response, including `usage.cost`, is always returned.
- **Safe by construction.** Credentials live only in the server environment,
  HTTPS is enforced, redirects are refused, request and response sizes and time
  are bounded.

## Quick start

### 1. Get an API key

- **OpenRouter** (default): create a key at [openrouter.ai](https://openrouter.ai/) and
  set `OPENROUTER_API_KEY`. Jev is listed as
  [`typesafe/jev-latest`](https://openrouter.ai/typesafe); the bare `jev-latest` id also works.
- **TypeSafe**: create a key at [typesafe.ai](https://typesafe.ai/) and set
  `TYPESAFE_API_KEY` together with `JEV_PROVIDER=typesafe`.

### 2. Install the server

Pick one:

**Binary.** Download the archive for your platform from
[Releases](https://github.com/freepik-company/jev-mcp/releases) (Linux, macOS
and Windows, amd64 and arm64), verify it against `checksums.txt` and put
`jev-mcp` on your `PATH`.

**Container.** Images are published for every tagged release:

```sh
docker pull ghcr.io/freepik-company/jev-mcp:latest
```

**Go toolchain.**

```sh
go install github.com/freepik-company/jev-mcp/cmd/jev-mcp@latest
```

### 3. Register it in your MCP client

The server speaks MCP over stdio and reads the credential from its environment.
Never put the key in tool arguments.

**Claude Code**

```sh
claude mcp add jev -e OPENROUTER_API_KEY="$OPENROUTER_API_KEY" -- jev-mcp
```

**Claude Desktop, Cursor and other JSON-configured clients**

```json
{
  "mcpServers": {
    "jev": {
      "command": "jev-mcp",
      "env": { "OPENROUTER_API_KEY": "sk-or-..." }
    }
  }
}
```

**Container instead of a binary** (use `-i`, never `-t`; the key is inherited
from the client's environment):

```json
{
  "mcpServers": {
    "jev": {
      "command": "docker",
      "args": ["run", "--rm", "-i", "-e", "OPENROUTER_API_KEY", "ghcr.io/freepik-company/jev-mcp:latest"]
    }
  }
}
```

For TypeSafe directly, add `"JEV_PROVIDER": "typesafe"` and `TYPESAFE_API_KEY`
instead of the OpenRouter key.

The server is also listed in the [MCP Registry](https://registry.modelcontextprotocol.io)
as `io.github.freepik-company/jev-mcp`, so clients that browse the registry can
install it from there.

## Configuration

| Variable | Default / purpose |
| --- | --- |
| `JEV_PROVIDER` | `openrouter` or `typesafe` (default: `openrouter`) |
| `OPENROUTER_API_KEY` | OpenRouter credential |
| `TYPESAFE_API_KEY` | TypeSafe credential |
| `API_KEY` | Overrides the selected provider's credential |
| `JEV_MODEL` | `jev-latest`; overridable per tool call with `model` |
| `BASE_URL` | Provider API root; `/v1/systemone` is appended |

Default roots: `https://openrouter.ai/api` and `https://api.typesafe.ai`.
`BASE_URL` must be HTTPS; plain HTTP is accepted on loopback only, for local
proxies and tests. Credentials belong in the server environment, never in tool
arguments.

## Tools

| Tool | Input | Result |
| --- | --- | --- |
| `decide` | Shared `state` and named `questions` | Raw typed answers and provider metadata |
| `classify` | `items`, `categories`, `instructions` | Category per item, confidence and probabilities |
| `verify` | `claims`, `evidence` | `supported`, `contradicted` or `insufficient_evidence` per claim |
| `rerank` | `query`, `candidates`, optional `top_k` | Descending relevance scores on a 0–4 rubric |
| `list_models` | No arguments | Compatible model IDs and the configured default |

Items, claims, evidence and candidates are arrays of `{ "id": "unique-id", "text": "..." }`
with 1–64 entries. Categories map category IDs to descriptions. The three task tools
accept an optional `model`, make one inference call, and return `results` plus the
unchanged provider `response` (including `response.usage.cost` when supplied).
Classification and verification preserve input order; ranking preserves it for ties.
`top_k` filters results after evaluating every candidate. Inputs are never truncated.

Verification uses only supplied evidence; its verdict is a model judgment, not a
proof or approval. No automatic acceptance thresholds are imposed. Model discovery
uses OpenRouter's decisions catalogue or TypeSafe's native catalogue, without inference.

### Example: `decide`

```json
{
  "state": {"ticket": "Please refund the duplicate charge."},
  "questions": {
    "department": {
      "type": "choice",
      "instructions": "Which team should handle this ticket?",
      "criteria": {"billing": "Payments and refunds", "technical": "Software defects"}
    },
    "refund": {"type": "noul", "instructions": "Is a refund requested?"},
    "urgency": {
      "type": "score",
      "instructions": "How urgent is this ticket?",
      "criteria": ["Routine", "Service blocked", "Active financial harm"]
    }
  }
}
```

`choice` selects a supplied option; `noul` returns P(true); `score` returns a
probability-weighted rubric index. Full provider output is preserved, including
score legends and `usage.cost` when available.

### Example: `classify`

```json
{
  "instructions": "Route each support message to the right queue.",
  "categories": {
    "billing": "Charges, invoices and refunds",
    "technical": "Bugs, errors and outages",
    "other": "Anything else"
  },
  "items": [
    {"id": "m1", "text": "I was charged twice this month."},
    {"id": "m2", "text": "The export button does nothing."}
  ]
}
```

Returns one `{ "id", "category", "confidence", "probabilities" }` per item, in
input order, plus the full provider `response`.

### Limits and failure modes

Unknown input fields and incomplete or inconsistent answers fail the call.
Requests have a 30-second timeout, a 1 MiB input limit, and a 2 MiB response limit.
Paid calls are not automatically retried. Provider errors surface only their HTTP
status, never the response body, so a credential echoed by a provider cannot
reach the model. See the [input](internal/mcp/input.schema.json) and
[output](internal/mcp/output.schema.json) schemas for the complete contract.

## Development

Only Docker is required:

```sh
make test       # formatting, vet, race-enabled unit and integration tests
make image      # runtime image, tagged jev-mcp:local
make release    # archives and checksums for every platform in dist/
```

`cmd/jev-mcp` contains the entry point. `internal/` separates configuration,
the System One client and answer validation, the task layer and the MCP protocol.
CI verifies builds and tests on every push and pull request; version tags publish
release archives, the container image on GHCR and the entry in the MCP Registry.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the workflow and design principles,
[SECURITY.md](SECURITY.md) for reporting vulnerabilities and
[CHANGELOG.md](CHANGELOG.md) for release notes.

## License

[Apache License 2.0](LICENSE). Jev and System One are products of
[TypeSafe AI](https://typesafe.ai/); this project is an independent client and
is not affiliated with TypeSafe or OpenRouter.
