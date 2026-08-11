import type {
  Asset,
  Entitlement,
  GenerationTask,
  LedgerEntry,
  PromptVersion,
  PromptReferenceAsset,
  RetouchTicket,
  RetouchTicketStatus,
  TaskStatus,
  User,
} from '@/types/domain'
import {
  DEMO_ACCOUNT,
  isFinalTaskStatus,
  MOCK_REDEMPTION_CODES,
  RETOUCH_TICKET_CONFIG,
  RETOUCH_TICKET_TIMING,
} from '@/config'
import { createId } from '@/utils/id'

interface MockUser extends User {
  password: string
}

interface MockCode {
  code: string
  credits: number
  expiresAt: string
  redeemedBy?: string
  redeemedAt?: string
}

interface MockTask extends Omit<GenerationTask, 'status'> {
  status: TaskStatus | 'failed'
  ownerId: string
  scenario: 'normal' | 'partial' | 'failure'
  refundMaterialized: boolean
}

interface MockAsset extends Asset {
  ownerId: string
}

interface MockPromptVersion extends PromptVersion {
  ownerId: string
  mode: 'text-to-image' | 'image-to-image'
  sourceAssetIds: string[]
  referenceAssets: PromptReferenceAsset[]
}

interface MockRetouchTicket extends RetouchTicket {
  ownerId: string
  acceptedAt?: string
}

interface MockDb {
  users: MockUser[]
  entitlements: Record<string, Entitlement>
  ledger: Record<string, LedgerEntry[]>
  codes: MockCode[]
  tasks: MockTask[]
  assets: MockAsset[]
  prompts: MockPromptVersion[]
  retouchTickets: MockRetouchTicket[]
  idempotency: Record<string, unknown>
}

const DB_KEY = 'yingyan:mock-db:v1'
const REMEMBERED_SESSION_KEY = 'yingyan:mock-session'
const TAB_SESSION_KEY = 'yingyan:mock-tab-session'

const defaultPrompt: PromptVersion = {
  id: 'prompt_demo',
  source: '将人物调整为自然通透的杂志人像，保留真实皮肤质感。',
  confirmedAt: new Date(Date.now() - 60_000).toISOString(),
  sections: {
    subject: '成年女性半身人像，保持五官比例和人物身份',
    scene: '简洁的灰色摄影棚背景',
    style: '自然、高级的时尚杂志人像',
    composition: '居中半身构图，视线自然',
    details: '保留皮肤纹理，均匀肤色，整理碎发',
    negative: '避免过度磨皮、塑料质感和五官变形',
    output: '3:4 竖图，细节清晰',
  },
}

