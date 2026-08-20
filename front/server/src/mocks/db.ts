import { MOCK_CONFIG } from '@/config'
import { createSeedDb } from '@/mocks/seed'
import type {
  MockAdmin,
  MockDb,
  MockModel,
  MockProvider,
  MockRedemptionBatch,
  MockRedemptionCode,
  MockTask,
  MockTicket,
  MockUser,
} from '@/mocks/schema'
import type {
  AIModel,
  AIProvider,
  AdminSession,
  AuditEvent,
  ManageRetouchTicket,
  ManageRetouchTicketSummary,
  ManagedGenerationTask,
  ManagedGenerationTaskSummary,
  ManagedUser,
  ManagedUserDetail,
  RedemptionBatch,
  RedemptionCode,
  RedemptionCodeDetail,
  RetouchSLA,
} from '@/types/domain'
import { createId } from '@/utils/id'
import {
  deriveRedemptionStatus,
  isExpiringSoon,
  maskRedemptionCode,
} from '@/utils/redemption'

export function readDb(): MockDb {
  const raw = localStorage.getItem(MOCK_CONFIG.databaseKey)
  if (raw) {
    try {
      const value = JSON.parse(raw) as Partial<MockDb>
      return {
        ...createSeedDb(),
        ...value,
        idempotency: value.idempotency ?? {},
      }
    } catch {
      localStorage.removeItem(MOCK_CONFIG.databaseKey)
    }
  }
  const seed = createSeedDb()
  writeDb(seed)
  return seed
}

export function writeDb(db: MockDb): void {
  localStorage.setItem(MOCK_CONFIG.databaseKey, JSON.stringify(db))
}

export function resetDb(): void {
  localStorage.removeItem(MOCK_CONFIG.databaseKey)
  localStorage.removeItem(MOCK_CONFIG.rememberedSessionKey)
  sessionStorage.removeItem(MOCK_CONFIG.tabSessionKey)
}

export function getSessionAdminId(): string | null {
  return (
    sessionStorage.getItem(MOCK_CONFIG.tabSessionKey) ??
    localStorage.getItem(MOCK_CONFIG.rememberedSessionKey)
  )
}

export function setSessionAdminId(adminId: string, remember = false): void {
  clearSession()
  ;(remember ? localStorage : sessionStorage).setItem(
    remember
      ? MOCK_CONFIG.rememberedSessionKey
      : MOCK_CONFIG.tabSessionKey,
    adminId,
  )
}

export function clearSession(): void {
  localStorage.removeItem(MOCK_CONFIG.rememberedSessionKey)
  sessionStorage.removeItem(MOCK_CONFIG.tabSessionKey)
}

export function publicAdmin(admin: MockAdmin): AdminSession {
  return {
    id: admin.id,
    email: admin.email,
    name: admin.name,
    role: admin.role,
    permissions: admin.permissions,
    status: admin.status,
    csrfToken: `csrf_${admin.id}`,
    createdAt: admin.createdAt,
  }
}

export function publicProvider(provider: MockProvider): AIProvider {
  return {
    id: provider.id,
    name: provider.name,
    code: provider.code,
    protocol: provider.protocol,
    baseUrl: provider.baseUrl,
    maskedApiKey: provider.maskedApiKey,
    enabled: provider.enabled,
    connectionStatus: provider.connectionStatus,
    lastTest: provider.lastTest,
    note: provider.note,
    createdAt: provider.createdAt,
    updatedAt: provider.updatedAt,
  }
}

export function publicModel(model: MockModel, db: MockDb): AIModel {
  const provider = db.providers.find((item) => item.id === model.providerId)
  const bindingId =
    model.type === 'chat'
      ? db.bindings.chatModelId
      : db.bindings.imageModelId
  return {
    ...model,
    providerName: provider?.name ?? '未知服务商',
    isPlatformModel: bindingId === model.id,
  }
}

export function publicBatch(
  batch: MockRedemptionBatch,
  db: MockDb,
): RedemptionBatch {
  const codes = db.codes.filter((item) => item.batchId === batch.id)
  const counts: RedemptionBatch['counts'] = {
    unused: 0,
    redeemed: 0,
    expired: 0,
    disabled: 0,
  }
  codes.forEach((item) => {
    counts[deriveRedemptionStatus(item)] += 1
  })
  return {
    ...batch,
    quantity: codes.length,
    counts,
    usageRate: codes.length ? counts.redeemed / codes.length : 0,
  }
}

