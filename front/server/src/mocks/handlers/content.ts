import { http } from 'msw'
import {
  TERMINAL_RETOUCH_STATUSES,
  TERMINAL_TASK_STATUSES,
} from '@/config'
import {
  appendAudit,
  publicTask,
  publicTaskSummary,
} from '@/mocks/db'
import {
  dbAndAdmin,
  idempotentMutation,
  MockApiError,
  readJson,
  requestId,
  respond,
} from '@/mocks/helpers'
import { ErrorCode } from '@/types/api'
import type {
  AssetKind,
  AuditResult,
  TaskStatus,
  WorkspaceMode,
} from '@/types/domain'
import { paginate } from '@/utils/pagination'

function pages(url: URL) {
  return {
    page: Number(url.searchParams.get('page') || 1),
    pageSize: Number(url.searchParams.get('pageSize') || 20),
  }
}

export const contentHandlers = [
  http.get('/api/manage/generation-tasks', ({ request }) =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      const url = new URL(request.url)
      const keyword = url.searchParams.get('keyword')?.trim().toLowerCase()
      const status = url.searchParams.get('status') as TaskStatus | null
      const mode = url.searchParams.get('mode') as WorkspaceMode | null
      const userId = url.searchParams.get('userId')
      const providerId = url.searchParams.get('providerId')
      const modelId = url.searchParams.get('modelId')
      const hasRetouchTicket = url.searchParams.get('hasRetouchTicket')
      const items = db.tasks
        .filter((item) => {
          const user = db.users.find((entry) => entry.id === item.ownerId)
          const hasTicket = db.tickets.some(
            (entry) => entry.taskId === item.id,
          )
          return (
            (!keyword ||
              item.id.toLowerCase().includes(keyword) ||
              item.title.toLowerCase().includes(keyword) ||
              user?.email.toLowerCase().includes(keyword)) &&
            (!status || item.status === status) &&
            (!mode || item.mode === mode) &&
            (!userId || item.ownerId === userId) &&
            (!providerId ||
              item.executionSnapshot.providerId === providerId) &&
            (!modelId || item.executionSnapshot.modelId === modelId) &&
            (hasRetouchTicket === null ||
              (hasRetouchTicket === 'true' ? hasTicket : !hasTicket))
          )
        })
        .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
        .map((item) => publicTaskSummary(item, db))
      return paginate(items, pages(url).page, pages(url).pageSize)
    }),
  ),

  http.get('/api/manage/generation-tasks/:taskId', ({ params }) =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      const task = db.tasks.find((item) => item.id === params.taskId)
      if (!task) {
        throw new MockApiError(404, ErrorCode.NotFound, '生成任务不存在')
      }
      return publicTask(task, db)
    }),
  ),

  http.get('/api/manage/assets', ({ request }) =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      const url = new URL(request.url)
      const keyword = url.searchParams.get('keyword')?.trim().toLowerCase()
      const kind = url.searchParams.get('kind') as AssetKind | null
      const userId = url.searchParams.get('userId')
      const taskId = url.searchParams.get('taskId')
      const ticketId = url.searchParams.get('ticketId')
      const retained = url.searchParams.get('retained')
      const items = db.assets
        .filter(
          (item) =>
            (!keyword ||
              item.name.toLowerCase().includes(keyword) ||
              item.ownerEmail.toLowerCase().includes(keyword)) &&
            (!kind || item.kind === kind) &&
            (!userId || item.ownerId === userId) &&
            (!taskId || item.taskId === taskId) &&
            (!ticketId || item.ticketId === ticketId) &&
            (retained === null ||
              (retained === 'true' ? item.retained : !item.retained)),
        )
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
      return paginate(items, pages(url).page, pages(url).pageSize)
    }),
  ),

  http.get('/api/manage/assets/:assetId', ({ params }) =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      const asset = db.assets.find((item) => item.id === params.assetId)
      if (!asset) {
        throw new MockApiError(404, ErrorCode.NotFound, '图片资产不存在')
      }
      return asset
    }),
  ),

  http.post('/api/manage/assets/:assetId/signed-url', ({ params, request }) =>
    respond(() => {
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, null, () => {
        const asset = db.assets.find((item) => item.id === params.assetId)
        if (!asset || asset.deletedAt || !asset.previewUrl) {
          throw new MockApiError(
            404,
            ErrorCode.NotFound,
            '图片文件不存在或已清理',
          )
        }
        const expiresAt = new Date(Date.now() + 10 * 60_000).toISOString()
        appendAudit(db, admin, {
          action: 'asset.sign_url',
          resourceType: 'asset',
          resourceId: asset.id,
          result: 'success',
          requestId: requestId(request),
        })
        return {
          url: `${asset.previewUrl}?mockSignature=active`,
          expiresAt,
        }
      })
    }),
  ),

  http.post('/api/manage/assets/:assetId/retain', ({ params, request }) =>
    respond(async () => {
      const payload = await readJson<{
        retained: boolean
        reason: string
      }>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        const asset = db.assets.find((item) => item.id === params.assetId)
        if (!asset) {
          throw new MockApiError(404, ErrorCode.NotFound, '图片资产不存在')
        }
        if (!payload.reason?.trim()) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '请填写留存操作原因',
          )
        }
        const previous = asset.retained
        asset.retained = Boolean(payload.retained)
        appendAudit(db, admin, {
          action: asset.retained ? 'asset.retain' : 'asset.release_retention',
          resourceType: 'asset',
          resourceId: asset.id,
          before: { retained: previous },
          after: { retained: asset.retained },
          reason: payload.reason.trim(),
          result: 'success',
          requestId: requestId(request),
        })
        return asset
      })
    }),
  ),

  http.post('/api/manage/assets/:assetId/cleanup', ({ params, request }) =>
    respond(async () => {
      const payload = await readJson<{ reason: string }>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        const asset = db.assets.find((item) => item.id === params.assetId)
        if (!asset) {
          throw new MockApiError(404, ErrorCode.NotFound, '图片资产不存在')
        }
        if (!payload.reason?.trim()) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '请填写提前清理原因',
          )
        }
        if (asset.retained) {
          throw new MockApiError(
            409,
            ErrorCode.InvalidPayload,
            '图片已设置长期保留，请先解除保留',
          )
        }
        const activeTask = db.tasks.some(
          (item) =>
            item.id === asset.taskId &&
            !TERMINAL_TASK_STATUSES.has(item.status),
        )
        const activeTicket = db.tickets.some(
          (item) =>
            item.id === asset.ticketId &&
            !TERMINAL_RETOUCH_STATUSES.has(item.status),
        )
        if (activeTask || activeTicket) {
          throw new MockApiError(
            409,
            ErrorCode.InvalidPayload,
            '图片仍被进行中的任务或工单使用，暂不能清理',
          )
        }
        const hadFile = Boolean(asset.previewUrl)
        asset.deletedAt = new Date().toISOString()
        asset.previewUrl = undefined
        appendAudit(db, admin, {
          action: 'asset.cleanup',
          resourceType: 'asset',
          resourceId: asset.id,
          before: { fileAvailable: hadFile },
          after: { fileAvailable: false, deletedAt: asset.deletedAt },
          reason: payload.reason.trim(),
          result: 'success',
          requestId: requestId(request),
        })
        return asset
      })
    }),
  ),

  http.get('/api/manage/audit-logs', ({ request }) =>
    respond(() => {
      const { db, admin } = dbAndAdmin('platform:manage')
      const url = new URL(request.url)
      const keyword = url.searchParams.get('keyword')?.trim().toLowerCase()
      const operatorId = url.searchParams.get('operatorId')
      const action = url.searchParams.get('action')
      const resourceType = url.searchParams.get('resourceType')
      const result = url.searchParams.get('result') as AuditResult | null
      const startAt = url.searchParams.get('startAt')
      const endAt = url.searchParams.get('endAt')
      const items = db.audits
        .filter(
          (item) =>
            (admin.role === 'platform_admin' ||
              item.operatorId === admin.id) &&
            (!keyword ||
              item.operatorEmail.toLowerCase().includes(keyword) ||
              item.resourceId.toLowerCase().includes(keyword)) &&
            (!operatorId || item.operatorId === operatorId) &&
            (!action || item.action === action) &&
            (!resourceType || item.resourceType === resourceType) &&
            (!result || item.result === result) &&
            (!startAt || item.createdAt >= startAt) &&
            (!endAt || item.createdAt <= endAt),
        )
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
      return paginate(items, pages(url).page, pages(url).pageSize)
    }),
  ),
]
