# 映研 Client API 接口契约

> 文档版本：v0.4
> 面向项目：`front/client` 用户端  
> 基础路径：`/api`  
> 更新时间：2026-08-13

本文档是用户端前后端联调和后端实现依据，覆盖当前 Client 已使用的全部接口，以及“AI 生成任务提交人工修图工单”的完整生命周期。人工修图管理项目与 Client 共用同一个后端，其管理接口单独维护在 [`serverApi.md`](./serverApi.md)。

## 1. 通用约定

### 1.1 响应协议

所有接口都返回 JSON。成功响应统一为：

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

示例：

```json
{
  "code": 0,
  "data": {
    "id": "user_01J...",
    "email": "demo@example.com"
  }
}
```

```json
{
  "code": 2002,
  "message": "剩余次数不足",
  "details": {
    "required": 4,
    "balance": 2
  }
}
```

约束：

- 只有 `code === 0` 表示成功。
- HTTP 2xx 携带非零 `code` 仍视为失败，但后端应优先返回与错误语义一致的 HTTP 状态码。
- 错误响应不需要 `data` 字段。
- 删除、退出等无业务数据的成功响应返回 `{ "code": 0, "data": null }`，不使用空响应体。
- 时间字段使用 ISO 8601 UTC 字符串，例如 `2026-07-30T08:30:00.000Z`。
- 次数、金额、数量均使用整数，不使用浮点数。

### 1.2 认证、权限与请求头

用户端采用 Cookie Session：

```http
Cookie: yy_user_session=...
```

Axios 已配置 `withCredentials: true`。后端应设置 `HttpOnly`、`Secure`（HTTPS 环境）和合适的 `SameSite` 属性。

通用请求头：

| 请求头 | 是否必需 | 用途 |
| --- | --- | --- |
| `Accept: application/json` | 是 | 声明响应格式 |
| `Content-Type: application/json` | JSON 请求是 | JSON 请求体 |
| `X-Request-Id` | 建议 | Client 为每次请求生成；后端应透传到响应头和日志 |
| `Idempotency-Key` | 指定写接口是 | 防止兑换、生成、工单提交及次数操作被重复执行 |

除 `POST /api/auth/register`、`POST /api/auth/login` 以及用于检查登录态的 `GET /api/auth/session` 外，本文用户端接口均要求有效用户 Session。`POST /api/auth/logout` 在 Session 已失效时仍按成功处理。

### 1.3 幂等规则

以下接口必须支持 `Idempotency-Key`：

- `POST /api/redemptions/claim`
- `POST /api/notices/ai-processing/ack`
- `POST /api/assets`
- `DELETE /api/assets/:assetId`
- `POST /api/prompts/confirm`
- `POST /api/generations`
- `POST /api/tasks/:taskId/cancel`
- `POST /api/tasks/:taskId/retouch-tickets`
- `POST /api/retouch-tickets/:ticketId/quote/accept`
- `POST /api/retouch-tickets/:ticketId/cancel`
- `POST /api/retouch-tickets/:ticketId/confirm`
- `POST /api/retouch-tickets/:ticketId/revisions`

后端以“身份 + 请求路径 + 幂等键”为唯一范围：

- 相同键和相同请求体重复提交时，返回第一次成功结果，不重复扣次数或创建记录。
- 相同键但请求体不同，返回 HTTP `409`、错误码 `4002`。
- 幂等结果至少保留 24 小时。

### 1.4 管理端接口

管理端权限、分页、报价、开工、交付、拒单及履约失败退款接口统一见
[`serverApi.md`](./serverApi.md)，不在本用户端文档重复定义。

### 1.5 接口总览

