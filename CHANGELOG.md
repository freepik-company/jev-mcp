# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Open-source project files: contributing guide, code of conduct, security
  policy, issue and pull request templates, code owners and Dependabot.
- `server.json` and a CI job that publishes each release to the
  [MCP Registry](https://registry.modelcontextprotocol.io).
- Multi-arch container image published to `ghcr.io/freepik-company/jev-mcp` on
  every release, labelled for MCP Registry ownership verification.
- Windows (amd64, arm64) release archives.
- `govulncheck` runs in CI.

### Changed

- The server reports the release version to MCP clients instead of a hard-coded
  string; it is injected at build time and reads `dev` for local builds.
- All comments, error messages and test output are in English.
- GitHub Actions are pinned to commit SHAs.

## [0.2.0] - 2026-09-21

### Added

- `classify`, `verify` and `rerank` tools built on `decide`, preserving IDs,
  input order and the full provider response.
- `list_models` tool adapting the OpenRouter and TypeSafe catalogues.

## [0.1.0] - 2026-09-21

### Added

- Standalone stdio MCP server with the `decide` tool for Jev / System One
  typed decisions through OpenRouter or TypeSafe.
- Answer validation, bounded request and response sizes, no credential leaks,
  no automatic retries of paid calls.
- Container image and cross-platform release archives built by CI.

[Unreleased]: https://github.com/freepik-company/jev-mcp/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/freepik-company/jev-mcp/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/freepik-company/jev-mcp/releases/tag/v0.1.0