function createSeed(): MockDb {
  const userId = 'user_demo'
  const now = Date.now()
  const demoUser: MockUser = {
    id: userId,
    email: DEMO_ACCOUNT.email,
    password: DEMO_ACCOUNT.password,
    createdAt: new Date(now - 86_400_000 * 12).toISOString(),
    status: 'active',
  }

  const completedTask: MockTask = {
    id: 'task_demo_completed',
    ownerId: userId,
    mode: 'image-to-image',
    title: '自然光人像精修',
    status: 'completed',
    prompt: defaultPrompt,
    settings: {
      aspectRatio: '3:4',
      outputCount: 2,
      referenceStrength: 68,
    },
    assets: [
      {
        id: 'asset_demo_source',
        name: '人物原图.jpg',
        kind: 'source',
        mimeType: 'image/jpeg',
        size: 282_400,
        previewUrl: '/demo/source-portrait.jpg',
        uploadProgress: 100,
      },
      {
        id: 'asset_demo_ref',
        name: '海岸穿搭参考.jpg',
        kind: 'reference',
        role: 'style',
        mimeType: 'image/jpeg',
        size: 192_200,
        previewUrl: '/demo/style-coast.jpg',
        uploadProgress: 100,
      },
    ],
    requestedCount: 2,
    successfulCount: 2,
    reservedCredits: 2,
    spentCredits: 2,
    refundedCredits: 0,
    progress: 100,
    results: [
      {
        id: 'result_demo_1',
        url: '/demo/result-blue.jpg',
        width: 1200,
        height: 1800,
      },
      {
        id: 'result_demo_2',
        url: '/demo/auth-studio.jpg',
        width: 1800,
        height: 2700,
      },
    ],
    createdAt: new Date(now - 7_200_000).toISOString(),
    updatedAt: new Date(now - 7_190_000).toISOString(),
    scenario: 'normal',
    refundMaterialized: true,
  }

  const partialTask: MockTask = {
    ...completedTask,
    id: 'task_demo_partial',
    title: '旅行照片风格统一',
    status: 'partial',
    requestedCount: 3,
    successfulCount: 2,
    reservedCredits: 3,
    spentCredits: 2,
    refundedCredits: 1,
    settings: { ...completedTask.settings, outputCount: 3 },
    createdAt: new Date(now - 86_400_000).toISOString(),
    updatedAt: new Date(now - 86_390_000).toISOString(),
    scenario: 'partial',
  }

  return {
    users: [demoUser],
    entitlements: {
      [userId]: { balance: 8, canCreate: true, status: 'active' },
    },
    ledger: {
      [userId]: [
        {
          id: 'ledger_seed',
          type: 'redemption',
          amount: 10,
          balanceAfter: 10,
          description: '兑换 YINGYAN-START-10',
          createdAt: new Date(now - 86_400_000 * 10).toISOString(),
        },
        {
          id: 'ledger_spend',
          type: 'reserve',
          amount: -2,
          balanceAfter: 8,
          description: '任务预占 2 次',
          createdAt: completedTask.createdAt,
        },
      ],
    },
    codes: MOCK_REDEMPTION_CODES.map((item) => ({
      code: item.code,
      credits: item.credits,
      expiresAt: new Date(
        item.state === 'expired'
          ? now - 86_400_000
          : now + 86_400_000 * 365,
      ).toISOString(),
      ...(item.state === 'used'
        ? {
            redeemedBy: 'other_user',
            redeemedAt: new Date(now - 86_400_000).toISOString(),
          }
        : {}),
    })),
    tasks: [completedTask, partialTask],
    assets: completedTask.assets.map((asset) => ({
      ...asset,
      ownerId: userId,
    })),
    prompts: [
      {
        ...defaultPrompt,
        ownerId: userId,
        mode: 'image-to-image',
        sourceAssetIds: ['asset_demo_source'],
        referenceAssets: [
          { assetId: 'asset_demo_ref', role: 'style' },
        ],
      },
      {
        ...defaultPrompt,
        id: 'prompt_test',
        ownerId: userId,
        mode: 'text-to-image',
        sourceAssetIds: [],
        referenceAssets: [],
      },
    ],
    retouchTickets: [],
    idempotency: {},
  }
}

export function readDb(): MockDb {
  const raw = localStorage.getItem(DB_KEY)
  if (raw) {
    const stored = JSON.parse(raw) as Partial<MockDb>
    return {
      ...(stored as MockDb),
      assets: stored.assets ?? [],
      prompts: stored.prompts ?? [],
      retouchTickets: stored.retouchTickets ?? [],
      idempotency: stored.idempotency ?? {},
    }
  }

  const seed = createSeed()
  writeDb(seed)
  return seed
}

export function writeDb(db: MockDb): void {
  localStorage.setItem(DB_KEY, JSON.stringify(db))
}

export function resetDb(): void {
  localStorage.removeItem(DB_KEY)
  localStorage.removeItem(REMEMBERED_SESSION_KEY)
  sessionStorage.removeItem(TAB_SESSION_KEY)
}

export function getSessionUserId(): string | null {
  return (
    sessionStorage.getItem(TAB_SESSION_KEY) ??
    localStorage.getItem(REMEMBERED_SESSION_KEY)
  )
}

export function setSessionUserId(userId: string, remember: boolean): void {
  clearSession()
  ;(remember ? localStorage : sessionStorage).setItem(
    remember ? REMEMBERED_SESSION_KEY : TAB_SESSION_KEY,
    userId,
  )
}

export function clearSession(): void {
  localStorage.removeItem(REMEMBERED_SESSION_KEY)
  sessionStorage.removeItem(TAB_SESSION_KEY)
}

