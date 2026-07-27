# NewAPI Personal Failover Mode

Date: 2026-07-27
Status: Personal usage accuracy follow-up published

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
- [x] Push and verify the Docker workflow.

## Personal Usage Overview Follow-up

- [x] Remove the personal operations dashboard section and its queries.
- [x] Replace personal-mode currency and quota cards with real token totals.
- [x] Preserve the upstream balance dashboard when personal mode is disabled.
- [x] Add focused API and frontend aggregation tests.
- [x] Re-run frontend checks/build before the complete Go test suite.

## Personal Usage Accuracy And Presentation Follow-up

- [x] Remove remaining personal-mode balance, cost, pricing, and billing copy
  from the dashboard, profile, and usage logs without hiding operational data.
- [x] Include unflushed `quota_data` cache entries in exact total and recent
  token counts without adding another high-frequency aggregate query.
- [x] Report disabled data export explicitly and keep its operational settings
  reachable in personal mode.
- [x] Add component coverage for personal/standard mode, API errors, empty
  data, large integer formatting, and responsive card layout.
- [x] Re-run the required frontend and backend verification sequence and review
  standard-mode behavior.
- [x] Push the verified source and verify the Docker workflow.

### Personal Usage Verification

- Personal-mode cards show last-24-hour tokens, recorded total tokens, and
  request count; quota and currency values remain confined to standard mode.
- `/api/data/self/summary` returns exact decimal strings for recent and total
  tokens plus an explicit tracking flag. One aggregate query reads both sums;
  the response also includes the current process's pending cache generation.
- Successful cache flushes are transactional; failed flushes retain pending
  rows for retry. Disabling data export and graceful shutdown both attempt a
  final flush without recording new rows after export is disabled.
- Focused frontend tests: 76 passed across personal/standard mode, token API
  error/empty/large-number states, responsive layout, flow metrics, channels,
  keys, pagination, costs, policy violations, and failover traces.
- `bun run typecheck`, changed-file OXLint, `bun run format:check`, i18n sync,
  and the production frontend build passed.
- `GOMAXPROCS=1 go test -p 1 ./...` passed after the frontend build.
- `git diff --check`, `gofmt`, changed-file size review, sensitive-data review,
  and the standard-mode branch review passed.
- Feature commit `0c6d45746fb362d8901a97437d7e7906b2eca429` was pushed to
  `buglyz/new-api` `main`. Docker Action `30311816983` and job `90128878315`
  succeeded in 19m39s. No deployment was performed.

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
- Feature commit `f23fb2583c2152c19d5a1f46ab1119434b72cc4f` was pushed to
  `buglyz/new-api` `main`.
- Docker Action `30264459228` succeeded for amd64/arm64 with manifest digest
  `sha256:0fd1a68326b3afcd36ca605c7b01a623d6648204c65596ce38f3ce8716eb7751`
  and no annotations.
- After an explicit user request, the immutable SHA image was deployed to the
  existing `/opt/newapi` Compose service. Backup
  `/opt/newapi/backups/pre-fork-deploy-20260727-202643` passed `quick_check`;
  the new container is healthy with version `main-f23fb25`, and Caddy was not
  reloaded or restarted.
