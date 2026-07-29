# NewAPI self-use post-handoff P1

## Goal

Finish the focused follow-up after `a8988ef` without deploying production:

- remove deployment price-estimation calls and UI after the endpoint was removed;
- honor personal-circuit cooldowns without forced early half-open attempts;
- keep the model square as reference pricing while removing recharge-price semantics.

## TODO

- [x] Remove the stale deployment price API and callers.
- [x] Remove forced personal-circuit claims and add deterministic tests.
- [x] Remove recharge-price controls and URL state from the model square.
- [x] Run focused Go and frontend verification.
- [x] Review the focused diff; commit and push are handled immediately after this record update.

## Verification

- `go test ./service -count=1`
- `go test -race ./service -run 'TestPersonalCircuit|TestChannelSelectionHonorsCircuitCooldownAndSingleHalfOpenClaim' -count=1`
- `cd web && bun run typecheck`
- `cd web && bun run build`
- `git diff --check`
- Residual searches found no deployment `price-estimation` caller and no model-square recharge-price state or conversion path.

## Status

Complete. No production deployment was performed.
