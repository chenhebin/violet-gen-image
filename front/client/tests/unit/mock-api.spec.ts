import { setupServer } from 'msw/node'
import { afterAll, beforeAll, beforeEach, describe, expect, it } from 'vitest'
import {
  authApi,
  aiNoticeApi,
  entitlementApi,
  generationApi,
  promptApi,
  retouchTicketApi,
  taskApi,
} from '@/services/api'
import { RETOUCH_TICKET_TIMING } from '@/config'
import { httpClient } from '@/services/http'
import { AppError, ErrorCode } from '@/types/api'
import { handlers, resetMockRateLimits } from '@/mocks/handlers'
import { readDb, resetDb, writeDb } from '@/mocks/db'

const server = setupServer(...handlers)
const originalBaseUrl = httpClient.defaults.baseURL

const confirmedPromptId = 'prompt_test'

async function createRetouchTicket(
  idempotencyKey = 'retouch-create-test',
) {
  const task = await taskApi.get('task_demo_completed')
  return retouchTicketApi.create(
    task.id,
    {
      selectedResultIds: [task.results[0].id],
      requirement: '保留人物特征，调整肤色并清理背景杂物。',
      supplementalAssetIds: [],
    },
    idempotencyKey,
  )
}

async function materializeQuote(ticketId: string) {
  const db = readDb()
  const ticket = db.retouchTickets.find((item) => item.id === ticketId)
  if (!ticket) throw new Error('Retouch ticket not stored')
  ticket.createdAt = new Date(
    Date.now() - RETOUCH_TICKET_TIMING.quoteDelayMs - 100,
  ).toISOString()
  ticket.timeline[0].createdAt = ticket.createdAt
  writeDb(db)
  return retouchTicketApi.get(ticketId)
}

beforeAll(() => {
  httpClient.defaults.baseURL = 'http://localhost/api'
  server.listen({ onUnhandledRequest: 'error' })
})

afterAll(() => {
  server.close()
  httpClient.defaults.baseURL = originalBaseUrl
})

beforeEach(async () => {
  resetDb()
  resetMockRateLimits()
  server.resetHandlers()
  await authApi.login({
    email: 'demo@yingyan.local',
    password: 'Demo1234!',
    remember: false,
  })
  await aiNoticeApi.acknowledge('ai-processing-v2', 'test-ai-notice-ack')
})

