# NewAPI Personal Circuit Low-SLA Window Tuning

Date: 2026-08-06
Status: in progress

Scope:

- Low-SLA upstreams should only be tripped when they are *persistently*
  unavailable, not on a few transient blips.
- Transient failures (5xx / transport): open only when a 10-minute sliding
  window contains more than 9 failed attempts (>= 10) with no success.
  Any success clears the window immediately.
- Deterministic failures (auth / quota / model missing / channel config):
  open on the first occurrence (they never self-heal).
- 429 rate limiting no longer counts toward circuit failure samples.
- Fix a state-machine defect where a repeated failure during cooldown could
  downgrade an open circuit back to closed.

Target policy:

- Window: 10 minutes.
- Transient trip threshold: > 9 failures in the window (>= 10), all failing.
- Deterministic errors: trip on first occurrence (model 10m / auth 15m /
  channel config 10m cooldowns unchanged).
- Half-open lease: 2 minutes (unchanged); failed probe reopens with >= 2-step backoff.
- Success clears the failure window and closes the circuit.
