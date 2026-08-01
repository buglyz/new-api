# NewAPI 镜像体积优化

日期：2026-08-01

## 改动

- 最终运行时从 Debian slim 切换到固定 digest 的 Alpine 3.22。
- 移除未使用的 `libasan8` 及其 Debian 运行时安装逻辑。
- 删除 Knip 确认且无业务入口的旧前端组件和依赖，包括旧 Recharts、React Flow、轮播、可调整布局和 Tokenlens 模板组件。
- 普通折线、面积、柱状和饼图改用 `vchart-simple`；Flow Sankey 使用仅注册 Sankey 所需模块的自定义运行时。

## 验证

- Go 全量测试：通过。
- 前端测试：152/152 通过。
- 前端 typecheck：通过。
- 变更文件定向 lint：通过。
- Dockerfile `buildx --check`：通过。
- 前端 bundle：约从 56,923 KB / 16,443 KB gzip 降至 56,228 KB / 16,258 KB gzip。
- 本地完整 Docker 镜像构建在复制 `web/dist` 时因宿主机磁盘峰值不足失败；前端和 Go 编译步骤均已成功，GitHub runner 负责最终构建验证。

## 发布

- 待提交并推送 `main`。
- 待 GitHub Docker Action 成功后，用新 digest 更新 `/opt/newapi`，更新前备份 Compose 和 SQLite。
