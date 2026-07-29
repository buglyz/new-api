# NewAPI self-use README rewrite - 2026-07-29

## Goal

Replace the upstream-oriented marketing README with an accurate operator guide
for the self-use-only fork.

## Required content

- [x] State the single-owner, low-SLA multi-upstream use case first.
- [x] Preserve New API and QuantumNous attribution and license references.
- [x] Document retained and intentionally removed capabilities.
- [x] Document the billing-independent request path and failover boundaries.
- [x] Point installation and maintenance to `deploy/personal` and the fork image.
- [x] Verify local links, commands, Markdown structure, and absence of secrets.

## Verification

- `git diff --check`
- All relative Markdown links resolve locally.
- Image tags and maintenance commands match the checked-in workflow and scripts.
- Timeout defaults match `common/init.go`; circuit durations match
  `service/personal_circuit.go`.
- No credentials, production paths, or deployment-specific secrets are present.

## Constraints

- Do not deploy or modify `/opt/newapi`, containers, databases, or Caddy.
- Do not imply that billing, recharge, subscriptions, redemption, OAuth, or
  multi-user SaaS can be re-enabled in this branch.
- Do not present reference prices as request admission or settlement inputs.