| 方法 | 路径 | 用途 | 幂等键 | 成功 data |
| --- | --- | --- | --- | --- |
| POST | `/api/auth/register` | 注册并登录 | 否 | `User` |
| POST | `/api/auth/login` | 登录 | 否 | `User` |
| GET | `/api/auth/session` | 恢复登录态 | 否 | `User` |
| POST | `/api/auth/logout` | 退出 | 否 | `null` |
| GET | `/api/me` | 当前用户资料 | 否 | `User` |
| GET | `/api/entitlements` | 当前次数权益 | 否 | `Entitlement` |
| GET | `/api/usage/ledger` | 次数流水 | 否 | `LedgerEntry[]` |
| GET | `/api/notices/ai-processing` | 第三方 AI 处理告知与当前确认状态 | 否 | `AIProcessingNotice` |
| POST | `/api/notices/ai-processing/ack` | 确认当前告知版本 | 是 | `AIProcessingNotice` |
| POST | `/api/redemptions/preview` | 公开预检兑换码 | 否 | `RedemptionPreview` |
| POST | `/api/redemptions/claim` | 兑换次数 | 是 | `RedemptionResult` |
| POST | `/api/usage/quote` | AI 生成报价 | 否 | `UsageQuote` |
| POST | `/api/assets` | 上传素材 | 是 | `Asset` |
| DELETE | `/api/assets/:assetId` | 删除素材 | 是 | `null` |
| POST | `/api/prompts/optimize` | 优化提示词 | 否 | `PromptVersion` |
| POST | `/api/prompts/reference-prompt` | 分析参考图并生成参考提示词 | 否 | `ReferencePromptResult` |
| POST | `/api/prompts/confirm` | 确认提示词 | 是 | `PromptVersion` |
| POST | `/api/generations` | 创建 AI 任务并返回最新权益 | 是 | `GenerationCreateResult` |
| GET | `/api/tasks?page=1&pageSize=20` | AI 任务分页列表 | 否 | `PageResult<GenerationTask>` |
| GET | `/api/tasks/:taskId` | AI 任务详情 | 否 | `GenerationTask` |
| POST | `/api/tasks/:taskId/cancel` | 取消排队任务 | 是 | `GenerationTask` |
| GET | `/api/retouch-tickets?page=1&pageSize=20` | 用户人工工单分页列表 | 否 | `PageResult<RetouchTicket>` |
| GET | `/api/retouch-tickets/:ticketId` | 用户人工工单详情 | 否 | `RetouchTicket` |
| POST | `/api/tasks/:taskId/retouch-tickets` | 从 AI 任务提交人工工单 | 是 | `RetouchTicket` |
| POST | `/api/retouch-tickets/:ticketId/quote/accept` | 接受报价并预占次数 | 是 | `RetouchTicketBalanceResult` |
| POST | `/api/retouch-tickets/:ticketId/cancel` | 开工前取消人工工单 | 是 | `RetouchTicketBalanceResult` |
| POST | `/api/retouch-tickets/:ticketId/revisions` | 提交唯一一次修改 | 是 | `RetouchTicket` |
| POST | `/api/retouch-tickets/:ticketId/confirm` | 确认人工成片 | 是 | `RetouchTicket` |

### 1.6 本地联调模式

- `npm run dev` 使用 `.env`，默认关闭 MSW 并连接真实 Go 后端。
- `npm run dev:mock` 使用 `.env.mock`，显式启用 MSW 演示数据。
- `npm run dev:backend` 作为兼容命令，仍然关闭 MSW。
- Vite 将 `/api` 代理至 `http://127.0.0.1:8080`，业务代码始终只访问相对路径。
- 生产构建使用 `.env.production`，默认关闭 MSW。

## 2. 公共数据结构

### 2.1 用户与次数

```ts
interface User {
  id: string
  email: string
  createdAt: string
  status: 'active' | 'disabled'
}

interface AuthPayload {
  email: string
  password: string
  remember?: boolean
}

interface RegisterPayload extends AuthPayload {
  acceptedTerms: boolean
}

interface Entitlement {
  // 服务端返回的当前可用次数；前端不得根据报价自行推算余额。
  balance: number
  canCreate: boolean
  status: 'unredeemed' | 'active' | 'empty'
}

interface AIProcessingNotice {
  version: string
  title: string
  providerDisclosure: string
  securitySummary: string
  purpose: string
  processingScope: string[]
  retentionDays: number
  stopUseDescription: string
  acknowledged: boolean
  acknowledgedAt?: string
}

interface LedgerEntry {
  id: string
  type: 'redemption' | 'reserve' | 'release' | 'refund' | 'adjustment'
  amount: number
  balanceAfter: number
  description: string
  createdAt: string
}

interface RedemptionResult {
  added: number
  entitlement: Entitlement
}

interface RedemptionPreview {
  valid: true
  credits: number
  productName: string
  maskedCode: string
  expiresAt: string | null
}

interface UsageQuote {
  action: 'generate'
  cost: number
  balance: number
  canSubmit: boolean
}
```

流水语义：

- `redemption`：兑换码增加次数。
- `reserve`：创建 AI 任务或接受人工报价时预占次数，`amount` 为负数。
- `release`：任务尚未结算前取消或失败，释放预占次数，`amount` 为正数。
- `refund`：已结算输出失败或工单履约失败，退回次数，`amount` 为正数。
- `adjustment`：平台管理员人工调整，可正可负。

人工工单接受报价时，报价次数从 `Entitlement.balance` 原子预占并写入
`reserve` 流水；开始处理只结算现有预占，不再次减少余额，也不再写一条负数流水。

### 2.2 素材、提示词与生成任务

