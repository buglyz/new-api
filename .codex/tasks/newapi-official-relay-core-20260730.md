# NewAPI Official Relay Core Restore

## Goal

Restore the request transport, SSE streaming lifecycle, and stream termination behavior to official `v1.0.0-rc.22` semantics while preserving the current `relaykit` module layout and unrelated self-use product changes.

## Scope

- Outbound relay HTTP client timeout and cancellation behavior.
- `DoRequest` transport selection and request lifecycle.
- `StreamScannerHandler` timer, terminal state, and error propagation behavior.
- OpenAI Responses stream handler integration.
- Relay attempt wrapper code added only for custom request timeouts.

## Out Of Scope

- Frontend and dashboard changes.
- Self-use route and feature trimming.
- Channel CRUD, model mapping, authentication, and database schema.
- Production deployment or container changes.

## Checklist

- [x] Build an exact official-to-fork behavior map.
- [x] Restore official relay behavior with current imports/module structure.
- [x] Preserve unrelated user worktree changes.
- [x] Run focused unit tests and race tests.
- [x] Build the root module; `relaykit` was not touched.
- [x] Review the final diff for accidental self-use feature rollback.

## Verification

- `go test` for affected relay, service, and controller packages.
- Focused `go test -race` for stream lifecycle packages.
- Root Go build.
- `relaykit` independent build only if files under `relaykit/` change.

## Result

- Core request, transport, and streaming files match `upstream/main` exactly.
- Removed custom request-context binding, layered relay timeouts, abnormal-stream
  error propagation, non-stream attempt timeout wrapping, and failover wall-clock
  budgeting.
- Preserved relaykit integration, per-channel HTTP protocol/HTTP2 sharding,
  personal circuit breaking, attempt tracing, cooldown, and channel failover.
- Focused tests passed for relay helper/provider, service, and controller packages.
- Focused race tests passed for relay helper, OpenAI, Claude, controller, and HTTP
  client/service paths. Two unrelated pre-existing test races remain documented in
  task polling and Gemini tests.
- `bun run build` and `go build ./...` passed. Generated `web/dist` was removed
  after verification.
- No commit, push, deployment, service restart, or production configuration change.
