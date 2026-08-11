import { http } from 'msw'
import { APP_CONFIG, REDEMPTION_CONFIG } from '@/config'
import {
  appendAudit,
  publicBatch,
  publicCode,
  publicCodeDetail,
} from '@/mocks/db'
import {
  dbAndAdmin,
  idempotentMutation,
  MockApiError,
  readJson,
  requestId,
  respond,
} from '@/mocks/helpers'
import type { MockRedemptionCode } from '@/mocks/schema'
import { ErrorCode } from '@/types/api'
import type {
  CreateRedemptionBatchPayload,
  CreateRedemptionBatchResult,
  DisableRedemptionPayload,
  ExtendRedemptionPayload,
} from '@/types/domain'
import { createId } from '@/utils/id'
import { paginate } from '@/utils/pagination'
import {
  deriveRedemptionStatus,
  normalizeRedemptionCode,
} from '@/utils/redemption'

const CODE_ALPHABET = '23456789ABCDEFGHJKLMNPQRSTUVWXYZ'

function randomSegment(length: number): string {
  const values = new Uint32Array(length)
  crypto.getRandomValues(values)
  return Array.from(
    values,
    (value) => CODE_ALPHABET[value % CODE_ALPHABET.length],
  ).join('')
}

function generateCode(existing: Set<string>): string {
  for (let attempt = 0; attempt < 20; attempt += 1) {
    const value = `YY-${randomSegment(4)}-${randomSegment(4)}-${randomSegment(4)}`
    if (!existing.has(value)) return value
  }
  throw new MockApiError(
    500,
    ErrorCode.Unknown,
    '兑换码生成失败，请重新尝试',
  )
}

function pageValues(url: URL): { page: number; pageSize: number } {
  return {
    page: Number(url.searchParams.get('page') || 1),
    pageSize: Number(
      url.searchParams.get('pageSize') || APP_CONFIG.defaultPageSize,
    ),
  }
}

function selectedCodes(
  codes: MockRedemptionCode[],
  payload: { codeIds?: string[]; batchId?: string },
): MockRedemptionCode[] {
  if (payload.codeIds?.length) {
    const ids = new Set(payload.codeIds)
    return codes.filter((item) => ids.has(item.id))
  }
  if (payload.batchId) {
    return codes.filter((item) => item.batchId === payload.batchId)
  }
  throw new MockApiError(
    422,
    ErrorCode.InvalidPayload,
    '请选择兑换码或生成批次',
  )
}