export function publicCode(
  code: MockRedemptionCode,
  db: MockDb,
): RedemptionCode {
  const batch = db.batches.find((item) => item.id === code.batchId)
  const user = db.users.find((item) => item.id === code.redeemedBy)
  return {
    id: code.id,
    maskedCode: maskRedemptionCode(code.fullCode),
    batchId: code.batchId,
    batchName: batch?.name ?? '未知批次',
    productCode: code.productCode,
    credits: code.credits,
    status: deriveRedemptionStatus(code),
    expiresAt: code.expiresAt,
    redeemedBy: code.redeemedBy,
    redeemedByEmail: user?.email,
    redeemedAt: code.redeemedAt,
    disabledBy: code.disabledBy,
    disabledAt: code.disabledAt,
    disabledReason: code.disabledReason,
    createdAt: code.createdAt,
    expiringSoon: isExpiringSoon(code),
  }
}

export function publicCodeDetail(
  code: MockRedemptionCode,
  db: MockDb,
): RedemptionCodeDetail {
  return {
    ...publicCode(code, db),
    operationHistory: code.operationHistory,
  }
}

export function publicUser(user: MockUser): ManagedUser {
  return {
    id: user.id,
    email: user.email,
    status: user.status,
    balance: user.balance,
    totalRedeemed: user.totalRedeemed,
    totalConsumed: user.totalConsumed,
    taskCount: user.taskCount,
    ticketCount: user.ticketCount,
    lastLoginAt: user.lastLoginAt,
    createdAt: user.createdAt,
    disabledReason: user.disabledReason,
  }
}

export function publicTaskSummary(
  task: MockTask,
  db: MockDb,
): ManagedGenerationTaskSummary {
  const user = db.users.find((item) => item.id === task.ownerId)
  const provider = db.providers.find(
    (item) => item.id === task.executionSnapshot.providerId,
  )
  const model = db.models.find(
    (item) => item.id === task.executionSnapshot.modelId,
  )
  return {
    id: task.id,
    ownerId: task.ownerId,
    ownerEmail: user?.email ?? '未知用户',
    title: task.title,
    mode: task.mode,
    status: task.status,
    progress: task.progress,
    requestedCount: task.requestedCount,
    successfulCount: task.successfulCount,
    reservedCredits: task.reservedCredits,
    spentCredits: task.spentCredits,
    refundedCredits: task.refundedCredits,
    providerName: provider?.name ?? task.executionSnapshot.providerName,
    modelName: model?.displayName ?? task.executionSnapshot.modelName,
    hasRetouchTicket: db.tickets.some((item) => item.taskId === task.id),
    createdAt: task.createdAt,
    updatedAt: task.updatedAt,
  }
}

export function publicTask(task: MockTask, db: MockDb): ManagedGenerationTask {
  const ticket = db.tickets.find((item) => item.taskId === task.id)
  return {
    ...publicTaskSummary(task, db),
    sourceRequirement: task.sourceRequirement,
    optimizedPrompt: task.optimizedPrompt,
    confirmedPrompt: task.confirmedPrompt,
    settings: task.settings,
    assets: task.assetIds
      .map((id) => db.assets.find((item) => item.id === id))
      .filter((item) => item !== undefined),
    results: task.resultAssetIds
      .map((id) => db.assets.find((item) => item.id === id))
      .filter((item) => item !== undefined),
    executionSnapshot: task.executionSnapshot,
    providerAttempts: task.providerAttempts ?? [],
    errorMessage: task.errorMessage,
    retouchTicket: ticket ? publicTicketSummary(ticket, db) : undefined,
  }
}

export function publicTicketSummary(
  ticket: MockTicket,
  db: MockDb,
): ManageRetouchTicketSummary {
  materializeRetouchTiming(ticket)
  const user = db.users.find((item) => item.id === ticket.userId)
  return {
    id: ticket.id,
    ticketNo: ticket.ticketNo,
    taskId: ticket.taskId,
    taskTitle: ticket.taskTitle,
    status: ticket.status,
    quoteCredits: ticket.quote?.credits,
    sla: ticket.sla,
    user: {
      id: ticket.userId,
      email: user?.email ?? '未知用户',
      status: user?.status ?? 'disabled',
    },
    createdAt: ticket.createdAt,
    updatedAt: ticket.updatedAt,
  }
}

