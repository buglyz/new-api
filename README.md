<div align="center">

![New API](./web/public/logo.png)

# New API 自用故障转移版

面向单一管理员的多上游 AI 网关，用于聚合低 SLA、公益或免费上游。

[上游项目](https://github.com/QuantumNous/new-api) ·
[自用镜像](https://github.com/buglyz/new-api/pkgs/container/new-api) ·
[部署与回滚](./deploy/personal/README.zh-CN.md) ·
[上游文档](https://docs.newapi.pro/)

[![Publish personal Docker image](https://github.com/buglyz/new-api/actions/workflows/personal-docker.yml/badge.svg?branch=main)](https://github.com/buglyz/new-api/actions/workflows/personal-docker.yml)

</div>

> [!IMPORTANT]
> 这是 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 的自用分支，产品身份仍为 **New API**。本分支固定为单管理员模式，不是面向公众运营、售卖额度或管理多个用户的 SaaS 版本。

## 适用场景

这个分支只解决一件事：让一个人可以通过一组稳定的 API Key，统一使用多个质量不稳定、随时可能限流或下线的上游。

请求能否执行只由以下技术条件决定：

- API Key 和登录鉴权是否有效；
- Key、分组、模型和渠道是否匹配；
- 请求是否触发速率限制；
- 是否存在未处于冷却期的可用渠道；
- 请求格式和本地转换是否合法。

请求链不读取余额或金额额度决定是否放行，也不执行模型计价、预扣、结算、补扣或退款。数据库中的历史金额字段为 SQLite、MySQL、PostgreSQL 升级兼容而保留，但不参与自用请求链。

## 保留与移除

| 保留 | 明确移除或禁用 |
| --- | --- |
| 单一管理员登录、Session、Passkey、2FA | 注册、找回密码和第三方 OAuth 登录 |
| API Key、渠道、模型、分组和权限约束 | 多用户管理、用户排行和公开营销页面 |
| OpenAI、Claude、Gemini 等协议入口 | 钱包、充值、支付、订阅和兑换码 |
| 请求速率限制和模型速率限制 | 签到、推广、邀请和额度分发 |
| token 数、请求数、延迟、错误和重试链观测 | token 金额用量、渠道余额和部署价格估算接口 |
| 跨渠道重试、临时熔断、半开探测和恢复 | 基于余额、quota 或价格的请求许可 |
| 渠道测试、批量探测、性能与系统运维 | 自动恢复商业功能的兼容开关 |
| 登录保护的 `/pricing` 模型广场 | 价格换算、充值汇率和实际扣费 |

`/pricing` 中的价格字段只用于只读参考，不影响选路、请求许可或日志中的技术用量统计。

## 故障转移语义

### 重试

- `401`、`403`、`408`、`425`、`429`、全部 `5xx` 和明确的 `model_not_found` 可以切换渠道。
- 普通 `400`、`402`、`404`、`409`、`422` 和本地校验/转换错误不会跨渠道重试。
- 指定 `specific_channel_id` 时不会切换到其他渠道。
- 任务提交属于非幂等操作；可能已经被上游接收的 `408` 不会扩大重试。
- 流式请求只允许在首个有效 `data:` 到达前故障转移。已经输出部分 SSE 后，不会跨渠道拼接第二条流。

### 流式处理

- 默认等待首个有效事件 35 秒，空行、注释和 heartbeat 不重置首事件计时器。
- 收到有效事件后切换为 90 秒流空闲超时，可通过环境变量调整。
- `[DONE]` 和兼容 EOF 视为正常完成。
- 部分输出后的 timeout、scanner、转换 panic 或 ping 失败会记录失败，但不会再次写 JSON 响应或跨渠道拼流。
- 客户端主动断开会立即关闭上游响应体，不会误判为渠道故障或打开熔断。

### 熔断

- 鉴权或渠道级配置失败使用 15 分钟 channel-wide 冷却。
- `model_not_found` 使用 30 分钟 model scope 冷却。
- 其他可恢复故障从 30 秒开始退避，最长 15 分钟。
- 支持 `Retry-After` 秒数和 HTTP 日期格式，并受最大冷却时间限制。
- 冷却到期后只允许一个 half-open 请求；成功会清除命中的熔断 scope。
- 熔断状态保存在当前进程内，重启后清空。

## 支持的入口

| 协议或任务 | 主要入口 |
| --- | --- |
| OpenAI Chat Completions | `/v1/chat/completions` |
| OpenAI Responses | `/v1/responses` |
| OpenAI Responses Compact | `/v1/responses/compact` |
| OpenAI Realtime | `/v1/realtime` |
| Claude Messages | `/v1/messages` |
| Gemini | `/v1beta/models/...` |
| Embeddings、Images、Audio、Rerank | 对应 `/v1/...` 标准路径 |
| Midjourney、Suno 和异步任务 | `/mj/...`、`/suno/...` 及任务查询接口 |

渠道适配包含 OpenAI Compatible、Claude、Gemini、Azure、AWS Bedrock、New API、Sub2API、Advanced Custom 等上游类型。协议转换核心位于独立的 [`relaykit`](./relaykit/README.md) 模块。

## 快速开始

### 镜像

```bash
docker pull ghcr.io/buglyz/new-api:latest
```

可用平台为 `linux/amd64` 和 `linux/arm64`：

- `latest`、`main`：随 `main` 更新的滚动标签；
- `sha-<完整40位提交哈希>`：不可变提交标签，适合正式部署和回滚；
- digest：最严格的不可变引用。

### 单容器 SQLite 模板

仓库提供仅监听回环地址的 Compose 模板：

```bash
git clone https://github.com/buglyz/new-api.git
cd new-api/deploy/personal

cp .env.example .env
cp .image.env.example .image.env
chmod 600 .env .image.env
```

编辑 `.env`：

- 将 `SESSION_SECRET` 替换为独立随机值；
- 使用 HTTPS 反向代理时，保留 `SESSION_COOKIE_SECURE=true`；
- 将 `SESSION_COOKIE_TRUSTED_URL` 改为实际的精确 HTTPS Origin。

编辑 `.image.env`，首次测试可以使用滚动镜像，稳定运行后建议固定到完整 SHA 或 digest：

```env
NEW_API_IMAGE=ghcr.io/buglyz/new-api:latest
NEW_API_CONTAINER_NAME=new-api
NEW_API_PORT=18080
TZ=Asia/Shanghai
```

检查并启动：

```bash
docker compose --env-file .image.env -f compose.yml config --quiet
docker compose --env-file .image.env -f compose.yml up -d
docker compose --env-file .image.env -f compose.yml ps
```

默认访问地址为 `http://127.0.0.1:18080`。首次打开后只初始化一个管理员账号，再配置渠道、模型、分组和 API Key。

> [!WARNING]
> 不要将 SQLite 数据目录放在临时文件系统中，也不要在没有可信反向代理和 HTTPS 配置的情况下直接暴露管理界面。

## 备份、升级与回滚

[`deploy/personal/maintenance.sh`](./deploy/personal/maintenance.sh) 不会自动运行，只处理本目录的单容器 SQLite 部署。

```bash
cd deploy/personal

./maintenance.sh status
./maintenance.sh backup
./maintenance.sh upgrade ghcr.io/buglyz/new-api:sha-<完整40位提交哈希>
```

升级命令只接受完整 SHA 标签或 digest。它会先使用 SQLite 在线 `.backup` 创建备份并执行 `PRAGMA quick_check`；新镜像健康检查失败时自动恢复旧镜像，但不会自动回滚数据库。

```bash
# 只回滚镜像
./maintenance.sh rollback backups/<UTC时间戳>

# 显式恢复数据库和配置
./maintenance.sh rollback backups/<UTC时间戳> --restore-database --restore-config
```

有状态回滚会先备份当前状态。完整说明见 [个人模式部署与回滚](./deploy/personal/README.zh-CN.md)。外部 MySQL、PostgreSQL 或独立日志数据库必须使用对应数据库的一致性备份工具。

## 常用环境变量

| 变量 | 默认值 | 作用 |
| --- | ---: | --- |
| `SESSION_SECRET` | 无 | Session 签名密钥，多节点必须一致 |
| `SESSION_COOKIE_SECURE` | `false` | HTTPS 下启用 Secure Cookie 和严格 Origin 检查 |
| `SESSION_COOKIE_TRUSTED_URL` | 无 | 允许 refresh/logout 的精确 HTTPS Origin |
| `SQL_DSN` | 空 | 留空使用 SQLite；也支持 MySQL/PostgreSQL |
| `REDIS_CONN_STRING` | 空 | 可选 Redis 缓存和分布式限流 |
| `RELAY_CONNECT_TIMEOUT` | `10` | 上游连接超时，秒 |
| `RELAY_RESPONSE_HEADER_TIMEOUT` | `20` | 等待上游响应头超时，秒 |
| `RELAY_NON_STREAM_TIMEOUT` | `60` | 非流式单次请求超时，秒 |
| `RELAY_FAILOVER_BUDGET` | `90` | 一次请求的总故障转移预算，秒 |
| `STREAM_FIRST_EVENT_TIMEOUT` | `35` | 首个有效流事件超时，秒 |
| `STREAMING_TIMEOUT` | `90` | 流式空闲超时，秒 |
| `STREAM_SCANNER_MAX_BUFFER_MB` | `128` | 单个流事件允许的最大扫描缓冲 |
| `MAX_REQUEST_BODY_MB` | `128` | 解压后的最大请求体 |
| `ERROR_LOG_ENABLED` | `false` | 是否记录 relay 错误日志 |
| `NODE_NAME` | 自动生成 | 日志和实例观测中的节点名称 |

完整环境变量和提供商配置仍可参考 [New API 上游文档](https://docs.newapi.pro/)，但其中支付、计费、多用户和公开运营章节不适用于本分支。

## 从源码构建

需要 Go 1.25.1 或更高版本以及 Bun：

```bash
cd web
bun install --frozen-lockfile
bun run typecheck
bun run build

cd ..
go test ./...
go build -o new-api .
```

`relaykit` 是独立 Go module，需要单独验证：

```bash
cd relaykit
GOWORK=off go test ./...
GOWORK=off go build ./...
```

## 上游同步

本分支会持续跟踪 [QuantumNous/new-api](https://github.com/QuantumNous/new-api)，但不会自动接受上游改动。上游的大型协议重构、router、relay、billing 或数据库变更必须人工审查，防止重新引入商业入口或计费依赖。

同步后至少验证：

- relay 主链、Realtime、Responses Compact、异步任务、Midjourney 和渠道测试不调用计价/预扣/结算；
- 商业 API 不重新注册，`GET /api/pricing` 仍使用登录鉴权和统一响应包装；
- 流式首事件、部分输出、DONE/EOF、重试和熔断行为不回退；
- SQLite、MySQL、PostgreSQL 不执行破坏性金额字段迁移；
- 根模块、`relaykit`、前端类型检查和生产构建全部通过。

## 安全与使用边界

- 仅使用你有权访问的上游、账号、模型和接口。
- 不要在 Issue、日志截图、Compose 文件或聊天记录中提交 API Key、Session Secret、数据库 DSN 或完整上游错误正文。
- 公益/免费上游不等于可公开转售；本分支不提供支付、额度销售或多租户隔离能力。
- 对外提供生成式 AI 服务时，使用者自行承担所在地关于授权、内容安全、日志留存和合规运营的义务。

## 文档

- [个人模式部署与回滚](./deploy/personal/README.zh-CN.md)
- [登录 Session 与认证边界](./docs/authentication.md)
- [RelayKit 协议转换模块](./relaykit/README.md)
- [New API 官方文档](https://docs.newapi.pro/)
- [QuantumNous/new-api 上游仓库](https://github.com/QuantumNous/new-api)

仓库中的 `README.en.md`、`README.zh_CN.md`、`README.zh_TW.md`、`README.fr.md` 和 `README.ja.md` 保留为上游通用版本参考，其商业功能说明不代表本自用分支的可用能力。

## 许可证与归属

本项目基于 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 和 [One API](https://github.com/songquanpeng/one-api) 开发，遵循 [GNU Affero General Public License v3.0](./LICENSE)。同时请阅读 [`NOTICE`](./NOTICE) 和 [`THIRD-PARTY-LICENSES.md`](./THIRD-PARTY-LICENSES.md)。

AGPLv3 Section 7 的附加条款仍然适用。修改版必须在适当的法律声明和用户界面归属位置保留：

> Frontend design and development by New API contributors.

带用户界面的修改版还必须保留指向原项目的可见链接：<https://github.com/QuantumNous/new-api>。

如组织政策不允许使用 AGPLv3，或需要其他授权安排，请联系 [support@quantumnous.com](mailto:support@quantumnous.com)。
