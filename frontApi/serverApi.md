# 映研平台管理端 API 接口契约

> 文档版本：v0.2  
> 面向项目：`front/server` 映研平台管理端  
> 后端关系：与 `front/client` 共用同一个后端、用户、次数账户和工单数据  
> 管理接口基础路径：`/api/manage`  
> 更新时间：2026-07-30

本文档是平台管理端与 Go 后端联调的依据。用户端接口由
[`clientApi.md`](./clientApi.md) 维护；两个前端必须连接同一个 PostgreSQL
事实源。本文先完整定义人工修图状态机，并在第 10 节汇总兑换码、AI 服务、
用户、任务、资产及审计接口。

## 1. 通用约定

### 1.1 响应协议

成功响应统一为：

```ts
interface ApiSuccessResponse<T> {
  code: 0
  data: T
}
```

错误响应统一为：

```ts
interface ApiErrorResponse {
  code: number
  message: string
  details?: unknown
}
```

处理规则：

- 只有 `code === 0` 表示成功。
- HTTP 2xx 携带非零 `code` 仍按业务错误处理。
- 错误响应不要求 `data` 字段。
- 无业务数据的成功响应返回 `{ "code": 0, "data": null }`。
- 时间使用 ISO 8601 UTC 字符串。
- 次数、页码、数量均使用整数。

### 1.2 认证和管理权限

管理项目使用独立于普通用户的认证 Realm：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| POST | `/api/manage/auth/login` | 管理员邮箱密码登录 |
| GET | `/api/manage/auth/session` | 恢复管理 Session |
| POST | `/api/manage/auth/logout` | 退出管理端 |

认证采用 Cookie Session，Cookie 名固定为 `yy_manage_session`，与用户端
`yy_user_session` 分离。前端 Axios 需要设置：

```ts
withCredentials: true
```

登录和 Session 成功响应均为 `AdminSession`：

```ts
interface AdminSession {
  id: string
  email: string
  name: string
  role: 'platform_admin' | 'retouch_operator'
  permissions: Array<'platform:manage' | 'retouch:manage'>
  status: 'active' | 'disabled'
  csrfToken: string
  createdAt: string
}
```

`csrfToken` 只保存在当前页面内存。除登录外，管理端所有 `POST`、`PUT`、
`PATCH`、`DELETE` 请求必须携带 `X-CSRF-Token`。后端同时校验 Session、
CSRF Token 和 `Origin`；Token 错误返回 HTTP `403`。

除登录接口外，所有管理接口必须校验：

1. Session 有效。
2. 用户账号为 `active`。
3. 工单接口具备 `retouch:manage`。
4. 兑换码、AI 配置、用户、次数、全局任务、资产和审计接口具备
   `platform:manage`。

`platform_admin` 同时拥有两项权限；`retouch_operator` 只能访问工单和权限范围内
的 Dashboard。修图操作员通过工单详情获得履约所需素材投影，
不能访问全局用户、任务或资产接口。

普通用户访问管理接口返回：

```json
{
  "code": 1003,
  "message": "无权执行此管理操作"
}
```

对应 HTTP 状态码为 `403`。后端不能只依赖管理项目前端隐藏按钮，必须对每个
`/api/manage/**` 请求执行服务端权限校验。

### 1.3 通用请求头

| 请求头 | 是否必需 | 用途 |
| --- | --- | --- |
| `Accept: application/json` | 是 | 声明响应格式 |
| `Content-Type: application/json` | JSON 请求是 | JSON 请求体 |
| `X-Request-Id` | 建议 | 请求追踪；后端应写入日志并通过响应头透传 |
| `Idempotency-Key` | 所有管理写接口是 | 防止重复报价、开工、交付、拒单或退款 |
| `X-CSRF-Token` | 管理写接口是 | 取自登录或 Session 响应，只保存在内存 |

### 1.4 幂等规则

以下工单接口必须携带 `Idempotency-Key`：

- `POST /api/manage/retouch-tickets/:ticketId/quote`
- `POST /api/manage/retouch-tickets/:ticketId/start`
- `POST /api/manage/retouch-tickets/:ticketId/deliver`
- `POST /api/manage/retouch-tickets/:ticketId/reject`
- `POST /api/manage/retouch-tickets/:ticketId/fail`

