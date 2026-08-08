# New API 自用分支审查整改与遗留问题

## 审查信息

- 项目：buglyz/new-api
- 审查基线：main 分支
- 审查日期：2026-08-08
- 文档作用：记录针对 2026-08-07 多维审查报告的修复落地情况，以及本次**未修复**、留待后续跟进的问题清单

## 一、本次已修复

本次提交针对审查报告中的高优先级问题进行了修复，涵盖故障转移语义对齐、计费残留清除、编码规范与文档一致性四类。

### 1. 故障转移与熔断语义对齐

| 问题 | 修复 |
| --- | --- |
| 流式 `ticker.Reset` 在空行/注释过滤**之前**执行，首事件计时器被无意义重置 | `relay/helper/stream_scanner.go`：`ticker.Reset` 移至有效事件判定之后，空行、注释、heartbeat 不再重置计时器 |
| 无独立首事件超时，`STREAM_FIRST_EVENT_TIMEOUT` 代码不存在 | 新增 `constant.StreamFirstEventTimeout`（默认 35s），初始 ticker 用首事件超时，收到首个有效事件后切换为流空闲超时 |
| `STREAMING_TIMEOUT` 默认 300，与 README 的 90 不符 | `common/init.go`：默认值改为 90 |
| 熔断退避常量与 README 不符（model 10min、channel 10min、base 15s、max 5min） | `service/personal_circuit_policy.go`：改为 model 30min、channel 15min、base 30s、max 15min，并同步修正单元测试 |
| `Retry-After` 被解析但熔断器未使用 | `personalCircuitBackoff` 现在在 transient 失败路径使用上游 `Retry-After`（秒数/HTTP 日期），受最大冷却 15min 限制 |
| README 声称的 `RELAY_CONNECT_TIMEOUT`/`RELAY_RESPONSE_HEADER_TIMEOUT`/`RELAY_NON_STREAM_TIMEOUT`/`RELAY_FAILOVER_BUDGET` 均未实现 | 分别接入 transport 连接超时、响应头超时、非流式 client 超时，以及 relay/task 重试循环总预算（`common/constants.go`、`service/http_client.go`、`controller/relay.go`、`controller/relay_task.go`） |

### 2. 计费豁免遗留项

| 问题 | 修复 |
| --- | --- |
| `controller/midjourney.go` 对历史任务失败仍调用 `model.IncreaseUserQuota` 真实修改用户余额 | 改为仅清除遗留 quota 标记（`task.Quota = 0` + `Update()`），不触碰用户余额，与 `RefundTaskQuota` 的自用语义一致 |

### 3. 编码规范违规

| 问题 | 修复 |
| --- | --- |
| `model/task.go` 使用 MySQL 特有 `AUTO_INCREMENT` GORM 标签 | 改为 `gorm:"primaryKey"`，符合"让 GORM 处理主键生成"的约定 |
| `setting/console_setting/validation.go` 直接调用 `encoding/json` | 改为 `common.UnmarshalJsonStr`，移除 `encoding/json` 导入，符合 JSON 包装统一走 `common.*` 的规则 |

### 4. 死代码与文档清理

| 问题 | 修复 |
| --- | --- |
| `middleware/auth.go` 空函数 `WssAuth`（无调用者） | 删除 |
| `service/error.go` 中整段被注释的历史错误包装函数 | 删除 |
| `relay/relay_task.go` 过时注释"控制器负责 defer Refund 和成功后 Settle" | 改为自用构建不执行计价/额度结算的准确描述 |
| `AGENTS.md`/`CLAUDE.md` Go 版本写 1.22+，实际需要 1.25.1 | 更新为 1.25.1+，并对 billing 相关规则补充"自用分支不参与请求链"的说明 |
| `README.md` 环境变量表与实际实现不符 | 更新 `RELAY_NON_STREAM_TIMEOUT`、`RELAY_FAILOVER_BUDGET`、`STREAMING_TIMEOUT` 的默认值与语义说明 |

## 二、审查发现但未修复的遗留问题

以下问题在审查报告中识别，出于以下原因之一**本次未处理**：属于上游继承的设计债（多处同类）、涉及跨模块大改有回归风险，或属于明确的后续增量工作。按优先级列出。

### P1 — 建议尽快处理

1. **计费死代码仍物理存在（约 20+ 个函数 + `pkg/billingexpr` 模块）**
   - 位置：`service/billing.go`、`service/billing_session.go`、`service/funding_source.go`、`service/tiered_settle.go`、`service/epay.go`、`service/waffo_pancake.go`、`service/violation_fee.go`、`service/quota.go` 的 `PreConsumeTokenQuota`/`PostConsumeQuota` 等、`relay/helper/price.go` 的 `ModelPriceHelper`、`pkg/billingexpr/` 整个模块、`controller/billing.go`、`controller/channel-billing.go`。
   - 现状：请求链（relay/controller/middleware）均不调用，但保留了真实扣费实现（`PostConsumeQuota` 会调 `DecreaseUserQuota`）。
   - 风险：上游同步时可能被误重新接入请求链；约 4000+ 行死代码增加维护负担与攻击面。
   - 建议：要么删除这些只被死代码互相引用的计费实现，要么在包级加"self-use: dead code, do not invoke"注释。删除时注意 `relaykit` 模块独立性测试不受影响。