```ts
type WorkspaceMode = 'text-to-image' | 'image-to-image'
type AssetKind = 'source' | 'reference' | 'retouch-reference'
type ReferenceRole = 'style' | 'composition' | 'person' | 'detail'

interface Asset {
  id: string
  name: string
  kind: AssetKind
  role?: ReferenceRole
  mimeType: string
  size: number
  previewUrl?: string
  previewUrlExpiresAt?: string
  uploadProgress: number
}

interface PromptSections {
  subject: string
  scene: string
  style: string
  composition: string
  details: string
  negative: string
  output: string
  referencePrompt?: string
}

interface PromptVersion {
  id: string
  source: string
  sections: PromptSections
  confirmedAt?: string
}

interface GenerationSettings {
  aspectRatio: '1:1' | '3:4' | '4:3' | '9:16' | '16:9'
  outputCount: 1 | 2 | 3 | 4
  // 0 到 100 的整数。
  referenceStrength: number
}

type TaskStatus =
  | 'queued'
  | 'processing'
  | 'completed'
  | 'partial'
  | 'failed'
  | 'cancelled'

interface GenerationResult {
  id: string
  // 用于页面预览的短期签名地址。
  url: string
  // 带 Content-Disposition: attachment 的短期签名地址。
  downloadUrl: string
  width: number
  height: number
}

interface GenerationTask {
  id: string
  mode: WorkspaceMode
  title: string
  status: TaskStatus
  prompt: PromptVersion
  settings: GenerationSettings
  assets: Asset[]
  requestedCount: number
  successfulCount: number
  reservedCredits: number
  spentCredits: number
  refundedCredits: number
  progress: number
  results: GenerationResult[]
  // 无关联工单时不返回此字段。
  retouchTicket?: RetouchTicketSummary
  createdAt: string
  updatedAt: string
}

interface GenerationCreateResult {
  task: GenerationTask
  entitlement: Entitlement
}

interface PageResult<T> {
  items: T[]
  page: number
  pageSize: number
  total: number
  hasMore: boolean
}
```

后端不能信任生成请求中由 Client 回传的素材详情、次数成本、用户 ID 或图片 URL，必须根据 ID 和当前登录用户重新查询。

后端统一返回 `failed`。当前 Client 为兼容既有界面，在 Service 层将其映射为
`failed-refunded`；数据库和其他管理端不得保存或依赖该前端展示值。

### 2.3 人工修图工单

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
  status: 'active' | 'accepted' | 'invalidated' | 'expired'
  expiresAt: string
  remainingSeconds: number
}