第 10 节列出的批次生成、Reveal、Export、失效、延期、服务商与模型写入、
用户状态与次数调整、资产签名和清理等管理写接口同样必须携带幂等键。

后端以“管理员身份 + 请求路径 + 幂等键”为唯一范围：

- 相同键和相同请求内容重复提交，返回第一次成功结果。
- 相同键但请求内容不同，返回 HTTP `409`、错误码 `4002`。
- 报价不能重复创建，开工不能重复结算，交付不能重复生成文件记录，退款不能重复入账。
- 幂等结果至少保留 24 小时。

对于 `multipart/form-data` 交付接口，后端应在首次处理时记录幂等键和文件摘要；
同一幂等键重试时返回首次交付结果。

### 1.5 分页结构

```ts
interface PageResult<T> {
  items: T[]
  page: number
  pageSize: number
  total: number
  hasMore: boolean
}
```

- `page` 从 `1` 开始。
- 默认 `page=1&pageSize=20`。
- `pageSize` 最大为 `100`。

### 1.6 本地联调模式

- `npm run dev` 使用 `.env`，默认关闭 MSW 并连接真实 Go 后端。
- `npm run dev:mock` 使用 `.env.mock`，显式启用管理端 MSW。
- `npm run dev:backend` 作为兼容命令，仍然关闭 MSW。
- Vite 将 `/api` 代理至 `http://127.0.0.1:8080`。
- 生产构建使用 `.env.production`，默认关闭 MSW。

## 2. 接口总览

| 方法 | 路径 | 用途 | 幂等键 | 成功 data |
| --- | --- | --- | --- | --- |
| GET | `/api/manage/retouch-tickets` | 管理端工单列表 | 否 | `PageResult<ManageRetouchTicketSummary>` |
| GET | `/api/manage/retouch-tickets/:ticketId` | 管理端工单详情 | 否 | `ManageRetouchTicket` |
| POST | `/api/manage/retouch-tickets/:ticketId/quote` | 给出或修改报价 | 是 | `RetouchTicket` |
| POST | `/api/manage/retouch-tickets/:ticketId/start` | 确认开工并结算预占次数 | 是 | `RetouchTicket` |
| POST | `/api/manage/retouch-tickets/:ticketId/deliver` | 上传并交付人工成片 | 是 | `RetouchTicket` |
| POST | `/api/manage/retouch-tickets/:ticketId/reject` | 开工前拒绝接单 | 是 | `RetouchTicket` |
| POST | `/api/manage/retouch-tickets/:ticketId/fail` | 标记履约失败并退款 | 是 | `RetouchTicket` |

## 3. 数据结构

### 3.1 基础结构

```ts
interface User {
  id: string
  email: string
  createdAt: string
  status: 'active' | 'disabled'
}

interface Asset {
  id: string
  name: string
  kind: 'source' | 'reference' | 'retouch-reference'
  role?: 'style' | 'composition' | 'person' | 'detail'
  mimeType: string
  size: number
  previewUrl?: string
  uploadProgress: number
}

interface GenerationResult {
  id: string
  // 用于管理端预览的短期签名地址。
  url: string
  // 带 Content-Disposition: attachment 的短期签名下载地址。
  downloadUrl: string
  width: number
  height: number
}
```

### 3.2 工单状态

```ts
type RetouchTicketStatus =
  | 'submitted'
  | 'quote_pending'
  | 'accepted'
  | 'processing'
  | 'awaiting_confirmation'
  | 'delivered'
  | 'rejected'
  | 'cancelled'
```

| 状态 | 管理端含义 |
| --- | --- |
| `submitted` | 用户已提交，等待评估 |
| `quote_pending` | 已报价，等待用户接受 |
| `accepted` | 用户已接受报价并预占次数，等待管理员开工 |
| `processing` | 首次修图或返修处理中 |
| `awaiting_confirmation` | 已交付人工成片，等待用户确认或提交一次返修 |
| `delivered` | 用户已确认，工单结束 |
| `rejected` | 管理员拒单或平台履约失败 |
| `cancelled` | 用户在开工前取消 |

### 3.3 工单结构

