import { delay, http, HttpResponse } from 'msw'
import type {
  Asset,
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

let retouchMutationQueue: Promise<void> = Promise.resolve()

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

  http.get('*/api/usage/ledger', async () => {
    await delay(220)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    return ok(db.ledger[userId] ?? [])
  }),

  http.post('*/api/redemptions/claim', async ({ request }) => {
    await delay(520)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)

    const idempotencyKey = request.headers.get('Idempotency-Key')
    if (idempotencyKey && db.idempotency[idempotencyKey]) {
      return ok(db.idempotency[idempotencyKey])
    }

    const payload = (await request.json()) as { code: string }
    const normalized = payload.code.trim().toUpperCase()
    const redemption = db.codes.find((item) => item.code === normalized)
    if (!redemption) return fail(ErrorCode.CodeInvalid, '兑换码无效', 404)
    if (redemption.redeemedBy) {
      return fail(ErrorCode.CodeUsed, '兑换码已经使用', 409)
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
      description: `兑换 ${redemption.code}`,
    })
    const result = { added: redemption.credits, entitlement }
    if (idempotencyKey) db.idempotency[idempotencyKey] = result
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
    const asset: Asset = {
      id: createId('asset'),
      name: file.name,
      kind,
      role:
        typeof formData.get('role') === 'string'
          ? (formData.get('role') as Asset['role'])
          : undefined,
      mimeType: file.type,
      size: file.size,
      uploadProgress: 100,
    }
    db.assets.push({ ...asset, ownerId: userId })
    writeDb(db)
    return ok(asset, 201)
  }),

  http.delete('*/api/assets/:assetId', async ({ params }) => {
    await delay(100)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    db.assets = db.assets.filter(
      (asset) => asset.id !== params.assetId || asset.ownerId !== userId,
    )
    writeDb(db)
    return ok(null)
  }),

  http.post('*/api/prompts/optimize', async ({ request }) => {
    await delay(820)
    const { db, user } = requireSession()
    if (!user) return fail(ErrorCode.AuthRequired, '请先登录', 401)
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
    const stored = db.prompts.find(
      (prompt) => prompt.id === payload.id && prompt.ownerId === user.id,
    )
    if (!stored) return fail(6003, '提示词版本不存在', 404)
    stored.source = payload.source.trim()
    stored.sections = { ...payload.sections }
    stored.confirmedAt = new Date().toISOString()
    writeDb(db)
    return ok({
      id: stored.id,
      source: stored.source,
      sections: stored.sections,
      confirmedAt: stored.confirmedAt,
    })
  }),

  http.post('*/api/generations', async ({ request }) => {
    await delay(620)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)

    const idempotencyKey = request.headers.get('Idempotency-Key')
    if (idempotencyKey && db.idempotency[idempotencyKey]) {
      return ok(db.idempotency[idempotencyKey])
    }

    const payload = (await request.json()) as CreateGenerationPayload
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
    if (idempotencyKey) db.idempotency[idempotencyKey] = task
    writeDb(db)
    return ok(task, 202)
  }),

  http.get('*/api/tasks', async () => {
    await delay(220)
    const { db, user } = requireSession()
    if (!user) return fail(ErrorCode.AuthRequired, '请先登录', 401)
    return ok(
      db.tasks
        .filter((task) => task.ownerId === user.id)
        .map((task) => materializeTask(db, task)),
    )
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

  http.post('*/api/tasks/:taskId/cancel', async ({ params }) => {
    await delay(220)
    const { db, userId, user } = requireSession()
    if (!user || !userId) return fail(ErrorCode.AuthRequired, '请先登录', 401)
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

      const idempotencyKey = retouchIdempotencyKey(
        request,
        userId,
        'create',
      )
      if (!idempotencyKey) {
        return fail(
          ErrorCode.InvalidPayload,
          '缺少 Idempotency-Key 请求头',
          400,
        )
      }
      if (db.idempotency[idempotencyKey]) {
        return ok(db.idempotency[idempotencyKey] as RetouchTicket, 201)
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
        createdAt: now,
        updatedAt: now,
      }
      db.retouchTickets.unshift(ticket)
      syncTaskRetouchTicket(db, task.id)
      const result = publicRetouchTicket(ticket)
      db.idempotency[idempotencyKey] = result
      writeDb(db)
      return ok(result, 201)
    })
  }),

  http.get('*/api/retouch-tickets', async () => {
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
    return ok(tickets)
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
      const idempotencyKey = retouchIdempotencyKey(
        request,
        userId,
        `accept:${String(params.ticketId)}`,
      )
      if (!idempotencyKey) {
        return fail(
          ErrorCode.InvalidPayload,
          '缺少 Idempotency-Key 请求头',
          400,
        )
      }
      if (db.idempotency[idempotencyKey]) {
        return ok(
          db.idempotency[idempotencyKey] as RetouchTicketBalanceResult,
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
      db.idempotency[idempotencyKey] = result
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
      const idempotencyKey = retouchIdempotencyKey(
        request,
        userId,
        `cancel:${String(params.ticketId)}`,
      )
      if (!idempotencyKey) {
        return fail(
          ErrorCode.InvalidPayload,
          '缺少 Idempotency-Key 请求头',
          400,
        )
      }
      if (db.idempotency[idempotencyKey]) {
        return ok(
          db.idempotency[idempotencyKey] as RetouchTicketBalanceResult,
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
      db.idempotency[idempotencyKey] = result
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
      const idempotencyKey = retouchIdempotencyKey(
        request,
        userId,
        `confirm:${String(params.ticketId)}`,
      )
      if (!idempotencyKey) {
        return fail(
          ErrorCode.InvalidPayload,
          '缺少 Idempotency-Key 请求头',
          400,
        )
      }
      if (db.idempotency[idempotencyKey]) {
        return ok(db.idempotency[idempotencyKey] as RetouchTicket)
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
      db.idempotency[idempotencyKey] = result
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
      const idempotencyKey = retouchIdempotencyKey(
        request,
        userId,
        `revision:${String(params.ticketId)}`,
      )
      if (!idempotencyKey) {
        return fail(
          ErrorCode.InvalidPayload,
          '缺少 Idempotency-Key 请求头',
          400,
        )
      }
      if (db.idempotency[idempotencyKey]) {
        return ok(db.idempotency[idempotencyKey] as RetouchTicket)
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
      db.idempotency[idempotencyKey] = result
      writeDb(db)
      return ok(result)
    })
  }),
]