export const redemptionHandlers = [
  http.get('/api/manage/redemption-codes', ({ request }) =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      const url = new URL(request.url)
      const keyword = url.searchParams.get('keyword')?.trim().toUpperCase()
      const status = url.searchParams.get('status')
      const batchId = url.searchParams.get('batchId')
      const productCode = url.searchParams.get('productCode')
      const redeemedBy = url.searchParams.get('redeemedBy')?.toLowerCase()
      const expiringSoon = url.searchParams.get('expiringSoon') === 'true'
      const items = db.codes
        .filter((item) => {
          const publicItem = publicCode(item, db)
          return (
            (!keyword ||
              normalizeRedemptionCode(item.fullCode).includes(keyword) ||
              publicItem.maskedCode.toUpperCase().includes(keyword)) &&
            (!status || publicItem.status === status) &&
            (!batchId || item.batchId === batchId) &&
            (!productCode || item.productCode === productCode) &&
            (!redeemedBy ||
              publicItem.redeemedByEmail?.toLowerCase().includes(redeemedBy)) &&
            (!expiringSoon || publicItem.expiringSoon)
          )
        })
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
        .map((item) => publicCode(item, db))
      const { page, pageSize } = pageValues(url)
      return paginate(items, page, pageSize)
    }),
  ),

  http.get('/api/manage/redemption-codes/:codeId', ({ params }) =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      const code = db.codes.find((item) => item.id === params.codeId)
      if (!code) {
        throw new MockApiError(404, ErrorCode.NotFound, '兑换码不存在')
      }
      return publicCodeDetail(code, db)
    }),
  ),

  http.get('/api/manage/redemption-batches', ({ request }) =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      const url = new URL(request.url)
      const keyword = url.searchParams.get('keyword')?.trim().toLowerCase()
      const productCode = url.searchParams.get('productCode')
      const items = db.batches
        .filter(
          (item) =>
            (!keyword ||
              item.name.toLowerCase().includes(keyword) ||
              item.id.toLowerCase().includes(keyword)) &&
            (!productCode || item.productCode === productCode),
        )
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
        .map((item) => publicBatch(item, db))
      const { page, pageSize } = pageValues(url)
      return paginate(items, page, pageSize)
    }),
  ),

  http.get('/api/manage/redemption-batches/:batchId', ({ params }) =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      const batch = db.batches.find((item) => item.id === params.batchId)
      if (!batch) {
        throw new MockApiError(404, ErrorCode.NotFound, '生成批次不存在')
      }
      return publicBatch(batch, db)
    }),
  ),

  http.post('/api/manage/redemption-batches', ({ request }) =>
    respond(async () => {
      const payload = await readJson<CreateRedemptionBatchPayload>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        if (
          !payload.name?.trim() ||
          !Number.isInteger(payload.quantity) ||
          payload.quantity < REDEMPTION_CONFIG.minQuantity ||
          payload.quantity > REDEMPTION_CONFIG.maxQuantity ||
          !Number.isInteger(payload.creditsPerCode) ||
          payload.creditsPerCode < REDEMPTION_CONFIG.minCredits ||
          payload.productCode !== APP_CONFIG.productCode
        ) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '请填写有效的批次名称、数量、次数和商品标识',
          )
        }
        const neverExpires = Boolean(payload.neverExpires)
        const defaultExpiry = new Date(
          Date.now() +
            REDEMPTION_CONFIG.defaultValidityDays * 86_400_000,
        ).toISOString()
        const expiresAt = neverExpires
          ? null
          : (payload.expiresAt ?? defaultExpiry)
        if (
          expiresAt &&
          new Date(expiresAt).getTime() <= Date.now()
        ) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '有效期必须晚于当前时间',
          )
        }

        const batchId = createId('batch')
        const createdAt = new Date().toISOString()
        const batch = {
          id: batchId,
          name: payload.name.trim(),
          productCode: payload.productCode,
          quantity: payload.quantity,
          creditsPerCode: payload.creditsPerCode,
          expiresAt,
          neverExpires,
          note: payload.note?.trim() || undefined,
          createdBy: admin.id,
          createdAt,
        }
        const existing = new Set(db.codes.map((item) => item.fullCode))
        const codes = Array.from({ length: payload.quantity }, () => {
          const fullCode = generateCode(existing)
          existing.add(fullCode)
          return {
            id: createId('code'),
            fullCode,
            batchId,
            productCode: payload.productCode,
            credits: payload.creditsPerCode,
            expiresAt,
            createdAt,
            operationHistory: [
              {
                action: 'generated',
                operator: admin.email,
                createdAt,
              },
            ],
          } satisfies MockRedemptionCode
        })
        db.batches.unshift(batch)
        db.codes.unshift(...codes)
        appendAudit(db, admin, {
          action: 'redemption.batch.create',
          resourceType: 'redemption_batch',
          resourceId: batchId,
          after: {
            name: batch.name,
            quantity: batch.quantity,
            creditsPerCode: batch.creditsPerCode,
            expiresAt,
          },
          result: 'success',
          requestId: requestId(request),
        })
        return {
          batch: publicBatch(batch, db),
          codes: codes.map((item) => ({
            id: item.id,
            fullCode: item.fullCode,
            maskedCode: `YY-****-****-${item.fullCode.slice(-4)}`,
          })),
        } satisfies CreateRedemptionBatchResult
      })
    }),
  ),

  http.post('/api/manage/redemption-codes/:codeId/reveal', ({ params, request }) =>
    respond(() => {
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, null, () => {
        const code = db.codes.find((item) => item.id === params.codeId)
        if (!code) {
          throw new MockApiError(404, ErrorCode.NotFound, '兑换码不存在')
        }
        if (deriveRedemptionStatus(code) !== 'unused') {
          throw new MockApiError(
            409,
            ErrorCode.InvalidPayload,
            '只有未使用兑换码可以查看完整值',
          )
        }
        appendAudit(db, admin, {
          action: 'redemption.code.reveal',
          resourceType: 'redemption_code',
          resourceId: code.id,
          result: 'success',
          requestId: requestId(request),
        })
        return { id: code.id, fullCode: code.fullCode }
      })
    }),
  ),

  http.post('/api/manage/redemption-batches/:batchId/reveal', ({ params, request }) =>
    respond(() => {
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, null, () => {
        const batch = db.batches.find((item) => item.id === params.batchId)
        if (!batch) {
          throw new MockApiError(404, ErrorCode.NotFound, '生成批次不存在')
        }
        const codes = db.codes
          .filter(
            (item) =>
              item.batchId === batch.id &&
              deriveRedemptionStatus(item) === 'unused',
          )
          .map((item) => ({ id: item.id, fullCode: item.fullCode }))
        appendAudit(db, admin, {
          action: 'redemption.batch.reveal',
          resourceType: 'redemption_batch',
          resourceId: batch.id,
          after: { revealedCount: codes.length },
          result: 'success',
          requestId: requestId(request),
        })
        return codes
      })
    }),
  ),

  http.post('/api/manage/redemption-batches/:batchId/export', ({ params, request }) =>
    respond(() => {
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, null, () => {
        const batch = db.batches.find((item) => item.id === params.batchId)
        if (!batch) {
          throw new MockApiError(404, ErrorCode.NotFound, '生成批次不存在')
        }
        const items = db.codes.filter(
          (item) =>
            item.batchId === batch.id &&
            deriveRedemptionStatus(item) === 'unused',
        )
        const rows = [
          ['兑换码', '批次', '次数', '商品标识', '状态', '有效期'],
          ...items.map((item) => [
            item.fullCode,
            batch.name,
            String(item.credits),
            item.productCode,
            'unused',
            item.expiresAt ?? '永久有效',
          ]),
        ]
        appendAudit(db, admin, {
          action: 'redemption.batch.export',
          resourceType: 'redemption_batch',
          resourceId: batch.id,
          after: { exportedCount: items.length },
          result: 'success',
          requestId: requestId(request),
        })
        return {
          filename: `${batch.name.replaceAll(/[^\w\u4e00-\u9fa5-]/g, '_')}.csv`,
          csv: `\uFEFF${rows
            .map((row) =>
              row.map((value) => `"${value.replaceAll('"', '""')}"`).join(','),
            )
            .join('\n')}`,
        }
      })
    }),
  ),

  http.post('/api/manage/redemption-codes/disable', ({ request }) =>
    respond(async () => {
      const payload = await readJson<DisableRedemptionPayload>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        if (!payload.reason?.trim()) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '请填写失效原因',
          )
        }
        const targets = selectedCodes(db.codes, payload)
        let affected = 0
        const now = new Date().toISOString()
        targets.forEach((item) => {
          if (deriveRedemptionStatus(item) !== 'unused') return
          item.disabledAt = now
          item.disabledBy = admin.id
          item.disabledReason = payload.reason.trim()
          item.operationHistory.push({
            action: 'disabled',
            operator: admin.email,
            reason: payload.reason.trim(),
            createdAt: now,
          })
          affected += 1
        })
        appendAudit(db, admin, {
          action: 'redemption.disable',
          resourceType: payload.batchId
            ? 'redemption_batch'
            : 'redemption_code',
          resourceId: payload.batchId ?? `${targets.length}_codes`,
          reason: payload.reason.trim(),
          after: { affected, skipped: targets.length - affected },
          result: 'success',
          requestId: requestId(request),
        })
        return {
          affected,
          skipped: targets.length - affected,
          failed: 0,
        }
      })
    }),
  ),

  http.post('/api/manage/redemption-codes/extend', ({ request }) =>
    respond(async () => {
      const payload = await readJson<ExtendRedemptionPayload>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        if (!payload.reason?.trim()) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '请填写延期原因',
          )
        }
        if (
          payload.expiresAt &&
          new Date(payload.expiresAt).getTime() <= Date.now()
        ) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '新的有效期必须晚于当前时间',
          )
        }
        const targets = selectedCodes(db.codes, payload)
        let affected = 0
        const now = new Date().toISOString()
        targets.forEach((item) => {
          const status = deriveRedemptionStatus(item)
          if (!['unused', 'expired'].includes(status) || item.disabledAt) return
          item.expiresAt = payload.expiresAt
          item.operationHistory.push({
            action: 'extended',
            operator: admin.email,
            reason: payload.reason.trim(),
            createdAt: now,
          })
          affected += 1
        })
        appendAudit(db, admin, {
          action: 'redemption.extend',
          resourceType: payload.batchId
            ? 'redemption_batch'
            : 'redemption_code',
          resourceId: payload.batchId ?? `${targets.length}_codes`,
          reason: payload.reason.trim(),
          after: {
            affected,
            skipped: targets.length - affected,
            expiresAt: payload.expiresAt,
          },
          result: 'success',
          requestId: requestId(request),
        })
        return {
          affected,
          skipped: targets.length - affected,
          failed: 0,
        }
      })
    }),
  ),
]
