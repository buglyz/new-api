# NewAPI 熔断高/中优先级修复

日期：2026-08-03
状态：实现与验证完成，Git 推送受执行策略阻断

## 范围

修复个人自用模式下已确认的熔断与故障转移一致性问题：

- 流式扫描异常必须进入 relay 失败路径；已有上游流数据时不得拼接第二条流。
- 渠道模型映射、参数覆盖等渠道配置错误允许切换备用渠道；本地请求错误仍保持不重试。
- HTTP 408/425 重试时同时进入个人熔断。
- 成功恢复探测清理具体模型和渠道级 `*` 熔断。
- 可靠性路由预览与真实选择器一致处理渠道级 `*` 熔断。

## 约束

- 不创建后台任务。
- 不部署、不修改 `/opt/newapi`、生产容器、数据库或 Caddy。
- 不修改测试来掩盖实现错误；只添加真实边界回归覆盖。

## 验证

- [x] 受影响包定向 `go test`
- [x] 熔断与流状态定向 `go test -race`
- [x] `gofmt`、`git diff --check`、文件行数检查
- [ ] 本地提交并推送 `origin/main`（当前会话禁止修改 `.git`）

## 已验证命令

- `go test ./... -count=1`
- `go test -race ./relay/helper -run 'TestStreamStatusError|TestStreamScannerHandler_StreamStatus' -count=1`
- `go test -race ./service -run 'TestPersonalCircuit|TestClassifyRelayAttempt|TestChannelSelectionHonorsCircuitCooldownAndSingleHalfOpenClaim' -count=1`
- `go test -race ./controller -run 'TestShouldRetry|TestPersonalCircuitPreview' -count=1`

## 推送阻断

当前会话的执行策略将 `git add`、`git commit` 等 `.git` 写操作判定为需要审批，但审批策略为 `never`，因此无法创建提交或推送；代码修改仍保留在工作区，生产环境未触碰。
