# New API 自用分支完整静态审查报告

## 审查信息

- 项目：buglyz/new-api
- 本地仓库：/root/.ductor/workspace/newapi
- 审查基线：upstream/main...HEAD
- 当前提交：8750eb64
- 当前分支：main
- 审查日期：2026-08-01
- 工作区状态：审查开始时干净，报告文件为本次新增内容

本报告针对当前自用故障转移分支，重点检查转发链、重试语义、个人熔断、native channel monitor、部署入口和前后端契约。

## 执行约束

本次只进行了静态代码审查和仓库差异检查，未执行：

- Go 编译或构建
- 前端构建
- 单元测试或集成测试
- Docker 构建、启动或容器操作
- 服务启动或真实上游请求

因此，报告中的触发条件和影响是基于代码路径推导，未经过运行时验证。

## 并行审查范围

审查方向包括：

1. 权限、敏感信息和租户隔离
2. native channel monitor 后端和配置持久化
3. 前端页面与接口契约
4. Docker、README 和发布基线
5. 普通转发、流式转发和任务提交
6. 个人熔断、路由预览和故障恢复

## 结果摘要

| 等级 | 数量 | 主要影响 |
| --- | ---: | --- |
| 高 | 5 | 流式协议损坏、错误被记录为成功、备用渠道失效、任务重复提交、部署使用错误镜像 |
| 中 | 8 | 退避策略、配置持久化、监控状态准确性、熔断预览/恢复、重复选路、数据库压力、开发镜像构建 |

## 高优先级问题

### H1. 流式响应已写出后仍会跨渠道重试

位置：

- controller/relay.go:220-225
- relay/channel/claude/relay-claude.go:203-210

触发条件：

1. 上游流式响应已经写入首个或多个 SSE 事件。
2. 后续读取、转换或响应写入失败。
3. handler 返回可重试错误。

当前行为：

- controller/relay.go 在调用 handler 后直接执行 shouldRetry。
- shouldRetry 没有检查 c.Writer.Written()。
- defer 中的 Writer.Written() 检查只能阻止最后的 JSON 错误响应，不能阻止已经开始的下一次渠道请求。

影响：

- 第二个渠道的 SSE 事件可能追加到第一条已经开始的流中。
- 客户端收到重复、拼接或协议损坏的响应。
- OpenAI、Claude 等流式客户端可能无法解析后续事件。

修复方向：

- 响应已写出后禁止跨渠道重试。
- 最好区分首个有效事件之前的失败和首个有效事件之后的失败。
- 流式 handler 应返回带有明确“已输出”状态的错误，避免 controller 只依赖 HTTP 状态码。

### H2. 流式故障可能被当作成功

位置：

- relay/helper/stream_scanner.go:283-289
- relay/channel/gemini/relay-gemini.go:153-206
- relay/channel/dify/relay-dify.go:231-263
- controller/relay.go:211-214

触发条件：

- scanner 超时、scanner 错误、handler Stop、流式 JSON 解析失败或 Dify error 事件发生。

当前行为：

- StreamScannerHandler 将异常写入 StreamStatus，但没有将异常返回给调用方。
- Gemini 和 Dify handler 在 scanner 完成后仍返回 usage, nil。
- Dify handler 还会无条件发送 helper.Done。
- controller 因 newAPIError == nil 进入成功分支，调用 RecordPersonalCircuitSuccess。

影响：

- 截断流、错误流或不完整流可能被记录为成功。
- 个人熔断被清除，故障渠道会继续被选中。
- 监控和请求日志的成功率失真。
- 客户端可能收到看似正常的 DONE，但内容实际不完整。

修复方向：

- StreamScannerHandler 应返回结构化的流结束状态或致命错误。
- controller 在成功前检查 StreamStatus 是否为正常结束。
- scanner、转换、写入失败时不要发送正常的终止事件。

### H3. 模型映射错误阻止备用渠道切换

位置：

- controller/relay.go:298-302
- relay/compatible_handler.go:42-44
- relay/channel/gemini/relay-gemini.go:68-70
- relay/claude_handler.go:39-41

触发条件：

- 某渠道的 model mapping 配置无效、形成环或无法解析。

当前行为：

- handler 将错误包装为 channel:model_mapped_error，并设置 skip retry。
- shouldRetry 先检查 IsSkipRetryError，再检查 IsChannelError。
- 因此该错误不会进入备用渠道。

影响：

- 单个错误渠道的配置问题会直接暴露给客户端。
- 同模型的其他健康渠道不会被尝试。
- 与本分支文档中“模型不可用时跨渠道切换”的预期不一致。

