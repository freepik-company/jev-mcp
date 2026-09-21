# Contributing to Jev MCP

Thanks for your interest in improving Jev MCP. This document explains how to
report problems, propose changes and get a pull request merged.

## Ground rules

- Be kind and constructive. This project follows the [Code of Conduct](CODE_OF_CONDUCT.md).
- Report security issues privately. See [SECURITY.md](SECURITY.md); never open a public issue for them.
- Keep the scope of the server small: it exposes Jev / System One decisions
  over MCP and validates what the provider returns. Features that belong in the
  MCP client, in the provider or in a workflow engine are out of scope.

## Reporting bugs and requesting features

Use the [issue templates](https://github.com/freepik-company/jev-mcp/issues/new/choose).
A good bug report includes the jev-mcp version (`jev-mcp` reports it to the
client during initialization), the MCP client you use, the provider
(`JEV_PROVIDER`), the tool called and the redacted input. Never paste API keys.

## Development

Only Docker is required:

```sh
make test       # gofmt, go vet and race-enabled tests inside a container
make image      # builds jev-mcp:local
make release    # cross-compiles archives into dist/
```

With a local Go toolchain (see `go.mod` for the version) the same checks are:

```sh
gofmt -l . && go vet ./... && go test -race ./...
```

Integration tests spawn the real binary over stdio and a local HTTP server.
They skip under `go test -short`.

### Project layout

| Path | Purpose |
| --- | --- |
| `cmd/jev-mcp` | Entry point: loads configuration and serves MCP over stdio |
| `internal/config` | Environment resolution and validation, done once at start-up |
| `internal/systemone` | HTTP client for the System One contract and answer validation |
| `internal/judgment` | The `classify`, `verify` and `rerank` tasks built on `decide` |
| `internal/mcp` | Tool registration and the embedded JSON schemas (the public contract) |
| `server.json` | Metadata for the [MCP Registry](https://registry.modelcontextprotocol.io) |

## Pull requests

1. Fork the repository and create a branch from `main`.
2. Make the change with tests. Every fix should come with a test that fails
   without it; every behavioural change should update the README and, when it
   affects tool inputs or outputs, the schemas under `internal/mcp`.
3. Run `make test` and make sure `gofmt -l .` prints nothing.
4. Add an entry under **Unreleased** in [CHANGELOG.md](CHANGELOG.md).
5. Open the pull request against `main` and fill in the template. CI must be green.

Commit messages follow the [Conventional Commits](https://www.conventionalcommits.org/)
style already used in the history (`feat:`, `fix:`, `docs:`, `chore:`).
Pull requests are squash-merged, so the title becomes the commit message.

## Design principles worth knowing

- **Credentials never leave the server environment.** They are not accepted as
  tool arguments, not echoed in errors and not forwarded on redirects.
- **Paid calls are never retried automatically.** A failure is reported to the
  client, which decides.
- **Provider answers are validated, not trusted.** Choices must be among the
  offered options, probabilities must sum to one, scores must match their
  distribution. Invalid answers fail the call instead of being fixed up.
- **The provider response is preserved verbatim** so usage, cost and provider
  extensions are always available to the caller.

## Releasing (maintainers)

Tag `main` with a semantic version (`git tag v1.2.3 && git push --tags`). CI
verifies the build, publishes the archives and checksums to GitHub Releases,
pushes the multi-arch image to `ghcr.io/freepik-company/jev-mcp` and publishes
the new version to the MCP Registry from `server.json`. Move the **Unreleased**
section of the changelog under the new version in the same pull request that
prepares the release.

## License

By contributing you agree that your contributions are licensed under the
[Apache License 2.0](LICENSE).
