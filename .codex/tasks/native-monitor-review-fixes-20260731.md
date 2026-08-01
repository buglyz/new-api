# Native Monitor Review Fixes

## Scope

Fix the confirmed high- and medium-severity findings from the strict review of
commit `8b054626`. Preserve the user's pre-existing worktree changes and keep
the native single-container integration contract.

## TODO

- [x] Publish native monitor settings as an atomic immutable snapshot.
- [x] Cancel active monitor runs when monitoring is disabled.
- [x] Make system-task shutdown and cancellation terminal states truthful.
- [x] Restrict monitoring to safe supported endpoint types and quiet relay logs.
- [x] Redact structured credentials and upstream response bodies.
- [x] Retain a real 24-hour statistics window and clean removed targets.
- [x] Poll manual task terminal state and refresh history in the React UI.
- [x] Add focused regression coverage and run targeted checks.
- [ ] Re-run final static verification after request-log suppression changes.

## Excluded Existing Changes

- `web/src/features/dashboard/components/models/__tests__/model-chart-runtime.test.ts`
- `.codex/tasks/frontend-audit-20260729.md`
- `.codex/tasks/newapi-deploy-e00a04a5-20260730.md`
