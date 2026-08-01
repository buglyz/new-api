# New API 完整代码审查报告

- 审查日期：2026-08-01
- 仓库：`https://github.com/buglyz/new-api`
- 审查分支：`main`
- 基线提交：`8b054626 feat(monitor): integrate native channel monitoring`
- 审查对象：基线提交及当前工作区全部未提交修改、未跟踪文件
- 审查方式：多 agent 并行静态审查、差异交叉核对、调用链核对
- 报告状态：完成

## 结论

当前工作区包含一处确定的 Go 编译阻断问题，以及多个会影响监控页面、任务取消、配置约束、敏感错误信息和可靠性预览的运行时问题。除报告文件外，本次没有修改业务实现，也没有通过修改测试来掩盖问题。

严重性汇总：

- P0：1 项，确定阻断 Go 编译
- P1：7 项，影响核心功能、配置安全边界或任务控制
- P2：4 项，影响可用性、数据保留或预览准确性

## P0 阻断问题

### P0-1 `lower` 局部变量声明后未使用

- 位置：[controller/channel_monitor_probe.go:82](https://github.com/buglyz/new-api/blob/main/controller/channel_monitor_probe.go#L82)
- 证据：`channelMonitorEndpointType` 声明了 `lower := strings.ToLower(strings.TrimSpace(modelName))`，后续循环使用的是另一个名为 `candidate` 的变量，`lower` 没有任何读取。
- 影响：Go 编译器会将未使用的局部变量视为编译错误，当前监控实现无法通过 Go 编译。
- 建议：删除无用声明，或将其用于预期的模型判断逻辑；修复后补充定向编译验证。

## P1 高优先级问题

### P1-1 监控概览查询使用不存在的数据库列

- 位置：[model/channel_monitor_channels.go:6](https://github.com/buglyz/new-api/blob/main/model/channel_monitor_channels.go#L6)、[model/channel.go:56](https://github.com/buglyz/new-api/blob/main/model/channel.go#L56)
- 证据：概览查询选择 `other_settings`；`Channel.OtherSettings` 的 GORM 映射列名是 `settings`。
- 影响：数据库使用实际 schema 时，`GET` 监控概览会因未知列失败，前端无法加载监控页面。
- 建议：使用 GORM 字段映射或明确选择真实列名 `settings`，并覆盖 SQLite/MySQL/PostgreSQL 的概览查询契约。

### P1-2 监控超时 Context 没有传递到实际上游 HTTP 请求

- 位置：[controller/channel_monitor_probe.go:247-254](https://github.com/buglyz/new-api/blob/main/controller/channel_monitor_probe.go#L247)、[controller/channel_test_support.go:36-40](https://github.com/buglyz/new-api/blob/main/controller/channel_test_support.go#L36)、[relay/channel/api_request.go:307-330](https://github.com/buglyz/new-api/blob/main/relay/channel/api_request.go#L307)、[relay/channel/api_request.go:474-513](https://github.com/buglyz/new-api/blob/main/relay/channel/api_request.go#L474)
- 证据：监控层创建 `context.WithTimeout` 并把它放入 `c.Request`；`DoApiRequest` 随后用 `http.NewRequest` 新建请求，但没有调用 `req.WithContext(c.Request.Context())`，最终直接 `client.Do(req)`。
- 影响：配置的 `timeout_seconds` 和任务取消无法可靠中断正在进行的拨号、读取或流式响应。监控 worker 可能长期占用，任务禁用或节点停止也可能延迟完成。
- 建议：所有由 `gin.Context` 派生的上游 HTTP 请求显式继承请求 Context；同时为需要特殊传输的 WebSocket/其他 adaptor 定义一致的取消语义。

### P1-3 隐藏错误消息时用 `fmt.Printf` 输出原始错误，绕过 quiet 日志抑制

- 位置：[relaykit/types/error.go:413-419](https://github.com/buglyz/new-api/blob/main/relaykit/types/error.go#L413)
- 证据：`ErrOptionWithHideErrMsg` 在 Debug 模式直接 `fmt.Printf` 原始 `e.Err`，之后才把错误替换为安全文本；这条输出不经过 `logger.Log*`，也不检查请求日志抑制标记。
- 影响：自动监控等 quiet 请求的上游 URL、网络错误或其他原始错误内容可能进入标准输出/日志；隐藏错误消息的安全边界被绕过，错误中包含敏感信息时存在泄露风险。
- 建议：不要在该 option 中输出原始错误；若确需诊断，使用统一 logger、请求级抑制判断和已有敏感信息脱敏策略，并确保只向受控服务端日志输出。

### P1-4 通用选项接口可以绕过 native monitor 配置边界校验

- 位置：[controller/channel_monitor_api.go:153-194](https://github.com/buglyz/new-api/blob/main/controller/channel_monitor_api.go#L153)、[controller/option.go:124-143](https://github.com/buglyz/new-api/blob/main/controller/option.go#L124)、[model/option.go:222-247](https://github.com/buglyz/new-api/blob/main/model/option.go#L222)
- 证据：专用监控接口校验 interval、concurrency、timeout、retry 和 pattern；通用 `UpdateOption` 接受任意 key，带 `native_monitor_setting.` 前缀的值直接进入 `UpdateNativeMonitorSettingFromMap`，该路径没有复用 `normalizeChannelMonitorConfig` 的范围和 wildcard 校验。
- 影响：管理员可通过通用设置接口写入专用接口拒绝的值，导致调度周期、并发度、重试和排除规则与 API 契约不一致，极端值可能造成资源消耗或监控失效。
- 建议：为 native monitor 选项建立唯一的 schema/validator，并让专用接口和通用接口共同调用；拒绝未知字段，避免仅按前缀接受任意配置。

### P1-5 禁用监控时只能取消当前进程的任务

- 位置：[controller/channel_monitor_api.go:106-109](https://github.com/buglyz/new-api/blob/main/controller/channel_monitor_api.go#L106)、[service/system_task_lifecycle.go:83-103](https://github.com/buglyz/new-api/blob/main/service/system_task_lifecycle.go#L83)
- 证据：更新配置后调用 `CancelSystemTaskType`；任务运行表 `systemTaskRuns` 是进程内 map，取消操作只遍历当前进程注册的 `context.CancelFunc`，没有更新共享数据库任务状态或发布跨节点取消信号。
- 影响：多 master/多节点部署中，其他节点正在执行的监控任务不会因本节点禁用配置而立即取消，可能继续访问上游并写入监控结果。
- 建议：将取消意图写入共享任务状态或增加带版本/取消标记的数据库协议；各节点在探测批次和上游请求前检查共享状态，同时保留本地 Context 取消以快速退出。

### P1-6 空排除模式可能序列化为 `null`，导致前端设置页崩溃

- 位置：[setting/operation_setting/native_monitor_setting.go:84-86](https://github.com/buglyz/new-api/blob/main/setting/operation_setting/native_monitor_setting.go#L84)、[web/src/features/channel-monitor/components/channel-monitor-settings.tsx:53-57](https://github.com/buglyz/new-api/blob/main/web/src/features/channel-monitor/components/channel-monitor-settings.tsx#L53)
- 证据：`cloneNativeMonitorSetting` 使用 `append([]string(nil), setting.ExcludePatterns...)`；空切片会被克隆成 nil，JSON 编码后为 `null`。前端初始化表单时无条件调用 `settings.exclude_patterns.join('\n')`。
- 影响：监控配置为空排除列表时，设置页面可能抛出 `Cannot read properties of null`，无法正常渲染或保存配置。
- 建议：在后端保证空列表始终编码为 `[]`，并在前端 API 解码层对该字段做契约校验；不要用 UI 层静默掩盖响应结构漂移。

### P1-7 native monitor 的配置更新与任务运行之间缺少一致的取消/快照语义

- 位置：[controller/channel_monitor_api.go:88-109](https://github.com/buglyz/new-api/blob/main/controller/channel_monitor_api.go#L88)、[controller/channel_monitor_task.go:53-107](https://github.com/buglyz/new-api/blob/main/controller/channel_monitor_task.go#L53)
- 证据：任务开始时一次性读取设置并据此收集 targets、重试次数和并发；配置更新只取消当前进程的已注册任务并唤醒 runner，运行中的其他节点或已进入探测循环的任务不会重新确认配置版本。
- 影响：配置已显示为禁用或已修改时，旧任务仍可能继续完成部分探测并写入结果，用户看到的状态与实际执行批次不一致。
- 建议：给任务 payload 或共享配置增加版本/取消标识，在每个批次及持久化前检查；将配置快照和执行结果关联，避免旧任务覆盖新配置语义。

## P2 中优先级问题

### P2-1 监控轮询会覆盖用户尚未保存的表单编辑

- 位置：[web/src/features/channel-monitor/index.tsx:55-59](https://github.com/buglyz/new-api/blob/main/web/src/features/channel-monitor/index.tsx#L55)、[web/src/features/channel-monitor/components/channel-monitor-settings.tsx:67-69](https://github.com/buglyz/new-api/blob/main/web/src/features/channel-monitor/components/channel-monitor-settings.tsx#L67)
- 证据：任务存在时概览每 5 秒刷新；每次 `props.settings` 变化都调用 `form.reset(...)`，没有判断表单 dirty 状态。
- 影响：管理员在任务运行期间编辑数值或排除模式时，刷新可能无提示地丢失未保存输入。
- 建议：仅在表单未 dirty 时 reset，或在刷新导致服务端值变化时显示冲突提示。

### P2-2 前端把任务查询网络错误直接显示为任务失败

- 位置：[web/src/features/channel-monitor/hooks/use-channel-monitor-task.ts:48-59](https://github.com/buglyz/new-api/blob/main/web/src/features/channel-monitor/hooks/use-channel-monitor-task.ts#L48)
- 证据：`taskQuery.isError` 与服务端返回的 terminal failure 共用“Monitor run failed”分支，且查询设置 `retry: false`。
- 影响：一次临时网络错误会停止任务监听、清空 task ID，并误导用户认为后端任务已经失败；后续任务实际状态只能依靠手动刷新发现。
- 建议：区分 transport error 和任务 terminal status；对查询错误进行有限退避重试，并在无法确认状态时保留“状态未知”而不是失败。

### P2-3 监控历史清理在超限但 24 小时内没有可删旧记录时不会收敛

- 位置：[model/channel_monitor.go:98-115](https://github.com/buglyz/new-api/blob/main/model/channel_monitor.go#L98)
- 证据：当总数超过 `historyLimit` 时，只查询 `created_at < now-24h` 的记录作为待删集合；如果超限记录全部仍在 24 小时窗口内，`staleIDs` 为空，本次插入不会删除任何记录。
- 影响：手动频繁触发、短周期配置或历史配置变化时，单个 channel/model 的结果数量可能超过保留上限并持续增长，增加数据库空间和查询成本。
- 建议：清理应同时满足数量上限和时间窗口，按 `created_at,id` 保留最新 `historyLimit` 条；对全局历史增加定期清理兜底。

### P2-4 个人可靠性路由预览可能忽略渠道级熔断，也可能展示已禁用渠道

- 位置：[controller/personal_reliability.go:144-168](https://github.com/buglyz/new-api/blob/main/controller/personal_reliability.go#L144)、[service/personal_circuit.go:134-139](https://github.com/buglyz/new-api/blob/main/service/personal_circuit.go#L134)、[model/channel_route_preview.go:75-79](https://github.com/buglyz/new-api/blob/main/model/channel_route_preview.go#L75)
- 证据：预览只把 `circuit.Model == request.Model` 放入 `circuitByChannel`，忽略代表整个渠道的 `Model == "*"`；只有找到精确模型熔断时才调用 `PersonalCircuitCanAttempt`。候选查询按 ability `enabled` 过滤，但查询本身没有同时约束 `channel.status`。
- 影响：预览可能把实际被渠道级熔断阻断的候选显示为 `Closed/Eligible`，也可能把渠道状态已禁用但 ability 仍启用的记录展示给管理员；预览结果与真实路由选择不一致。
- 建议：复用真实选择器的渠道状态和熔断判定，统一处理精确模型与 `*` 渠道级状态，并为预览增加对应回归用例。

## 已确认的正向检查项

- 监控错误文本增加了对 key、配置 JSON、URL、query 参数和通用敏感信息的脱敏路径，但该路径仍无法抵消 P1-3 中 `fmt.Printf` 绕过统一日志的风险。
- 任务完成路径会根据 Context 错误把取消中的任务标记为失败，避免取消任务长期停留在运行状态；这不解决跨节点取消问题。
- 监控历史接口对请求 `limit` 做了上限限制；该限制不替代数据库层的总量清理。
- `git diff --check` 已通过。

## 验证边界

按用户要求，本次没有执行以下操作：

- 编译或运行 Go 程序
- 前端构建、打包或类型检查
- 单元测试、集成测试或端到端测试
- 启动开发服务或连接外部数据库/上游服务

因此，P0-1 是基于 Go 语言规则的静态确定性结论；其余问题是基于当前源码、数据库映射、调用链和前端状态逻辑的静态确认，仍建议在修复后执行受影响范围内的定向验证。

## 提交范围说明

本次提交将包含当前工作区在审查开始前已存在的全部修改、未跟踪文件和本报告。没有回退、覆盖或删除与本任务无关的用户修改。