2. **`main.go` 仍注册的两个商业化后台任务**
   - `main.go:116-121`：`CHANNEL_UPDATE_FREQUENCY` 激活 `controller.AutomaticallyUpdateChannels`（渠道余额自动查询，默认不启动）。
   - `main.go:127-128`：`service.StartSubscriptionQuotaResetTask()` **默认启动**，调用 `ExpireDueSubscriptions`/`ResetDueSubscriptions` 轮询订阅表。
   - 风险：订阅轮询任务与"钱包/订阅已移除"的定位不符，持续空转 DB 轮询。
   - 建议：若订阅功能确定不使用，移除 `StartSubscriptionQuotaResetTask` 启动调用及其依赖；`CHANNEL_UPDATE_FREQUENCY` 若不需要渠道余额查询可一并移除。

3. **前端仍保留计费设置界面**
   - 位置：`web/src/features/system-settings/billing/` 仍渲染 `payment`（Epay/Stripe/Creem/Waffo/WaffoPancake）和 `checkin`（签到奖励）section。
   - 缓解：`middleware/personal_mode.go` 已在后端拒绝写入绝大多数支付 option key，只读展示不产生实际功能。
   - 建议：移除 `payment`/`checkin` 两个 section 及其翻译键，保留 `quota`/`currency`/`model-pricing`/`group-pricing` 用于 `/pricing` 只读展示。

4. **`controller/user.go:570` 残留 `defaultConfig["personal"]["topup"] = true`**
   - 前端/codex 侧已无 topup 路由，后端仍下发该模块开关，可能让前端侧边栏渲染一个空入口。
   - 建议：改为 `false` 或删除。

### P2 — 低风险债务，按需清理

5. **controller 直接访问 `model.DB`（42 处）**
   - 上游 legacy 模式，非本次引入。严格分层要求 controller 通过 service 访问数据。
   - 建议：结合上游演进分批收敛，不阻塞本分支。

6. **Go 版本文档与构建环境不一致**
   - `go.mod` 与 `relaykit/go.mod` 为 1.25.1；`Dockerfile` 使用 `golang:1.26.1-alpine`。
   - `go.mod:3` 残留 `// +heroku goVersion go1.18` 注释，已过时。
   - 建议：统一 Dockerfile 与 go.mod 版本，删除 heroku 注释。

7. **`model/task.go` 的 `type:json` 字段无显式 TEXT 回退**
   - `Properties`/`PrivateData`/`Data` 使用 `gorm:"type:json"`。SQLite 会退化为 TEXT，但按 AGENTS.md 规范"数据库特有 JSON 列类型需有 TEXT 回退"，严格合规要求补显式 fallback（实际三库运行无问题，属规约层面的轻微违规）。

8. **`service` 包存在预存 flaky 测试**
   - `service/channel_affinity_usage_cache_test.go` 的 `TestObserveChannelAffinityUsageCacheByRelayFormat_*` 在原始代码上单独运行偶发失败（共享缓存状态未清理），与本次改动无关。
   - 建议：为这些测试增加状态隔离（清理全局 affinity 缓存）。

### 已评估但决定不处理

9. **`migrateDBFast` 并发自动迁移路径**
   - `model/main.go:318-399` 的 `migrateDBFast` 未启用，当前 `InitDB` 走 `migrateDB`。若未来启用需校验 GORM 并发迁移驱动安全。
   - 未处理理由：不影响当前运行。

10. **`relaykit` 与根模块 Go 版本一致性问题**
    - 双模块构建等价通过，无实质风险。仅版本注释需在 P2 第 6 项统一处理。

## 三、修复验证

- `go build`：根模块（除 `main.go` 依赖 `web/dist` 产物）全部编译通过；`relaykit` 独立模块 `go build ./...` 通过。
- `go vet`：受影响包（service/controller/middleware/relay/setting/common/model）通过。
- 单元测试：
  - `service` 熔断/重试/Retry-After 相关测试通过；
  - `relay/helper`（流式 scanner）通过；
  - `controller`/`router`/`middleware`/`model` 通过。
  - 已知例外：`service/channel_affinity_usage_cache_test.go` flaky（见遗留第 8 项，原始代码亦失败）。
- 完整 `go build ./...` 需先构建前端 `web/dist`（任务上常规构建流程要求）。

## 四、后续建议

1. 优先处理 P1 第 2 条（移除默认启动的订阅后台任务），与"订阅已移除"定位保持一致。
2. 计费死代码（P1 第 1 条）建议在下一次上游同步前先清理或加禁用标记，避免同步时被重新引入。
3. 前端计费页面（P1 第 3 条）与后端 `topup` 开关（P1 第 4 条）应在下一次前端改动窗口一并处理，减少用户在自用界面上看到无效入口的困扰。