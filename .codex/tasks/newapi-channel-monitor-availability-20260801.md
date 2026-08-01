# Native channel monitor availability bar

## Scope

Add a Uptime-style availability bar to the native channel monitor page while
keeping the existing single-container integration and channel/model grouping.

## TODO

- [x] Aggregate recent channel probe results into 24 hourly buckets.
- [x] Return availability series in the existing overview response.
- [x] Render fixed-position green, amber, red, and no-data segments per channel.
- [x] Keep the channel total success rate visible and model details collapsible.
- [x] Add focused backend coverage and run frontend checks.

## Verification

- `go test ./model -run 'ChannelMonitor|NativeMonitor' -count=1`
- `go test ./controller -run 'ChannelMonitor|NativeMonitor' -count=1`
- `bun test src/features/channel-monitor/lib/__tests__/channel-monitor-groups.test.ts`
- `bun run typecheck`
- Targeted `oxlint`
- `bun run build:check`
