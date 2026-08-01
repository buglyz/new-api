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

- 已提交并推送 `main`：`362cde72`。
- GitHub Docker Action `30701870609` 成功，镜像 manifest digest 为
  `sha256:b5bc355f98a26d32313c184409f384e6c3b5e9b5cef1e28f5db5a170ed57fc34`。
- 已更新 `/opt/newapi`，备份为 `compose-before-request-count-20260801-213848.yaml` 和
  `one-api-before-request-count-20260801-213848.db`。
- 容器运行版本为 `main-362cde7`，状态 `healthy`、重启次数 0；本地/公网 `/api/status`
  均返回成功，线上 SQLite `quick_check` 为 `ok`。
