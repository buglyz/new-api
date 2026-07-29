# NewAPI 调用分析页面卡死修复

日期：2026-07-29

## 问题

- Android Via 与 Chrome 进入 `/dashboard/models` 后页面卡死。
- 生产访问日志显示 `/api/data` 在 4-5ms 内返回约 7.4KB，卡死发生在响应后的图表挂载阶段。
- 当前页面最多处理 7 个时间点、14 个模型，不存在业务数据量爆炸。
- 页面通过通用 `@visactor/react-vchart` `VChart` 入口加载完整图表引擎；生产分块包含未使用的词云、Sankey、漏斗和 Treemap 等模块。

## 历史对比

- 最初官方 `v1.0.0-rc.21` 与当前代码均使用 VChart 2.1.4，调用分析组件源码一致。
- 旧官方镜像的共享图表分块传输约 406KB，当前镜像约 563KB；后续路由和依赖图变化扩大了共享块及页面驻留内存。
- 因此不能通过回退 VChart 版本解决，需收紧调用分析的导入边界。

## 最小修复

- 新增项目内的极小 React 挂载组件，直接实例化 VChart 官方包内置的 `vchart-simple` 构造器，避免 `@visactor/react-vchart` 适配层重新导入完整运行时。
- 移除该组件为设置主题而动态导入完整 `@visactor/vchart` 的逻辑；继续通过图表 spec 的 `theme` 字段切换主题。
- 每次图表规格变化时释放旧实例并重建，组件卸载时释放 Canvas、监听器和观察器。
- 不修改 API、数据聚合、筛选、图表选项和其他图表页面。

## 验证结果

- `bun test src/features/dashboard/components/models/__tests__/model-chart-runtime.test.ts`：通过，确认 area、bar、pie 已注册，wordCloud、Sankey、Funnel、Treemap 未注册。
- `bun run typecheck`：通过。
- 涉及文件 `oxlint`：通过。
- `bun run build`：通过，调用分析入口约 3.7KB（gzip 1.9KB），轻量 VChart 共享运行时约 696KB（gzip 184KB）。当前仍供其他页面使用的完整运行时约 1.44MB（gzip 408KB），个人模式的调用分析路径不会加载它。
- 构建产物扫描：调用分析分块未包含 WordCloud、Sankey、Funnel、Treemap 图表实现。
- `git diff --check`：通过。
- 非空非纯注释行：`model-charts.tsx` 121 行，`model-chart-runtime.tsx` 26 行，测试 11 行，均低于 TSX 200 行限制。

## 边界

- 本任务不修改 `/opt/newapi`、生产容器、数据库或 Caddy；生产更新需在源码修复、提交和镜像发布后单独执行。
- 个人模式下计费相关的“消费分布”图表不会挂载；其完整 VChart 路径不在本次可达页面中，故不扩大修改。
- i18n 与 Lobe 图标整包问题另行治理，不在本次最小修复中扩展。