```ts
interface RetouchTicketSummary {
  id: string
  ticketNo: string
  status: RetouchTicketStatus
  updatedAt: string
  quoteCredits?: number
}

interface RetouchTicketTimelineEntry {
  status: RetouchTicketStatus
  note?: string
  createdAt: string
}

interface RetouchQuote {
  id: string
  credits: number
  createdAt: string
}

interface RetouchRevision {
  message: string
  requestedAt: string
}

interface RetouchTicket {
  id: string
  ticketNo: string
  taskId: string
  taskTitle: string
  status: RetouchTicketStatus
  selectedResults: GenerationResult[]
  requirement: string
  supplementalAssets: Asset[]
  quote?: RetouchQuote
  timeline: RetouchTicketTimelineEntry[]
  reservedCredits: number
  spentCredits: number
  refundedCredits: number
  revision?: RetouchRevision
  deliverables: GenerationResult[]
  createdAt: string
  updatedAt: string
}

interface ManageRetouchTicketSummary extends RetouchTicketSummary {
  user: Pick<User, 'id' | 'email' | 'status'>
}

interface ManageSourceTask {
  id: string
  title: string
  mode: 'text-to-image' | 'image-to-image'
  status: GenerationTaskStatus
  modelName: string
  sourceRequirement: string
}

interface ManageRetouchTicket extends RetouchTicket {
  user: Pick<User, 'id' | 'email' | 'status'>
  sourceTaskDetail: ManageSourceTask
}
```

`sourceTaskDetail` 只返回修图履约必需的来源任务投影。平台管理员如需查看素材、
提示词版本、模型快照和次数结算等完整信息，应通过
`GET /api/manage/generation-tasks/:taskId` 查询；修图操作员无权访问该全量接口。

管理详情中的素材和图片 URL 必须是当前管理员有权访问的短期签名地址，不得返回对象
存储内部路径或永久公开地址。

## 4. 管理端查询接口

### 4.1 工单列表

#### `GET /api/manage/retouch-tickets`

用途：按状态、工单号、任务标题或用户邮箱检索工单。

Query 参数：

| 参数 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `status` | `RetouchTicketStatus` | 否 | 按单一状态筛选 |
| `keyword` | string | 否 | 匹配工单号、任务标题或用户邮箱，最长 100 字符 |
| `page` | integer | 否 | 默认 `1` |
| `pageSize` | integer | 否 | 默认 `20`，最大 `100` |

成功：HTTP `200`，`ApiSuccessResponse<PageResult<ManageRetouchTicketSummary>>`。

```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": "retouch_ticket_01J...",
        "ticketNo": "YY20260730-A1B2C3",
        "status": "submitted",
        "updatedAt": "2026-07-30T08:50:00.000Z",
        "user": {
          "id": "user_01J...",
          "email": "customer@example.com",
          "status": "active"
        }
      }
    ],
    "page": 1,
    "pageSize": 20,
    "total": 1,
    "hasMore": false
  }
}
```

默认按 `updatedAt` 倒序。列表查询不能返回其他业务域的敏感用户信息。

### 4.2 工单详情

#### `GET /api/manage/retouch-tickets/:ticketId`

用途：获取工单、用户、来源 AI 任务、原始素材、所选 AI 成片、要求和完整时间线。

路径参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `ticketId` | string | 人工修图工单 ID |

成功：HTTP `200`，`ApiSuccessResponse<ManageRetouchTicket>`。

主要错误：

- `404/7003` 工单不存在。
- `403/1003` 当前登录身份无管理权限。

## 5. 管理端操作接口

### 5.1 给出或修改报价

#### `POST /api/manage/retouch-tickets/:ticketId/quote`

用途：管理员评估需求后给出整数次数报价。

请求头：必须携带 `Idempotency-Key`。

请求体：

```ts
interface QuoteRetouchTicketPayload {
  credits: number // 1 到 999 的整数
  note?: string // 可选，最多 500 个字符
}
```

示例：

```json
{
  "credits": 3,
  "note": "包含人物皮肤、碎发和背景细节处理"
}
```

成功：HTTP `200`，`ApiSuccessResponse<RetouchTicket>`：

