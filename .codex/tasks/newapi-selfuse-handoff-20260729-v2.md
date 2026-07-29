# NewAPI 自用分支交接 v2

日期：2026-07-29

## 仓库状态

- 仓库：`/root/.ductor/workspace/newapi`
- 分支：`selfuse-only-trim-20260728`
- 核心提交：`a8988ef fix(self-use): remove billing and harden failover`
- P1 收尾提交：`f2f625b fix(self-use): honor cooldowns and remove price remnants`
- 远端：`origin/selfuse-only-trim-20260728` 已包含 `f2f625b`
- 未部署，未修改 `/opt/newapi` 或生产容器、数据库、Caddy。

## 用户目标

这是纯个人 NewAPI，用于聚合多个低 SLA 公益/免费上游：

- 请求许可和转发不依赖余额、额度、价格、预扣、结算或退款；
- 保留 API Key、渠道、模型、分组、限流、安全鉴权和技术用量观测；
- 保留多渠道重试、分层超时、熔断、half-open 与恢复；
- 保留侧栏模型广场 `/pricing`，价格仅是只读参考，不参与请求路径；
- 旧金额数据库列保留升级兼容，不做破坏性迁移。

## 已完成

### 去金额主链

- 普通 relay、Responses Compact、Realtime、异步任务、Midjourney、渠道测试均不再进行余额检查、模型计价、预扣、结算或退款。
- 技术日志保留 token、请求数、耗时、错误、渠道和流式状态，金额字段固定为零。
- API Key 不再因旧额度耗尽状态拒绝请求。
- 金额 API 与钱包、充值、兑换、订阅、多用户管理等前端入口已撤销。
- 部署创建和延期已删除对失效 `POST /api/deployments/price-estimation` 的调用、币种字段与金额 UI；部署操作本身保留。

### 模型广场

- `/pricing` 和 `/pricing/$modelId` 已恢复并要求登录，侧栏包含 `Model Square`。
- `GET /api/pricing` 保留前端所需统一契约。
- 标准、动态、分组参考价格仍显示，但 `rechargePrice` URL 状态、Standard/Recharge 切换和充值倍率换算实现已删除。
- 模型参考价格不进入请求鉴权、扣费或结算。

### 低 SLA 故障转移

- 默认分层超时：connect 10s、response header 20s、首个有效 SSE 35s、stream idle 90s、non-stream attempt 60s、failover budget 90s。
- 首个有效 `data:` 前超时可重试；blank/comment/heartbeat 不重置计时器。
- 已有部分输出后的 timeout/scanner/panic/ping 错误不切换第二上游、不拼接流，仍记录失败并打开熔断；已提交 SSE 后不追加 JSON。
- 重试：401/403、408、425、429、全部 5xx、结构化 `model_not_found`。
- 不重试：普通 400/402/404/409/422 和本地转换错误；`specific_channel_id` 不跨渠道。
- auth/no-key/config 使用 channel-wide 15 分钟熔断；model-not-found 使用 model scope 30 分钟。
- 429 `Retry-After` 支持 delta-seconds 和 HTTP-date，作为退避下限且最大 15 分钟。
- `f2f625b` 删除了全部候选冷却时的强制 claim。冷却到期前不会提前 half-open；到期后只允许一个请求持有 half-open lease。

## 本轮验证

- `go test ./service -count=1`：通过。
- `go test -race ./service -run 'TestPersonalCircuit|TestChannelSelectionHonorsCircuitCooldownAndSingleHalfOpenClaim' -count=1`：通过。
- `cd web && bun run typecheck`：通过。
- `cd web && bun run build`：通过。
- `git diff --check`：通过。
- 残留搜索：有效前端源码中无 `price-estimation` 调用，无模型广场 `rechargePrice`/`showRechargePrice`/`applyRechargeRate` 路径。
- 提交后已删除 ignored 的 `web/dist`，并运行 `go clean -cache -testcache`。

## 已知残余与边界

