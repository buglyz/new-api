# NewAPI final self-use audit P0/P1 fixes

Date: 2026-07-29
Status: Complete

## Scope

Fix only the P0/P1 findings confirmed during the final read-only audit of
`b539b4b..HEAD`. Do not deploy, push, or change production state.

## Findings

- `GET /api/pricing` is authenticated but bypasses the project-wide success
  response wrapper, leaving a one-off frontend contract.
- A scanner-observed `[DONE]` can win `StreamStatus.endOnce` before the queued
  data handler reports a fatal stop, incorrectly recording the attempt as a
  success and clearing its circuit.
- Task submission retries inherited HTTP 408 from the generic relay matrix,
  expanding retries for a non-idempotent request beyond the previous policy.
- `f2f625b` removed the forced-claim argument from `ClaimPersonalCircuit`, but
  two affinity-selection callers still pass it, so controller/root builds fail.

## TODO

- [x] Wrap the pricing payload with `common.ApiSuccess` and align frontend types.
- [x] Give fatal stream handler outcomes precedence over scanner DONE/EOF.
- [x] Keep HTTP 408 non-retryable for task submissions and add a regression test.
- [x] Keep legacy affinity callers source-compatible without allowing forced
  cooldown bypass, and restore builds.
- [x] Run focused Go tests, race checks, frontend checks, and file-size checks.
- [x] Review the final diff and create one local commit without pushing.

## Verification

- `go test ./controller ./relay/helper ./middleware ./service -count=1`
- `go test -race ./controller ./relay/helper ./service -run '<focused audit tests>' -count=1`
- `go test -race ./service -run 'TestPersonalCircuit|TestClaimPersonalCircuitLegacyForceCannotBypassCooldown|TestChannelSelectionHonorsCircuitCooldownAndSingleHalfOpenClaim' -count=1`
- `go test . -count=1` with the freshly built embedded frontend
- `go test ./router -count=1`
- `cd web && bun run typecheck`
- Focused `oxlint`, `oxfmt --check`, and personal-mode route tests
- `cd web && bun run build`
- `git diff --check`

Generated `web/dist` was removed after verification. No production state was
changed and no remote operation was performed.