- 状态变为 `quote_pending`。
- 生成新的 `{ id, credits, createdAt }` 报价对象。
- `note` 写入本次 `timeline.note`。
- 重新报价必须生成新的 `quote.id`，使旧报价不能再被用户接受。

允许状态：

- `submitted`：首次报价。
- `quote_pending`：用户接受前可以覆盖旧报价。

`accepted` 及之后不能修改报价。

### 5.2 开始处理

#### `POST /api/manage/retouch-tickets/:ticketId/start`

用途：管理员确认开工，并把用户接受报价时预占的次数结算为实际消费。

请求头：必须携带 `Idempotency-Key`。

请求体：无。

成功：HTTP `200`，`ApiSuccessResponse<RetouchTicket>`：

- 状态由 `accepted` 变为 `processing`。
- `spentCredits` 变为 `reservedCredits`。
- `reservedCredits` 作为历史预占数量保留，不清零。
- 只结算既有预占记录，不新增余额变动流水。
- 不再次减少用户可用余额。

主要错误：

- `409/7004` 工单未接受报价、已经开始或已经结束。
- `409/7005` 报价或次数预占记录不存在。

### 5.3 交付人工成片

#### `POST /api/manage/retouch-tickets/:ticketId/deliver`

用途：上传首次人工成片或返修后的人工成片。

请求头：必须携带 `Idempotency-Key`。

内容类型：`multipart/form-data`。

表单字段：

| 字段 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `files` | File[] | 是 | 1 到 4 个 JPG、PNG 或 WebP，每个最大 30MB |
| `note` | string | 否 | 交付说明，最多 500 字符 |

成功：HTTP `200`，`ApiSuccessResponse<RetouchTicket>`：

- 调用前状态必须为 `processing`。
- 上传文件转换为 `GenerationResult[]` 并写入 `deliverables`。
- 状态变为 `awaiting_confirmation`。
- 是否为返修交付由 `revision` 是否存在判断。
- `deliverables[].url` 返回受权限保护的预览和下载地址。

主要错误：

- `409/7004` 当前状态不允许交付。
- `413/6001` 文件过大。
- `422/6001` 文件类型、数量或交付说明无效。

### 5.4 拒绝接单

#### `POST /api/manage/retouch-tickets/:ticketId/reject`

用途：在用户接受报价前，因需求不合规或无法承接而关闭工单。

请求头：必须携带 `Idempotency-Key`。

请求体：

```ts
interface RejectRetouchTicketPayload {
  reason: string // 1 到 500 个字符
}
```

示例：

```json
{
  "reason": "当前需求超出人工修图服务范围"
}
```

成功：HTTP `200`，`ApiSuccessResponse<RetouchTicket>`：

- 仅允许 `submitted` 或 `quote_pending`。
- 状态变为 `rejected`。
- `reason` 写入最新一条 `timeline.note`。
- 因用户尚未接受报价，不发生退款。

### 5.5 标记履约失败

#### `POST /api/manage/retouch-tickets/:ticketId/fail`

用途：用户接受报价后，因平台、文件或履约原因无法继续时关闭工单并全额退款。

请求头：必须携带 `Idempotency-Key`。

请求体：

```ts
interface FailRetouchTicketPayload {
  reason: string // 1 到 500 个字符
}
```

示例：

```json
{
  "reason": "源文件损坏，人工流程无法继续"
}
```

成功：HTTP `200`，`ApiSuccessResponse<RetouchTicket>`：

- 允许状态为 `accepted`、`processing` 或 `awaiting_confirmation`。
- 状态变为 `rejected`。
- `reason` 写入最新一条 `timeline.note`。
- 无论次数处于预占还是已结算状态，都全额返还用户可用余额。
- `refundedCredits` 变为报价次数并写入一条 `refund` 流水。
- 同一工单只允许退款一次。

主要错误：

- `409/7004` 当前状态不允许失败处理。
- `409/4002` 幂等键重复使用但请求内容不一致。

## 6. 状态机和次数事务

### 6.1 状态流转