interface RetouchSLA {
  stage: 'quote' | 'first-delivery' | 'revision' | 'completed'
  dueAt: string | null
  overdue: boolean
  remainingSeconds: number | null
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
  sla: RetouchSLA
  createdAt: string
  updatedAt: string
}
```

`deliverables[].url` 可以是短期签名下载地址。详情接口每次调用都应返回当前有效地址，不应把对象存储的内部路径暴露给 Client。Client 根据状态和字段推导可用操作：

- `quote_pending` 且存在 `quote`：可接受报价或取消。
- `submitted`、`quote_pending`、`accepted`：可取消。
- `awaiting_confirmation` 且不存在 `revision`：可确认或提交一次返修。
- `awaiting_confirmation` 且已存在 `revision`：只可确认。
- `awaiting_confirmation`、`delivered` 且存在 `deliverables`：可下载。

## 3. 当前 Client 接口

### 3.1 认证

#### `POST /api/auth/register`

用途：注册用户并建立登录 Session。

请求体：

```ts
RegisterPayload
```

校验：

- `email` 必须是合法邮箱，服务端统一转为小写。
- `password` 最少 8 个字符。
- `acceptedTerms` 必须为 `true`。
- `remember` 缺省为 `true`。

成功：HTTP `201`，`ApiSuccessResponse<User>`。

主要错误：`422/6001` 注册信息无效，`409/6001` 邮箱已注册。

#### `POST /api/auth/login`

用途：邮箱密码登录并建立 Session。

请求体：

```ts
AuthPayload
```

成功：HTTP `200`，`ApiSuccessResponse<User>`。

主要错误：`401/1001` 邮箱或密码错误，`403/1002` 账号已停用。

#### `GET /api/auth/session`

用途：页面刷新后恢复当前登录态。

请求参数：无。

成功：HTTP `200`，`ApiSuccessResponse<User>`。

主要错误：`401/1001` 未登录或 Session 失效。

#### `POST /api/auth/logout`

用途：销毁当前 Session。

请求体：无。

成功：HTTP `200`，`ApiSuccessResponse<null>`。重复退出也应成功。

#### `GET /api/me`

用途：获取当前用户资料，供账号抽屉或其他 Client 使用。

请求参数：无。

成功：HTTP `200`，`ApiSuccessResponse<User>`。

主要错误：`401/1001` 未登录。

### 3.2 次数、流水与兑换

#### `GET /api/entitlements`

用途：获取当前可用次数以及是否可生成。

请求参数：无。

成功：HTTP `200`，`ApiSuccessResponse<Entitlement>`。

#### `GET /api/usage/ledger`

用途：获取当前用户次数流水，按 `createdAt` 倒序返回。

请求参数：无。

成功：HTTP `200`，`ApiSuccessResponse<LedgerEntry[]>`。

#### `GET /api/notices/ai-processing`

用途：读取第三方 AI 处理告知和当前用户是否已确认。首次进入创作工作台时调用；告知版本变化时必须重新确认。

成功：`ApiSuccessResponse<AIProcessingNotice>`。

#### `POST /api/notices/ai-processing/ack`

用途：确认指定告知版本。请求头必须携带 `Idempotency-Key`；确认前可以浏览任务记录，但上传素材、提示词优化、参考图分析和生成接口都会再次校验确认状态。

请求体：`{ "version": "ai-processing-v2" }`。

成功：`ApiSuccessResponse<AIProcessingNotice>`；版本过期返回 `409`。

#### `POST /api/redemptions/claim`

用途：兑换咸鱼订单提供的兑换码。

请求头：必须携带 `Idempotency-Key`。

请求体：

```json
{
  "code": "YINGYAN-START-10"
}
```

后端先执行 `trim` 和大写标准化，再原子校验、核销和增加次数。

成功：HTTP `200`，`ApiSuccessResponse<RedemptionResult>`。

主要错误：

- `404/3001` 兑换码无效。
- `409/3002` 兑换码已使用。
- `410/3003` 兑换码已过期。
- `409/3004` 兑换码与当前商品不匹配。

#### `POST /api/redemptions/preview`

用途：在未登录状态下只读预检闲鱼领取链接中的兑换码，用于展示可领取次数、商品名称、掩码和有效期。此接口不核销兑换码、不增加次数，也不保证后续正式领取一定成功。

认证：公开接口，不要求用户 Session。后端按客户端 IP 限制为每分钟最多 20 次。

请求体：

```json
{
  "code": "YY-AKK9-XD9M-9DSM"
}
```

成功：HTTP `200`，`ApiSuccessResponse<RedemptionPreview>`。

```json
{
  "code": 0,
  "data": {
    "valid": true,
    "credits": 10,
    "productName": "AI 生图服务",
    "maskedCode": "YY-AKK9-****-9DSM",
    "expiresAt": "2026-09-30T15:59:59Z"
  }
}
```

主要错误：`404/3001` 无效或已禁用、`409/3002` 已使用、`410/3003` 已过期、`409/3004` 商品不匹配、`429/4001` 请求过于频繁。错误不得返回兑换用户、完整兑换码或内部数据库 ID。

快捷领取页使用 `/claim?code=<完整兑换码>`，首次读取后立即清理 Query；待领取码和重试幂等键只允许存入当前标签页的 `sessionStorage`，成功及明确终态后删除。

#### `POST /api/usage/quote`

用途：在生成前计算预计消耗，并判断当前余额是否足够。此接口只报价，不预占次数。

请求体：

```json
{
  "action": "generate",
  "outputCount": 4
}
```

校验：`outputCount` 是 `1 | 2 | 3 | 4`。

成功：

```json
{
  "code": 0,
  "data": {
    "action": "generate",
    "cost": 4,
    "balance": 8,
    "canSubmit": true
  }
}
```

### 3.3 素材

#### `POST /api/assets`

用途：上传原图或参考图。

内容类型：`multipart/form-data`。

表单字段：

| 字段 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `file` | File | 是 | JPG、PNG 或 WebP，单张最大 15MB |
| `kind` | `source \| reference \| retouch-reference` | 是 | 创作原图、创作参考图或人工修图补充参考图 |
| `role` | `style \| composition \| person \| detail` | 否 | `kind=reference` 时使用 |

成功：HTTP `201`，`ApiSuccessResponse<Asset>`。

请求头必须携带 `Idempotency-Key`。相同用户、路径和 Key 重试同一文件会返回第一次上传结果；同一 Key 对应不同文件或字段会返回 `409` 幂等冲突。

说明：

- 后端保存素材所有者，后续请求必须验证素材属于当前用户。
- 人工工单的 `supplementalAssetIds` 只接受以 `retouch-reference` 上传的素材。
- 上传使用独立的 120 秒超时，不受普通 JSON 请求 20 秒超时限制。
- Client 会把文件缓存在 IndexedDB，因此 Mock 响应可不返回 `previewUrl`。
  正式后端返回短期签名 `previewUrl` 时，Client 优先使用该地址，不会被本地缓存覆盖。
- 一个创作草稿最多使用 8 张素材，后端在生成请求时必须再次校验。

主要错误：`413/6001` 文件过大，`422/6001` 文件类型或字段错误。

#### `DELETE /api/assets/:assetId`

用途：删除当前用户尚未被任务锁定的素材。

路径参数：`assetId`，素材 ID。

成功：HTTP `200`，`ApiSuccessResponse<null>`。

请求头必须携带 `Idempotency-Key`，重复删除只返回第一次成功结果。

主要错误：`404/6002` 素材不存在，`409/6001` 素材已被任务使用。

#### `GET /api/assets/:assetId/url`

用途：刷新当前用户素材的短期预览或下载地址。

Query 参数 `purpose` 可选值为 `preview` 或 `download`，默认是 `preview`。
成功：`ApiSuccessResponse<{ url: string; expiresAt: string }>`。签名地址不会写入数据库，
`expiresAt` 为 UTC 时间；地址即将过期或加载失败时 Client 可重新请求一次。

### 3.4 提示词

#### `POST /api/prompts/optimize`

用途：免费优化用户的原始需求，返回可编辑的分区提示词。

请求体：

```json
{
  "source": "将人物修成自然通透的杂志人像，保留皮肤质感",
  "mode": "image-to-image",
  "sourceAssetIds": ["asset_source_01J..."],
  "referenceAssets": [
    {
      "assetId": "asset_reference_01J...",
      "role": "style"
    }
  ],
  "referencePrompt": "参考图的清透杂志氛围，50mm 人像镜头，柔和侧逆光，真实材质细节"
}
```

校验：

- `source` 去除首尾空白后长度为 6 到 600 个字符。
- `mode` 为 `text-to-image` 或 `image-to-image`。
- `sourceAssetIds` 和 `referenceAssets[].assetId` 必须属于当前用户。
- `referenceAssets[].role` 只能为 `style`、`composition`、`person` 或 `detail`。
- 图生图必须至少提供一张 `sourceAssetIds`；参考图不能单独作为图生图原图。
- 第二阶段合并时，`referencePrompt` 是由参考图分析接口返回的文字描述；后端只把原图发送给对话模型和生图模型，参考图不会再次作为生图图片输入。
- 后端根据素材 ID 读取私有图片并形成模型输入，不能信任 Client 回传的 URL。

成功：HTTP `200`，`ApiSuccessResponse<PromptVersion>`，首次优化不返回 `confirmedAt`。

#### `POST /api/prompts/confirm`

用途：保存用户最终确认的提示词版本。

请求体：

```ts
{
  id: string
  source: string
  sections: PromptSections
}
```

成功：HTTP `200`，`ApiSuccessResponse<PromptVersion>`，其中 `confirmedAt` 必须存在。

主要错误：`404/6003` 提示词版本不存在，`422/6001` 内容不合法。

#### `POST /api/prompts/reference-prompt`

用途：读取用户上传的参考图，调用对话模型生成一段 Midjourney 风格的参考提示词。该接口只返回文字，不创建生成任务，也不消耗用户次数。

请求体：

```json
{
  "referenceAssets": [
    { "assetId": "asset_reference_01J...", "role": "style" }
  ]
}
```

返回：

```ts
interface ReferencePromptResult {
  prompt: string
  referenceAssets: PromptReferenceAsset[]
}
```

提示词至少覆盖主体、场景、风格、镜头与光影、细节修饰。Client 展示并允许用户修改后，再把文字传入 `/api/prompts/optimize` 做最终合并。

### 3.5 AI 生成

#### `POST /api/generations`

用途：最终创建 AI 生成任务，并按输出张数原子预占次数。

请求头：必须携带 `Idempotency-Key`。

请求体：

```ts
interface CreateGenerationPayload {
  promptVersionId: string
  assetIds: string[]
  settings: GenerationSettings
}
```

示例：

```json
{
  "promptVersionId": "prompt_01J...",
  "assetIds": ["asset_source_01J...", "asset_reference_01J..."],
  "settings": {
    "aspectRatio": "3:4",
    "outputCount": 2,
    "referenceStrength": 68
  }
}
```

成功：HTTP `202`，`ApiSuccessResponse<GenerationTask>`，初始状态为 `queued`。

业务规则：

- `promptVersionId` 必须属于当前用户且已经确认；模式和标题由该版本派生。
- `assetIds` 必须属于当前用户；图生图至少关联一张 `source` 素材。
- 输出数量为 1 到 4，每张消耗 1 次。
- 提交时再次校验余额并在同一事务内扣减可用余额、创建任务和写入 `reserve` 流水。
- 全部失败全额退款，部分失败按失败张数退款。
- 请求中出现 `prompt`、`assets`、`mode`、`title`、`mockScenario` 等非契约字段时，
  正式后端返回 `422/6001`，不得静默信任。

主要错误：`409/2002` 次数不足，`422/6001` 提示词、素材或设置无效。

### 3.6 AI 任务

#### `GET /api/tasks`

用途：获取当前用户 AI 生成任务分页，按 `createdAt` 倒序返回。

Query 参数：`page` 默认 `1`，`pageSize` 默认 `20`，最大 `100`。

成功：HTTP `200`，`ApiSuccessResponse<PageResult<GenerationTask>>`。

每条任务按关联情况返回可选字段 `retouchTicket`：

- 未提交过人工工单时不返回此字段，任务详情显示“申请人工精修”。
- 最新工单仍在有效状态时返回 `RetouchTicketSummary`，按钮显示“查看人工修图记录”并跳转到该工单。
- 最新工单为 `cancelled` 或 `rejected` 时仍返回其摘要，任务详情同时提供“重新申请人工精修”和“查看历史工单”。

#### `GET /api/tasks/:taskId`

用途：获取当前用户指定 AI 任务的完整详情和最新生成状态。

路径参数：`taskId`，AI 任务 ID。

成功：HTTP `200`，`ApiSuccessResponse<GenerationTask>`。

主要错误：`404/6004` 任务不存在或不属于当前用户。

#### `POST /api/tasks/:taskId/cancel`

用途：取消仍在排队的 AI 任务并全额退回预占次数。

路径参数：`taskId`，AI 任务 ID。

请求体：无。

成功：HTTP `200`，`ApiSuccessResponse<GenerationTask>`。

主要错误：`404/6004` 任务不存在，`409/6001` 任务已开始处理或已结束。

## 4. 用户端人工修图接口

### 4.1 工单列表

#### `GET /api/retouch-tickets`

用途：加载“人工修图记录”Tab，展示当前用户的工单和状态。

Query 参数：`page` 默认 `1`，`pageSize` 默认 `20`，最大 `100`。

成功：HTTP `200`，`ApiSuccessResponse<PageResult<RetouchTicket>>`，按 `updatedAt` 倒序返回。Client 只轮询当前页。

示例：

```json
{
  "code": 0,
  "data": [
    {
      "id": "retouch_ticket_01J...",
      "ticketNo": "YY20260730-A1B2C3",
      "taskId": "task_01J...",
      "taskTitle": "自然光人像精修",
      "status": "quote_pending",
      "selectedResults": [
        {
          "id": "result_01J...",
          "url": "/media/result-01.webp",
          "width": 1200,
          "height": 1800
        }
      ],
      "requirement": "减轻面部过度磨皮并整理碎发",
      "supplementalAssets": [],
      "quote": {
        "id": "quote_01J...",
        "credits": 3,
        "createdAt": "2026-07-30T09:00:00.000Z"
      },
      "timeline": [
        {
          "status": "submitted",
          "note": "人工修图需求已提交",
          "createdAt": "2026-07-30T08:50:00.000Z"
        },
        {
          "status": "quote_pending",
          "note": "人工修图报价 3 次",
          "createdAt": "2026-07-30T09:00:00.000Z"
        }
      ],
      "reservedCredits": 0,
      "spentCredits": 0,
      "refundedCredits": 0,
      "deliverables": [],
      "createdAt": "2026-07-30T08:50:00.000Z",
      "updatedAt": "2026-07-30T09:00:00.000Z"
    }
  ]
}
```

### 4.2 工单详情

#### `GET /api/retouch-tickets/:ticketId`

用途：获取工单要求、报价、结算、交付文件和时间线。

路径参数：`ticketId`，工单 ID。

成功：HTTP `200`，`ApiSuccessResponse<RetouchTicket>`。

主要错误：`404/7003` 工单不存在或不属于当前用户。

### 4.3 从 AI 任务提交人工工单

#### `POST /api/tasks/:taskId/retouch-tickets`

用途：用户对 AI 成片不满意时，从一条有成功结果的生成任务创建人工修图工单。

请求头：必须携带 `Idempotency-Key`。

路径参数：`taskId`，来源 AI 任务 ID。

请求体：

```ts
interface CreateRetouchTicketPayload {
  // 必须属于路径 taskId 的成功生成结果。
  selectedResultIds: string[] // 1 到 4 个，不能重复
  requirement: string // 去除首尾空白后 1 到 1000 个字符
  supplementalAssetIds: string[] // 0 到 4 个，不能重复
}
```

示例：

```json
{
  "selectedResultIds": ["result_01J...", "result_01K..."],
  "requirement": "保留第一张的人物状态，减轻面部过度磨皮，整理右侧碎发，并自然弱化背景路人。",
  "supplementalAssetIds": ["asset_01J..."]
}
```

成功：HTTP `201`，`ApiSuccessResponse<RetouchTicket>`，初始状态为 `submitted`，并生成全局唯一、便于人工沟通的 `ticketNo`。

资格校验：

- 来源任务必须属于当前用户，状态为 `completed` 或 `partial`。
- 必须选择 1 到 4 个属于该任务 `results` 的结果 ID。
- 补充素材必须属于当前用户，且上传时 `kind` 为 `retouch-reference`。
- 同一 AI 任务同时只能有一个非 `rejected`、非 `cancelled` 工单；被拒绝或取消后允许重新提交，新工单成为任务上的最新关联。
- 创建工单不扣次数。

主要错误：

- `404/7001` 来源任务不存在或不可提交。
- `409/7002` 该任务已有人工工单；`details` 返回现有 `ticketId`。
- `422/7001` 来源任务没有可提交的成功结果，或结果 ID 不属于该任务。
- `422/6001` 请求字段或补充素材不合法。

成功后，`GET /api/tasks` 和 `GET /api/tasks/:taskId` 中的 `retouchTicket` 必须立即可见。

### 4.4 接受报价

#### `POST /api/retouch-tickets/:ticketId/quote/accept`

用途：用户确认管理员报价，并原子预占相应次数。

请求头：必须携带 `Idempotency-Key`。

请求体：

```json
{
  "quoteId": "quote_01J..."
}
```

`quoteId` 必须等于详情中当前 `quote.id`，防止用户接受已经被管理员替换的旧报价。

成功：

```ts
interface RetouchTicketBalanceResult {
  ticket: RetouchTicket
  entitlement: Entitlement
}
```

HTTP `200`，状态由 `quote_pending` 变为 `accepted`，`reservedCredits` 变为报价次数。

事务要求：

1. 锁定用户次数账户和工单。
2. 再次确认工单仍为 `quote_pending`。
3. 校验余额不小于报价。
4. 从可用 `balance` 扣除报价次数并记录工单预占。
5. 更新工单状态后提交事务。

接受成功时写一条 `reserve` 流水。管理员开始处理只结算现有预占，
不再次扣减余额或写第二条负数流水。

主要错误：`404/7003` 工单不存在，`409/7004` 状态不允许，`409/7005` 报价不存在或已失效，`409/2002` 次数不足。

### 4.5 取消工单

#### `POST /api/retouch-tickets/:ticketId/cancel`

用途：用户在人工处理开始前取消工单。

请求头：必须携带 `Idempotency-Key`。

请求体：无。

成功：

```ts
interface RetouchTicketBalanceResult {
  ticket: RetouchTicket
  entitlement: Entitlement
}
```

HTTP `200`，状态变为 `cancelled`：

- `submitted` 或 `quote_pending` 取消时没有次数变化。
- `accepted` 取消时全额释放预占次数，`refundedCredits` 变为预占次数，并写入 `release` 流水。
- `processing` 及之后不允许用户取消。

主要错误：`404/7003` 工单不存在，`409/7004` 当前状态不可取消。

### 4.6 提交一次修改意见

#### `POST /api/retouch-tickets/:ticketId/revisions`

用途：用户查看人工交付后不满意时，提交唯一一次免费修改要求。

请求头：必须携带 `Idempotency-Key`。

请求体：

```ts
interface CreateRetouchRevisionPayload {
  message: string // 去除首尾空白后 1 到 500 个字符
}
```

成功：HTTP `200`，`ApiSuccessResponse<RetouchTicket>`：

- 调用前状态必须为 `awaiting_confirmation`。
- `revision` 从缺省变为 `{ message, requestedAt }`。
- 工单状态回到 `processing`。
- `deliverables` 暂时清空，时间线保留首次交付和返修记录；管理员再次交付后返回新的 `deliverables`。
- 本次修改不重新报价、不额外扣次数。

主要错误：`404/7003` 工单不存在，`409/7004` 当前状态不允许，`409/7006` 已使用修改机会，`422/6001` 修改要求无效。

### 4.7 确认人工成片

#### `POST /api/retouch-tickets/:ticketId/confirm`

用途：用户确认人工成片并结束工单。

请求头：必须携带 `Idempotency-Key`。

请求体：无。

成功：HTTP `200`，`ApiSuccessResponse<RetouchTicket>`，状态由 `awaiting_confirmation` 变为 `delivered`。

说明：

- 首次交付且尚未使用修改机会时，用户可选择确认或提交一次修改。
- 第二次交付只能确认，不再允许提交修改。
- `deliverables` 中的 URL 在 `awaiting_confirmation` 和 `delivered` 状态均可下载。

主要错误：`404/7003` 工单不存在，`409/7004` 当前状态不允许。

## 5. 管理端接口说明

人工修图管理项目与 Client 使用同一个后端，但 `/api/manage/**` 接口不属于用户端
调用范围。管理权限、分页、报价、开工、交付、拒单及履约失败退款的完整合同见
[`serverApi.md`](./serverApi.md)。

## 6. 人工工单状态机与次数规则

### 6.1 状态流转

| 状态 | 用户端文案 | 含义 |
| --- | --- | --- |
| `submitted` | 已提交 | 用户已提交需求，等待人工评估 |
| `quote_pending` | 待确认报价 | 管理员已报价，等待用户确认 |
| `accepted` | 已接受报价 | 用户已确认并预占次数，等待开工 |
| `processing` | 处理中 | 修图师正在处理首次成片或返修 |
| `awaiting_confirmation` | 待确认 | 人工成片已交付，等待用户确认或申请一次返修 |
| `delivered` | 已交付 | 用户已确认，工单结束 |
| `rejected` | 已拒绝 | 管理员拒单或平台履约失败，工单结束 |
| `cancelled` | 已取消 | 用户在开工前取消，工单结束 |

```text
创建
  submitted
    ├─ 管理员报价 -> quote_pending
    │    ├─ 用户接受 -> accepted
    │    │    ├─ 管理员开工 -> processing
    │    │    │    ├─ 管理员交付 -> awaiting_confirmation
    │    │    │    │    ├─ 用户确认 -> delivered
    │    │    │    │    └─ 用户仅一次修改 -> processing
    │    │    │    │         └─ 管理员再次交付 -> awaiting_confirmation
    │    │    │    │              └─ 用户确认 -> delivered
    │    │    │    └─ 平台失败 -> rejected + 全额退款
    │    │    ├─ 用户取消 -> cancelled + 释放预占
    │    │    └─ 平台失败 -> rejected + 释放预占
    │    ├─ 用户取消 -> cancelled
    │    └─ 管理员拒绝 -> rejected
    ├─ 用户取消 -> cancelled
    └─ 管理员拒绝 -> rejected
```

终态为 `delivered`、`rejected`、`cancelled`。终态不能再转换。

### 6.2 次数账务

| 事件 | 可用余额 | 工单计数字段 | 流水 |
| --- | --- | --- | --- |
| 创建工单 | 不变 | 三个字段均为 `0` | 无 |
| 管理员报价 | 不变 | 三个字段均为 `0` | 无 |
| 用户接受报价 | 减少报价次数 | `reservedCredits=报价` | 写 `reserve` |
| 管理员开始处理 | 不变 | `spentCredits=报价` | 不新增余额流水 |
| 开工前用户取消 | 增加全部预占次数 | `refundedCredits=报价` | 写 `release` |
| 开工前平台失败 | 增加全部预占次数 | `refundedCredits=报价` | 写 `release` |
| 开工后履约失败 | 增加全部已结算次数 | `refundedCredits=报价` | 写 `refund` |
| 正常交付与确认 | 不变 | 计数字段不变 | 无 |

次数账户、工单计数、状态和流水必须在同一个数据库事务内更新。并发接受报价时不能透支；并发退款时不能重复入账。

## 7. HTTP 与业务错误码

| HTTP | code | 常量建议 | 场景 |
| --- | ---: | --- | --- |
| 401 | 1001 | `AUTH_REQUIRED` | 未登录、Session 失效、登录凭据错误 |
| 403 | 1002 | `ACCOUNT_DISABLED` | 用户账号停用 |
| 403 | 1003 | `FORBIDDEN` | 无管理端权限或无权访问资源 |
| 409 | 2001 | `REDEMPTION_REQUIRED` | 未兑换，不能生成 |
| 409 | 2002 | `INSUFFICIENT_CREDITS` | 次数不足 |
| 404 | 3001 | `CODE_INVALID` | 兑换码无效 |
| 409 | 3002 | `CODE_USED` | 兑换码已使用 |
| 410 | 3003 | `CODE_EXPIRED` | 兑换码已过期 |
| 409 | 3004 | `PRODUCT_MISMATCH` | 兑换码商品不匹配 |
| 429 | 4001 | `RATE_LIMITED` | 请求过于频繁 |
| 409 | 4002 | `DUPLICATE_REQUEST` | 幂等键复用但请求内容不同 |
| 500 | 5001 | `TASK_FAILED_REFUNDED` | AI 任务失败且已退款 |
| 422 | 6001 | `INVALID_PAYLOAD` | 通用参数或文件校验失败 |
| 404 | 6002 | `ASSET_NOT_FOUND` | 素材不存在 |
| 404 | 6003 | `PROMPT_NOT_FOUND` | 提示词版本不存在 |
| 404 | 6004 | `TASK_NOT_FOUND` | AI 任务不存在 |
| 404/422 | 7001 | `RETOUCH_TASK_NOT_ELIGIBLE` | AI 任务不存在、未结算或没有合格成片 |
| 409 | 7002 | `RETOUCH_TICKET_ALREADY_EXISTS` | AI 任务已有关联工单 |
| 404 | 7003 | `RETOUCH_TICKET_NOT_FOUND` | 人工工单不存在 |
| 409 | 7004 | `RETOUCH_INVALID_STATUS` | 当前状态不允许该操作 |
| 409 | 7005 | `RETOUCH_QUOTE_INVALID` | 报价不存在、已失效或预占记录不存在 |
| 409 | 7006 | `RETOUCH_REVISION_LIMIT_REACHED` | 唯一一次修改机会已使用 |
| 500 | 9999 | `UNKNOWN` | 未归类服务异常 |

不存在或不属于当前用户的资源统一返回 `404`，避免通过错误差异泄露他人资源是否存在。

## 8. 联调验收清单

- 注册、登录、Session 恢复、退出和账号停用响应符合统一协议。
- 兑换和 AI 生成相同幂等键不会重复加减次数。
- AI 任务全部失败、部分失败和排队取消的退款金额正确。
- 只有 `completed`、`partial` 且至少一个成功结果的任务可提交人工工单。
- 工单只能选择来源任务内的 1 到 4 个 AI 结果，并且每个任务同时只能有一个未取消、未拒绝的工单。
- 工单创建后，任务列表从缺少 `retouchTicket` 变为返回关联摘要。
- 接受报价的余额校验、次数预占和状态变化是同一原子事务。
- `accepted` 状态用户取消会释放预占；`processing` 后用户不能取消。
- 管理员开工只结算一次，平台失败只退款一次。
- 首次交付允许提交一次修改；第二次交付只能确认。
- 用户只能读取自己的任务、素材、工单和下载文件。
- 管理端所有写操作有权限校验、操作审计和幂等保护。
