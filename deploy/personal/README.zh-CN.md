# 个人模式部署与回滚

本目录提供单容器 SQLite 部署模板和显式维护命令，不会自动修改已有部署。

## 初始化

```bash
cd deploy/personal
cp .env.example .env
cp .image.env.example .image.env
chmod 600 .env .image.env
```

填写 `.env` 的会话密钥和可信 HTTPS Origin，并把 `.image.env` 的镜像改为不可变的完整提交标签或 digest。首次启动前可用 `docker compose --env-file .image.env -f compose.yml config --quiet` 检查配置。

## 备份与升级

```bash
./maintenance.sh backup
./maintenance.sh upgrade ghcr.io/buglyz/new-api:sha-<完整40位提交哈希>
```

备份使用 SQLite 在线 `.backup`，随后执行 `PRAGMA quick_check`；同时保存 Compose、应用环境文件和当前镜像。备份目录使用唯一的 UTC 时间前缀名称且权限为 `0700`，其中可能包含会话配置，不应上传或提交。

升级只接受 digest 或完整提交 SHA 标签。健康检查失败时脚本自动恢复旧镜像；数据库不会被自动回滚。

## 回滚

```bash
./maintenance.sh rollback backups/<UTC时间戳>
./maintenance.sh rollback backups/<UTC时间戳> --restore-database --restore-config
```

第一条只回滚镜像。第二条会停止容器并恢复 SQLite 和应用配置，属于有状态回滚；执行前脚本仍会再创建一份当前状态备份。

此模板只处理本目录的 SQLite。使用外部 MySQL、PostgreSQL 或独立日志数据库时，应使用对应数据库的原生一致性备份工具，不能把文件复制当作数据库备份。
