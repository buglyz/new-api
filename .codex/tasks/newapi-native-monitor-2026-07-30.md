# NewAPI Native Monitor

## Goal

Integrate `newapi-monitor--` as a native NewAPI feature that is built into the existing single NewAPI container. It must use the existing database, channels, credentials, scheduler, authentication and React application; no sidecar service, port, iframe, reverse proxy, config file, or file history is permitted.

## Boundaries

- Preserve pre-existing worktree changes and do not touch `/opt/newapi`, Caddy, containers, production databases, or production credentials.
- Reuse NewAPI channel URL, protocol, credentials, model mappings, groups, settings, authentication, response helpers, and system task leases.
- Persist configuration and results through GORM with SQLite, MySQL, and PostgreSQL compatibility. Results must never expose secrets, full upstream credentials, or request bodies.
- Default monitoring to disabled so upgrades do not consume upstream quota.

## Integration Contract

| Monitor source capability | NewAPI destination |
| --- | --- |
| configured APIs and keys | existing channels and credentials |
| `/v1/models` discovery | existing upstream model-update task |
| minimal probe request | existing relay-backed channel test path |
| JSON history | GORM results with retention per channel-model |
| periodic/manual probe | DB-leased system task |
| standalone status/admin page | root-admin React route and API |

## TODO

- [x] Record scope and preserve observed user changes.
- [x] Audit monitor source and NewAPI channel, model-update, test, scheduler, settings and UI paths.
- [x] Add native settings, migration, history model, API and cancellable scheduled task.
- [x] Replace iframe page with root-admin native management UI, route and translations.
- [x] Add focused coverage for default-disabled behavior, retries, retention, sanitization and shutdown cancellation.
- [ ] Run focused Go tests, race detector, frontend checks and single-container build in GitHub Actions.

## Verification Results

- Source-only checks passed: `gofmt -d`, `git diff --check`, locale JSON parsing,
  and forbidden standalone-integration reference scanning.
- Local compilation and builds were intentionally skipped because the host is
  resource constrained. The pushed commit delegates the single-container build
  to the existing `personal-docker.yml` GitHub Actions workflow.