export function publicTicket(
  ticket: MockTicket,
  db: MockDb,
): ManageRetouchTicket {
  materializeRetouchTiming(ticket)
  const task = db.tasks.find((item) => item.id === ticket.taskId)
  if (!task) throw new Error('来源任务不存在')
  return {
    ...publicTicketSummary(ticket, db),
    selectedResults: ticket.selectedResults,
    requirement: ticket.requirement,
    supplementalAssets: ticket.supplementalAssetIds
      .map((id) => db.assets.find((item) => item.id === id))
      .filter((item) => item !== undefined),
    quote: ticket.quote,
    timeline: ticket.timeline,
    reservedCredits: ticket.reservedCredits,
    spentCredits: ticket.spentCredits,
    refundedCredits: ticket.refundedCredits,
    revision: ticket.revision,
    deliverables: ticket.deliverables,
    sourceTaskDetail: publicTask(task, db),
  }
}

function materializeRetouchTiming(ticket: MockTicket): void {
  const now = Date.now()
  if (ticket.quote?.status === 'active' && Date.parse(ticket.quote.expiresAt) <= now) {
    ticket.quote.status = 'expired'
    ticket.quote.remainingSeconds = 0
  }
  if (ticket.quote) {
    ticket.quote.remainingSeconds = ticket.quote.status === 'active'
      ? Math.max(0, Math.floor((Date.parse(ticket.quote.expiresAt) - now) / 1000))
      : 0
  }

  let stage: RetouchSLA['stage'] = 'completed'
  let dueAt: string | null = null
  if (ticket.status === 'submitted' || ticket.status === 'quote_pending') {
    stage = 'quote'
    dueAt = new Date(Date.parse(ticket.createdAt) + 24 * 60 * 60_000).toISOString()
  } else if (ticket.status === 'accepted' || ticket.status === 'processing' || ticket.status === 'awaiting_confirmation') {
    if (ticket.revision) {
      stage = 'revision'
      dueAt = new Date(Date.parse(ticket.revision.requestedAt) + 24 * 60 * 60_000).toISOString()
    } else {
      stage = 'first-delivery'
      dueAt = new Date(Date.parse(ticket.acceptedAt ?? ticket.updatedAt) + 48 * 60 * 60_000).toISOString()
    }
  }
  const dueMs = dueAt ? Date.parse(dueAt) : 0
  const remaining = dueMs ? Math.max(0, Math.floor((dueMs - now) / 1000)) : null
  ticket.sla = {
    stage,
    dueAt,
    overdue: Boolean(dueMs && dueMs <= now),
    remainingSeconds: remaining,
  }
}

export function publicUserDetail(
  user: MockUser,
  db: MockDb,
): ManagedUserDetail {
  return {
    ...publicUser(user),
    ledger: db.ledger
      .filter((item) => item.userId === user.id)
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt)),
    redemptionCodes: db.codes
      .filter((item) => item.redeemedBy === user.id)
      .map((item) => publicCode(item, db)),
    tasks: db.tasks
      .filter((item) => item.ownerId === user.id)
      .map((item) => publicTaskSummary(item, db)),
    tickets: db.tickets
      .filter((item) => item.userId === user.id)
      .map((item) => publicTicketSummary(item, db)),
  }
}

function sanitizeSnapshot(
  value: Record<string, unknown> | undefined,
): Record<string, unknown> | undefined {
  if (!value) return undefined
  const blocked = /(api.?key|password|full.?code|signed.?url|preview.?url)/i
  return Object.fromEntries(
    Object.entries(value).map(([key, entry]) => [
      key,
      blocked.test(key) ? '[REDACTED]' : entry,
    ]),
  )
}

export function appendAudit(
  db: MockDb,
  admin: MockAdmin,
  input: Omit<
    AuditEvent,
    | 'id'
    | 'operatorId'
    | 'operatorEmail'
    | 'operatorRole'
    | 'createdAt'
  >,
): AuditEvent {
  const event: AuditEvent = {
    ...input,
    id: createId('audit'),
    operatorId: admin.id,
    operatorEmail: admin.email,
    operatorRole: admin.role,
    before: sanitizeSnapshot(input.before),
    after: sanitizeSnapshot(input.after),
    createdAt: new Date().toISOString(),
  }
  db.audits.unshift(event)
  return event
}
