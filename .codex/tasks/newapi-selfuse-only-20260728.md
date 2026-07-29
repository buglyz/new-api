# NewAPI Self-Use Only Trim

Date: 2026-07-28
Status: Complete

## Objective

Trim the fork down to self-use-only operation. Keep the personal gateway
workflow for one operator and remove unrelated public, multi-user, billing,
subscription, redemption, and marketing features from both backend and
frontend.

## Keep

- Channels, models, groups, API keys, relay, logs, performance, system tasks
- Routing, retries, auto-disable/recovery, personal reliability tools
- Local admin login, session refresh/logout, Passkey, 2FA
- System settings needed for self-hosted operations

## Remove

- Registration, password reset, email verification, OAuth login/bind
- Public marketing/legal/pricing/rankings/about surfaces
- Wallet/top-up/payment/subscription/redemption/check-in/affiliate flows
- Multi-user CRUD and related admin pages
- Standard-mode switching and mode-dependent UI branches

## Checklist

- [x] Force self-use mode as the only supported mode in setup/status/options
- [x] Remove disabled feature routes from Gin router
- [x] Delete unused frontend routes/features/navigation
- [x] Delete standalone backend feature files no longer referenced
- [x] Regenerate frontend route tree/build artifacts as needed
- [x] Remove the obsolete route-deny compatibility middleware
- [x] Add a regression test for retired and retained self-use API routes
- [x] Run focused backend, frontend, and production-build verification

## Verification

- `go test .`
- `go test ./router ./middleware ./controller ./model ./service`
- `bun run typecheck`
- `bun test src/lib/legacy-route.test.ts src/lib/__tests__/personal-mode.test.ts`
- `bun run build`
