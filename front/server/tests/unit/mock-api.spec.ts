import { setupServer } from 'msw/node'
import {
  afterAll,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
} from 'vitest'
import { handlers } from '@/mocks/handlers'
import { resetDb } from '@/mocks/db'
import { resetMockRateLimits } from '@/mocks/helpers'
import { authApi } from '@/services/auth'
import { auditApi } from '@/services/audits'
import { apiRequest, httpClient } from '@/services/http'
import { providerApi } from '@/services/providers'
import { redemptionApi } from '@/services/redemption'
import { retouchApi } from '@/services/retouch'
import { dashboardApi } from '@/services/dashboard'
import { userApi } from '@/services/users'
import { ErrorCode } from '@/types/api'

const originalBaseUrl = httpClient.defaults.baseURL
const server = setupServer(...handlers)

async function loginAdmin(): Promise<void> {
  await authApi.login({
    email: 'admin@yingyan.local',
    password: 'Admin1234!',
    remember: false,
  })
}

async function loginRetouch(): Promise<void> {
  await authApi.login({
    email: 'retouch@yingyan.local',
    password: 'Retouch1234!',
    remember: false,
  })
}

beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' })
  httpClient.defaults.baseURL = new URL('/api', window.location.href).toString()
})

beforeEach(() => {
  resetDb()
  resetMockRateLimits()
})

afterAll(() => {
  server.close()
  httpClient.defaults.baseURL = originalBaseUrl
})

