# 映研后端

Go、Gin、GORM、PostgreSQL 与 MinIO 组成的模块化单体。API 和 Worker 是两个独立进程，
数据库是用户端与管理端唯一的业务事实源。

## 本地启动

```bash
cp .env.example .env
docker compose up -d postgres minio minio-init
docker compose --profile tools run --rm migrate
docker compose --profile tools run --rm seed
docker compose up -d api worker
```

健康检查：

```bash
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

三端共用的 OpenAPI 3.1 契约位于
[`openapi/openapi.yaml`](openapi/openapi.yaml)。它覆盖当前 Router 注册的全部接口，
可用 `make openapi-check` 做基础结构校验；更详细的中文业务说明继续见
`../frontApi/clientApi.md` 和 `../frontApi/serverApi.md`。

迁移和演示数据初始化均为显式命令。API 启动不会自动改表，生产环境也不会注册演示数据
重置接口。`cmd/seed` 在 `APP_ENV=production` 时默认拒绝执行。

## 初始化账号

账号初始化不再内置账号或密码。唯一的平台管理员通过 `bootstrap-admin` 创建：

| 角色 | 邮箱变量 | 密码变量 |
| --- | --- | --- |
| 平台管理员 | `PLATFORM_ADMIN_EMAIL` | `PLATFORM_ADMIN_PASSWORD` |

`PLATFORM_ADMIN_NAME` 可用于设置平台管理员名称。创建成功后即可从部署配置中移除
`PLATFORM_ADMIN_PASSWORD`；`seed` 不读取或修改平台管理员密码。

运行 `seed` 时提供三组互不相同的邮箱，其中平台管理员只需提供邮箱：

| 角色 | 邮箱变量 | 密码变量 |
| --- | --- | --- |
| 平台管理员 | `PLATFORM_ADMIN_EMAIL` | 不需要 |
| 客户前端用户 | `CLIENT_USER_EMAIL` | `CLIENT_USER_PASSWORD` |
| 修图管理员 | `RETOUCH_ADMIN_EMAIL` | `RETOUCH_ADMIN_PASSWORD` |

平台管理员应先通过 `bootstrap-admin` 创建；`seed` 只验证并引用该账号。其余两项密码必须为
8 到 72 字节；邮箱必须有效且三者不能相同。生产环境执行 `seed`
还需显式设置 `ALLOW_ACCOUNT_SEED=true`。这些变量只应注入一次性工具容器，不需要配置给 API 或 Worker。

未使用兑换码：`YINGYAN-START-10`、`YINGYAN-PRO-30`。

## 常用命令

```bash
go mod tidy
go test ./...
go vet ./...
make openapi-check
go run ./cmd/api
go run ./cmd/worker
```

正式环境必须替换 `.env.example` 中的数据库密码、MinIO 密钥、加密主密钥和 Pepper。
真实 AI 服务商的 Base URL 与 API Key 通过管理端写入数据库，不写入环境示例或源码。

## 前端联调与 AI 配置

用户端和管理端的 `npm run dev` 默认都连接本机 `http://127.0.0.1:8080` 的真实
Go 后端。只有显式执行 `npm run dev:mock` 才会启用浏览器内的 MSW 演示数据；
Mock 中配置的服务商、模型、用户和任务不会写入 PostgreSQL。

真实提示词优化链路需要在管理端依次完成：

1. 新建 AI 服务商并保存真实 Base URL 和 API Key。
2. 对服务商执行连接测试。
3. 新建 `chat` 模型，开启提示词优化能力并执行模型测试。
4. 将已通过测试的聊天模型设为平台对话模型。
5. 在 Client 填写需求并点击“优化提示词”。

聊天模型勾选“图片理解”时，模型测试会附带一张最小 PNG，同时验证文字对话与
图片输入；未通过该测试的模型不能设为平台模型。

API 日志会依次输出 `prompt_optimization_provider_call_started` 与
`prompt_optimization_provider_call_succeeded`；失败时输出
`prompt_optimization_provider_call_failed`。日志只记录服务商编码、模型 ID、
耗时、上游请求 ID 和 Token 用量，不记录 API Key、图片或完整提示词。

如果 macOS 代理软件把公网域名解析到 `198.18.0.0/15` Fake-IP 网段，本地
Docker 容器的 SSRF 防护会返回 `unsafe_url`。仅在这种本机开发场景下，可以在
被 Git 忽略的 `backend/.env` 中设置：

```dotenv
PROVIDER_ALLOW_PRIVATE_NETWORK=true
```

生产环境必须保持为 `false`，并使用能解析服务商真实公网地址的 DNS。