```text
创建
  submitted
    ├─ 管理员报价 -> quote_pending
    │    ├─ 用户接受 -> accepted
    │    │    ├─ 管理员开工 -> processing
    │    │    │    ├─ 管理员交付 -> awaiting_confirmation
    │    │    │    │    ├─ 用户确认 -> delivered
    │    │    │    │    └─ 用户仅一次返修 -> processing
    │    │    │    │         └─ 管理员再次交付 -> awaiting_confirmation
    │    │    │    │              └─ 用户确认 -> delivered
    │    │    │    └─ 履约失败 -> rejected + 全额退款
    │    │    ├─ 用户取消 -> cancelled + 释放预占
    │    │    └─ 履约失败 -> rejected + 释放预占
    │    ├─ 用户取消 -> cancelled
    │    └─ 管理员拒绝 -> rejected
    ├─ 用户取消 -> cancelled
    └─ 管理员拒绝 -> rejected
```

终态为 `delivered`、`rejected`、`cancelled`。终态不能再转换。

### 6.2 次数账务

| 事件 | 可用余额 | 工单计数字段 | 流水 |
| --- | --- | --- | --- |
| 用户创建工单 | 不变 | 三个字段均为 `0` | 无 |
| 管理员报价 | 不变 | 三个字段均为 `0` | 无 |
| 用户接受报价 | 减少报价次数 | `reservedCredits=报价` | 写 `reserve` |
| 管理员开始处理 | 不变 | `spentCredits=报价` | 不新增余额流水 |
| 开工前用户取消 | 增加全部预占次数 | `refundedCredits=报价` | 写 `release` |
| 开工前履约失败 | 增加全部预占次数 | `refundedCredits=报价` | 写 `release` |
| 开工后履约失败 | 增加全部已结算次数 | `refundedCredits=报价` | 写 `refund` |
| 正常交付和用户确认 | 不变 | 计数字段不变 | 无 |

次数账户、工单计数、工单状态和流水必须在同一个数据库事务内更新：

- 并发接受报价不能透支。
- 并发开工不能重复结算。
- 并发失败操作不能重复退款。
- 开工后用户端不能自行取消；只能由管理端根据履约结果处理。

## 7. 错误码

| HTTP | code | 常量建议 | 场景 |
| --- | ---: | --- | --- |
| 401 | 1001 | `AUTH_REQUIRED` | 未登录或 Session 失效 |
| 403 | 1002 | `ACCOUNT_DISABLED` | 管理员账号停用 |
| 403 | 1003 | `FORBIDDEN` | 缺少接口要求的管理权限 |
| 409 | 2002 | `INSUFFICIENT_CREDITS` | 用户接受报价时次数不足 |
| 429 | 4001 | `RATE_LIMITED` | 请求过于频繁 |
| 409 | 4002 | `DUPLICATE_REQUEST` | 幂等键相同但请求内容不同 |
| 422 | 6001 | `INVALID_PAYLOAD` | 通用参数或文件校验失败 |
| 404 | 6004 | `TASK_NOT_FOUND` | 来源 AI 任务不存在 |
| 404 | 7003 | `RETOUCH_TICKET_NOT_FOUND` | 人工工单不存在 |
| 409 | 7004 | `RETOUCH_INVALID_STATUS` | 当前状态不允许管理操作 |
| 409 | 7005 | `RETOUCH_QUOTE_INVALID` | 报价或预占记录不存在 |
| 500 | 9999 | `UNKNOWN` | 未归类服务异常 |

不存在或无权查看的资源统一返回 `404`，避免通过错误差异泄露资源是否存在。

## 8. 后端实现要求

- 管理操作记录 `operatorId`、`ticketId`、`X-Request-Id`、动作、结果和操作时间。
- 报价、开工、交付、拒单和退款均需要审计日志。
- 文件上传先校验数量、类型和大小，再写对象存储和数据库。
- 交付文件使用受权限保护的短期签名 URL，不能直接暴露存储密钥或内部地址。
- 开工事务同时锁定工单和次数记录，保证只结算一次。
- 履约失败事务同时更新状态、返还次数并写退款流水。
- 管理列表查询建立 `status`、`ticketNo`、`updatedAt` 和用户邮箱相关索引。
- 所有状态变化都追加时间线，禁止覆盖历史节点。

## 9. 联调验收清单

