# NewAPI Personal Failover Mode

Date: 2026-07-27
Status: Operations visibility verified

## Objective

Adapt the existing self-use mode for one operator aggregating multiple low-SLA
public upstreams. Keep routing, retries, channel recovery, logs, models, API
keys, security, and system operations; remove multi-user, billing, promotion,
subscription, redemption, and public marketing surfaces while the mode is on.

## Constraints

- Reuse `SelfUseModeEnabled`; no migration or new dependency.
- Preserve standard-mode behavior when the switch is off.
- Preserve QuantumNous/new-api licensing and branding notices.
- Do not deploy or modify the running NewAPI service.
- Push the completed source change to `buglyz/new-api` `main` only after tests.

## Checklist

- [x] Add the backend personal-mode capability matrix and stable 403 contract.
- [x] Normalize `/api/status` for personal mode.
- [x] Filter personal-mode routes, navigation, profile, dashboard, and settings.
- [x] Preserve failover operations and prioritize channels, logs, and API keys.
- [x] Add focused backend and frontend decision tests.
- [x] Add a multi-architecture GHCR image workflow for the personal fork.
- [x] Complete frontend lint, format, typecheck, tests, and production build.
- [x] Complete focused and full Go tests after `web/dist` exists.
- [x] Review the final diff and clean generated artifacts.

## Operations Visibility Follow-up

- [x] Add a full-pagination channel attention view with probe freshness.
- [x] Surface retry chains and request-correlated log navigation.
- [x] Add full-pagination API key risk filtering without revealing keys.
- [x] Add a personal operations overview using bounded, low-frequency queries.
- [x] Re-run frontend checks/build before the full Go test suite.
- [x] Push the verified commit and record the Docker Action result externally.

## Verification

- Personal operations tests: 23 passed across pagination, channel attention,
  API key risks, failover traces, and standard-mode decision coverage.
- `bun run i18n:sync`: all seven locale key sets complete; Simplified and
  Traditional Chinese operations copy translated.
- Changed-file OXLint and the 1,069-file protected-header format check passed.
- `bun run typecheck` and the production `bun run build` passed.
- `GOMAXPROCS=1 go test -p 1 ./...`: passed after the final frontend build.
- `git diff --check`: passed.
- Size review: every new TS/TSX file is within the workspace limit; modified
  upstream files that remain over the limit were already over at the baseline.
- Personal Docker workflow targets `ghcr.io/buglyz/new-api` for amd64/arm64
  with `latest`, `main`, and immutable commit-SHA tags.

## Baseline Notes

- Full-project OXLint still reports existing errors in unrelated upstream files.
  The changed-file lint gate is clean; unrelated lint debt was not modified.
- `copyright:check` still reports the untouched upstream file
  `web/src/features/channels/lib/channel-field-update.ts`. All new frontend
  source files retain the project's protected copyright header.
