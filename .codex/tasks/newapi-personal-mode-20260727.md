# NewAPI Personal Failover Mode

Date: 2026-07-27
Status: Reliability control loop locally verified; pending push and Docker verification

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

## Reliability Control Loop Follow-up

- [x] Record structured per-attempt relay traces without request bodies, keys,
  or raw upstream error messages.
- [x] Add personal-mode channel-and-model temporary circuits with bounded
  backoff, half-open recovery, and an all-candidates-cooling fallback.
- [x] Notify only on deduplicated circuit state transitions.
- [x] Add bounded batch probing/recovery and a read-only route preview.
- [x] Surface circuit state, attempt outcomes, and maintenance tools in the
  personal operations UI with complete i18n coverage.
- [x] Add reversible SQLite backup/upgrade/rollback tooling and documentation.
- [x] Keep documentation-only changes out of Docker builds, refresh QEMU, and
  add a conflict-visible upstream synchronization workflow.
- [x] Re-run frontend checks/build before the complete Go test suite and review
  standard-mode behavior.
- [ ] Push and verify the Docker workflow.

### Reliability Policy

- Temporary circuits are enabled only while `SelfUseModeEnabled` is true.
- Transport failures, HTTP 429, and HTTP 5xx use exponential backoff from 30
  seconds to 15 minutes; `model_not_found` opens only the affected
  channel/model pair for 30 minutes.
- HTTP 401/403 remain governed by the existing hard auto-disable path.
- Circuit state is intentionally process-local and is reported as volatile;
  restarting the process clears temporary cooldowns.
- Attempt logs contain identifiers, timing, outcome enums, status/error codes,
  retry decisions, and request correlation only. They never contain prompts,
  responses, channel keys, or raw upstream error text.

### Local Verification

- Focused frontend tests: 26 passed across personal mode, full pagination,
  channel attention, API key risks, failover traces, and reliability batches.
- `bun run typecheck`, changed-file OXLint, protected-header format check, i18n
  synchronization, and the production frontend build passed.
- `GOMAXPROCS=1 go test -p 1 ./...` passed after the final frontend build.
- `git diff --check`, YAML parsing, ShellCheck, Compose expansion, and an
  isolated SQLite backup smoke test passed.
- `copyright:check` still identifies the untouched upstream
  `web/src/features/channels/lib/channel-field-update.ts`; new frontend files
  retain protected headers.
