# Design boundary

`main.go` selects provider configuration and runs stdio. `server.go` owns the MCP
tool and public schemas. `client.go` sends one System One request and checks the
returned answers against its questions. No file reads, database, Altherium import,
agent orchestration, chat model adapter or secret store is built into the server.

The same protocol client serves OpenRouter and TypeSafe. Only configuration
changes. `BASE_URL` provides a compatible endpoint escape hatch without creating
another provider implementation or making URLs/credentials model-controlled.

## Community regression references

Reviewed on 2026-09-21. This is an independent implementation, not a fork.

| Reference | Requirement covered here |
| --- | --- |
| [jkudish/jev-mcp #9](https://github.com/jkudish/jev-mcp/issues/9) | Missing/malformed answers cannot turn into pass/absent/approved; the entire call fails |
| [#19](https://github.com/jkudish/jev-mcp/issues/19) | Unknown top-level and nested question fields fail before network I/O |
| [#10](https://github.com/jkudish/jev-mcp/issues/10) | A Choice must agree with the maximum probability, not just belong to the option set |
| [#20](https://github.com/jkudish/jev-mcp/issues/20) | Score distributions, legends and confidence survive intact; scores and distributions must agree |
| [#12](https://github.com/jkudish/jev-mcp/issues/12) | All providers share the same missing-envelope/error handling |
| [#13](https://github.com/jkudish/jev-mcp/issues/13) | Tests cover the actual public tool and every supported provider configuration |
| [#4](https://github.com/jkudish/jev-mcp/issues/4) | Explicit configurable compatible API root and credential |

`TestInvalidInputNeverReachesProvider`,
`TestProviderFailuresAreErrorsWithoutSecretsOrRetries`,
`TestDecideMCPContract`, `TestProviderConfiguration`, and `TestStdioRoundTrip`
exercise these boundaries. Live model accuracy and direct TypeSafe availability
require their own credentials and evidence; unit tests cannot establish them.
