import { delay, http, HttpResponse, type DefaultBodyType } from 'msw'
import type {
  Asset,
  GenerationTask,
  PromptReferenceAsset,
  PromptSections,
  PromptVersion,
  RegisterPayload,
  RetouchTicket,
} from '@/types/domain'
import type {
  CreateGenerationPayload,
  CreateRetouchTicketPayload,
  OptimizePromptPayload,
  RetouchTicketBalanceResult,
} from '@/services/api'
import {
  ASSET_CONFIG,
  AUTH_CONFIG,
  PROMPT_CONFIG,
  RETOUCH_TICKET_CONFIG,
} from '@/config'
import { ErrorCode } from '@/types/api'
import { createId } from '@/utils/id'
import {
  addLedger,
  clearSession,
  currentEntitlement,
  getSessionUserId,
  materializeRetouchTicket,
  materializeTask,
  publicRetouchTicket,
  publicUser,
  readDb,
  setSessionUserId,
  syncTaskRetouchTicket,
  transitionRetouchTicket,
  writeDb,
  createRetouchTicketNumber,
  type MockRetouchTicket,
  type MockTask,
} from './db'

function ok<T>(data: T, status = 200) {
  return HttpResponse.json({ code: 0 as const, data }, { status })
}

function fail(code: number, message: string, status = 400, details?: unknown) {
  return HttpResponse.json({ code, message, details }, { status })
}

function requireSession() {
  const db = readDb()
  const userId = getSessionUserId()
  const user = db.users.find((item) => item.id === userId)
  return { db, userId, user }
}

function maskRedemptionCode(code: string): string {
  const parts = code.split('-')
  if (parts.length < 3) return `${code.slice(0, 3)}****${code.slice(-3)}`
  return parts
    .map((part, index) => index === 0 || index === parts.length - 1 ? part : '****')
    .join('-')
}

let retouchMutationQueue: Promise<void> = Promise.resolve()
const mockRateWindows = new Map<string, { startedAt: number; count: number }>()

export function resetMockRateLimits(): void {
  mockRateWindows.clear()
}

async function withRetouchMutation<T>(operation: () => T | Promise<T>) {
  const previous = retouchMutationQueue
  let release = () => {}
  retouchMutationQueue = new Promise<void>((resolve) => {
    release = resolve
  })
  await previous
  try {
    return await operation()
  } finally {
    release()
  }
}

function retouchIdempotencyKey(
  request: Request,
  userId: string,
  operation: string,
): string | null {
  const value = request.headers.get('Idempotency-Key')?.trim()
  return value ? `retouch:${userId}:${operation}:${value}` : null
}

function retouchIdempotency(
  db: ReturnType<typeof readDb>,
  request: Request,
  userId: string,
  operation: string,
  payload: unknown,
) {
  const key = retouchIdempotencyKey(request, userId, operation)
  if (!key) {
    return {
      key: null,
      digest: '',
      conflict: false,
      replay: undefined as unknown,
    }
  }
  const digest = requestDigest(payload)
  const previousDigest = db.idempotencyDigests[key]
  return {
    key,
    digest,
    conflict: Boolean(previousDigest && previousDigest !== digest),
    replay: db.idempotency[key],
  }
}

function requestDigest(value: unknown): string {
  if (value === undefined) return ''
  if (value === null || typeof value !== 'object') return JSON.stringify(value)
  if (Array.isArray(value)) return `[${value.map(requestDigest).join(',')}]`
  return `{${Object.entries(value as Record<string, unknown>)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, item]) => `${JSON.stringify(key)}:${requestDigest(item)}`)
    .join(',')}}`
}

function userIdempotency(
  db: ReturnType<typeof readDb>,
  request: Request,
  userId: string,
  path: string,
  payload: unknown,
) {
  const key = request.headers.get('Idempotency-Key')?.trim()
  if (!key) return { key: null, replay: undefined as unknown }
  const scope = `user:${userId}:${request.method}:${path}:${key}`
  const digest = requestDigest(payload)
  const previousDigest = db.idempotencyDigests[scope]
  if (previousDigest && previousDigest !== digest) {
    return { key, conflict: true, scope, replay: undefined as unknown }
  }
  return { key, conflict: false, scope, replay: db.idempotency[scope] }
}

function rateLimited(scope: string, limit: number): HttpResponse<DefaultBodyType> | null {
  const now = Date.now()
  const current = mockRateWindows.get(scope)
  const windowMs = 60_000
  const entry = !current || now - current.startedAt >= windowMs
    ? { startedAt: now, count: 1 }
    : { startedAt: current.startedAt, count: current.count + 1 }
  mockRateWindows.set(scope, entry)
  if (entry.count <= limit) return null
  const retryAfter = Math.max(1, Math.ceil((windowMs - (now - entry.startedAt)) / 1000))
  return HttpResponse.json(
    { code: ErrorCode.RateLimited, message: '请求过于频繁，请稍后重试' },
    { status: 429, headers: { 'Retry-After': String(retryAfter) } },
  )
}

function paginate<T>(items: T[], page = 1, pageSize = 20) {
  const safePage = Math.max(1, page)
  const safeSize = Math.min(100, Math.max(1, pageSize))
  const start = (safePage - 1) * safeSize
  return {
    items: items.slice(start, start + safeSize),
    page: safePage,
    pageSize: safeSize,
    total: items.length,
    hasMore: start + safeSize < items.length,
  }
}

