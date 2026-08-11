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
import { authApi } from '@/services/auth'
import { apiRequest, httpClient } from '@/services/http'
import { providerApi } from '@/services/providers'
import { redemptionApi } from '@/services/redemption'
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
