# Jev MCP

A stdio MCP server for [Jev / System One](https://github.com/typesafe-ai/typesafe-sdk-js)
typed decisions, through **OpenRouter or TypeSafe directly**.
One tool, `decide`, returns answers, probabilities, confidence, and provider usage.

## Quick start

Download a binary from [Releases](https://github.com/freepik-company/jev-mcp/releases)
and register `jev-mcp` as a stdio server in your MCP client. Supply
`OPENROUTER_API_KEY` through the server's environment.

Or build and run with Docker:

```sh
make image
docker run --rm -i -e OPENROUTER_API_KEY jev-mcp:local
```

For TypeSafe directly:

```sh
docker run --rm -i -e JEV_PROVIDER=typesafe -e TYPESAFE_API_KEY jev-mcp:local
```

Set the key in your shell before running these commands. Use `-i` without `-t`.
Example client configuration, with the credential inherited from its environment:

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
Credentials belong in the server environment, never in tool arguments.

## Tool: `decide`

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

Unknown input fields and incomplete or inconsistent answers fail the call.
Requests have a 30-second timeout, a 1 MiB input limit, and a 2 MiB response limit.
Paid calls are not automatically retried. See the [input](internal/mcp/input.schema.json)
and [output](internal/mcp/output.schema.json) schemas for the complete contract.

## Development

Only Docker is required:

```sh
make test       # formatting, vet, race-enabled unit and integration tests
make image      # runtime image
make release    # Linux/macOS amd64/arm64 archives and checksums in dist/
```

`cmd/jev-mcp` contains the entry point. `internal/` separates configuration,
the System One client and answer validation, and the MCP protocol.
CI verifies builds and tests; version tags publish release archives.

## License

[Apache License 2.0](LICENSE).
