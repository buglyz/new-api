# NewAPI Personal Failover Mode

Date: 2026-07-27
Status: Source complete

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

## Verification

- `bun test src/lib/__tests__/personal-mode.test.ts`: 9 passed.
- `bun run typecheck`: passed.
- OXLint on every changed and new frontend source file: passed.
- `bun run format:check`: passed across 1,051 files.
- `bun run build:check`: passed; production assets built successfully.
- `go test -p 1 ./middleware ./controller`: passed.
- `GOMAXPROCS=1 go test -p 1 ./...`: passed after the frontend build.
- `git diff --check`: passed.
- Personal Docker workflow targets `ghcr.io/buglyz/new-api` for amd64/arm64
  with `latest`, `main`, and immutable commit-SHA tags.

## Baseline Notes

- Full-project OXLint still reports existing errors in unrelated upstream files.
  The changed-file lint gate is clean; unrelated lint debt was not modified.
- `copyright:check` still reports the untouched upstream file
  `web/src/features/channels/lib/channel-field-update.ts`. All new frontend
  source files retain the project's protected copyright header.