- 无 `retouch:manage` 权限不能访问任何 `/api/manage/retouch-tickets/**` 接口。
- 管理列表分页、状态筛选和关键字搜索结果正确。
- 工单详情包含用户、来源任务、原始素材、所选成片和时间线。
- 相同幂等键重复报价不会创建第二个报价。
- 用户接受报价前可以重新报价，接受后不能改价。
- 开工只允许从 `accepted` 进入，并且只结算一次。
- 交付只允许从 `processing` 进入，文件限制由后端再次校验。
- 首次交付后用户可返修一次，返修后管理端可以再次交付。
- 拒单只允许发生在 `submitted` 或 `quote_pending`。
- 履约失败只退款一次，余额、工单字段和流水保持一致。
- 每次状态变化都能在用户端“人工修图记录”中看到。

## 10. 平台管理接口汇总

本节覆盖 `front/server` 除人工工单外的全部接口。所有接口都要求
`platform:manage`，只有第 10.1 节 Dashboard 会为 `retouch_operator`
返回权限范围内的精简数据。

### 10.1 Dashboard

| 方法与路径 | 参数 | 成功 data | 用途 |
| --- | --- | --- | --- |
| `GET /api/manage/dashboard` | 无 | `DashboardData` | 兑换库存、当前平台模型、AI 健康、失败任务及待处理工单 |

```ts
interface DashboardData {
  metrics: Array<{
    key: string
    label: string
    value: number
    tone: 'neutral' | 'positive' | 'warning' | 'danger'
  }>
  currentModels: {
    chat?: ModelSummary
    image?: ModelSummary
  }
  alerts: DashboardAlert[]
  pendingTickets: ManageRetouchTicketSummary[]
  recentBatches: RedemptionBatch[]
}
```

### 10.2 兑换码和批次

| 方法与路径 | 参数 | 成功 data | 用途 |
| --- | --- | --- | --- |
| `GET /api/manage/redemption-codes` | `page,pageSize,keyword,status,batchId,productCode,redeemedBy,expiringSoon` | `PageResult<RedemptionCode>` | 筛选兑换码 |
| `GET /api/manage/redemption-codes/:codeId` | 路径 ID | `RedemptionCodeDetail` | 查看掩码信息和操作历史 |
| `GET /api/manage/redemption-batches` | `page,pageSize,keyword,productCode` | `PageResult<RedemptionBatch>` | 查询生成批次 |
| `GET /api/manage/redemption-batches/:batchId` | 路径 ID | `RedemptionBatch` | 查看批次统计 |
| `POST /api/manage/redemption-batches` | `CreateRedemptionBatchPayload` | `CreateRedemptionBatchResult` | 原子生成 1–500 个兑换码 |
| `POST /api/manage/redemption-codes/:codeId/reveal` | 无 | `{ id, fullCode }` | 单次查看未使用完整码 |
| `POST /api/manage/redemption-batches/:batchId/reveal` | 无 | `{ id, fullCode }[]` | 查看批次内可展示完整码 |
| `POST /api/manage/redemption-batches/:batchId/export` | 无 | `{ filename, csv }` | 导出批次 CSV |
| `POST /api/manage/redemption-codes/disable` | `DisableRedemptionPayload` | `BulkMutationResult` | 失效未使用兑换码 |
| `POST /api/manage/redemption-codes/extend` | `ExtendRedemptionPayload` | `BulkMutationResult` | 延长未使用或自然过期兑换码 |

```ts
interface CreateRedemptionBatchPayload {
  name: string
  quantity: number
  creditsPerCode: number
  productCode: string
  expiresAt?: string | null
  neverExpires?: boolean
  note?: string
}

interface DisableRedemptionPayload {
  codeIds?: string[]
  batchId?: string
  reason: string
}

interface ExtendRedemptionPayload {
  codeIds?: string[]
  batchId?: string
  expiresAt: string | null
  reason: string
}

interface BulkMutationResult {
  affected: number
  skipped: number
  failed: number
}
```

`CreateRedemptionBatchResult` 返回 `batch` 和仅本次响应可见的
`codes: { id, fullCode, maskedCode }[]`。Reveal、Export 和生成响应必须设置
`Cache-Control: no-store` 并写敏感读取审计。

### 10.3 AI 服务商、模型和平台绑定