describe('management mock API', () => {
  it('restores the platform administrator session', async () => {
    await loginAdmin()
    const session = await authApi.session()

    expect(session.role).toBe('platform_admin')
    expect(session.permissions).toEqual([
      'platform:manage',
      'retouch:manage',
    ])
  })

  it('enforces platform permissions on every platform endpoint', async () => {
    await loginRetouch()

    await expect(redemptionApi.listCodes()).rejects.toEqual(
      expect.objectContaining({
        code: ErrorCode.Forbidden,
        message: '无权执行此管理操作',
      }),
    )
  })

  it('replays a batch creation with the same idempotency key', async () => {
    await loginAdmin()
    const payload = {
      name: '幂等测试批次',
      quantity: 2,
      creditsPerCode: 6,
      productCode: 'yingyan-client',
      neverExpires: true,
    }
    const request = () =>
      apiRequest<{ batch: { id: string }; codes: unknown[] }>({
        method: 'POST',
        url: '/manage/redemption-batches',
        data: payload,
        headers: { 'Idempotency-Key': 'idem-test-create-batch' },
      })

    const first = await request()
    const second = await request()

    expect(second.batch.id).toBe(first.batch.id)
    expect(second.codes).toHaveLength(2)
    const batches = await redemptionApi.listBatches({ pageSize: 100 })
    expect(
      batches.items.filter((item) => item.name === payload.name),
    ).toHaveLength(1)

    await expect(
      apiRequest({
        method: 'POST',
        url: '/manage/redemption-batches',
        data: { ...payload, quantity: 3 },
        headers: { 'Idempotency-Key': 'idem-test-create-batch' },
      }),
    ).rejects.toEqual(
      expect.objectContaining({ code: ErrorCode.DuplicateRequest }),
    )
  })

  it('exposes SLA metrics and accepts a fresh quote with a 48-hour expiry', async () => {
    await loginAdmin()
    const dashboard = await dashboardApi.get()
    expect(dashboard.metrics.map((metric) => metric.key)).toEqual(
      expect.arrayContaining(['overdueTickets', 'dueSoonTickets']),
    )

    const tickets = await retouchApi.list({ status: 'submitted', pageSize: 100 })
    const quoted = await retouchApi.quote(tickets.items[0].id, 2, '按需求评估')
    expect(quoted.status).toBe('quote_pending')
    expect(quoted.quote?.status).toBe('active')
    expect(quoted.quote?.remainingSeconds).toBeGreaterThanOrEqual(48 * 60 * 60 - 1)
    expect(quoted.quote?.remainingSeconds).toBeLessThanOrEqual(48 * 60 * 60)
    expect(Date.parse(quoted.quote!.expiresAt) - Date.parse(quoted.quote!.createdAt)).toBe(48 * 60 * 60_000)
  })

  it('filters management tickets by overdue and due-soon SLA', async () => {
    await loginAdmin()
    const overdue = await retouchApi.list({ sla: 'overdue', pageSize: 100 })
    const dueSoon = await retouchApi.list({ sla: 'due-soon', pageSize: 100 })

    expect(overdue.items.every((item) => item.sla.overdue)).toBe(true)
    expect(
      dueSoon.items.every(
        (item) => !item.sla.overdue && (item.sla.remainingSeconds ?? Infinity) <= 24 * 60 * 60,
      ),
    ).toBe(true)
  })

  it('returns Retry-After when management quote requests exceed the mock limit', async () => {
    await loginAdmin()
    const tickets = await retouchApi.list({ status: 'submitted', pageSize: 100 })
    const ticketId = tickets.items[0].id
    const attempts = await Promise.allSettled(
      Array.from({ length: 11 }, (_, index) =>
        retouchApi.quote(ticketId, index + 1, `限流测试 ${index + 1}`),
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

  it('renames a redemption batch and keeps code references in sync', async () => {
    await loginAdmin()
    const batches = await redemptionApi.listBatches({ pageSize: 100 })
    const target = batches.items[0]
    const renamed = await redemptionApi.updateBatch(target.id, {
      name: '  八月咸鱼发码批次  ',
    })
    const [detail, codes, audits] = await Promise.all([
      redemptionApi.getBatch(target.id),
      redemptionApi.listCodes({ batchId: target.id, pageSize: 100 }),
      auditApi.list({
        action: 'redemption_batch.rename',
        resourceType: 'redemption_batch',
        pageSize: 100,
      }),
    ])

    expect(renamed.name).toBe('八月咸鱼发码批次')
    expect(detail.name).toBe(renamed.name)
    expect(codes.items.every((item) => item.batchName === renamed.name)).toBe(true)
    expect(audits.items[0]).toEqual(
      expect.objectContaining({
        action: 'redemption_batch.rename',
        resourceId: target.id,
        before: { name: target.name },
        after: { name: renamed.name },
      }),
    )
  })

  it('rejects an invalid redemption batch name', async () => {
    await loginAdmin()
    const batches = await redemptionApi.listBatches({ pageSize: 1 })

    await expect(
      redemptionApi.updateBatch(batches.items[0].id, { name: '   ' }),
    ).rejects.toEqual(
      expect.objectContaining({
        code: ErrorCode.InvalidPayload,
        message: '批次名称不能为空',
      }),
    )
  })

  it('exports one URL-encoded Xianyu inventory item per unused code', async () => {
    await loginAdmin()
    const batches = await redemptionApi.listBatches({ pageSize: 100 })
    const target = batches.items.find((item) => item.counts.unused > 0)
    if (!target) throw new Error('No batch with unused codes')

    const exported = await redemptionApi.exportBatch(target.id, 'xianyu')
    const lines = exported.content ? exported.content.split('\n') : []

    expect(exported).toMatchObject({
      format: 'xianyu',
      mediaType: 'text/plain;charset=utf-8',
      count: target.counts.unused,
    })
    expect(lines).toHaveLength(target.counts.unused)
    expect(lines[0]).toMatch(
      /^浏览器打开领取并开始创作吧～：https:\/\/img\.daidaiweb\.cn\/claim\?code=YY-[A-Z0-9-]+ ｜ 兑换码：YY-[A-Z0-9-]+$/,
    )
    const [, claimCode, backupCode] = lines[0].match(
      /\?code=([^ ]+) ｜ 兑换码：(.+)$/,
    ) ?? []
    expect(decodeURIComponent(claimCode)).toBe(backupCode)
  })

  it('rejects a credit adjustment that would create a negative balance', async () => {
    await loginAdmin()

    await expect(
      userApi.adjustCredits('user_mia', {
        amount: -7,
        reason: '错误扣减测试',
      }),
    ).rejects.toEqual(
      expect.objectContaining({
        code: ErrorCode.InsufficientCredits,
      }),
    )
  })

  it('atomically switches the single platform image model', async () => {
    await loginAdmin()
    const alternate = await providerApi.createModel({
      providerId: 'provider_test1',
      displayName: 'Photon Alternate',
      modelId: 'photon-alternate-v1',
      type: 'image',
      enabled: true,
      capabilities: {
        textToImage: true,
        imageToImage: true,
      },
    })
    await providerApi.testModel(alternate.id)
    const bindings = await providerApi.bindModel('image', alternate.id)
    const models = await providerApi.listModels()

    expect(bindings.imageModelId).toBe(alternate.id)
    expect(
      models.filter(
        (item) => item.type === 'image' && item.isPlatformModel,
      ),
    ).toHaveLength(1)
  })

  it('keeps a successful model test when non-runtime fields are edited', async () => {
    await loginAdmin()
    const created = await providerApi.createModel({
      providerId: 'provider_test1',
      displayName: 'Stable Test Model',
      modelId: 'stable-test-model-v1',
      type: 'image',
      enabled: true,
      capabilities: {
        textToImage: true,
        imageToImage: true,
      },
    })
    const tested = await providerApi.testModel(created.id)
    const renamed = await providerApi.updateModel(created.id, {
      displayName: 'Stable Test Model Renamed',
      enabled: true,
      capabilities: {
        imageToImage: true,
        textToImage: true,
      },
    })

    expect(tested.connectionStatus).toBe('healthy')
    expect(renamed.connectionStatus).toBe('healthy')
    expect(renamed.lastTestAt).toBe(tested.lastTestAt)
  })

  it('deletes unused models and empty providers but protects active configuration', async () => {
    await loginAdmin()

    await expect(
      providerApi.deleteModel('model_test1_image'),
    ).rejects.toEqual(expect.objectContaining({ code: ErrorCode.InvalidPayload }))
    await expect(
      providerApi.deleteProvider('provider_test1'),
    ).rejects.toEqual(expect.objectContaining({ code: ErrorCode.InvalidPayload }))

    const provider = await providerApi.createProvider({
      name: 'Disposable Provider',
      code: 'disposable_provider',
      baseUrl: 'https://disposable.example/v1',
      apiKey: 'sk-disposable-provider-key',
      enabled: true,
    })
    const model = await providerApi.createModel({
      providerId: provider.id,
      displayName: 'Disposable Model',
      modelId: 'disposable-model-v1',
      type: 'chat',
      enabled: true,
      capabilities: {
        promptOptimization: true,
        visionInput: true,
      },
    })

    await providerApi.deleteModel(model.id)
    await providerApi.deleteProvider(provider.id)

    const [providers, models] = await Promise.all([
      providerApi.listProviders(),
      providerApi.listModels(),
    ])
    expect(providers.some((item) => item.id === provider.id)).toBe(false)
    expect(models.some((item) => item.id === model.id)).toBe(false)
  })

  it('does not expose API keys or full codes in list payloads', async () => {
    await loginAdmin()
    const [providers, codes] = await Promise.all([
      providerApi.listProviders(),
      redemptionApi.listCodes({ pageSize: 100 }),
    ])

    expect(providers[0]).not.toHaveProperty('apiKey')
    expect(codes.items[0]).not.toHaveProperty('fullCode')
  })
})
