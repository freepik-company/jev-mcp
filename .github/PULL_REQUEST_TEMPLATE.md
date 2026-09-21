## What

<!-- One or two sentences: what changes and why. Link the issue if there is one. -->

## How to verify

<!-- Commands or steps a reviewer can run. `make test` must pass. -->

## Checklist

- [ ] Tests cover the change (a fix has a test that fails without it)
- [ ] README and, if tool inputs or outputs changed, the schemas under `internal/mcp` are updated
- [ ] `CHANGELOG.md` has an entry under **Unreleased**
- [ ] No credentials or private endpoints in code, tests or fixtures