修复方向：

- 明确区分“本地请求错误”和“当前渠道配置错误”。
- 对渠道级模型映射错误允许切换渠道，或不要同时设置 skip retry。
- 如果特定场景必须禁止重试，应使用更精确的错误分类，而不是覆盖所有 channel: 错误。

### H4. 非幂等任务提交仍可能重复创建任务

位置：

- controller/relay_task.go:121-142
- controller/relay_task.go:181-205
- relay/relay_task.go:157-170

触发条件：

1. 代理向上游发送任务创建请求。
2. 上游已经接受或创建任务，但代理收到 500、502、503、429 等响应，或者读取响应/解析响应失败。
3. shouldRetryTaskRelay 根据状态码切换渠道并再次提交。

当前行为：

- 代码只排除了 408。
- 普通 5xx 和 429 仍然可能再次发送同一任务。
- 第一次创建的 upstream task id 未被保存，第二次结果覆盖第一次结果。

影响：

- 上游可能创建重复任务。
- 第一次任务成为无法查询或无法回收的孤儿任务。
- 视频、图片等昂贵或长耗时任务会产生重复资源。

修复方向：

- 任务提交默认按非幂等请求处理，发送后不要仅凭 HTTP 状态重试。
- 如果必须重试，引入贯穿代理和上游的幂等键，并确保上游按幂等键去重。
- 至少区分“请求未发出”与“请求已发出但响应未知”。

### H5. 默认部署入口仍使用官方镜像

位置：

- docker-compose.yml:19
- README.zh_CN.md:169-184、338
- README.en.md:131-146、298

当前行为：

- 本分支 README 前部介绍 ghcr.io/buglyz/new-api。
- 但根目录 docker-compose.yml 和中英文文档的常规 Docker 命令仍使用 calciumion/new-api:latest。
- 用户按仓库常规入口部署时，实际运行的是官方镜像，而不是本分支构建。

影响：

- native channel monitor、个人熔断和自定义转发逻辑不会生效。
- 部署结果与当前源代码不一致，且不会产生明显启动错误。

修复方向：

- 统一根目录 compose、README 中的默认镜像。
- 如果必须保留上游说明，应明确标记为“上游部署示例”，并把自用镜像替换成可直接复制的默认配置。

## 中优先级问题

### M1. Retry-After 只用于熔断，不用于当前请求退避

位置：

- controller/relay.go:220
- service/error.go:88-93
- service/personal_circuit.go:348-356

当前行为：

- 上游 429 的 Retry-After 会被解析并用于个人熔断时间。
- 当前请求在 shouldRetry 返回后立即继续尝试下一渠道。
- 如果多个渠道都返回 429，请求会在一个循环内快速消耗重试次数。

影响：

- 未遵守上游的请求级退避建议。
- 多渠道共享同一上游配额时，可能形成限流风暴。

修复方向：

- 明确区分“当前渠道冷却”和“当前请求退避”。
- 对请求级重试遵守 Retry-After，并对等待时间设置上限。
- 若设计上允许立即切换备用渠道，应在接口契约和日志中明确说明该策略。

### M2. 更新 native monitor 配置时忽略数据库错误

位置：

- model/option.go:227-251
- controller/option.go:367-388

当前行为：

- UpdateOption 忽略 DB.FirstOrCreate 和 DB.Save 的返回错误。
- 数据库写入失败后仍更新进程内 native monitor 配置。
- controller 最终仍返回 success: true。

影响：

- 当前进程看起来已使用新配置，重启后却恢复旧配置。
- 管理员误以为配置已持久化。
- 多节点部署时各节点可能出现不同配置。

修复方向：

- 检查并返回每个数据库操作错误。
- 数据库提交成功后再更新进程内状态。
- 配置更新接口应在持久化失败时返回失败，不记录成功审计事件。

### M3. 监控概览会展示过期的最新状态

位置：

- controller/channel_monitor_api.go:28-30
- model/channel_monitor.go:146-170

当前行为：

- 24 小时窗口只用于统计 Samples24H 和 SuccessRate24H。
- latestRows 查询仍从全部历史记录中取最新一条。
- 超过 24 小时未探测的目标仍会显示其历史健康状态。

影响：

- 页面可能把长期未探测的渠道显示为当前健康。
- unhealthy 筛选不能表达“数据已过期”。

修复方向：

- 增加 freshness/stale 字段。
- 或在概览接口中把超过窗口的目标标记为 stale，并让前端区别显示。

### M4. 路由预览忽略渠道级全模型熔断

位置：

- controller/personal_reliability.go:149-168
- service/personal_circuit.go:134-139

