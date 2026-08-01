# 渠道请求次数展示

日期：2026-08-01

## 范围

- 为渠道增加持久化 `request_count`，在消费记录链路中按渠道原子递增。
- 渠道桌面表格、移动卡片和标签聚合行显示请求次数，移除“已使用 / 剩余”。
- 复制渠道时默认重置请求次数，并禁止通过渠道更新接口写入服务端字段。
- 增加 Go 和前端回归测试。

## 验证

- `go test ./...`：通过。
- `cd web && bun test`：通过，151/151。
- `cd web && bun run typecheck`：通过。
- `cd web && bun run build`：通过。
- 变更文件定向 lint：通过。
- `git diff --check`：通过。

## 发布与部署

- 待提交并推送 `main`。
- 待 GitHub Docker Action 成功后更新 `/opt/newapi`，更新前备份 Compose 和 SQLite，并验证容器健康及本地/公网 `/api/status`。
