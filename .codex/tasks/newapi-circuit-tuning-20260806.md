# NewAPI Personal Circuit Tuning

Date: 2026-08-06
Status: in progress

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
