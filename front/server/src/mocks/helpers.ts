import { HttpResponse, delay, type DefaultBodyType } from 'msw'
import { MOCK_CONFIG } from '@/config'
import {
  appendAudit,
  getSessionAdminId,
  readDb,
  writeDb,
} from '@/mocks/db'
import type { MockAdmin, MockDb } from '@/mocks/schema'
import {
  ErrorCode,
  type ApiErrorResponse,
  type ApiSuccessResponse,
} from '@/types/api'
import type { AdminPermission } from '@/types/domain'

export class MockApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: number,
    message: string,
    readonly details?: unknown,
    readonly retryAfterSeconds?: number,
  ) {
    super(message)
  }
}

const mockRateWindows = new Map<string, { startedAt: number; count: number }>()

export function resetMockRateLimits(): void {
  mockRateWindows.clear()
}

export function enforceRateLimit(scope: string, limit: number): void {
  const now = Date.now()
  const windowMs = 60_000
  const current = mockRateWindows.get(scope)
  const entry = !current || now - current.startedAt >= windowMs
    ? { startedAt: now, count: 1 }
    : { startedAt: current.startedAt, count: current.count + 1 }
  mockRateWindows.set(scope, entry)
  if (entry.count <= limit) return
  const retryAfterSeconds = Math.max(
    1,
    Math.ceil((windowMs - (now - entry.startedAt)) / 1000),
  )
  throw new MockApiError(
    429,
    ErrorCode.RateLimited,
    '请求过于频繁，请稍后重试',
    undefined,
    retryAfterSeconds,
  )
}

export function ok<T>(
  data: T,
  init?: ResponseInit,
): HttpResponse<DefaultBodyType> {
  return HttpResponse.json<ApiSuccessResponse<T>>(
    { code: 0, data },
    { status: 200, ...init },
  ) as HttpResponse<DefaultBodyType>
}

export function fail(error: MockApiError): HttpResponse<DefaultBodyType> {
  return HttpResponse.json<ApiErrorResponse>(
    {
      code: error.code,
      message: error.message,
      ...(error.details === undefined ? {} : { details: error.details }),
    },
    {
      status: error.status,
      headers: error.retryAfterSeconds
        ? { 'Retry-After': String(error.retryAfterSeconds) }
        : undefined,
    },
  ) as HttpResponse<DefaultBodyType>
}

export async function respond<T>(
  operation: () => Promise<T> | T,
): Promise<HttpResponse<DefaultBodyType>> {
  try {
    await delay(MOCK_CONFIG.latencyMs)
    return ok(await operation())
  } catch (error) {
    if (error instanceof MockApiError) return fail(error)
    return fail(
      new MockApiError(
        500,
        ErrorCode.Unknown,
        '服务暂时不可用，请稍后重试',
        error instanceof Error ? error.message : error,
      ),
    )
  }
}

export function requireAdmin(
  db: MockDb,
  permission?: AdminPermission,
): MockAdmin {
  const adminId = getSessionAdminId()
  const admin = db.admins.find((item) => item.id === adminId)
  if (!admin) {
    throw new MockApiError(401, ErrorCode.AuthRequired, '登录状态已失效')
  }
  if (admin.status !== 'active') {
    throw new MockApiError(403, ErrorCode.AccountDisabled, '管理员账号已停用')
  }
  if (permission && !admin.permissions.includes(permission)) {
    throw new MockApiError(403, ErrorCode.Forbidden, '无权执行此管理操作')
  }
  return admin
}

export function requestId(request: Request): string {
  return request.headers.get('X-Request-Id') || 'req_mock'
}

function stableStringify(value: unknown): string {
  if (value === undefined) return ''
  if (value === null || typeof value !== 'object') return JSON.stringify(value)
  if (Array.isArray(value)) return `[${value.map(stableStringify).join(',')}]`
  return `{${Object.entries(value as Record<string, unknown>)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, entry]) => `${JSON.stringify(key)}:${stableStringify(entry)}`)
    .join(',')}}`
}

export function idempotentMutation<T>(
  db: MockDb,
  admin: MockAdmin,
  request: Request,
  payload: unknown,
  operation: () => T,
): T {
  const key = request.headers.get('Idempotency-Key')
  if (!key) {
    throw new MockApiError(
      422,
      ErrorCode.InvalidPayload,
      '缺少 Idempotency-Key 请求头',
    )
  }
  const path = new URL(request.url).pathname
  const scope = `${admin.id}:${path}:${key}`
  const fingerprint = stableStringify(payload)
  const previous = db.idempotency[scope]
  if (previous) {
    if (previous.fingerprint !== fingerprint) {
      appendAudit(db, admin, {
        action: 'idempotency.conflict',
        resourceType: 'request',
        resourceId: path,
        result: 'failure',
        requestId: requestId(request),
      })
      writeDb(db)
      throw new MockApiError(
        409,
        ErrorCode.DuplicateRequest,
        '幂等键已用于不同的请求内容',
      )
    }
    return previous.result as T
  }

  const result = operation()
  db.idempotency[scope] = {
    operatorId: admin.id,
    path,
    fingerprint,
    result,
    createdAt: new Date().toISOString(),
  }
  writeDb(db)
  return result
}

export async function readJson<T>(request: Request): Promise<T> {
  try {
    return (await request.json()) as T
  } catch {
    throw new MockApiError(
      422,
      ErrorCode.InvalidPayload,
      '请求数据格式不正确',
    )
  }
}

export function dbAndAdmin(permission?: AdminPermission): {
  db: MockDb
  admin: MockAdmin
} {
  const db = readDb()
  return { db, admin: requireAdmin(db, permission) }
}