- 旧 billing/quota service 和数据库列仍作为历史升级兼容代码存在，但当前 router/relay 请求主链不调用。物理删除需要单独设计 SQLite/MySQL/PostgreSQL 迁移，不应在小修中处理。
- 模型广场当前仍使用原有 `PublicLayout`，但路由有登录保护。是否迁入控制台布局属于产品布局选择，不是当前安全缺陷。
- 全量前端 lint 存在大量仓库旧问题；不得通过全局 ignore/skip 掩盖。
- 全量 `go test -race ./service` 曾暴露旧异步视频轮询共享 Task 竞态；本次相关熔断定向 race 已通过。该竞态应独立定位和修复。
- 若继续修改旧超限文件，必须按职责拆分；不要为纯行数指标进行全仓重构。

## 给下一个 Agent 的 Prompt

```text
继续处理 /root/.ductor/workspace/newapi 的 selfuse-only-trim-20260728 分支。先完整阅读：
1. /root/.ductor/workspace/AGENTS.md
2. 仓库 AGENTS.md 和 web/AGENTS.md
3. .codex/tasks/newapi-selfuse-handoff-20260729-v2.md

当前远端基线是 f2f625b（origin/selfuse-only-trim-20260728），工作树预期干净。禁止 reset/checkout/stash/revert，禁止部署或修改 /opt/newapi、生产容器、数据库或 Caddy，除非用户另行明确授权。不要重做 a8988ef 和 f2f625b 已完成的内容。

用户目标：纯个人聚合多个低 SLA 公益/免费上游。请求必须完全不依赖余额、额度、金额、模型计价、预扣、结算或退款；保留 API Key、渠道、模型、分组、鉴权、限流、token/请求数/延迟/错误观测、重试和熔断。模型广场 /pricing 保留在侧栏，价格仅作为只读参考。旧数据库金额列保留兼容，不做 DROP，SQLite/MySQL/PostgreSQL 必须安全。

下一步先做只读最终审计，不要默认需要大改：

1. 从 b539b4b..HEAD 审查完整 diff，确认普通 relay、Responses Compact、Realtime、async task、Midjourney、channel test 不调用 ModelPriceHelper、PreConsumeBilling、SettleBilling、PostConsumeQuota、PreWssConsumeQuota，也不读取用户/token quota 决定请求许可。
2. 核对 router 与直接 HTTP 边界：支付、充值、订阅、兑换、多用户、token usage、channel balance、deployment price-estimation 等金额接口不注册；GET /api/pricing 走项目统一响应包装和登录鉴权，契约与前端一致。
3. 深审流式状态机：首个有效 data 前 timeout 可重试；blank/comment/heartbeat 不重置；部分输出后的 timeout/scanner/panic/ping 不跨渠道、不拼流但记录失败并熔断；DONE/兼容 EOF 成功；已写 SSE 后不追加 JSON。
4. 深审选择与重试：401/403/408/425/429/全部5xx/model_not_found 可跨渠道；普通400/402/404/409/422和本地转换错误不重试；specific_channel_id 不跨渠道；任务类非幂等请求不应被无意扩大重试。
5. 深审 circuit：channel-wide 15m、model scope 30m、Retry-After 两种格式及 cap、冷却期不可 claim、到期后单请求 half-open、成功清除命中 scope、动态渠道 SetupContextForSelectedChannel 失败后记录正确渠道身份并继续候选。
6. 检查 /pricing、详情深链、侧栏、登录跳转和 API 契约；确认可达 UI 无 Balance/Wallet/Topup/Subscription/Redemption 或 deployment price-estimation 残留。不要恢复 recharge price 换算。
7. 运行交接文档列出的定向验证，并补充最小必要测试。区分本次问题和仓库旧 lint/race 债务，禁止全局 skip/ignore，禁止改测试掩盖实现。
8. 检查本次修改/新增文件限制：Go <=400、TS <=300、TSX <=200。旧超限文件只有在确实继续修改时按职责拆分，不做无关全面重构。

若确认无 P0/P1，直接输出证据化审计结论和剩余风险，不制造改动。若发现真实问题，先创建新的 .codex/tasks 记录，做最小修复、定向测试和独立本地提交。不要 push，除非用户再次明确要求。最终使用简体中文报告文件/行号、行为影响、测试和 commit。
```