export function publicUser(user: MockUser): User {
  return {
    id: user.id,
    email: user.email,
    createdAt: user.createdAt,
    status: user.status,
  }
}

export function currentEntitlement(db: MockDb, userId: string): Entitlement {
  const entitlement = db.entitlements[userId] ?? {
    balance: 0,
    canCreate: false,
    status: 'unredeemed',
  }
  return {
    ...entitlement,
    canCreate: entitlement.balance > 0,
    status:
      entitlement.balance > 0
        ? 'active'
        : db.ledger[userId]?.some((entry) => entry.type === 'redemption')
          ? 'empty'
          : 'unredeemed',
  }
}

export function addLedger(
  db: MockDb,
  userId: string,
  entry: Omit<LedgerEntry, 'id' | 'createdAt'>,
): void {
  const record: LedgerEntry = {
    ...entry,
    id: createId('ledger'),
    createdAt: new Date().toISOString(),
  }
  db.ledger[userId] = [record, ...(db.ledger[userId] ?? [])]
}

export function materializeTask(db: MockDb, task: MockTask): MockTask {
  materializeLatestRetouchTicket(db, task)

  if (task.status === 'failed' || isFinalTaskStatus(task.status)) {
    return task
  }

  const elapsed = Date.now() - new Date(task.createdAt).getTime()
  if (elapsed < 2_500) {
    task.status = 'queued'
    task.progress = Math.min(22, Math.round((elapsed / 2_500) * 22))
  } else if (elapsed < 7_000) {
    task.status = 'processing'
    task.progress = Math.min(92, 22 + Math.round(((elapsed - 2_500) / 4_500) * 70))
  } else {
    const failedCount =
      task.scenario === 'failure'
        ? task.requestedCount
        : task.scenario === 'partial'
          ? 1
          : 0
    const successCount = task.requestedCount - failedCount
    task.status =
      failedCount === 0
        ? 'completed'
        : successCount === 0
          ? 'failed'
          : 'partial'
    task.successfulCount = successCount
    task.spentCredits = successCount
    task.refundedCredits = failedCount
    task.progress = 100
    task.updatedAt = new Date().toISOString()
    task.results = createResults(successCount)

    if (failedCount > 0 && !task.refundMaterialized) {
      if (task.ownerId) {
        const entitlement = currentEntitlement(db, task.ownerId)
        entitlement.balance += failedCount
        db.entitlements[task.ownerId] = entitlement
        addLedger(db, task.ownerId, {
          type: 'refund',
          amount: failedCount,
          balanceAfter: entitlement.balance,
          description: `退回 ${failedCount} 张失败图片的次数`,
        })
      }
      task.refundMaterialized = true
    }
  }

  writeDb(db)
  return task
}

export function createResults(count: number) {
  const sources = [
    '/demo/result-blue.jpg',
    '/demo/auth-studio.jpg',
    '/demo/source-portrait.jpg',
    '/demo/style-coast.jpg',
  ]
  return Array.from({ length: count }, (_, index) => ({
    id: createId('result'),
    url: sources[index % sources.length],
    width: index % 2 === 0 ? 1200 : 1800,
    height: index % 2 === 0 ? 1800 : 2700,
  }))
}

export function publicRetouchTicket(
  ticket: MockRetouchTicket,
): RetouchTicket {
  const result = { ...ticket } as RetouchTicket & {
    ownerId?: string
    acceptedAt?: string
  }
  delete result.ownerId
  delete result.acceptedAt
  return result
}

export function createRetouchTicketNumber(): string {
  const date = new Date().toISOString().slice(0, 10).replaceAll('-', '')
  const suffix = createId('retouch').split('_').at(-1)?.slice(0, 6) ?? 'ticket'
  return `YY${date}-${suffix.toUpperCase()}`
}

export function transitionRetouchTicket(
  ticket: MockRetouchTicket,
  status: RetouchTicketStatus,
  note?: string,
  createdAt = new Date().toISOString(),
): void {
  if (ticket.status === status) return
  ticket.status = status
  ticket.updatedAt = createdAt
  ticket.timeline.push({
    status,
    ...(note ? { note } : {}),
    createdAt,
  })
}

