# New API 自用分支完整审查报告

## 审查信息

- 项目：buglyz/new-api（fork 自 QuantumNous/new-api，单管理员自用故障转移版）
- 审查方式：并行多 agent 静态审查（4 个维度）+ 人工复核关键代码路径
- 审查日期：2026-08-07
- 基线：main 分支
- 结论总览：核心故障转移/熔断链路与 README 描述存在系统性偏差；计费移除在请求链上彻底但在代码与文档层有大量残留；安全边界良好；架构（relaykit 拆分、部署脚本）质量高。

## 执行约束

本次为静态代码审查与仓库差异分析，未执行：

- 前端构建（web/dist 产物缺失，完整 `go build ./...` 无法执行）
- Docker 构建/启动
- 真实上游请求或运行时验证
- 报告中的触发条件基于代码路径推导

## 一、故障转移与熔断实现

### 1.1 重试决策逻辑 —— 良好，基本符合 README

符合设计：

- 状态码分类正确：`setting/operation_setting/status_code_ranges.go:21-28` 定义可重试范围 401/403/408/425/429/500-599。
- `specific_channel_id` 指定时禁止切换：`controller/relay.go:298-300`、`controller/relay_task.go:191-193` 均正确拦截。
- 任务提交非幂等，`408` 不扩大重试：`controller/relay_task.go:197-201`。
- 流式部分输出后不重试：`controller/relay_retry.go:10-12` 检查 `ReceivedResponseCount > 0`。

潜在问题：

- `controller/relay.go:315-316`：状态码 `<100 || >599` 时返回 `true`（可重试），对异常非标准码（如 600+）可能错误重试。

### 1.2 流式故障转移边界 —— 修复前与文档不符（已修复）

修复前发现：

- `relay/helper/stream_scanner.go` 的 `ticker.Reset(streamingTimeout)` 在空行/注释过滤前执行，空行、comments、heartbeat 都会重置计时器，与"空行注释 heartbeat 不重置首事件计时器"矛盾。
- 无独立首事件计时器，`STREAM_FIRST_EVENT_TIMEOUT` 环境变量在代码中完全不存在；只有统一的 `STREAMING_TIMEOUT`（默认 300s）。

符合文档的部分：

- `[DONE]`/EOF 正常完成：`stream_scanner.go:276-280`。
- 客户端断开不触发熔断：`stream_scanner.go:298-301`。
- 部分输出后不重新写响应/拼流：`relay_retry.go`。

### 1.3 熔断器实现 —— 修复前退避常量与文档不一致（已修复）

修复前：

| 文档描述 | 代码实际 |
| --- | --- |
| model_not_found：30 分钟 | `personalCircuitModelBackoff = 10min` |
| 渠道级配置失败：15 分钟 | `personalCircuitChannelBackoff = 10min` |
| 其他可恢复故障：30s 起，最长 15min | `base = 15s`、`max = 5min` |

- `Retry-After` 被解析（`service/error.go:90,153`）但熔断器从未使用（已修复）。

符合文档：

- half-open 只允许一个探测：`personal_circuit.go:101-132`。
- 成功后清除 scope：`recordSuccess`。
- 进程内状态、重启清空：`var personalCircuits`。
- 并发安全：`sync.Mutex` 保护所有访问。
- 429 不计入熔断（upstream 存活只是繁忙）：`personal_circuit_policy.go:31-38`。

### 1.4 超时配置链 —— 修复前缺 5 个环境变量（已修复 4 个）

修复前 `STREAM_FIRST_EVENT_TIMEOUT`、`RELAY_FAILOVER_BUDGET`、`RELAY_NON_STREAM_TIMEOUT`、`RELAY_CONNECT_TIMEOUT`、`RELAY_RESPONSE_HEADER_TIMEOUT` 在代码中均不存在，只有 `RELAY_TIMEOUT` 与 `STREAMING_TIMEOUT`（默认 300）。

### 1.5 评价：需改进 → 修复后良好

## 二、单管理员模式安全边界

### 2.1 移除功能的彻底性 —— 良好

- `router/api-router.go` 不注册注册/找回密码/OAuth/充值/兑换码/签到/邀请等路由。
- `router/retired_frontend_routes_test.go:28-67` 断言 9 个退役路由不存在，防止上游同步误引入。
- `middleware/personal_mode.go:13-87` 拒绝写入 75 个商业 feature 的 option key。

残留（不可达）：

