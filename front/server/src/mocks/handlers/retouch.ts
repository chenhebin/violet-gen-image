import { http } from 'msw'
import { RETOUCH_UPLOAD_CONFIG } from '@/config'
import {
  appendAudit,
  publicTicket,
  publicTicketSummary,
  publicUser,
} from '@/mocks/db'
import {
  dbAndAdmin,
  idempotentMutation,
  MockApiError,
  readJson,
  requestId,
  respond,
} from '@/mocks/helpers'
import type { MockDb, MockTicket } from '@/mocks/schema'
import { ErrorCode } from '@/types/api'
import type { RetouchTicketStatus } from '@/types/domain'
import { createId } from '@/utils/id'
import { paginate } from '@/utils/pagination'

function requireTicket(db: MockDb, ticketId: string): MockTicket {
  const ticket = db.tickets.find((item) => item.id === ticketId)
  if (!ticket) {
    throw new MockApiError(
      404,
      ErrorCode.RetouchNotFound,
      '人工修图工单不存在',
    )
  }
  return ticket
}

function requireStatus(
  ticket: MockTicket,
  allowed: RetouchTicketStatus[],
  message: string,
): void {
  if (!allowed.includes(ticket.status)) {
    throw new MockApiError(
      409,
      ErrorCode.RetouchInvalidStatus,
      message,
      { status: ticket.status, allowed },
    )
  }
}

function appendTimeline(
  ticket: MockTicket,
  status: RetouchTicketStatus,
  action: string,
  note?: string,
): void {
  const now = new Date().toISOString()
  ticket.status = status
  ticket.updatedAt = now
  ticket.timeline.push({
    status,
    action,
    note,
    createdAt: now,
  })
}

