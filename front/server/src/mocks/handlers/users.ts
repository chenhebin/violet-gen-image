import { http } from 'msw'
import { APP_CONFIG } from '@/config'
import {
  appendAudit,
  publicUser,
  publicUserDetail,
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
  AdjustCreditsPayload,
  UserStatus,
} from '@/types/domain'
import { createId } from '@/utils/id'
import { paginate } from '@/utils/pagination'

function pageValues(url: URL): { page: number; pageSize: number } {
  return {
    page: Number(url.searchParams.get('page') || 1),
    pageSize: Number(
      url.searchParams.get('pageSize') || APP_CONFIG.defaultPageSize,
    ),
  }
}

export const userHandlers = [
  http.get('/api/manage/users', ({ request }) =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      const url = new URL(request.url)
      const keyword = url.searchParams.get('keyword')?.trim().toLowerCase()
      const status = url.searchParams.get('status') as UserStatus | null
      const minBalance = url.searchParams.has('minBalance')
        ? Number(url.searchParams.get('minBalance'))
        : undefined
      const maxBalance = url.searchParams.has('maxBalance')
        ? Number(url.searchParams.get('maxBalance'))
        : undefined
      const hasTasks = url.searchParams.get('hasTasks')
      const hasTickets = url.searchParams.get('hasTickets')
      const items = db.users
        .filter(
          (item) =>
            (!keyword ||
              item.id.toLowerCase().includes(keyword) ||
              item.email.toLowerCase().includes(keyword)) &&
            (!status || item.status === status) &&
            (minBalance === undefined || item.balance >= minBalance) &&
            (maxBalance === undefined || item.balance <= maxBalance) &&
            (hasTasks === null ||
              (hasTasks === 'true' ? item.taskCount > 0 : item.taskCount === 0)) &&
            (hasTickets === null ||
              (hasTickets === 'true'
                ? item.ticketCount > 0
                : item.ticketCount === 0)),
        )
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
        .map(publicUser)
      const { page, pageSize } = pageValues(url)
      return paginate(items, page, pageSize)
    }),
  ),

  http.get('/api/manage/users/:userId', ({ params }) =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      const user = db.users.find((item) => item.id === params.userId)
      if (!user) {
        throw new MockApiError(404, ErrorCode.NotFound, '用户不存在')
      }
      return publicUserDetail(user, db)
    }),
  ),

  http.post('/api/manage/users/:userId/status', ({ params, request }) =>
    respond(async () => {
      const payload = await readJson<{
        status: UserStatus
        reason: string
      }>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        const user = db.users.find((item) => item.id === params.userId)
        if (!user) {
          throw new MockApiError(404, ErrorCode.NotFound, '用户不存在')
        }
        if (
          !['active', 'disabled'].includes(payload.status) ||
          !payload.reason?.trim()
        ) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '请选择账号状态并填写操作原因',
          )
        }
        const previous = user.status
        user.status = payload.status
        user.disabledReason =
          payload.status === 'disabled' ? payload.reason.trim() : undefined
        appendAudit(db, admin, {
          action:
            payload.status === 'disabled' ? 'user.disable' : 'user.enable',
          resourceType: 'user',
          resourceId: user.id,
          before: { status: previous },
          after: { status: user.status },
          reason: payload.reason.trim(),
          result: 'success',
          requestId: requestId(request),
        })
        return publicUser(user)
      })
    }),
  ),

  http.post('/api/manage/users/:userId/reset-password', ({ params, request }) =>
    respond(() => {
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, null, () => {
        const user = db.users.find((item) => item.id === params.userId)
        if (!user) {
          throw new MockApiError(404, ErrorCode.NotFound, '用户不存在')
        }
        const temporaryPassword = `Tmp-${Math.random()
          .toString(36)
          .slice(2, 8)}-9A`
        user.password = temporaryPassword
        user.mustChangePassword = true
        const expiresAt = new Date(Date.now() + 30 * 60_000).toISOString()
        appendAudit(db, admin, {
          action: 'user.reset_password',
          resourceType: 'user',
          resourceId: user.id,
          after: { temporaryPassword, expiresAt },
          result: 'success',
          requestId: requestId(request),
        })
        return { temporaryPassword, expiresAt }
      })
    }),
  ),

  http.post('/api/manage/users/:userId/adjust-credits', ({ params, request }) =>
    respond(async () => {
      const payload = await readJson<AdjustCreditsPayload>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        const user = db.users.find((item) => item.id === params.userId)
        if (!user) {
          throw new MockApiError(404, ErrorCode.NotFound, '用户不存在')
        }
        if (
          !Number.isInteger(payload.amount) ||
          payload.amount === 0 ||
          !payload.reason?.trim()
        ) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '调整次数必须为非零整数，并填写原因',
          )
        }
        const balanceBefore = user.balance
        const balanceAfter = balanceBefore + payload.amount
        if (balanceAfter < 0) {
          throw new MockApiError(
            409,
            ErrorCode.InsufficientCredits,
            '用户剩余次数不足，无法完成扣减',
            { balance: balanceBefore, requested: payload.amount },
          )
        }
        user.balance = balanceAfter
        const ledger = {
          id: createId('ledger'),
          userId: user.id,
          type: 'adjustment' as const,
          amount: payload.amount,
          balanceBefore,
          balanceAfter,
          description:
            payload.amount > 0 ? '管理员增加次数' : '管理员扣减次数',
          reason: payload.reason.trim(),
          referenceNo: payload.referenceNo?.trim() || undefined,
          operatorId: admin.id,
          createdAt: new Date().toISOString(),
        }
        db.ledger.unshift(ledger)
        appendAudit(db, admin, {
          action: 'user.adjust_credits',
          resourceType: 'user',
          resourceId: user.id,
          before: { balance: balanceBefore },
          after: { balance: balanceAfter, amount: payload.amount },
          reason: payload.reason.trim(),
          result: 'success',
          requestId: requestId(request),
        })
        return { user: publicUser(user), ledger }
      })
    }),
  ),

  http.get('/api/manage/usage-ledger', ({ request }) =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      const url = new URL(request.url)
      const userId = url.searchParams.get('userId')
      const items = db.ledger
        .filter((item) => !userId || item.userId === userId)
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
      const { page, pageSize } = pageValues(url)
      return paginate(items, page, pageSize)
    }),
  ),
]