- `controller/user.go` 的 `Register`/`GetAllUsers`/`SearchUsers`/`DeleteUser`/`ManageUser`/`GetAffCode` 等函数保留但无路由。
- `controller/misc.go:292` `ResetPassword` 保留但无路由。

### 2.2 单管理员模式强制 —— 良好

- `controller/setup.go:103-111` 初始化只创建一个 root，`SelfUseModeEnabled=true`、`DemoSiteEnabled=false`（`setup.go:122-124`、`personal_mode.go:15` 禁改）。
- `controller/misc.go:87-90` 状态接口硬编码自用/注册关闭。

潜在问题：

- `PostSetup` 在已初始化时返回 200 而非 403。
- 无显式用户总数=1 限制，依赖路由移除隔离。

### 2.3 认证与 Session 安全 —— 优秀

- `common/session_cookie.go`: secure 与 trusted URL 互相强制、仅 https、精确 Origin（无通配）。
- `common/init.go:50-57`：`SESSION_SECRET` 设为默认值时启动失败。
- `middleware/auth.go:502-508`：`specific_channel_id` 仅管理员可用。
- `middleware/auth.go:427-441`：token IP 白名单。

### 2.4 请求许可边界 —— 符合文档

`Distribute`/`TokenAuth` 完全不检查 quota/balance，只检查渠道状态、分组权限、模型限制、熔断。

## 3、计费移除完整性

### 3.1 请求链计费残留 —— 基本彻底，一处残留（已修复）

- relay 主链、任务链无 pre-consume/settle/refund 调用；`RecordConsumeLog` Quota=0。
- **修复前**：`controller/midjourney.go:204-231` 对历史任务失败仍 `model.IncreaseUserQuota` 真实修改用户余额 —— 本次已改为仅清除遗留标记。

### 3.2 死代码 —— 大量存在（未处理，见遗留）

`service/billing.go`、`billing_session.go`、`funding_source.go`、`tiered_settle.go`、`violation_fee.go`、`epay.go`、`waffo_pancake.go`、`quota.go` 预扣/结算函数、`relay/helper/price.go`、`pkg/billingexpr` 模块、`controller/billing.go`、`channel-billing.go`。约 4000+ 行。

### 3.3 文档不一致 —— 已部分修复

AGENTS.md billing 规则、`relay/relay_task.go:111` 过时注释（已修复）、前端计费设置 UI（未修复）、`controller/user.go:570 topup=true`（未修复）。

### 3.4 结论：计费移除核心路径彻底，但残留量大

## 4、架构与代码质量

### 4.1 relaykit 模块独立性 —— 优秀（亮点）

- 独立 go.mod（1.25.1），零根模块依赖。
- `relaykit/relayconvert/boundary_test.go` 用 `go/parser` 静态检查 import，禁止引入根模块与 gin，`allowedViolations` 为空历史耦合已清零。
- 依赖注入交互（`kitutil.SetLogging` 等），边界清晰。

### 4.2 分层 —— 良好

controller 直访 `model.DB`（42 处）为上游遗留，非本分支引入。

### 4.3 代码质量 —— 良好

`common/quota_math.go` 饱和+审计设计；错误处理一致；并发安全措施到位。

### 4.4 数据库兼容性 —— 良好，局部违规

- `lockForUpdate` helper 正确（SQLite 跳过，MySQL/PG 加锁）。
- 迁移三库分支完备（`migrateTokenModelLimitsToText` 等）。
- 违规：`model/task.go:49` AUTO_INCREMENT（已修复）；`setting/console_setting/validation.go` 直调 `encoding/json`（已修复）；`model/task.go` 的 `type:json` 无显式 TEXT 回退（未修复）。

### 4.5 构建与文档 —— 良好

Go 版本不一致：go.mod 1.25.1，Dockerfile `golang:1.26.1`，`go.mod` 残留 heroku 注释（未修复）。

### 4.6 部署配置 —— 优秀（亮点）

`deploy/personal/maintenance.sh`：不可变镜像校验、SQLite 在线备份 + `PRAGMA quick_check`、自动镜像回滚；compose 仅监听 loopback。

## 结论与遗留

详见 `docs/reviews/newapi-review-followup-20260808.md`（遗留问题优先级、缓存、验证与后续建议）。修复验证：根模块（除 web/dist）与 relaykit 独立模块 `go build` 通过，`go vet` 通过，相关单元测试通过（除预存 `channel_affinity_usage_cache_test.go` flaky，与本次改动无关，在原始代码亦失败）。