| 方法与路径 | 参数 | 成功 data | 用途 |
| --- | --- | --- | --- |
| `GET /api/manage/ai-providers` | 无 | `AIProvider[]` | 服务商列表 |
| `POST /api/manage/ai-providers` | `CreateAIProviderPayload` | `AIProvider` | 新增 OpenAI Compatible 服务商 |
| `PATCH /api/manage/ai-providers/:providerId` | `UpdateAIProviderPayload` | `AIProvider` | 修改名称、Base URL、启用状态和备注 |
| `DELETE /api/manage/ai-providers/:providerId` | 无 | `null` | 删除不再使用且不含模型的服务商配置 |
| `POST /api/manage/ai-providers/:providerId/test` | 无 | `AIProvider` | 测试鉴权和基础连接 |
| `POST /api/manage/ai-providers/:providerId/rotate-key` | `{ apiKey }` | `AIProvider` | 覆盖并轮换密钥 |
| `GET /api/manage/ai-models` | `providerId?` | `AIModel[]` | 查询模型 |
| `POST /api/manage/ai-models` | `CreateAIModelPayload` | `AIModel` | 新增聊天或生图模型 |
| `PATCH /api/manage/ai-models/:modelId` | `UpdateAIModelPayload` | `AIModel` | 修改模型和能力 |
| `DELETE /api/manage/ai-models/:modelId` | 无 | `null` | 删除未绑定的平台模型配置 |
| `POST /api/manage/ai-models/:modelId/test` | 无 | `AIModel` | 测试模型能力 |
| `GET /api/manage/platform-model-bindings` | 无 | `PlatformModelBindings` | 查询当前平台模型 |
| `POST /api/manage/platform-model-bindings` | `{ type, modelId }` | `PlatformModelBindings` | 原子切换或解除平台模型 |

```ts
interface CreateAIProviderPayload {
  name: string
  code: string
  baseUrl: string
  apiKey: string
  enabled?: boolean
  note?: string
}

interface CreateAIModelPayload {
  providerId: string
  displayName: string
  modelId: string
  type: 'chat' | 'image'
  enabled?: boolean
  capabilities: {
    promptOptimization?: boolean
    visionInput?: boolean
    textToImage?: boolean
    imageToImage?: boolean
  }
}

interface AIModel {
  id: string
  providerId: string
  providerName: string
  displayName: string
  modelId: string
  type: 'chat' | 'image'
  enabled: boolean
  connectionStatus: 'untested' | 'healthy' | 'error'
  capabilities: CreateAIModelPayload['capabilities']
  lastTestAt?: string
  lastTest?: {
    testedAt: string
    success: boolean
    message: string
  }
  createdAt: string
  updatedAt: string
  isPlatformModel: boolean
}

interface PlatformModelBindings {
  chatModelId: string | null
  imageModelId: string | null
}
```

所有 `AIProvider` 响应只能返回 `maskedApiKey`。服务商 Base URL、API Key，
或模型 ID、能力声明发生实际变化后，对应测试状态变为 `untested`；修改展示名称、
备注或启停状态不会清除已有测试结果。`enabled`、`connectionStatus` 和
`isPlatformModel` 分别表示“配置是否启用”“最近能力测试是否通过”和“是否承担平台路由”，
三者相互独立。只有启用、测试健康且能力满足的平台模型可以绑定；聊天和生图绑定各自最多一条。

删除模型前必须先解除平台绑定，否则返回 HTTP `409`。删除服务商前必须先删除其下
所有模型，否则返回 HTTP `409`。删除只移除当前路由配置和对应的能力测试记录，历史生成
任务保留创建时保存的服务商、模型名称及配置版本快照，不会被级联删除。

模型能力测试根据已声明能力依次调用上游接口。文生图使用
`POST /v1/images/generations`，图生图使用 multipart
`POST /v1/images/edits`。生图测试最长可能等待 180 秒。上游测试失败仍返回
最新 `AIModel`，通过 `connectionStatus: 'error'` 和 `lastTest.message` 提供经过
脱敏的失败阶段与 HTTP 状态；不得在响应、日志或 cURL 预览中返回真实 API Key。

### 10.4 用户和次数

