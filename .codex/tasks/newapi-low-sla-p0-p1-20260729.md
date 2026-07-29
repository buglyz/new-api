# NewAPI Low-SLA P0/P1 Reliability

Date: 2026-07-29
Status: Handoff after implementation commit

## Objective

Strengthen the self-use-only gateway for multiple low-SLA upstreams without
changing public API request or success response contracts.

## P0

- Treat a stream timeout before the first meaningful event as retryable.
- Treat an abnormal stream after partial output as failed for circuit health,
  but never append a second upstream stream to the partial response.
- Do not let blank lines, comments, or keepalives satisfy the first-event timer.
- Add response-header, first-event, stream-idle, non-stream attempt, and total
  failover-budget controls with conservative self-use defaults.

## P1

- Retry transient failures including 408, 425, 429, 5xx, 504, and 524.
- Avoid retrying deterministic request failures such as generic 400 and 422.
- Add channel-wide temporary circuits for authentication and channel
  configuration/key failures while keeping model-not-found model-scoped.
- Honor upstream Retry-After for 429 cooldowns when available.

## Defaults

- Response header: 20s
- First meaningful stream event: 35s
- Stream idle: 90s
- Non-stream attempt: 60s
- Total failover retry budget: 90s

## Verification

- [x] Focused stream timeout tests
- [x] Retry policy tests
- [x] Circuit scope and cooldown tests
- [x] HTTP timeout tests
- [x] `go test` for affected packages
- [x] Focused race tests for circuit, timeout, and stream state
- [x] Root package build/test
- [x] Frontend typecheck, focused tests, and production build
- [ ] Full-repository lint baseline cleanup (pre-existing, out of scope)
- [ ] Full service race cleanup for asynchronous video polling (pre-existing)
- [ ] Follow-up review described in the handoff document
