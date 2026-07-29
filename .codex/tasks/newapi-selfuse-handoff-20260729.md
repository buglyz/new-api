# NewAPI 自用改造交接

日期：2026-07-29

## 当前状态

- 仓库：`/root/.ductor/workspace/newapi`
- 分支：`selfuse-only-trim-20260728`
- 基线提交：`b539b4b refactor(self-use): trim unsupported SaaS surfaces`
- 本次目标提交信息：`fix(self-use): remove billing and harden failover`
- 不部署、不修改 `/opt/newapi`；旧数据库金额列保留兼容，不做破坏性迁移。

## 已实现

1. 请求主链去金额：普通 relay、Responses Compact、Realtime、异步任务、Midjourney 与渠道测试不再读取余额、计算模型价格、预扣、结算或退款；技术用量日志保留 token、请求数、延迟、错误、渠道与流状态，`Quota=0`。
2. API Key 不再因旧额度耗尽状态拒绝；用户、Token、渠道金额字段不再通过 JSON/UI 暴露。金额 API 路由（token usage、渠道余额、部署估价及旧 billing dashboard）不再注册。
3. 删除前端钱包、充值、兑换码、订阅和多用户管理孤立模块及余额入口；个人 Dashboard、资料页和日志用户弹窗不再展示余额/额度。
4. 恢复登录保护的 `/pricing`、`/pricing/$modelId` 与侧栏 `Model Square`。`GET /api/pricing` 返回现有模型广场契约；价格仅作为只读参考元数据，不参与请求许可或结算。
5. 低 SLA：connect 10s、response header 20s、首个有效 SSE 35s、stream idle 90s、non-stream attempt 60s、failover budget 90s；relay HTTP client 不使用整请求短 timeout。
6. 首个有效 `data:` 前超时可重试；blank/comment/heartbeat 不重置；部分输出后的异常不切第二上游、不拼流，仍记录失败并熔断；已写 SSE 后不追加 JSON。
7. 重试矩阵为 401/403、408、425、429、全部 5xx 和结构化 `model_not_found`；普通 400/402/404/409/422 不重试；`specific_channel_id` 不跨渠道。
8. auth/no-key/config 使用 channel-wide 15 分钟熔断；model-not-found 使用 model scope 30 分钟；429 支持两种 `Retry-After`，作为指数退避下限且 cap 15 分钟；half-open 单请求。
9. 动态渠道在无 Key/配置失败时保留渠道身份、记录 channel-wide circuit 并切换下一个候选；自用模式不再永久 auto-disable 渠道。
10. 修复 logger 无锁计数器竞态；`personal_circuit.go` 已按职责拆分。

## 已通过验证

- 受影响 Go 包逐包测试：controller、relay、relay/helper、service、middleware、model、router、setting/operation_setting。
- `go test . -count=1`。
- 定向 `-race`：流式首事件/idle/heartbeat、熔断/Retry-After/budget/header timeout、controller non-stream/specific-channel retry。
- 前端 `bun run typecheck`。
- 前端相关测试 18/18：API Key attention、personal-mode、summary cards。
- `bun run build`，生产构建成功。
- `git diff --check` 和敏感凭据模式扫描无问题。

## 已知残余

- 全量 `bun run lint` 仍失败，错误广泛存在于本次未修改的旧文件（品牌图标、系统设置、图表、通用组件等）；未开启 ignore，也未为本次提交扩大重构。
- 全量 `go test -race ./service` 暴露旧异步视频轮询测试的共享 logger/task 对象竞态；logger 计数器已修，视频轮询共享 Task 仍需独立治理。本次相关定向 race 均通过。
- 多个旧文件超过仓库行数阈值；新增 `controller/pricing.go`、timeout/circuit/usage 文件均在阈值内。建议后续只拆本次仍需维护的旧大文件，不做一次性全面重构。
- 旧 billing/quota service 实现和数据库字段仍在源码中作为升级兼容，但当前请求/router 主链不再调用。后续若要物理删除，必须先做 SQLite/MySQL/PostgreSQL 历史数据迁移设计。

## 给下一个 Agent 的 Prompt

```text
继续审查和优化 /root/.ductor/workspace/newapi 的 selfuse-only-trim-20260728 分支。先完整阅读 /root/.ductor/workspace/AGENTS.md、仓库 AGENTS.md、web/AGENTS.md，以及 .codex/tasks/newapi-selfuse-handoff-20260729.md。以交接文件记录的最新已推送提交为基线，禁止 reset/checkout/stash/revert，禁止部署或修改 /opt/newapi，除非用户另行明确授权。

用户目标：这是纯个人 NewAPI，用于同时聚合多个低 SLA 公益/免费上游。请求必须完全不依赖余额、额度、价格、预扣或结算；保留 API Key、渠道、模型、分组、日志、token/请求数/延迟/错误观测、路由重试与熔断。模型广场 /pricing 必须保留在侧栏，价格仅是只读参考元数据。旧数据库金额列保留兼容，不做 DROP，SQLite/MySQL/PostgreSQL 均要安全。

先做只读复核，不要立即大改：
1. 对比最新提交完整 diff，确认 relay、Responses Compact、Realtime、async task、Midjourney、channel test 都不调用 ModelPriceHelper/PreConsumeBilling/SettleBilling/PostConsumeQuota/PreWssConsumeQuota，也不读取 GetUserQuota 决定许可。
2. 确认所有旧金额 API 不注册，/api/pricing 使用 UserAuth 且返回前端真实契约；直接 HTTP 不能绕过前端进入支付、订阅、兑换、多用户管理。
3. 深审低 SLA：首有效 SSE 前 timeout 可 retry；blank/comment/heartbeat 不重置；partial output 后 timeout/scanner/panic/ping 不跨渠道且开熔断；已提交 SSE 不追加 JSON；specific_channel_id 不跨渠道；401/403/408/425/429/5xx/model_not_found retry，普通 400/402/404/409/422 不 retry。
4. 重点复核动态渠道 SetupContextForSelectedChannel 失败后的 continue 路径、channel-wide/model circuit、half-open 单请求、Retry-After cap、成功清理 circuit，以及任务接口非幂等重试范围。
5. 检查前端 /pricing 登录保护、侧栏和详情深链；检查仍可达页面是否残留 Balance/Wallet/Topup/Subscription/Redemption 文案或调用已撤路由。
6. 运行交接文件列出的验证。全量 lint 是仓库既有基线，先区分本次改动文件与旧债；禁止开启 skip/ignore。全量 service race 的视频轮询共享 Task 竞态可作为独立最小修复，但不要改测试掩盖生产问题。
7. 检查本次新增文件行数；对 controller/relay.go、channel-test.go、channels-columns.tsx 等旧大文件，只在实际继续修改时按职责拆分。

若发现真实 P0/P1，先写新的 .codex/tasks 任务文件并做最小修复、定向测试、单独提交。不要 push 或部署，除非用户再次明确要求。最终用简体中文报告 findings、commit、测试与残余风险。
```
