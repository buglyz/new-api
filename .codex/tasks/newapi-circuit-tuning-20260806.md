# NewAPI Personal Circuit Tuning

Date: 2026-08-06
Status: done

Scope:

- Tune personal-mode circuit backoffs for low-SLA upstream failover.
- Keep authentication failures slower than transient failures.
- Prevent half-open probes from being duplicated while the probe lease is valid.
- Add focused regression coverage and run race-enabled tests.

Target policy:

- Transient base backoff: 15 seconds.
- Transient maximum backoff: 5 minutes.
- Model-unavailable backoff: 10 minutes.
- Authentication backoff: 15 minutes.
- Channel/configuration backoff: 10 minutes.
- Half-open probe watchdog lease: 2 minutes.
- Closed circuits open after 3 consecutive qualifying failures.
- A successful attempt clears the pending failure streak.

Result (2026-08-06):

- Implemented and pushed (main, synced): `17261a6c` circuit backoff tuning + half-open lease fix, `34771f72` publish quality gate, `a451a976` gofmt, `57bc46a1` open after 3 consecutive failures, `5d5093db` Dify unreachable-return fix.
- GitHub Actions Run `31105797537` passed backend/frontend quality gates, race tests, and multi-arch publish; image `main-5d5093d` (digest `sha256:08773d3d…`).
- Deployed to local new-api container (127.0.0.1:18080, served at https://new.fleey.site): healthy, zero restarts, SQLite quick_check ok, 17/24 channels enabled.
- Live failover verified: 502 (fengwind) → 403 (方舟) → success (freemodel), `/v1/responses` 200 with stream eof; next request served by recovered channel.
- Rollback image tagged `local/new-api:rollback-threshold-20260806-213823` (pre-update `17261a6c` build).
- Local + public `/api/status` both 200; disk usage normal.
