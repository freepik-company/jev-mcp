# Validation evidence

2026-09-21, initial `v0.1.0` implementation.

## Offline

- `make test`: gofmt check, `go vet`, race-enabled unit and integration tests passed.
- Real subprocess over stdio: MCP initialization and a `decide` call passed against
  a local HTTP stand-in.
- Both provider configurations, all three primitives, structured and legacy text
  results, cost preservation, strict inputs, malformed responses, cancellation,
  redirects, bounded I/O, rounding boundaries and probability consistency tested.
- `make image` built the non-root runtime image.
- `make release` built Linux/macOS amd64/arm64 archives and checksums.

## Live OpenRouter

The Linux amd64 release binary ran as a temporary process in the production
environment, using an existing credential there. No credential was copied into
this repository or into the MCP request. The application and agent configuration
were not changed. The same reusable `scripts/smoke.py` drove the actual stdio MCP
protocol, including tool discovery.

Synthetic input: a customer was charged twice and explicitly asks for a refund.
All three questions went through one `decide` call:

| Question | Type | Returned value |
| --- | --- | --- |
| Department | Choice | `billing`, confidence `1`, full distribution preserved |
| Refund requested | Noul | `0.99` |
| Request explicitness | Score | `2` on a three-level rubric, confidence `1`, full legend/distribution preserved |

Provider: `TypeSafe`, routed by OpenRouter.
Model: `typesafe/jev-1.13-20260917`.
Usage: 403 input tokens, 70 output tokens, `0.000016926` USD.

This proves the endpoint, authentication, binary, stdio protocol and all three
primitives work together. One synthetic case is not an accuracy benchmark or a
calibration claim.

## Not live-verified

TypeSafe direct uses the same request/response contract and passes the provider
configuration tests, but no direct TypeSafe credential was available for a live
call. macOS and arm64 artifacts were cross-compiled, not executed on target hosts.
