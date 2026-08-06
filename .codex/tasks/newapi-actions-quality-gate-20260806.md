# NewAPI GitHub Actions Quality Gate

Date: 2026-08-06
Status: in progress

Scope:

- Move routine compilation and test pressure from the production host to GitHub Actions.
- Gate personal multi-architecture image publishing on backend and frontend validation.
- Keep the existing image tags, provenance, SBOM, and immutable digest deployment flow.

Quality gate:

- Root Go module format, vet, build, and tests.
- Independent relaykit vet, build, and tests with `GOWORK=off`.
- Focused race tests for personal failover, retry, and stream-state behavior.
- Frontend frozen dependency install, typecheck, tests, and production build.

Validation note:

- The first workflow run found and corrected pre-existing formatting debt in
  `controller/misc.go`; the quality gate remains a full-repository Go format check.

Deployment policy:

- The publish job runs only after both quality jobs pass.
- Local verification is limited to workflow syntax and diff review.