describe('mock API business rules', () => {
  it('previews a redemption without consuming it', async () => {
    const preview = await entitlementApi.previewRedemption('YINGYAN-START-10')
    const before = await entitlementApi.get()

    expect(preview).toMatchObject({
      valid: true,
      credits: 10,
      maskedCode: expect.not.stringContaining('START'),
    })
    expect(before.balance).toBe(8)

    const claimed = await entitlementApi.redeem(
      'YINGYAN-START-10',
      'preview-then-claim-key',
    )
    expect(claimed.entitlement.balance).toBe(18)
  })

  it('redeems exactly once for the same idempotency key', async () => {
    const first = await entitlementApi.redeem(
      'YINGYAN-START-10',
      'redeem-test-key',
    )
    const retry = await entitlementApi.redeem(
      'YINGYAN-START-10',
      'redeem-test-key',
    )

    expect(first.entitlement.balance).toBe(18)
    expect(retry).toEqual(first)

    await expect(
      entitlementApi.redeem('YINGYAN-START-10', 'another-key'),
    ).rejects.toMatchObject({
      code: ErrorCode.CodeUsed,
      details: { claimedByCurrentUser: true },
    })
  })

  it('rejects a redemption idempotency key reused with a different code', async () => {
    await entitlementApi.redeem('YINGYAN-START-10', 'redeem-conflict-key')

    await expect(
      entitlementApi.redeem('YINGYAN-SECOND-5', 'redeem-conflict-key'),
    ).rejects.toMatchObject({
      code: ErrorCode.DuplicateRequest,
    })
  })

  it('validates prompt optimization asset ownership by trusted IDs', async () => {
    await expect(
      promptApi.optimize({
        source: '参考上传图片优化成杂志风格',
        mode: 'image-to-image',
        sourceAssetIds: ['asset_not_owned'],
        referenceAssets: [],
      }),
    ).rejects.toMatchObject({
      code: ErrorCode.InvalidPayload,
      message: '提示词素材不存在或不可用',
    })
  })

  it('creates a textual reference prompt before prompt composition', async () => {
    const result = await promptApi.describeReferences([
      { assetId: 'asset_demo_ref', role: 'style' },
    ])

    expect(result.prompt).toContain('杂志氛围')
    expect(result.referenceAssets).toEqual([
      { assetId: 'asset_demo_ref', role: 'style' },
    ])
  })

  it('prevents concurrent generation from overdrawing credits', async () => {
    const payload = {
      promptVersionId: confirmedPromptId,
      assetIds: [],
      settings: {
        aspectRatio: '3:4' as const,
        outputCount: 4 as const,
        referenceStrength: 60,
      },
    }

    const settled = await Promise.allSettled([
      generationApi.create(payload, 'generation-a'),
      generationApi.create(payload, 'generation-b'),
      generationApi.create(payload, 'generation-c'),
    ])
    const successes = settled.filter((item) => item.status === 'fulfilled')
    const failures = settled.filter((item) => item.status === 'rejected')

    expect(successes).toHaveLength(2)
    expect(failures).toHaveLength(1)
    expect((failures[0] as PromiseRejectedResult).reason).toMatchObject({
      code: ErrorCode.InsufficientCredits,
    })
    expect((await entitlementApi.get()).balance).toBe(0)
  })

  it('refunds every failed output and records the ledger entry', async () => {
    const task = await generationApi.create(
      {
        promptVersionId: confirmedPromptId,
        assetIds: [],
        settings: {
          aspectRatio: '1:1',
          outputCount: 2,
          referenceStrength: 50,
        },
      },
      'generation-refund',
    )

    const db = readDb()
    const stored = db.tasks.find((item) => item.id === task.id)
    if (!stored) throw new Error('Task not stored')
    stored.scenario = 'failure'
    stored.createdAt = new Date(Date.now() - 10_000).toISOString()
    writeDb(db)

    const finished = await taskApi.get(task.id)
    const entitlement = await entitlementApi.get()
    const ledger = await entitlementApi.ledger()

    expect(finished.status).toBe('failed-refunded')
    expect(finished.refundedCredits).toBe(2)
    expect(entitlement.balance).toBe(8)
    expect(ledger[0]).toMatchObject({ type: 'refund', amount: 2 })
  })

  it('creates directly from a raw requirement without prompt optimization', async () => {
    const task = await generationApi.create(
      {
        source: '小猫四爪朝天，躺在地上',
        referenceAssets: [],
        assetIds: [],
        settings: {
          aspectRatio: '1:1',
          outputCount: 1,
          referenceStrength: 50,
        },
      },
      'generation-direct',
    )

    expect(task.entitlement).toMatchObject({
      balance: 7,
      canCreate: true,
    })
    expect(task.prompt).toMatchObject({
      source: '小猫四爪朝天，躺在地上',
      sections: {
        subject: '',
        scene: '',
        style: '',
        composition: '',
        details: '',
        negative: '',
        output: '',
      },
    })
    expect(task.prompt.confirmedAt).toBeTruthy()
    expect((await entitlementApi.get()).balance).toBe(7)
  })

  it('rejects an unconfirmed prompt version when one is supplied', async () => {
    const request = generationApi.create(
      {
        promptVersionId: 'prompt_not_confirmed',
        assetIds: [],
        settings: {
          aspectRatio: '1:1',
          outputCount: 1,
          referenceStrength: 50,
        },
      },
      'generation-unconfirmed',
    )

    await expect(request).rejects.toBeInstanceOf(AppError)
    await expect(request).rejects.toMatchObject({
      code: ErrorCode.InvalidPayload,
    })
    expect((await entitlementApi.get()).balance).toBe(8)
  })

  it('rejects a prompt confirmation key reused with different content', async () => {
    const first = await promptApi.confirm(
      confirmedPromptId,
      '第一版需求',
      {
        subject: '第一版需求',
        scene: '',
        style: '',
        composition: '',
        details: '',
        negative: '',
        output: '',
      },
      'prompt-confirm-conflict',
    )
    expect(first.confirmedAt).toBeTruthy()

    await expect(
      promptApi.confirm(
        confirmedPromptId,
        '第二版需求',
        first.sections,
        'prompt-confirm-conflict',
      ),
    ).rejects.toMatchObject({ code: ErrorCode.DuplicateRequest })
  })

  it('rejects a generation key reused with different settings', async () => {
    await generationApi.create(
      {
        promptVersionId: confirmedPromptId,
        assetIds: [],
        settings: { aspectRatio: '1:1', outputCount: 1, referenceStrength: 50 },
      },
      'generation-conflict-key',
    )

    await expect(
      generationApi.create(
        {
          promptVersionId: confirmedPromptId,
          assetIds: [],
          settings: { aspectRatio: '3:4', outputCount: 1, referenceStrength: 50 },
        },
        'generation-conflict-key',
      ),
    ).rejects.toMatchObject({ code: ErrorCode.DuplicateRequest })
  })

  it('requires an eligible generation task with successful results', async () => {
    const task = await generationApi.create(
      {
        promptVersionId: confirmedPromptId,
        assetIds: [],
        settings: {
          aspectRatio: '1:1',
          outputCount: 1,
          referenceStrength: 50,
        },
      },
      'generation-for-retouch-eligibility',
    )

    await expect(
      retouchTicketApi.create(
        task.id,
        {
          selectedResultIds: ['missing-result'],
          requirement: '需要人工精修',
          supplementalAssetIds: [],
        },
        'retouch-ineligible',
      ),
    ).rejects.toMatchObject({
      code: ErrorCode.RetouchTaskNotEligible,
    })
  })

  it('creates idempotently and rejects a second active ticket', async () => {
    const first = await createRetouchTicket('retouch-create-idempotent')
    const retry = await createRetouchTicket('retouch-create-idempotent')

    expect(retry).toEqual(first)
    expect((await taskApi.get(first.taskId)).retouchTicket).toMatchObject({
      id: first.id,
      ticketNo: first.ticketNo,
      status: 'submitted',
    })
    await expect(
      createRetouchTicket('retouch-create-duplicate'),
    ).rejects.toMatchObject({
      code: ErrorCode.RetouchTicketAlreadyExists,
    })
  })

  it('rejects a retouch creation key reused with different requirements', async () => {
    const task = await taskApi.get('task_demo_completed')
    await retouchTicketApi.create(
      task.id,
      {
        selectedResultIds: [task.results[0].id],
        requirement: '第一版人工要求',
        supplementalAssetIds: [],
      },
      'retouch-create-conflict',
    )

    await expect(
      retouchTicketApi.create(
        task.id,
        {
          selectedResultIds: [task.results[0].id],
          requirement: '第二版人工要求',
          supplementalAssetIds: [],
        },
        'retouch-create-conflict',
      ),
    ).rejects.toMatchObject({ code: ErrorCode.DuplicateRequest })
  })

  it('returns Retry-After when a mock endpoint exceeds its rate limit', async () => {
    const attempts = await Promise.allSettled(
      Array.from({ length: 21 }, () =>
        entitlementApi.previewRedemption('YINGYAN-START-10'),
      ),
    )
    const rejected = attempts.find(
      (item): item is PromiseRejectedResult => item.status === 'rejected',
    )
    expect(rejected?.reason).toMatchObject({
      code: ErrorCode.RateLimited,
      retryAfterSeconds: expect.any(Number),
    })
  })

  it('quotes three credits, reserves them once, and prevents overdrawing', async () => {
    const ticket = await createRetouchTicket('retouch-create-quote')
    const quoted = await materializeQuote(ticket.id)

    expect(quoted.status).toBe('quote_pending')
    expect(quoted.quote?.credits).toBe(3)

    const accepted = await retouchTicketApi.acceptQuote(
      ticket.id,
      quoted.quote!.id,
      'retouch-accept-once',
    )
    const retry = await retouchTicketApi.acceptQuote(
      ticket.id,
      quoted.quote!.id,
      'retouch-accept-once',
    )

    expect(accepted.ticket.status).toBe('accepted')
    expect(accepted.ticket.reservedCredits).toBe(3)
    expect(retry).toEqual(accepted)
    expect((await entitlementApi.get()).balance).toBe(5)

    const secondTask = await taskApi.get('task_demo_partial')
    const secondTicket = await retouchTicketApi.create(
      secondTask.id,
      {
        selectedResultIds: [secondTask.results[0].id],
        requirement: '统一画面色调并清理背景',
        supplementalAssetIds: [],
      },
      'retouch-create-overdraw',
    )
    const secondQuote = await materializeQuote(secondTicket.id)
    const db = readDb()
    db.entitlements.user_demo.balance = 2
    writeDb(db)

    await expect(
      retouchTicketApi.acceptQuote(
        secondTicket.id,
        secondQuote.quote!.id,
        'retouch-accept-overdraw',
      ),
    ).rejects.toMatchObject({
      code: ErrorCode.InsufficientCredits,
    })
    expect((await entitlementApi.get()).balance).toBe(2)
  })

  it('serializes concurrent quote acceptance so balance cannot go negative', async () => {
    const completedTask = await taskApi.get('task_demo_completed')
    const partialTask = await taskApi.get('task_demo_partial')
    const [firstTicket, secondTicket] = await Promise.all([
      retouchTicketApi.create(
        completedTask.id,
        {
          selectedResultIds: [completedTask.results[0].id],
          requirement: '人工精修第一组图片',
          supplementalAssetIds: [],
        },
        'retouch-concurrent-create-a',
      ),
      retouchTicketApi.create(
        partialTask.id,
        {
          selectedResultIds: [partialTask.results[0].id],
          requirement: '人工精修第二组图片',
          supplementalAssetIds: [],
        },
        'retouch-concurrent-create-b',
      ),
    ])
    const [firstQuote, secondQuote] = await Promise.all([
      materializeQuote(firstTicket.id),
      materializeQuote(secondTicket.id),
    ])
    const db = readDb()
    db.entitlements.user_demo.balance = 5
    writeDb(db)

    const settled = await Promise.allSettled([
      retouchTicketApi.acceptQuote(
        firstTicket.id,
        firstQuote.quote!.id,
        'retouch-concurrent-accept-a',
      ),
      retouchTicketApi.acceptQuote(
        secondTicket.id,
        secondQuote.quote!.id,
        'retouch-concurrent-accept-b',
      ),
    ])

    expect(
      settled.filter((result) => result.status === 'fulfilled'),
    ).toHaveLength(1)
    const rejection = settled.find((result) => result.status === 'rejected')
    expect((rejection as PromiseRejectedResult).reason).toMatchObject({
      code: ErrorCode.InsufficientCredits,
    })
    expect((await entitlementApi.get()).balance).toBe(2)
  })

  it('releases an accepted ticket reservation when cancelled before processing', async () => {
    const ticket = await createRetouchTicket('retouch-create-cancel')
    const quoted = await materializeQuote(ticket.id)
    await retouchTicketApi.acceptQuote(
      ticket.id,
      quoted.quote!.id,
      'retouch-accept-cancel',
    )

    const cancelled = await retouchTicketApi.cancel(
      ticket.id,
      'retouch-cancel-once',
    )
    const retry = await retouchTicketApi.cancel(
      ticket.id,
      'retouch-cancel-once',
    )
    const ledger = await entitlementApi.ledger()

    expect(cancelled.ticket.status).toBe('cancelled')
    expect(cancelled.ticket.refundedCredits).toBe(3)
    expect(cancelled.entitlement.balance).toBe(8)
    expect(retry).toEqual(cancelled)
    expect(ledger[0]).toMatchObject({ type: 'release', amount: 3 })
  })

  it('allows a new ticket after the previous ticket is cancelled', async () => {
    const first = await createRetouchTicket('retouch-create-first')
    await retouchTicketApi.cancel(first.id, 'retouch-cancel-first')

    const reopened = await createRetouchTicket('retouch-create-reopened')
    const task = await taskApi.get(first.taskId)

    expect(reopened.id).not.toBe(first.id)
    expect(reopened.status).toBe('submitted')
    expect(task.retouchTicket).toMatchObject({
      id: reopened.id,
      status: 'submitted',
    })
  })

  it('advances to review and permits exactly one revision', async () => {
    const ticket = await createRetouchTicket('retouch-create-progress')
    const quoted = await materializeQuote(ticket.id)
    await retouchTicketApi.acceptQuote(
      ticket.id,
      quoted.quote!.id,
      'retouch-accept-progress',
    )

    let db = readDb()
    let stored = db.retouchTickets.find((item) => item.id === ticket.id)
    if (!stored) throw new Error('Retouch ticket not stored')
    stored.acceptedAt = new Date(
      Date.now() -
        RETOUCH_TICKET_TIMING.acceptedDelayMs -
        RETOUCH_TICKET_TIMING.processingDurationMs -
        100,
    ).toISOString()
    writeDb(db)

    const awaiting = await retouchTicketApi.get(ticket.id)
    expect(awaiting.status).toBe('awaiting_confirmation')
    expect(awaiting.spentCredits).toBe(3)
    expect(awaiting.deliverables).toHaveLength(1)
    expect(awaiting.timeline.map((entry) => entry.status)).toEqual(
      expect.arrayContaining(['accepted', 'processing', 'awaiting_confirmation']),
    )

    const revising = await retouchTicketApi.requestRevision(
      ticket.id,
      '请降低磨皮强度，保留更多皮肤纹理。',
      'retouch-revision-once',
    )
    expect(revising.status).toBe('processing')

    db = readDb()
    stored = db.retouchTickets.find((item) => item.id === ticket.id)
    if (!stored?.revision) throw new Error('Revision not stored')
    stored.revision.requestedAt = new Date(
      Date.now() - RETOUCH_TICKET_TIMING.revisionProcessingMs - 100,
    ).toISOString()
    writeDb(db)

    const revised = await retouchTicketApi.get(ticket.id)
    expect(revised.status).toBe('awaiting_confirmation')
    expect(revised.deliverables).toHaveLength(1)
    await expect(
      retouchTicketApi.requestRevision(
        ticket.id,
        '再次调整',
        'retouch-revision-twice',
      ),
    ).rejects.toMatchObject({
      code: ErrorCode.RetouchRevisionLimitReached,
    })

    const delivered = await retouchTicketApi.confirm(
      ticket.id,
      'retouch-confirm-delivery',
    )
    expect(delivered.status).toBe('delivered')
  })
})
