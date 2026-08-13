# 映研三端 Docker 部署

## 部署结构

生产环境构建三个应用镜像，业务代码仍保持三个独立模块：

| 镜像 | 来源 | 职责 |
| --- | --- | --- |
| `yingyan-client` | `front/client` | Client 静态页面、统一 HTTP 入口与反向代理 |
| `yingyan-admin` | `front/server` | 管理端静态页面，仅在 Docker 内网访问 |
| `yingyan-backend` | `backend` | 同一镜像复用为 API、Worker、Migration 和初始化工具 |

运行时另有 PostgreSQL 和 MinIO，它们是基础设施容器，不属于应用镜像。

```text
HTTPS 反向代理
       |
       v
yingyan-client :8080
  |-- /                  -> Client SPA
  |-- /manage/**         -> yingyan-admin
  |-- /manage-assets/**  -> yingyan-admin
  |-- /api/**            -> Go API
  `-- /yingyan-assets/** -> MinIO 私有桶的签名读取

Go API <-> PostgreSQL
Go API <-> MinIO
Worker <-> PostgreSQL / MinIO / AI 服务商
```

浏览器只访问一个 HTTPS 域名。用户 Session、管理 Session、API 和签名图片均为同源请求；Client 和管理端都无法读取 AI 服务商 Key 或 MinIO 内部地址。

登录和其他带浏览器 `Origin` 的 API 请求必须从 `PUBLIC_ORIGIN` 对应的 HTTPS 地址发起。
生产环境会同时启用精确 Origin 校验与 Secure Session Cookie，因此宿主机的
`http://NAS-IP:端口` 只适合检查页面和 `/healthz`，不能作为登录入口。不要为兼容 HTTP
内网地址而关闭 `COOKIE_SECURE`；需要登录时统一访问公开的 HTTPS 域名。

## 首次准备

需要 Docker Engine、Docker Compose v2，以及一个已经完成 HTTPS 的域名。复制部署环境模板：

```bash
cp .env.deploy.example .env.deploy
```

生成彼此独立的随机值：

```bash
openssl rand -hex 24
openssl rand -hex 32
openssl rand -hex 32
openssl rand -hex 32
openssl rand -hex 32
```

填写 `.env.deploy` 时注意：

- `PUBLIC_ORIGIN` 填完整 HTTPS Origin，例如 `https://image.example.com`，末尾不加 `/`。
- `PUBLIC_ORIGIN` 必须与浏览器地址栏中的协议、域名和端口完全一致；Cloudflare Tunnel
  使用 `https://img.example.com` 时就不能填写 NAS IP、容器名或 `http://` 地址。
- 第一项随机值可作为 `POSTGRES_PASSWORD`；它必须与 `DATABASE_URL` 内的密码完全一致。
- 其余随机值分别用于 `MINIO_SECRET_KEY`、`ENCRYPTION_KEY`、`TOKEN_PEPPER`、`REDEMPTION_PEPPER`，不要复用。
- `DATABASE_URL` 示例为 `postgres://yingyan:实际密码@postgres:5432/yingyan?sslmode=disable`。
- `.env.deploy` 已被 Git 忽略，不能放入镜像、提交到仓库或通过聊天传递。
- 生产环境固定拒绝 HTTP AI 服务商和私网 AI 地址。只有本地开发环境才可按需放开。

## HTTPS 入口

默认只有 `127.0.0.1:8088` 被发布。宿主机 Nginx、Caddy、NAS 反向代理或 Cloudflare Tunnel 应将完整请求转发到该地址，并保留原始 Host、Query 和 HTTPS 协议。

宿主机 Nginx 的核心配置示例：

```nginx
location / {
    proxy_pass http://127.0.0.1:8088;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto https;
    client_max_body_size 128m;
    proxy_read_timeout 240s;
}
```

不能单独改写 `/yingyan-assets/` 的 Host、路径或查询参数，否则 MinIO 签名会失效。若反向代理不在同一宿主机，按实际网络设计修改 `APP_BIND_ADDRESS`，不要公开 PostgreSQL 或 MinIO API。

## 构建与启动

先检查环境、构建三个镜像、显式执行迁移，再启动服务：

```bash
make deploy
make deploy-ps
```

`make deploy` 不会写入演示数据，也不会自动创建账号。API 本身也不会在启动时迁移数据库。

首次部署需要显式创建唯一的初始平台管理员：

```bash
make deploy-bootstrap-admin
```

该命令读取 `.env.deploy` 中的 `PLATFORM_ADMIN_EMAIL`、`PLATFORM_ADMIN_PASSWORD` 和
`PLATFORM_ADMIN_NAME`。密码要求 12 到 72 字节。初始化程序使用数据库事务和锁，并且仅在
`admin_accounts` 为空时成功；已有任意管理员后再次执行会拒绝操作。创建成功后，从部署配置中
移除 `PLATFORM_ADMIN_PASSWORD`；后续 Seed 不需要它。

如需创建预设账号和种子数据，在 `.env.deploy` 中临时填写以下变量：

```dotenv
ALLOW_ACCOUNT_SEED=true
PLATFORM_ADMIN_EMAIL=
CLIENT_USER_EMAIL=
CLIENT_USER_PASSWORD=
RETOUCH_ADMIN_EMAIL=
RETOUCH_ADMIN_PASSWORD=
```

三组邮箱必须有效且互不相同；客户用户和修图管理员密码长度为 8 到 72 字节。先确保
`PLATFORM_ADMIN_EMAIL` 对应的管理员已通过 `bootstrap-admin` 创建，然后手动执行：

```bash
make deploy-account-seed
```

`PLATFORM_ADMIN_*` 对应唯一的平台管理员，Seed 只引用它，不会覆盖其密码；
`CLIENT_USER_*` 对应客户前端用户；`RETOUCH_ADMIN_*` 对应管理端修图管理员。该命令还会写入
演示兑换码、演示工作区和占位 AI 服务商，只应在明确需要这些种子数据时运行。同一组邮箱
再次执行会更新客户用户和修图管理员的密码；已有 Seed 数据后不允许更换客户用户邮箱，
避免演示数据串到其他账号。执行成功后将 `ALLOW_ACCOUNT_SEED` 改回 `false`，并从部署配置中
移除 `CLIENT_USER_PASSWORD` 和 `RETOUCH_ADMIN_PASSWORD`。API 和 Worker 不需要任何账号初始化变量。

常用命令：

```bash
make deploy-logs
make deploy-restart
make deploy-down
```

`make deploy-down` 只停止容器，不删除数据库或图片 Volume。不要对该项目执行 `docker compose down -v`，除非已经确认要永久删除全部业务数据。

## 发布与回滚

每次发布使用新的不可变 `IMAGE_TAG`：

1. 修改 `.env.deploy` 中的 `IMAGE_TAG`。
2. 执行 `make deploy-build`。
3. 执行 `make deploy-migrate`。
4. 执行 `make deploy-up` 并检查 `make deploy-ps`、日志和关键页面。

应用回滚时把 `IMAGE_TAG` 改回上一版本并再次执行 `make deploy-up`。数据库 Migration 当前为只向前执行；若新版本包含不兼容的数据迁移，必须使用发布前备份恢复数据库，不能只回滚镜像。

## 数据备份

数据库和图片必须作为同一发布批次备份。PostgreSQL 可在线导出：

```bash
mkdir -p backups
docker compose --env-file .env.deploy -f compose.production.yaml \
  exec -T postgres pg_dump -U yingyan -d yingyan -Fc \
  > backups/yingyan-postgres.dump
```

MinIO 数据位于 `yingyan-production_minio_data`。可使用 MinIO Client 做在线镜像备份，或在维护窗口停止写入后对该 Docker Volume 做快照。恢复演练必须同时验证：

- `schema_migrations` 版本正确。
- 用户余额、预占和流水一致。
- 任务、工单与素材记录能关联。
- MinIO 对象可通过签名 URL 下载。

## 本地查看运行数据

PostgreSQL 不发布宿主机端口，可以直接在 Docker Desktop 打开 `postgres` 容器的 Exec，或执行：

```bash
docker compose --env-file .env.deploy -f compose.production.yaml \
  exec postgres psql -U yingyan -d yingyan
```

MinIO Console 仅监听宿主机 `http://127.0.0.1:9001`，账号来自 `.env.deploy`。该 Console 不应通过公网反向代理。

## 健康与故障定位

- 外部入口：`GET /healthz`
- API：容器内 `GET /health/ready`
- Worker：容器内 `GET :8081/health/ready`
- PostgreSQL：`pg_isready`
- MinIO：`GET /minio/health/live`

排查顺序：

1. `make deploy-ps` 确认依赖健康。
2. `make deploy-logs` 查看 API、Worker 和网关日志。
3. 检查 `.env.deploy` 的 `PUBLIC_ORIGIN` 与浏览器实际域名是否完全一致。
4. 图片 403 时检查反向代理是否保留 Host、桶路径与原查询参数。
5. AI 任务排队不推进时检查 Worker、平台模型绑定和服务商连接测试。
