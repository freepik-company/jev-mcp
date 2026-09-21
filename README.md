# Jev MCP

An independent stdio MCP server for typed decisions with Jev/System One, using
**OpenRouter or TypeSafe directly**. One tool, `decide`, takes a shared `state`
and named `choice`, `noul`, or `score` questions. It returns the provider's answers,
probabilities, confidence, model metadata, and usage, including cost when supplied.

No Altherium dependency. The only external Go dependency is the official
[MCP SDK](https://github.com/modelcontextprotocol/go-sdk).

## Run with Docker

```sh
make image
# Set OPENROUTER_API_KEY securely in your shell first. Never put its value in a command.
docker run --rm -i -e OPENROUTER_API_KEY jev-mcp:local
```

Use TypeSafe directly:

```sh
# Set TYPESAFE_API_KEY securely in your shell first.
docker run --rm -i -e JEV_PROVIDER=typesafe -e TYPESAFE_API_KEY jev-mcp:local
```

`-i` is required for stdio; **do not add `-t`**. The process waits for MCP messages,
not an interactive prompt. JSON-RPC is the only output on stdout.

For clients that launch executables, download a binary from
[Releases](https://github.com/freepik-company/jev-mcp/releases), verify its archive
against `checksums.txt`, extract it, and register `jev-mcp` as a stdio server.
The repository is private, so downloads require GitHub access. Releases include
Linux and macOS binaries for amd64 and arm64; Windows users can use Docker.

Example MCP client configuration (the client process must inherit the environment):

```json
{
  "mcpServers": {
    "jev": {
      "command": "docker",
      "args": ["run", "--rm", "-i", "-e", "OPENROUTER_API_KEY", "jev-mcp:local"]
    }
  }
}
```

If a client filters its environment, configure the credential in that client's
protected server environment. Credentials are never tool arguments. Altherium
resolves `${secrets:...}` into the child process environment; it does not require
a reverse proxy.

## Configuration

| Variable | Meaning |
| --- | --- |
| `JEV_PROVIDER` | `openrouter` (default) or `typesafe`; explicit selection, no guessing from key format |
| `OPENROUTER_API_KEY` | Credential for OpenRouter |
| `TYPESAFE_API_KEY` | Credential for TypeSafe |
| `API_KEY` | Optional provider-neutral override; takes precedence over the provider-specific variable |
| `JEV_MODEL` | Default System One model, `jev-latest`; `decide.model` can override it |
| `BASE_URL` | Optional compatible API root; `/v1/systemone` is appended while retaining its path prefix |

Provider defaults:

| Provider | Base URL | Final endpoint |
| --- | --- | --- |
| OpenRouter | `https://openrouter.ai/api` | `https://openrouter.ai/api/v1/systemone` |
| TypeSafe | `https://api.typesafe.ai` | `https://api.typesafe.ai/v1/systemone` |

OpenRouter accepts `jev-latest` and maps bare TypeSafe IDs to its own namespace.
For a pinned release use an ID supported by the selected provider. An OpenRouter
prefixed ID is not necessarily valid at TypeSafe. See
[OpenRouter's System One integration](https://openrouter.ai/docs/guides/community/typesafe-sdk)
and [TypeSafe's request/response types](https://github.com/typesafe-ai/typesafe-sdk-js/blob/main/src/types.ts).

## Tool: `decide`

```json
{
  "state": {"ticket": "Please refund the duplicate charge on my account."},
  "questions": {
    "department": {
      "type": "choice",
      "instructions": "Which team should handle this ticket?",
      "criteria": {
        "billing": "Charges, payments, refunds",
        "technical": "Software defects and outages",
        "other": "None of the other teams fits"
      }
    },
    "refund": {"type": "noul", "instructions": "Does the customer ask for a refund?"},
    "urgency": {
      "type": "score",
      "instructions": "How urgently does this ticket need attention?",
      "criteria": ["Routine question", "Customer unable to use the service", "Active financial harm"]
    }
  }
}
```

- **Choice:** one option you supplied, its distribution and a separate confidence.
- **Noul:** P(true), between 0 and 1; not a boolean, and no confidence field is invented.
- **Score:** probability-weighted rubric index, between 0 and `len(criteria)-1`.
  The original legend, distribution and confidence are preserved.

The tool requires explicit instructions and a non-null state to keep judgments
intentional. Choice requires at least two alternatives; Score requires at least
two levels. Unknown input fields, including nested question fields, are rejected
before any paid API call. All questions are sent together, without truncation or
extra model-generated instructions. See the versioned
[input](input.schema.json) and [output](output.schema.json) schemas.

Use probabilities and confidence as separate signals. This server never turns a
missing answer into a favorable decision, never invents `auto`/`approve` policies,
and does not replace application authorization or Altherium governance gates.
See [recipes](docs/recipes.md) for routing, evidence checks and scoring.

## Failure behavior and limits

- 30-second HTTP timeout, caller cancellation, 1 MiB serialized request limit,
  2 MiB response limit. Oversized input is rejected, not truncated.
- No automatic retries of paid calls. HTTP errors report status, never the raw
  upstream body, transport URL, headers or credential.
- HTTPS is required; HTTP is allowed only for explicitly configured loopback
  endpoints. Redirects are rejected, including same-host redirects.
- Missing, null, extra or mismatched answers, unoffered options, incomplete
  probability maps and scores outside their rubrics fail the entire call.
- Distributions must sum to 1 within `0.01 + 1e-12`; a Choice must be an argmax
  (ties allowed within `1e-12`); a Score must match the weighted mean within
  `0.01 * (number of levels - 1) + 1e-12`, and its legend must match the requested
  rubric. These are **our explicit rounding allowances**, not provider guarantees.
- Failure is an MCP error, not a business decision or an all-clear response.
- `usage.cost` is passed through when provided. Nothing writes Altherium's run
  budget, and TypeSafe responses without a cost are not assigned an invented one.

## Development and releases

Only Docker is required. All build/test commands run inside containers:

```sh
make test       # gofmt check, go vet, race-enabled tests; no API key or paid calls
make image      # minimal non-root runtime image
make release    # Linux/macOS amd64/arm64 archives and SHA-256 checksums in dist/
```

Tests exercise MCP initialize/list/call, actual stdio subprocess startup, provider
selection, exact HTTP payload/auth, all three primitives, unchanged cost and
metadata, input rejection, invalid upstream answers, cancellation, redirects,
limits and credential redaction. Mock coverage is not a claim about model accuracy.

For an opt-in, paid live smoke test, run `scripts/smoke.py <server command>` with
Python 3 inside your test container and the provider credential in its environment.
It initializes MCP, discovers `decide`, and checks all three primitives in one
synthetic request. It prints results and usage, never the credential. See
[validation evidence](docs/validation.md) for what was actually tested live.

CI runs the same container commands. A `v*` tag publishes archives only after
verification succeeds. See [design notes](docs/design.md) for the boundary and
the community issues used to select regression cases.
