# NewAPI upstream sync - 2026-07-29

## Goal

Integrate `QuantumNous/new-api:main` into the personal self-use fork while
preserving the fork's billing-independent relay path, private-only HTTP/UI
surface, retry/circuit behavior, and database compatibility.

## Upstream baseline

- Local baseline: `5934ea06317b30300ab9757bf47e1a9e158e98e3`
- Upstream baseline: `c27d1ef` (`upstream/main`)
- Merge base: `bc14c18f6024e79cba1c08d02cd007796e12d668`
- Divergence before merge: 18 local commits, 20 upstream commits

## Work items

- [x] Fetch and classify upstream changes.
- [x] Merge upstream without restoring removed SaaS/payment surfaces.
- [x] Preserve billing-independent relay, retry, stream, and circuit behavior.
- [x] Verify router/API/UI self-use contracts and database compatibility.
- [x] Run focused Go, race, frontend, formatting, and size checks.
- [x] Commit the reviewed integration locally and update `main` without force.

## Constraints

- Do not deploy or modify `/opt/newapi`, production containers, databases, or Caddy.
- Do not use reset, checkout, stash, revert, or force push.
- Keep legacy monetary database columns for SQLite/MySQL/PostgreSQL compatibility.
- Keep `/pricing` authenticated and read-only; do not restore wallet, recharge,
  subscription, redemption, multi-user, channel balance, token usage, or
  deployment price-estimation routes/UI.

## Integration note

Adapter tests run without `common.Init`; their stream timeout globals therefore
remain zero. The stream scanner now applies the same 35s/90s defaults used by
`positiveEnv` so an uninitialized zero value cannot become an immediate timeout.

Client cancellation during an SSE write is treated as a client-side terminal,
not an upstream transport failure, so partial output cannot open a healthy
channel circuit.

## Verification

- `go test ./... -count=1`
- `go test -race ./controller -run 'TestShouldRetry|TestGetPricing|TestRunRelayAttempt' -count=1`
- `go test -race ./relay/helper -run 'TestStreamScannerHandler...|TestStreamRelayError...' -count=1`
- `cd relaykit && GOWORK=off go test ./... -count=1`
- `cd web && bun run typecheck`
- `cd web && bun test` for New API channel, login session, personal mode, and legacy routes
- `cd web && bun run build`
- `git diff --check`
- Added Go/TS/TSX files comply with 400/300/200 effective-line limits; three
  upstream test files were split by responsibility without changing assertions.