| 方法与路径 | 参数 | 成功 data | 用途 |
| --- | --- | --- | --- |
| `GET /api/manage/users` | `page,pageSize,keyword,status,minBalance,maxBalance,hasTasks,hasTickets` | `PageResult<ManagedUser>` | 查询用户 |
| `GET /api/manage/users/:userId` | 路径 ID | `ManagedUserDetail` | 查看用户、流水、兑换、任务和工单 |
| `POST /api/manage/users/:userId/status` | `{ status, reason }` | `ManagedUser` | 启用或停用账号 |
| `POST /api/manage/users/:userId/reset-password` | 无 | `ResetPasswordResult` | 生成仅展示一次的临时密码 |
| `POST /api/manage/users/:userId/adjust-credits` | `AdjustCreditsPayload` | `{ user, ledger }` | 原子人工调次 |
| `GET /api/manage/usage-ledger` | `page,pageSize,userId?` | `PageResult<AdjustmentLedger>` | 查询不可修改次数流水 |

```ts
type LedgerType =
  | 'redemption'
  | 'reserve'
  | 'release'
  | 'refund'
  | 'adjustment'

interface AdjustCreditsPayload {
  amount: number
  reason: string
  referenceNo?: string
}

interface ResetPasswordResult {
  temporaryPassword: string
  expiresAt: string
}
```

调次 `amount` 必须为非零整数，调整后余额不能为负。用户余额、流水和审计必须
在同一事务提交。停用用户后立即撤销其用户端 Session。

### 10.5 生成任务

| 方法与路径 | 参数 | 成功 data | 用途 |
| --- | --- | --- | --- |
| `GET /api/manage/generation-tasks` | `page,pageSize,keyword,status,mode,userId,providerId,modelId,hasRetouchTicket` | `PageResult<ManagedGenerationTaskSummary>` | 查询全平台生成任务 |
| `GET /api/manage/generation-tasks/:taskId` | 路径 ID | `ManagedGenerationTask` | 查看需求、提示词、素材、结算、模型快照和输出 |

任务状态统一为
`queued | processing | completed | partial | failed | cancelled`。
`ManagedGenerationTask` 额外返回 `sourceRequirement`、`optimizedPrompt`、
`confirmedPrompt`、`settings`、`assets`、`results`、`executionSnapshot`、
可选 `errorMessage` 和关联工单。管理端不得使用 Client 的
`failed-refunded` 展示别名。

### 10.6 资产

| 方法与路径 | 参数 | 成功 data | 用途 |
| --- | --- | --- | --- |
| `GET /api/manage/assets` | `page,pageSize,keyword,kind,userId,taskId,ticketId,retained` | `PageResult<ManagedAsset>` | 查询图片元数据 |
| `GET /api/manage/assets/:assetId` | 路径 ID | `ManagedAsset` | 查看资产详情 |
| `POST /api/manage/assets/:assetId/signed-url` | 无 | `{ url, expiresAt }` | 获取短期预览或下载地址 |
| `POST /api/manage/assets/:assetId/retain` | `{ retained, reason }` | `ManagedAsset` | 长期保留或解除保留 |
| `POST /api/manage/assets/:assetId/cleanup` | `{ reason }` | `ManagedAsset` | 提前清理可删除对象 |

`ManagedAsset` 只返回对象元数据和可选短期 `previewUrl`，绝不返回 MinIO
Object Key、访问密钥或永久公开地址。进行中任务、进行中工单或长期保留资产
不能清理。

### 10.7 审计

| 方法与路径 | 参数 | 成功 data | 用途 |
| --- | --- | --- | --- |
| `GET /api/manage/audit-logs` | `page,pageSize,keyword,operatorId,action,resourceType,result,startAt,endAt` | `PageResult<AuditEvent>` | 追踪敏感读取和管理操作，仅平台管理员可用 |

`AuditEvent` 包含操作者、角色、动作、资源、脱敏前后快照、原因、结果、
Request ID、IP、设备和时间。完整兑换码、API Key、密码、Session Token 和
签名 URL 必须在写入审计前递归脱敏。

`POST /api/manage/mock/reset` 仅由前端 MSW 演示模式提供，真实 Go API 和生产
路由中不得注册。