export function syncTaskRetouchTicket(
  db: MockDb,
  taskId: string,
): void {
  const task = db.tasks.find((item) => item.id === taskId)
  if (!task) return
  const latest = db.retouchTickets
    .filter((ticket) => ticket.taskId === taskId)
    .sort(
      (left, right) =>
        new Date(right.createdAt).getTime() -
        new Date(left.createdAt).getTime(),
    )[0]

  if (!latest) {
    delete task.retouchTicket
    return
  }

  task.retouchTicket = {
    id: latest.id,
    ticketNo: latest.ticketNo,
    status: latest.status,
    updatedAt: latest.updatedAt,
    ...(latest.quote ? { quoteCredits: latest.quote.credits } : {}),
  }
}

function createRetouchDeliverables(ticket: MockRetouchTicket) {
  const sources =
    ticket.selectedResults.length > 0
      ? ticket.selectedResults
      : createResults(1)
  return sources.map((source, index) => ({
    ...source,
    id: createId(`retouch_result_${index + 1}`),
  }))
}

function materializeLatestRetouchTicket(
  db: MockDb,
  task: MockTask,
): void {
  const latest = db.retouchTickets
    .filter((ticket) => ticket.taskId === task.id)
    .sort(
      (left, right) =>
        new Date(right.createdAt).getTime() -
        new Date(left.createdAt).getTime(),
    )[0]
  if (latest) materializeRetouchTicket(db, latest)
  else delete task.retouchTicket
}

export function materializeRetouchTicket(
  db: MockDb,
  ticket: MockRetouchTicket,
): MockRetouchTicket {
  const now = Date.now()

  if (ticket.status === 'submitted') {
    const submittedAt = new Date(ticket.createdAt).getTime()
    if (now - submittedAt >= RETOUCH_TICKET_TIMING.quoteDelayMs) {
      const quotedAt = new Date(
        submittedAt + RETOUCH_TICKET_TIMING.quoteDelayMs,
      ).toISOString()
      ticket.quote = {
        id: createId('quote'),
        credits: RETOUCH_TICKET_CONFIG.quoteCredits,
        createdAt: quotedAt,
      }
      transitionRetouchTicket(
        ticket,
        'quote_pending',
        `人工修图报价 ${RETOUCH_TICKET_CONFIG.quoteCredits} 次`,
        quotedAt,
      )
    }
  }

  if (
    (ticket.status === 'accepted' || ticket.status === 'processing') &&
    !ticket.revision
  ) {
    const acceptedAt = new Date(
      ticket.acceptedAt ?? ticket.updatedAt,
    ).getTime()
    const processingAt =
      acceptedAt + RETOUCH_TICKET_TIMING.acceptedDelayMs
    const reviewAt =
      processingAt + RETOUCH_TICKET_TIMING.processingDurationMs

    if (now >= processingAt && ticket.status === 'accepted') {
      transitionRetouchTicket(
        ticket,
        'processing',
        '修图师已开始处理',
        new Date(processingAt).toISOString(),
      )
    }
    if (now >= reviewAt && ticket.status === 'processing') {
      ticket.spentCredits = ticket.quote?.credits ?? ticket.reservedCredits
      ticket.deliverables = createRetouchDeliverables(ticket)
      transitionRetouchTicket(
        ticket,
        'awaiting_confirmation',
        '人工成片已上传，请确认',
        new Date(reviewAt).toISOString(),
      )
    }
  }

  if (ticket.status === 'processing' && ticket.revision) {
    const requestedAt = new Date(ticket.revision.requestedAt).getTime()
    if (
      now - requestedAt >= RETOUCH_TICKET_TIMING.revisionProcessingMs
    ) {
      const reviewAt = new Date(
        requestedAt + RETOUCH_TICKET_TIMING.revisionProcessingMs,
      ).toISOString()
      ticket.deliverables = createRetouchDeliverables(ticket)
      transitionRetouchTicket(
        ticket,
        'awaiting_confirmation',
        '返修成片已上传，请再次确认',
        reviewAt,
      )
    }
  }

  syncTaskRetouchTicket(db, ticket.taskId)
  writeDb(db)
  return ticket
}

export type { MockAsset, MockDb, MockRetouchTicket, MockTask }