当前行为：

- 实际选路会优先使用 channel + "*" 的熔断条目。
- 路由预览只收集 circuit.Model == request.Model 的条目。
- 预览中该渠道仍显示为 closed/eligible，实际选择器却会跳过。

影响：

- 管理员看到的预览与真实路由不一致。
- 不能准确判断最高可用优先级。

修复方向：

- 预览合并具体模型条目和渠道级 "*" 条目。
- 复用与实际选择器一致的 canAttempt 判断和状态显示。

### M5. 成功探测不能清除渠道级全模型熔断

位置：

- controller/channel-test.go:859-864
- service/personal_circuit.go:215-235

当前行为：

- 成功探测后只按 testedModel 调用 ResetPersonalCircuit。
- reset 在传入具体模型时不会删除同渠道的 "*" 条目。
- 后续实际选路仍被渠道级熔断拦截。

影响：

- 渠道已经恢复，但成功探测无法恢复流量。
- 需要等待冷却到期或手动进行全渠道重置。

修复方向：

- 成功探测渠道时同时清除具体模型和 "*" 熔断。
- 或提供明确的 channel-level reset API。

### M6. 渠道上下文初始化失败可能重复选择同一渠道

位置：

- controller/relay.go:278-280
- service/channel_select.go:129-140
- model/channel_cache.go:139-143

当前行为：

- SetupContextForSelectedChannel 失败时 getChannel 返回已选渠道并触发重试。
- 非 SelfUseMode 的普通选择路径不会根据 use_channel 排除已尝试渠道。
- 当只有一个候选渠道时，缓存选择器会在每次重试返回同一渠道。

影响：

- 多 Key 全部禁用、渠道元数据异常等场景会重复消耗重试次数。
- 日志看似发生了多次重试，实际没有尝试新的备用渠道。

修复方向：

- 选路层统一维护当前请求已尝试渠道集合，不应只在 SelfUseMode 过滤。
- 对渠道上下文初始化错误进行明确分类，避免对不可恢复的同一渠道重复重试。

### M7. 监控结果持久化的数据库往返随目标数量线性增长

位置：

- controller/channel_monitor_task.go:110-148
- model/channel_monitor.go:86-113

当前行为：

- 监控探测可以并发执行，但结果持久化逐个目标执行独立事务。
- 每个目标至少包含最新记录查询、插入、数量统计和历史清理等数据库操作。

影响：

- 大量渠道/模型时，单次任务的数据库往返次数线性增长。
- 监控间隔较小时，持久化开销可能挤压正常业务数据库连接。

修复方向：

- 先批量写入结果，再按目标批量清理历史记录。
- retention 清理可从每条结果事务中移出，改为独立批处理。
- 保留任务级取消和失败回滚语义。

### M8. Dockerfile.dev 在复制本地 relaykit 前执行依赖下载

位置：

- Dockerfile.dev:14-17
- go.mod:165-170

当前行为：

- go.mod 使用 replace 指向本地 ./relaykit。
- Dockerfile.dev 先只复制主模块 go.mod/go.sum 就执行 go mod download。
- ./relaykit/go.mod 在该层尚未存在。

影响：

- 干净环境下开发镜像构建存在因本地 replace 目录缺失而失败的风险。
- 修改缓存层后问题可能只在 CI 或新机器上出现。

修复方向：

- 在 go mod download 前复制 relaykit/go.mod。
- 保持与生产 Dockerfile 中已有的处理一致。

## 已检查但未确认问题

1. 安全方向未发现可由当前代码证实的权限绕过、敏感信息泄露或租户隔离问题。
2. 前端方向未发现明确的用户可见接口契约回归。
3. HTTP 客户端方向未形成独立 finding：当前请求路径普遍使用请求上下文，RelayTimeout 为 0 也可能是为流式请求保留的设计。
4. git diff --check 已通过。

## 建议修复顺序

1. 先修复 H1、H2，确保流式协议不会拼接且截断流不会被记为成功。
2. 修复 H3、H4，恢复备用渠道语义并阻止非幂等任务重复提交。
3. 统一 H5 的 compose、README 和镜像发布入口。
4. 修复 M2、M4、M5，避免管理界面和实际运行状态不一致。
5. 处理 M1、M3、M6，再优化 M7、M8。

## 审查结论

当前提交具备较完整的个人故障转移和监控基础，但流式请求、任务提交和部署入口仍存在会直接影响生产行为的缺陷。尤其是流式错误处理和非幂等任务重试，不能仅作为日志或可观测性问题处理，应在请求控制流程中阻止错误状态被转换为成功或重复提交。