export const handlers = [
  http.post('*/api/auth/register', async ({ request }) => {
    await delay(480)
    const payload = (await request.json()) as RegisterPayload
    const db = readDb()
    const email = payload.email.trim().toLowerCase()

    if (
      !payload.acceptedTerms ||
      payload.password.length < AUTH_CONFIG.minimumPasswordLength ||
      !email
    ) {
      return fail(ErrorCode.InvalidPayload, '请检查注册信息', 422)
    }
    if (db.users.some((item) => item.email === email)) {
      return fail(ErrorCode.InvalidPayload, '该邮箱已经注册', 409)
    }

    const user = {
      id: createId('user'),
      email,
      password: payload.password,
      createdAt: new Date().toISOString(),
      status: 'active' as const,
    }
    db.users.push(user)
    db.entitlements[user.id] = {
      balance: 0,
      canCreate: false,
      status: 'unredeemed',
    }
    db.ledger[user.id] = []
    writeDb(db)
    setSessionUserId(user.id, payload.remember ?? true)
    return ok(publicUser(user), 201)
  }),

  http.post('*/api/auth/login', async ({ request }) => {
    await delay(420)
    const payload = (await request.json()) as {
      email: string
      password: string
      remember?: boolean
    }
    const db = readDb()
    const user = db.users.find(
      (item) =>
        item.email === payload.email.trim().toLowerCase() &&
        item.password === payload.password,
    )
    if (!user) {
      return fail(ErrorCode.AuthRequired, '邮箱或密码不正确', 401)
    }
    if (user.status === 'disabled') {
      return fail(ErrorCode.AccountDisabled, '账号当前不可用', 403)
    }

    setSessionUserId(user.id, payload.remember ?? false)
    return ok(publicUser(user))
  }),

  http.post('*/api/auth/logout', async () => {
    await delay(160)
    clearSession()
    return ok(null)
  }),

  http.get('*/api/auth/session', async () => {
    await delay(120)
    const { user } = requireSession()
    return user
      ? ok(publicUser(user))
      : fail(ErrorCode.AuthRequired, '登录状态已失效', 401)
  }),

  http.get('*/api/me', () => {
    const { user } = requireSession()
    return user
      ? ok(publicUser(user))
      : fail(ErrorCode.AuthRequired, '请先登录', 401)
  }),

  http.get('*/api/entitlements', async () => {
    await delay(180)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    return ok(currentEntitlement(db, userId))
  }),

  http.get('*/api/notices/ai-processing', async () => {
    await delay(100)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    const acknowledged = db.aiNoticeVersions[userId] === 'ai-processing-v2'
    return ok({
      version: 'ai-processing-v2',
      title: '关于第三方 AI 处理的告知',
      providerDisclosure: '当您主动使用提示词优化或图片生成功能时，映研只会把完成当前任务所需的图片和提示词发送给平台配置的第三方 AI 服务商；不会公开展示您的内容，也不会提供给其他用户。',
      securitySummary: '平台采用权限隔离、私有存储和短期签名地址保护您的素材。服务商只收到当前任务所需的内容；您的密码、兑换码、次数余额等不会发送给服务商。平台 API Key 由服务端保管，不会向其他用户展示或写入普通日志。',
      purpose: '仅用于完成您主动点击的提示词优化、文生图和图生图任务，不用于广告或向其他用户展示。',
      processingScope: ['当前任务所需的原图和参考图（参考图仅用于提示词分析，不会作为图生图输入）', '当前任务的需求、已确认提示词和生成参数', '不会发送您的账号密码、兑换码、次数余额或其他用户信息'],
      retentionDays: 90,
      stopUseDescription: '素材默认在任务完成或结束后保留 90 天，用于查看结果和处理关联服务。您可以随时停止后续使用；已经发送给第三方服务商的请求无法撤回。',
      acknowledged,
      ...(acknowledged ? { acknowledgedAt: new Date().toISOString() } : {}),
    })
  }),

  http.post('*/api/notices/ai-processing/ack', async ({ request }) => {
    await delay(120)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    const key = request.headers.get('Idempotency-Key')?.trim()
    if (!key) return fail(ErrorCode.InvalidPayload, '缺少 Idempotency-Key 请求头', 422)
    const payload = (await request.json()) as { version?: string }
    if (payload.version !== 'ai-processing-v2') return fail(ErrorCode.InvalidPayload, '告知版本已更新，请重新加载', 409)
    const scope = `notice:${userId}:${key}`
    if (db.idempotency[scope]) return ok(db.idempotency[scope])
    const acknowledgedAt = new Date().toISOString()
    db.aiNoticeVersions[userId] = 'ai-processing-v2'
    const result = {
      version: 'ai-processing-v2',
      title: '关于第三方 AI 处理的告知',
      providerDisclosure: '当您主动使用提示词优化或图片生成功能时，映研只会把完成当前任务所需的图片和提示词发送给平台配置的第三方 AI 服务商；不会公开展示您的内容，也不会提供给其他用户。',
      securitySummary: '平台采用权限隔离、私有存储和短期签名地址保护您的素材。服务商只收到当前任务所需的内容；您的密码、兑换码、次数余额等不会发送给服务商。平台 API Key 由服务端保管，不会向其他用户展示或写入普通日志。',
      purpose: '仅用于完成您主动点击的提示词优化、文生图和图生图任务，不用于广告或向其他用户展示。',
      processingScope: ['当前任务所需的原图和参考图（参考图仅用于提示词分析，不会作为图生图输入）', '当前任务的需求、已确认提示词和生成参数', '不会发送您的账号密码、兑换码、次数余额或其他用户信息'],
      retentionDays: 90,
      stopUseDescription: '素材默认在任务完成或结束后保留 90 天，用于查看结果和处理关联服务。您可以随时停止后续使用；已经发送给第三方服务商的请求无法撤回。',
      acknowledged: true,
      acknowledgedAt,
    }
    db.idempotency[scope] = result
    writeDb(db)
    return ok(result)
  }),

  http.get('*/api/usage/ledger', async () => {
    await delay(220)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    return ok(db.ledger[userId] ?? [])
  }),

  http.post('*/api/redemptions/preview', async ({ request }) => {
    const limited = rateLimited('redemption-preview', 20)
    if (limited) return limited
    await delay(240)
    const payload = (await request.json()) as { code: string }
    const normalized = payload.code.trim().toUpperCase()
    const db = readDb()
    const redemption = db.codes.find((item) => item.code === normalized)
    if (!redemption) return fail(ErrorCode.CodeInvalid, '兑换码无效', 404)
    if (redemption.redeemedBy) return fail(ErrorCode.CodeUsed, '兑换码已经使用', 409)
    if (new Date(redemption.expiresAt).getTime() < Date.now()) {
      return fail(ErrorCode.CodeExpired, '兑换码已过期', 410)
    }
    return ok({
      valid: true as const,
      credits: redemption.credits,
      productName: `AI 生图 ${redemption.credits} 次`,
      maskedCode: maskRedemptionCode(redemption.code),
      expiresAt: redemption.expiresAt,
    })
  }),

  http.post('*/api/redemptions/claim', async ({ request }) => {
    const limited = rateLimited('redemption-claim', 10)
    if (limited) return limited
    await delay(520)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)

    const payload = (await request.json()) as { code: string }
    const idempotency = userIdempotency(db, request, userId, '/api/redemptions/claim', payload)
    if (!idempotency.key) return fail(ErrorCode.InvalidPayload, '缺少 Idempotency-Key 请求头', 422)
    if (idempotency.conflict) return fail(ErrorCode.DuplicateRequest, '幂等键已用于不同的请求内容', 409)
    if (idempotency.replay) return ok(idempotency.replay)
    const normalized = payload.code.trim().toUpperCase()
    const redemption = db.codes.find((item) => item.code === normalized)
    if (!redemption) return fail(ErrorCode.CodeInvalid, '兑换码无效', 404)
    if (redemption.redeemedBy) {
      return fail(
        ErrorCode.CodeUsed,
        '兑换码已经使用',
        409,
        redemption.redeemedBy === userId
          ? { claimedByCurrentUser: true }
          : undefined,
      )
    }
    if (new Date(redemption.expiresAt).getTime() < Date.now()) {
      return fail(ErrorCode.CodeExpired, '兑换码已过期', 410)
    }

    const entitlement = currentEntitlement(db, userId)
    entitlement.balance += redemption.credits
    entitlement.canCreate = true
    entitlement.status = 'active'
    db.entitlements[userId] = entitlement
    redemption.redeemedBy = userId
    redemption.redeemedAt = new Date().toISOString()
    addLedger(db, userId, {
      type: 'redemption',
      amount: redemption.credits,
      balanceAfter: entitlement.balance,
      description: `兑换 ${maskRedemptionCode(redemption.code)}`,
    })
    const result = { added: redemption.credits, entitlement }
    db.idempotencyDigests[idempotency.scope] = requestDigest(payload)
    db.idempotency[idempotency.scope] = result
    writeDb(db)
    return ok(result)
  }),

  http.post('*/api/usage/quote', async ({ request }) => {
    await delay(160)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    const payload = (await request.json()) as { outputCount: number }
    const entitlement = currentEntitlement(db, userId)
    const cost = Math.max(1, Math.min(4, payload.outputCount))
    return ok({
      action: 'generate' as const,
      cost,
      balance: entitlement.balance,
      canSubmit: entitlement.balance >= cost,
    })
  }),

  http.post('*/api/assets', async ({ request }) => {
    await delay(280)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    const formData = await request.formData()
    const file = formData.get('file')
    if (!(file instanceof File)) {
      return fail(ErrorCode.InvalidPayload, '请选择图片文件', 422)
    }
    if (!ASSET_CONFIG.acceptedMimeTypes.some((type) => type === file.type)) {
      return fail(ErrorCode.InvalidPayload, '仅支持图片文件', 422)
    }
    if (file.size > ASSET_CONFIG.maxFileSize) {
      return fail(
        ErrorCode.InvalidPayload,
        `单张图片不能超过 ${ASSET_CONFIG.maxFileSizeLabel}`,
        422,
      )
    }

    const requestedKind = formData.get('kind')
    const kind =
      requestedKind === 'source' ||
      requestedKind === 'reference' ||
      requestedKind === 'retouch-reference'
      ? requestedKind
      : 'reference'
    if ((kind === 'source' || kind === 'reference') && db.aiNoticeVersions[userId] !== 'ai-processing-v2') {
      return fail(ErrorCode.AINoticeRequired, '请先确认第三方 AI 处理告知', 428)
    }
    const role = typeof formData.get('role') === 'string'
      ? (formData.get('role') as Asset['role'])
      : undefined
    const idempotency = userIdempotency(db, request, userId, '/api/assets', {
      name: file.name,
      type: file.type,
      size: file.size,
      kind,
      role,
    })
    if (!idempotency.key) {
      return fail(ErrorCode.InvalidPayload, '缺少 Idempotency-Key 请求头', 422)
    }
    if (idempotency.conflict) {
      return fail(ErrorCode.DuplicateRequest, '幂等键已用于不同的请求内容', 409)
    }
    if (idempotency.replay) return ok(idempotency.replay as Asset, 201)
    const asset: Asset = {
      id: createId('asset'),
      name: file.name,
      kind,
      role,
      mimeType: file.type,
      size: file.size,
      uploadProgress: 100,
    }
    db.assets.push({ ...asset, ownerId: userId })
    db.idempotencyDigests[idempotency.scope] = requestDigest({
      name: file.name, type: file.type, size: file.size, kind, role,
    })
    db.idempotency[idempotency.scope] = asset
    writeDb(db)
    return ok(asset, 201)
  }),

  http.delete('*/api/assets/:assetId', async ({ params, request }) => {
    await delay(100)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    const idempotency = userIdempotency(
      db,
      request,
      userId,
      `/api/assets/${String(params.assetId)}`,
      { assetId: params.assetId },
    )
    if (!idempotency.key) {
      return fail(ErrorCode.InvalidPayload, '缺少 Idempotency-Key 请求头', 422)
    }
    if (idempotency.conflict) {
      return fail(ErrorCode.DuplicateRequest, '幂等键已用于不同的请求内容', 409)
    }
    if (idempotency.replay !== undefined) return ok(null)
    db.assets = db.assets.filter(
      (asset) => asset.id !== params.assetId || asset.ownerId !== userId,
    )
    db.idempotencyDigests[idempotency.scope] = requestDigest({ assetId: params.assetId })
    db.idempotency[idempotency.scope] = null
    writeDb(db)
    return ok(null)
  }),

  http.get('*/api/assets/:assetId/url', async ({ params, request }) => {
		await delay(60)
		const { db, userId, user } = requireSession()
		if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
		const asset = db.assets.find((item) => item.id === params.assetId && item.ownerId === userId)
		if (!asset) return fail(ErrorCode.AssetNotFound, '素材不存在', 404)
		const expiresAt = new Date(Date.now() + 15 * 60_000).toISOString()
		const fallback = asset.kind === 'source' ? '/demo/source-portrait.jpg' : '/demo/style-coast.jpg'
		return ok({ url: fallback, expiresAt, purpose: new URL(request.url).searchParams.get('purpose') ?? 'preview' })
	}),

  http.post('*/api/prompts/reference-prompt', async ({ request }) => {
    const { userId } = requireSession()
    const limited = rateLimited(`reference-prompt:${userId ?? 'anonymous'}`, 12)
    if (limited) return limited
    await delay(760)
    const { db, user } = requireSession()
    if (!user) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    if (db.aiNoticeVersions[user.id] !== 'ai-processing-v2') {
      return fail(ErrorCode.AINoticeRequired, '请先确认第三方 AI 处理告知', 428)
    }
    const payload = (await request.json()) as {
      referenceAssets: PromptReferenceAsset[]
    }
    if (!payload.referenceAssets?.length) {
      return fail(ErrorCode.InvalidPayload, '请先上传参考图', 422)
    }
    const valid = payload.referenceAssets.every((reference) =>
      db.assets.some(
        (asset) =>
          asset.id === reference.assetId &&
          asset.ownerId === user.id &&
          asset.kind === 'reference',
      ),
    )
    if (!valid) {
      return fail(ErrorCode.InvalidPayload, '参考图素材不存在或不可用', 422)
    }
    return ok({
      prompt:
        '主体保持原图人物与身份关系，参考图的清透杂志氛围与自然色彩；' +
        '柔和侧逆光，50mm 人像镜头，浅景深，背景层次干净；' +
        '保留真实皮肤纹理与材质细节，画面克制、自然、无水印。',
      referenceAssets: payload.referenceAssets,
    })
  }),

  http.post('*/api/prompts/optimize', async ({ request }) => {
    const { userId } = requireSession()
    const limited = rateLimited(`prompt-optimize:${userId ?? 'anonymous'}`, 12)
    if (limited) return limited
    await delay(820)
    const { db, user } = requireSession()
    if (!user) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    if (db.aiNoticeVersions[user.id] !== 'ai-processing-v2') {
      return fail(ErrorCode.AINoticeRequired, '请先确认第三方 AI 处理告知', 428)
    }
    const payload = (await request.json()) as OptimizePromptPayload
    if (payload.source.trim().length < PROMPT_CONFIG.minLength) {
      return fail(ErrorCode.InvalidPayload, '请再具体描述想要的画面', 422)
    }
    const sourceAssets = payload.sourceAssetIds.map((assetId) =>
      db.assets.find(
        (asset) =>
          asset.id === assetId &&
          asset.ownerId === user.id &&
          asset.kind === 'source',
      ),
    )
    const referenceAssets = payload.referenceAssets.map((reference) =>
      db.assets.find(
        (asset) =>
          asset.id === reference.assetId &&
          asset.ownerId === user.id &&
          asset.kind === 'reference',
      ),
    )
    if (
      sourceAssets.some((asset) => !asset) ||
      referenceAssets.some((asset) => !asset)
    ) {
      return fail(ErrorCode.InvalidPayload, '提示词素材不存在或不可用', 422)
    }
    if (referenceAssets.length > 0 && sourceAssets.length === 0) {
      return fail(ErrorCode.InvalidPayload, '图生图必须上传待修改原图', 422)
    }
    if (referenceAssets.length > 0 && !payload.referencePrompt?.trim()) {
      return fail(ErrorCode.InvalidPayload, '请先完成参考图提示词分析', 422)
    }
    const mode =
      sourceAssets.length + referenceAssets.length > 0
        ? 'image-to-image'
        : 'text-to-image'

    const sections: PromptSections = {
      subject: payload.source.trim(),
      scene:
        mode === 'image-to-image'
          ? '保留原图环境关系，清理分散注意力的背景细节'
          : '安静而有层次的真实场景，主体与环境关系自然',
      style: '克制、高级的当代杂志摄影，真实材质与细腻层次',
      composition: '主体清晰，画面重心稳定，保留舒适呼吸空间',
      details: '保留真实纹理，优化光影过渡、色彩统一和边缘细节',
      negative: '避免过度磨皮、塑料质感、肢体错误、文字和水印',
      output: '高清成片，细节自然，适合社交媒体与个人保存',
      referencePrompt: payload.referencePrompt,
    }
    const result: PromptVersion = {
      id: createId('prompt'),
      source: payload.source.trim(),
      sections,
    }
    db.prompts.push({
      ...result,
      ownerId: user.id,
      mode,
      sourceAssetIds: [...payload.sourceAssetIds],
      referenceAssets: payload.referenceAssets.map((item) => ({ ...item })),
    })
    writeDb(db)
    return ok(result)
  }),

  http.post('*/api/prompts/confirm', async ({ request }) => {
    await delay(260)
    const { db, user } = requireSession()
    if (!user) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    const payload = (await request.json()) as PromptVersion
    const idempotency = userIdempotency(db, request, user.id, '/api/prompts/confirm', payload)
    if (!idempotency.key) return fail(ErrorCode.InvalidPayload, '缺少 Idempotency-Key 请求头', 422)
    if (idempotency.conflict) return fail(ErrorCode.DuplicateRequest, '幂等键已用于不同的请求内容', 409)
    if (idempotency.replay) return ok(idempotency.replay)
    const stored = db.prompts.find(
      (prompt) => prompt.id === payload.id && prompt.ownerId === user.id,
    )
    if (!stored) return fail(6003, '提示词版本不存在', 404)
    stored.source = payload.source.trim()
    stored.sections = { ...payload.sections }
    stored.confirmedAt = new Date().toISOString()
    const result = {
      id: stored.id,
      source: stored.source,
      sections: stored.sections,
      confirmedAt: stored.confirmedAt,
    }
    db.idempotencyDigests[idempotency.scope] = requestDigest(payload)
    db.idempotency[idempotency.scope] = result
    writeDb(db)
    return ok(result)
  }),

  http.post('*/api/generations', async ({ request }) => {
    const { userId: sessionUserId } = requireSession()
    const limited = rateLimited(`generation:${sessionUserId ?? 'anonymous'}`, 8)
    if (limited) return limited
    await delay(620)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    if (db.aiNoticeVersions[userId] !== 'ai-processing-v2') {
      return fail(ErrorCode.AINoticeRequired, '请先确认第三方 AI 处理告知', 428)
    }

    const payload = (await request.json()) as CreateGenerationPayload
    const idempotency = userIdempotency(db, request, userId, '/api/generations', payload)
    if (!idempotency.key) return fail(ErrorCode.InvalidPayload, '缺少 Idempotency-Key 请求头', 422)
    if (idempotency.conflict) return fail(ErrorCode.DuplicateRequest, '幂等键已用于不同的请求内容', 409)
    if (idempotency.replay) return ok(idempotency.replay)
    const assets = payload.assetIds.map((assetId) =>
      db.assets.find(
        (asset) => asset.id === assetId && asset.ownerId === userId,
      ),
    )
    if (assets.some((asset) => !asset)) {
      return fail(ErrorCode.InvalidPayload, '素材不存在或不可用', 422)
    }
    const mode = assets.length > 0 ? 'image-to-image' : 'text-to-image'
    const hasPromptVersion = Boolean(payload.promptVersionId)
    const source = payload.source?.trim() ?? ''
    if (hasPromptVersion === Boolean(source)) {
      return fail(
        ErrorCode.InvalidPayload,
        '请选择已确认的提示词方案，或直接填写画面需求',
        422,
      )
    }

    let prompt = hasPromptVersion
      ? db.prompts.find(
          (item) =>
            item.id === payload.promptVersionId &&
            item.ownerId === userId &&
            item.confirmedAt,
        )
      : undefined
    if (hasPromptVersion && !prompt) {
      return fail(ErrorCode.InvalidPayload, '已确认提示词版本不存在', 404)
    }
    if (prompt && prompt.mode !== mode) {
      return fail(
        ErrorCode.InvalidPayload,
        '素材已经变化，请重新优化并确认提示词',
        422,
      )
    }
    if (!prompt) {
      if (
        source.length < PROMPT_CONFIG.minLength ||
        source.length > PROMPT_CONFIG.maxLength
      ) {
        return fail(ErrorCode.InvalidPayload, '请先填写完整的画面需求', 422)
      }
      const references = payload.referenceAssets ?? []
      const referenceAssetIds = assets
        .filter((asset) => asset?.kind === 'reference')
        .map((asset) => asset?.id)
      if (
        references.length !== referenceAssetIds.length ||
        references.some(
          (reference) =>
            !referenceAssetIds.includes(reference.assetId) ||
            !['style', 'composition', 'person', 'detail'].includes(
              reference.role,
            ),
        )
      ) {
        return fail(ErrorCode.InvalidPayload, '参考图素材或用途无效', 422)
      }
      const confirmedAt = new Date().toISOString()
      prompt = {
        id: createId('prompt'),
        ownerId: userId,
        source,
        mode,
        sourceAssetIds: assets
          .filter((asset) => asset?.kind === 'source')
          .map((asset) => asset?.id as string),
        referenceAssets: references.map((reference) => ({ ...reference })),
        sections: {
          subject: '',
          scene: '',
          style: '',
          composition: '',
          details: '',
          negative: '',
          output: '',
        },
        confirmedAt,
      }
      db.prompts.push(prompt)
    }

    const cost = payload.settings.outputCount
    const entitlement = currentEntitlement(db, userId)
    if (entitlement.balance < cost) {
      return fail(ErrorCode.InsufficientCredits, '剩余次数不足', 409, {
        required: cost,
        balance: entitlement.balance,
      })
    }

    entitlement.balance -= cost
    db.entitlements[userId] = entitlement
    const taskId = createId('task')
    addLedger(db, userId, {
      type: 'reserve',
      amount: -cost,
      balanceAfter: entitlement.balance,
      description: `任务 ${taskId} · 生成 ${cost} 张图片`,
    })

    const now = new Date().toISOString()
    const task: MockTask = {
      id: taskId,
      ownerId: userId,
      mode,
      title: prompt.source.slice(0, 20),
      status: 'queued',
      prompt: {
        id: prompt.id,
        source: prompt.source,
        sections: { ...prompt.sections },
        confirmedAt: prompt.confirmedAt,
      },
      settings: payload.settings,
      assets: assets
        .filter((asset): asset is NonNullable<typeof asset> => Boolean(asset))
        .map((asset) => {
          const result = { ...asset } as Asset & { ownerId?: string }
          delete result.ownerId
          return result
        }),
      requestedCount: cost,
      successfulCount: 0,
      reservedCredits: cost,
      spentCredits: 0,
      refundedCredits: 0,
      progress: 0,
      results: [],
      createdAt: now,
      updatedAt: now,
      scenario: 'normal',
      refundMaterialized: false,
    }
    db.tasks.unshift(task)
    const result = {
      task,
      entitlement: currentEntitlement(db, userId),
    }
    db.idempotencyDigests[idempotency.scope] = requestDigest(payload)
    db.idempotency[idempotency.scope] = result
    writeDb(db)
    return ok(result, 202)
  }),

  http.get('*/api/tasks', async ({ request }) => {
    await delay(220)
    const { db, user } = requireSession()
    if (!user) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    const url = new URL(request.url)
    const tasks = db.tasks
      .filter((task) => task.ownerId === user.id)
      .map((task) => materializeTask(db, task))
    return ok(paginate(
      tasks,
      Number(url.searchParams.get('page') ?? 1),
      Number(url.searchParams.get('pageSize') ?? 20),
    ))
  }),

  http.get('*/api/tasks/:taskId', async ({ params }) => {
    await delay(160)
    const { db, user } = requireSession()
    if (!user) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    const task = db.tasks.find(
      (item) => item.id === params.taskId && item.ownerId === user.id,
    )
    return task ? ok(materializeTask(db, task)) : fail(6004, '任务不存在', 404)
  }),

  http.post('*/api/tasks/:taskId/cancel', async ({ params, request }) => {
    await delay(220)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    const idempotency = userIdempotency(
      db,
      request,
      userId,
      `/api/tasks/${String(params.taskId)}/cancel`,
      { taskId: params.taskId },
    )
    if (!idempotency.key) {
      return fail(ErrorCode.InvalidPayload, '缺少 Idempotency-Key 请求头', 422)
    }
    if (idempotency.conflict) {
      return fail(ErrorCode.DuplicateRequest, '幂等键已用于不同的请求内容', 409)
    }
    if (idempotency.replay) return ok(idempotency.replay as GenerationTask)
    const task = db.tasks.find(
      (item) => item.id === params.taskId && item.ownerId === user.id,
    )
    if (!task) return fail(6004, '任务不存在', 404)
    materializeTask(db, task)
    if (task.status !== 'queued') {
      return fail(ErrorCode.InvalidPayload, '只有排队中的任务可以取消', 409)
    }

    task.status = 'cancelled'
    task.refundedCredits = task.reservedCredits
    task.updatedAt = new Date().toISOString()
    task.progress = 0
    const entitlement = currentEntitlement(db, userId)
    entitlement.balance += task.reservedCredits
    db.entitlements[userId] = entitlement
    addLedger(db, userId, {
      type: 'release',
      amount: task.reservedCredits,
      balanceAfter: entitlement.balance,
      description: `取消任务 ${task.id}，退回次数`,
    })
    db.idempotencyDigests[idempotency.scope] = requestDigest({ taskId: params.taskId })
    db.idempotency[idempotency.scope] = task
    writeDb(db)
    return ok(task)
  }),

  http.post('*/api/tasks/:taskId/retouch-tickets', async ({
    params,
    request,
  }) => {
    await delay(260)
    const payload = (await request.json()) as CreateRetouchTicketPayload

    return withRetouchMutation(() => {
      const { db, userId, user } = requireSession()
      if (!user || !userId) {
        return fail(ErrorCode.AuthRequired, '请先登录', 401)
      }

      const idempotency = retouchIdempotency(
        db,
        request,
        userId,
        'create',
        { taskId: params.taskId, ...payload },
      )
      if (!idempotency.key) {
        return fail(
          ErrorCode.InvalidPayload,
          '缺少 Idempotency-Key 请求头',
          400,
        )
      }
      if (idempotency.conflict) {
        return fail(ErrorCode.DuplicateRequest, '幂等键已用于不同的请求内容', 409)
      }
      if (idempotency.replay) {
        return ok(idempotency.replay as RetouchTicket, 201)
      }

      const task = db.tasks.find(
        (item) => item.id === params.taskId && item.ownerId === userId,
      )
      if (!task) {
        return fail(
          ErrorCode.RetouchTaskNotEligible,
          '生成任务不存在或不可提交人工修图',
          404,
        )
      }
      materializeTask(db, task)
      if (
        (task.status !== 'completed' && task.status !== 'partial') ||
        task.successfulCount < 1 ||
        task.results.length < 1
      ) {
        return fail(
          ErrorCode.RetouchTaskNotEligible,
          '只有存在成功成片的已结算任务可以提交人工修图',
          422,
        )
      }

      const activeTicket = db.retouchTickets.find(
        (ticket) =>
          ticket.taskId === task.id &&
          ticket.ownerId === userId &&
          ticket.status !== 'rejected' &&
          ticket.status !== 'cancelled',
      )
      if (activeTicket) {
        return fail(
          ErrorCode.RetouchTicketAlreadyExists,
          '该生成任务已有关联的人工修图工单',
          409,
          {
            ticketId: activeTicket.id,
            ticketNo: activeTicket.ticketNo,
            status: activeTicket.status,
          },
        )
      }

      const selectedResultIds = payload.selectedResultIds
      const supplementalAssetIds = payload.supplementalAssetIds
      const requirement = payload.requirement?.trim()
      if (
        !Array.isArray(selectedResultIds) ||
        selectedResultIds.length < 1 ||
        selectedResultIds.length > RETOUCH_TICKET_CONFIG.maxSelectedResults ||
        new Set(selectedResultIds).size !== selectedResultIds.length ||
        !requirement ||
        requirement.length > RETOUCH_TICKET_CONFIG.requirementMaxLength ||
        !Array.isArray(supplementalAssetIds) ||
        supplementalAssetIds.length >
          RETOUCH_TICKET_CONFIG.maxSupplementalAssets ||
        new Set(supplementalAssetIds).size !== supplementalAssetIds.length
      ) {
        return fail(
          ErrorCode.InvalidPayload,
          '请检查成片选择、修图要求和补充参考图',
          422,
        )
      }

      const selectedResults = selectedResultIds.map((resultId) =>
        task.results.find((result) => result.id === resultId),
      )
      if (selectedResults.some((result) => !result)) {
        return fail(
          ErrorCode.RetouchTaskNotEligible,
          '所选成片不属于当前生成任务',
          422,
        )
      }

      const supplementalAssets = supplementalAssetIds.map((assetId) =>
        db.assets.find(
          (asset) =>
            asset.id === assetId &&
            asset.ownerId === userId &&
            asset.kind === 'retouch-reference',
        ),
      )
      if (supplementalAssets.some((asset) => !asset)) {
        return fail(
          ErrorCode.InvalidPayload,
          '补充参考图不存在或不可用',
          422,
        )
      }

      const now = new Date().toISOString()
      const ticket: MockRetouchTicket = {
        id: createId('retouch_ticket'),
        ticketNo: createRetouchTicketNumber(),
        ownerId: userId,
        taskId: task.id,
        taskTitle: task.title,
        status: 'submitted',
        selectedResults: selectedResults.filter(
          (result): result is NonNullable<typeof result> => Boolean(result),
        ),
        requirement,
        supplementalAssets: supplementalAssets
          .filter(
            (asset): asset is NonNullable<typeof asset> => Boolean(asset),
          )
          .map((asset) => {
            const result = { ...asset } as Asset & { ownerId?: string }
            delete result.ownerId
            return result
          }),
        timeline: [
          {
            status: 'submitted',
            note: '人工修图需求已提交',
            createdAt: now,
          },
        ],
        reservedCredits: 0,
        spentCredits: 0,
        refundedCredits: 0,
        deliverables: [],
        sla: {
          stage: 'quote',
          dueAt: new Date(Date.parse(now) + 24 * 60 * 60_000).toISOString(),
          overdue: false,
          remainingSeconds: 24 * 60 * 60,
        },
        createdAt: now,
        updatedAt: now,
      }
      db.retouchTickets.unshift(ticket)
      syncTaskRetouchTicket(db, task.id)
      const result = publicRetouchTicket(ticket)
      db.idempotencyDigests[idempotency.key] = idempotency.digest
      db.idempotency[idempotency.key] = result
      writeDb(db)
      return ok(result, 201)
    })
  }),

  http.get('*/api/retouch-tickets', async ({ request }) => {
    await delay(220)
    const { db, user } = requireSession()
    if (!user) return fail(ErrorCode.AuthRequired, '请先登录', 401)

    const tickets = db.retouchTickets
      .filter((ticket) => ticket.ownerId === user.id)
      .map((ticket) => materializeRetouchTicket(db, ticket))
      .sort(
        (left, right) =>
          new Date(right.updatedAt).getTime() -
          new Date(left.updatedAt).getTime(),
      )
      .map(publicRetouchTicket)
    const url = new URL(request.url)
    return ok(paginate(
      tickets,
      Number(url.searchParams.get('page') ?? 1),
      Number(url.searchParams.get('pageSize') ?? 20),
    ))
  }),

  http.get('*/api/retouch-tickets/:ticketId', async ({ params }) => {
    await delay(160)
    const { db, user } = requireSession()
    if (!user) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    const ticket = db.retouchTickets.find(
      (item) => item.id === params.ticketId && item.ownerId === user.id,
    )
    if (!ticket) {
      return fail(ErrorCode.RetouchTicketNotFound, '人工修图工单不存在', 404)
    }
    return ok(publicRetouchTicket(materializeRetouchTicket(db, ticket)))
  }),

  http.post('*/api/retouch-tickets/:ticketId/quote/accept', async ({
    params,
    request,
  }) => {
    await delay(220)
    const payload = (await request.json()) as { quoteId?: string }

    return withRetouchMutation(() => {
      const { db, userId, user } = requireSession()
      if (!user || !userId) {
        return fail(ErrorCode.AuthRequired, '请先登录', 401)
      }
      const idempotency = retouchIdempotency(
        db,
        request,
        userId,
        `accept:${String(params.ticketId)}`,
        { ticketId: params.ticketId, quoteId: payload.quoteId },
      )
      if (!idempotency.key) {
        return fail(
          ErrorCode.InvalidPayload,
          '缺少 Idempotency-Key 请求头',
          400,
        )
      }
      if (idempotency.conflict) {
        return fail(ErrorCode.DuplicateRequest, '幂等键已用于不同的请求内容', 409)
      }
      if (idempotency.replay) {
        return ok(
          idempotency.replay as RetouchTicketBalanceResult,
        )
      }

      const ticket = db.retouchTickets.find(
        (item) => item.id === params.ticketId && item.ownerId === userId,
      )
      if (!ticket) {
        return fail(ErrorCode.RetouchTicketNotFound, '人工修图工单不存在', 404)
      }
      materializeRetouchTicket(db, ticket)
      if (ticket.status !== 'quote_pending') {
        return fail(
          ErrorCode.RetouchInvalidStatus,
          '当前工单状态不能接受报价',
          409,
        )
      }
      if (!ticket.quote || ticket.quote.id !== payload.quoteId) {
        return fail(
          ErrorCode.RetouchQuoteInvalid,
          '报价不存在或已失效',
          409,
        )
      }
      if (ticket.quote.status !== 'active' || Date.parse(ticket.quote.expiresAt) <= Date.now()) {
        ticket.quote.status = 'expired'
        ticket.quote.remainingSeconds = 0
        writeDb(db)
        return fail(ErrorCode.RetouchQuoteInvalid, '报价已过期，请等待管理员重新报价', 409)
      }

      const entitlement = currentEntitlement(db, userId)
      if (entitlement.balance < ticket.quote.credits) {
        return fail(
          ErrorCode.InsufficientCredits,
          '剩余次数不足，无法接受人工修图报价',
          409,
          {
            required: ticket.quote.credits,
            balance: entitlement.balance,
          },
        )
      }

      entitlement.balance -= ticket.quote.credits
      entitlement.canCreate = entitlement.balance > 0
      entitlement.status = entitlement.balance > 0 ? 'active' : 'empty'
      db.entitlements[userId] = entitlement
      ticket.quote.status = 'accepted'
      ticket.quote.remainingSeconds = 0
      ticket.reservedCredits = ticket.quote.credits
      ticket.acceptedAt = new Date().toISOString()
      transitionRetouchTicket(
        ticket,
        'accepted',
        `已接受报价并预占 ${ticket.quote.credits} 次`,
        ticket.acceptedAt,
      )
      addLedger(db, userId, {
        type: 'reserve',
        amount: -ticket.quote.credits,
        balanceAfter: entitlement.balance,
        description: `人工修图工单 ${ticket.ticketNo} 预占次数`,
      })
      syncTaskRetouchTicket(db, ticket.taskId)

      const result: RetouchTicketBalanceResult = {
        ticket: publicRetouchTicket(ticket),
        entitlement,
      }
      db.idempotencyDigests[idempotency.key] = idempotency.digest
      db.idempotency[idempotency.key] = result
      writeDb(db)
      return ok(result)
    })
  }),

  http.post('*/api/retouch-tickets/:ticketId/cancel', async ({
    params,
    request,
  }) => {
    await delay(180)

    return withRetouchMutation(() => {
      const { db, userId, user } = requireSession()
      if (!user || !userId) {
        return fail(ErrorCode.AuthRequired, '请先登录', 401)
      }
      const idempotency = retouchIdempotency(
        db,
        request,
        userId,
        `cancel:${String(params.ticketId)}`,
        { ticketId: params.ticketId },
      )
      if (!idempotency.key) {
        return fail(
          ErrorCode.InvalidPayload,
          '缺少 Idempotency-Key 请求头',
          400,
        )
      }
      if (idempotency.conflict) {
        return fail(ErrorCode.DuplicateRequest, '幂等键已用于不同的请求内容', 409)
      }
      if (idempotency.replay) {
        return ok(
          idempotency.replay as RetouchTicketBalanceResult,
        )
      }

      const ticket = db.retouchTickets.find(
        (item) => item.id === params.ticketId && item.ownerId === userId,
      )
      if (!ticket) {
        return fail(ErrorCode.RetouchTicketNotFound, '人工修图工单不存在', 404)
      }
      materializeRetouchTicket(db, ticket)
      if (
        ticket.status !== 'submitted' &&
        ticket.status !== 'quote_pending' &&
        ticket.status !== 'accepted'
      ) {
        return fail(
          ErrorCode.RetouchInvalidStatus,
          '当前工单状态不能取消',
          409,
        )
      }

      const entitlement = currentEntitlement(db, userId)
      if (ticket.status === 'accepted' && ticket.reservedCredits > 0) {
        entitlement.balance += ticket.reservedCredits
        entitlement.canCreate = true
        entitlement.status = 'active'
        db.entitlements[userId] = entitlement
        ticket.refundedCredits = ticket.reservedCredits
        addLedger(db, userId, {
          type: 'release',
          amount: ticket.reservedCredits,
          balanceAfter: entitlement.balance,
          description: `取消人工修图工单 ${ticket.ticketNo}，退回次数`,
        })
      }

      transitionRetouchTicket(ticket, 'cancelled', '用户已取消工单')
      syncTaskRetouchTicket(db, ticket.taskId)
      const result: RetouchTicketBalanceResult = {
        ticket: publicRetouchTicket(ticket),
        entitlement,
      }
      db.idempotencyDigests[idempotency.key] = idempotency.digest
      db.idempotency[idempotency.key] = result
      writeDb(db)
      return ok(result)
    })
  }),

  http.post('*/api/retouch-tickets/:ticketId/confirm', async ({
    params,
    request,
  }) => {
    await delay(180)

    return withRetouchMutation(() => {
      const { db, userId, user } = requireSession()
      if (!user || !userId) {
        return fail(ErrorCode.AuthRequired, '请先登录', 401)
      }
      const idempotency = retouchIdempotency(
        db,
        request,
        userId,
        `confirm:${String(params.ticketId)}`,
        { ticketId: params.ticketId },
      )
      if (!idempotency.key) {
        return fail(
          ErrorCode.InvalidPayload,
          '缺少 Idempotency-Key 请求头',
          400,
        )
      }
      if (idempotency.conflict) {
        return fail(ErrorCode.DuplicateRequest, '幂等键已用于不同的请求内容', 409)
      }
      if (idempotency.replay) {
        return ok(idempotency.replay as RetouchTicket)
      }

      const ticket = db.retouchTickets.find(
        (item) => item.id === params.ticketId && item.ownerId === userId,
      )
      if (!ticket) {
        return fail(ErrorCode.RetouchTicketNotFound, '人工修图工单不存在', 404)
      }
      materializeRetouchTicket(db, ticket)
      if (ticket.status !== 'awaiting_confirmation') {
        return fail(
          ErrorCode.RetouchInvalidStatus,
          '只有待确认的工单可以确认交付',
          409,
        )
      }

      transitionRetouchTicket(ticket, 'delivered', '用户已确认人工成片')
      syncTaskRetouchTicket(db, ticket.taskId)
      const result = publicRetouchTicket(ticket)
      db.idempotencyDigests[idempotency.key] = idempotency.digest
      db.idempotency[idempotency.key] = result
      writeDb(db)
      return ok(result)
    })
  }),

  http.post('*/api/retouch-tickets/:ticketId/revisions', async ({
    params,
    request,
  }) => {
    await delay(180)
    const payload = (await request.json()) as { message?: string }

    return withRetouchMutation(() => {
      const { db, userId, user } = requireSession()
      if (!user || !userId) {
        return fail(ErrorCode.AuthRequired, '请先登录', 401)
      }
      const idempotency = retouchIdempotency(
        db,
        request,
        userId,
        `revision:${String(params.ticketId)}`,
        { ticketId: params.ticketId, message: payload.message },
      )
      if (!idempotency.key) {
        return fail(
          ErrorCode.InvalidPayload,
          '缺少 Idempotency-Key 请求头',
          400,
        )
      }
      if (idempotency.conflict) {
        return fail(ErrorCode.DuplicateRequest, '幂等键已用于不同的请求内容', 409)
      }
      if (idempotency.replay) {
        return ok(idempotency.replay as RetouchTicket)
      }

      const ticket = db.retouchTickets.find(
        (item) => item.id === params.ticketId && item.ownerId === userId,
      )
      if (!ticket) {
        return fail(ErrorCode.RetouchTicketNotFound, '人工修图工单不存在', 404)
      }
      materializeRetouchTicket(db, ticket)
      if (ticket.status !== 'awaiting_confirmation') {
        return fail(
          ErrorCode.RetouchInvalidStatus,
          '只有待确认的工单可以申请返修',
          409,
        )
      }
      if (ticket.revision) {
        return fail(
          ErrorCode.RetouchRevisionLimitReached,
          '本工单的一次返修机会已使用',
          409,
        )
      }

      const message = payload.message?.trim()
      if (
        !message ||
        message.length > RETOUCH_TICKET_CONFIG.revisionMaxLength
      ) {
        return fail(ErrorCode.InvalidPayload, '请填写有效的返修要求', 422)
      }

      const requestedAt = new Date().toISOString()
      ticket.revision = { message, requestedAt }
      ticket.deliverables = []
      transitionRetouchTicket(
        ticket,
        'processing',
        '用户已提交一次返修要求',
        requestedAt,
      )
      syncTaskRetouchTicket(db, ticket.taskId)
      const result = publicRetouchTicket(ticket)
      db.idempotencyDigests[idempotency.key] = idempotency.digest
      db.idempotency[idempotency.key] = result
      writeDb(db)
      return ok(result)
    })
  }),
]