export const retouchHandlers = [
  http.get('/api/manage/retouch-tickets', ({ request }) =>
    respond(() => {
      const { db } = dbAndAdmin('retouch:manage')
      const url = new URL(request.url)
      const keyword = url.searchParams.get('keyword')?.trim().toLowerCase()
      const status = url.searchParams.get('status') as
        | RetouchTicketStatus
        | null
      const page = Number(url.searchParams.get('page') || 1)
      const pageSize = Number(url.searchParams.get('pageSize') || 20)
      const items = db.tickets
        .filter((item) => {
          const user = db.users.find((entry) => entry.id === item.userId)
          return (
            (!status || item.status === status) &&
            (!keyword ||
              item.ticketNo.toLowerCase().includes(keyword) ||
              item.taskTitle.toLowerCase().includes(keyword) ||
              user?.email.toLowerCase().includes(keyword))
          )
        })
        .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
        .map((item) => publicTicketSummary(item, db))
      return paginate(items, page, pageSize)
    }),
  ),

  http.get('/api/manage/retouch-tickets/:ticketId', ({ params }) =>
    respond(() => {
      const { db } = dbAndAdmin('retouch:manage')
      return publicTicket(requireTicket(db, String(params.ticketId)), db)
    }),
  ),

  http.post('/api/manage/retouch-tickets/:ticketId/quote', ({ params, request }) =>
    respond(async () => {
      const payload = await readJson<{
        credits: number
        note?: string
      }>(request)
      const { db, admin } = dbAndAdmin('retouch:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        const ticket = requireTicket(db, String(params.ticketId))
        requireStatus(
          ticket,
          ['submitted', 'quote_pending'],
          '当前工单状态不允许报价',
        )
        if (
          !Number.isInteger(payload.credits) ||
          payload.credits < 1 ||
          payload.credits > 999 ||
          (payload.note?.length ?? 0) > 500
        ) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '报价须为 1 到 999 的整数，说明不能超过 500 字',
          )
        }
        const previousQuote = ticket.quote?.credits
        ticket.quote = {
          id: createId('quote'),
          credits: payload.credits,
          createdAt: new Date().toISOString(),
        }
        appendTimeline(
          ticket,
          'quote_pending',
          'quoted',
          payload.note?.trim(),
        )
        appendAudit(db, admin, {
          action: 'retouch.quote',
          resourceType: 'retouch_ticket',
          resourceId: ticket.id,
          before: { quoteCredits: previousQuote },
          after: { quoteCredits: payload.credits },
          result: 'success',
          requestId: requestId(request),
        })
        return publicTicket(ticket, db)
      })
    }),
  ),

  http.post('/api/manage/retouch-tickets/:ticketId/start', ({ params, request }) =>
    respond(() => {
      const { db, admin } = dbAndAdmin('retouch:manage')
      return idempotentMutation(db, admin, request, null, () => {
        const ticket = requireTicket(db, String(params.ticketId))
        requireStatus(ticket, ['accepted'], '只有已接受报价的工单可以开工')
        if (!ticket.quote || ticket.reservedCredits <= 0) {
          throw new MockApiError(
            409,
            ErrorCode.RetouchQuoteInvalid,
            '工单报价或次数预占记录不存在',
          )
        }
        ticket.spentCredits = ticket.reservedCredits
        appendTimeline(ticket, 'processing', 'started', '人工修图已开工')
        appendAudit(db, admin, {
          action: 'retouch.start',
          resourceType: 'retouch_ticket',
          resourceId: ticket.id,
          after: { status: ticket.status, spentCredits: ticket.spentCredits },
          result: 'success',
          requestId: requestId(request),
        })
        return publicTicket(ticket, db)
      })
    }),
  ),

  http.post('/api/manage/retouch-tickets/:ticketId/deliver', ({ params, request }) =>
    respond(async () => {
      const form = await request.formData()
      const files = form
        .getAll('files')
        .filter((item): item is File => item instanceof File)
      const note = String(form.get('note') || '').trim()
      const fingerprint = {
        files: files.map((item) => ({
          name: item.name,
          size: item.size,
          type: item.type,
        })),
        note,
      }
      const { db, admin } = dbAndAdmin('retouch:manage')
      return idempotentMutation(db, admin, request, fingerprint, () => {
        const ticket = requireTicket(db, String(params.ticketId))
        requireStatus(ticket, ['processing'], '只有处理中的工单可以交付')
        if (
          files.length < 1 ||
          files.length > RETOUCH_UPLOAD_CONFIG.maxFiles ||
          note.length > 500 ||
          files.some(
            (item) =>
              item.size > RETOUCH_UPLOAD_CONFIG.maxFileSize ||
              !RETOUCH_UPLOAD_CONFIG.allowedTypes.includes(
                item.type as (typeof RETOUCH_UPLOAD_CONFIG.allowedTypes)[number],
              ),
          )
        ) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '请上传 1 到 4 个 JPG、PNG 或 WebP 文件，每个不超过 30MB',
          )
        }
        ticket.deliverables = files.map(() => ({
          id: createId('retouch_result'),
          url: '/demo/auth-studio.jpg',
          width: 1200,
          height: 1800,
        }))
        appendTimeline(
          ticket,
          'awaiting_confirmation',
          ticket.revision ? 'revision_delivered' : 'delivered',
          note || '人工成片已交付',
        )
        appendAudit(db, admin, {
          action: 'retouch.deliver',
          resourceType: 'retouch_ticket',
          resourceId: ticket.id,
          after: {
            status: ticket.status,
            deliverableCount: ticket.deliverables.length,
          },
          result: 'success',
          requestId: requestId(request),
        })
        return publicTicket(ticket, db)
      })
    }),
  ),

  http.post('/api/manage/retouch-tickets/:ticketId/reject', ({ params, request }) =>
    respond(async () => {
      const payload = await readJson<{ reason: string }>(request)
      const { db, admin } = dbAndAdmin('retouch:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        const ticket = requireTicket(db, String(params.ticketId))
        requireStatus(
          ticket,
          ['submitted', 'quote_pending'],
          '当前工单状态不允许拒单',
        )
        if (!payload.reason?.trim() || payload.reason.length > 500) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '请填写 1 到 500 字的拒单原因',
          )
        }
        appendTimeline(
          ticket,
          'rejected',
          'rejected',
          payload.reason.trim(),
        )
        appendAudit(db, admin, {
          action: 'retouch.reject',
          resourceType: 'retouch_ticket',
          resourceId: ticket.id,
          reason: payload.reason.trim(),
          after: { status: ticket.status },
          result: 'success',
          requestId: requestId(request),
        })
        return publicTicket(ticket, db)
      })
    }),
  ),

  http.post('/api/manage/retouch-tickets/:ticketId/fail', ({ params, request }) =>
    respond(async () => {
      const payload = await readJson<{ reason: string }>(request)
      const { db, admin } = dbAndAdmin('retouch:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        const ticket = requireTicket(db, String(params.ticketId))
        requireStatus(
          ticket,
          ['accepted', 'processing', 'awaiting_confirmation'],
          '当前工单状态不允许标记履约失败',
        )
        if (!payload.reason?.trim() || payload.reason.length > 500) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '请填写 1 到 500 字的失败原因',
          )
        }
        if (ticket.refundedCredits > 0) {
          throw new MockApiError(
            409,
            ErrorCode.RetouchInvalidStatus,
            '该工单已经完成退款',
          )
        }
        const refund = ticket.quote?.credits ?? ticket.reservedCredits
        if (refund <= 0) {
          throw new MockApiError(
            409,
            ErrorCode.RetouchQuoteInvalid,
            '工单报价或预占记录不存在',
          )
        }
        const user = db.users.find((item) => item.id === ticket.userId)
        if (!user) {
          throw new MockApiError(404, ErrorCode.NotFound, '工单用户不存在')
        }
        const before = user.balance
        user.balance += refund
        ticket.refundedCredits = refund
        appendTimeline(
          ticket,
          'rejected',
          'fulfillment_failed',
          payload.reason.trim(),
        )
        db.ledger.unshift({
          id: createId('ledger'),
          userId: user.id,
          type: 'refund',
          amount: refund,
          balanceBefore: before,
          balanceAfter: user.balance,
          description: `人工修图工单 ${ticket.ticketNo} 履约失败退款`,
          reason: payload.reason.trim(),
          operatorId: admin.id,
          createdAt: new Date().toISOString(),
        })
        appendAudit(db, admin, {
          action: 'retouch.fail',
          resourceType: 'retouch_ticket',
          resourceId: ticket.id,
          reason: payload.reason.trim(),
          after: {
            status: ticket.status,
            refundedCredits: refund,
            user: publicUser(user),
          },
          result: 'success',
          requestId: requestId(request),
        })
        return publicTicket(ticket, db)
      })
    }),
  ),
]
