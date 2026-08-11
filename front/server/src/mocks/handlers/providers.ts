import { http } from 'msw'
import {
  appendAudit,
  publicModel,
  publicProvider,
} from '@/mocks/db'
import {
  dbAndAdmin,
  idempotentMutation,
  MockApiError,
  readJson,
  requestId,
  respond,
} from '@/mocks/helpers'
import { ErrorCode } from '@/types/api'
import type {
  CreateAIModelPayload,
  CreateAIProviderPayload,
  ModelType,
  UpdateAIModelPayload,
  UpdateAIProviderPayload,
} from '@/types/domain'
import { createId } from '@/utils/id'

function validateBaseUrl(value: string): string {
  let url: URL
  try {
    url = new URL(value)
  } catch {
    throw new MockApiError(
      422,
      ErrorCode.InvalidPayload,
      'Base URL 格式不正确',
    )
  }
  const hostname = url.hostname.toLowerCase()
  if (
    url.protocol !== 'https:' ||
    hostname === 'localhost' ||
    hostname === '127.0.0.1' ||
    hostname === '::1' ||
    /^10\./.test(hostname) ||
    /^192\.168\./.test(hostname) ||
    /^169\.254\./.test(hostname)
  ) {
    throw new MockApiError(
      422,
      ErrorCode.InvalidPayload,
      'Base URL 必须使用 HTTPS，且不能指向本机或内网地址',
    )
  }
  return value.replace(/\/+$/, '')
}

function maskKey(apiKey: string): string {
  return apiKey.length <= 8
    ? '••••••••'
    : `${apiKey.slice(0, 3)}-••••••••${apiKey.slice(-5)}`
}

function sameCapabilities(
  type: ModelType,
  current: UpdateAIModelPayload['capabilities'],
  next: UpdateAIModelPayload['capabilities'],
): boolean {
  if (type === 'chat') {
    return (
      Boolean(current?.promptOptimization) ===
        Boolean(next?.promptOptimization) &&
      Boolean(current?.visionInput) === Boolean(next?.visionInput)
    )
  }
  return (
    Boolean(current?.textToImage) === Boolean(next?.textToImage) &&
    Boolean(current?.imageToImage) === Boolean(next?.imageToImage)
  )
}

export const providerHandlers = [
  http.get('/api/manage/ai-providers', () =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      return db.providers
        .slice()
        .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
        .map(publicProvider)
    }),
  ),

  http.post('/api/manage/ai-providers', ({ request }) =>
    respond(async () => {
      const payload = await readJson<CreateAIProviderPayload>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        const code = payload.code?.trim().toLowerCase()
        if (
          !payload.name?.trim() ||
          !code ||
          !/^[a-z][a-z0-9_-]{1,31}$/.test(code) ||
          !payload.apiKey?.trim()
        ) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '请填写有效的服务商名称、编码和 API Key',
          )
        }
        if (db.providers.some((item) => item.code === code)) {
          throw new MockApiError(
            409,
            ErrorCode.DuplicateRequest,
            '服务商编码已存在',
          )
        }
        const now = new Date().toISOString()
        const provider = {
          id: createId('provider'),
          name: payload.name.trim(),
          code,
          protocol: 'openai-compatible' as const,
          baseUrl: validateBaseUrl(payload.baseUrl),
          apiKey: payload.apiKey.trim(),
          maskedApiKey: maskKey(payload.apiKey.trim()),
          enabled: payload.enabled ?? true,
          connectionStatus: 'untested' as const,
          note: payload.note?.trim() || undefined,
          createdAt: now,
          updatedAt: now,
        }
        db.providers.unshift(provider)
        appendAudit(db, admin, {
          action: 'provider.create',
          resourceType: 'ai_provider',
          resourceId: provider.id,
          after: {
            name: provider.name,
            code: provider.code,
            baseUrl: provider.baseUrl,
            apiKey: provider.apiKey,
            enabled: provider.enabled,
          },
          result: 'success',
          requestId: requestId(request),
        })
        return publicProvider(provider)
      })
    }),
  ),

  http.patch('/api/manage/ai-providers/:providerId', ({ params, request }) =>
    respond(async () => {
      const payload = await readJson<UpdateAIProviderPayload>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        const provider = db.providers.find(
          (item) => item.id === params.providerId,
        )
        if (!provider) {
          throw new MockApiError(404, ErrorCode.NotFound, '服务商不存在')
        }
        const before = publicProvider(provider)
        if (payload.enabled === false) {
          const boundIds = new Set([
            db.bindings.chatModelId,
            db.bindings.imageModelId,
          ])
          if (
            db.models.some(
              (item) =>
                item.providerId === provider.id && boundIds.has(item.id),
            )
          ) {
            throw new MockApiError(
              409,
              ErrorCode.InvalidPayload,
              '请先切换或解除当前平台模型，再停用服务商',
            )
          }
        }
        const connectionChanged =
          payload.baseUrl !== undefined &&
          validateBaseUrl(payload.baseUrl) !== provider.baseUrl
        if (payload.name !== undefined) provider.name = payload.name.trim()
        if (payload.baseUrl !== undefined) {
          provider.baseUrl = validateBaseUrl(payload.baseUrl)
        }
        if (payload.enabled !== undefined) provider.enabled = payload.enabled
        if (payload.note !== undefined) {
          provider.note = payload.note.trim() || undefined
        }
        provider.updatedAt = new Date().toISOString()
        if (connectionChanged) {
          provider.connectionStatus = 'untested'
          provider.lastTest = undefined
          db.models
            .filter((item) => item.providerId === provider.id)
            .forEach((item) => {
              item.connectionStatus = 'untested'
              item.lastTestAt = undefined
              item.lastTest = undefined
            })
        }
        appendAudit(db, admin, {
          action: 'provider.update',
          resourceType: 'ai_provider',
          resourceId: provider.id,
          before: {
            name: before.name,
            baseUrl: before.baseUrl,
            enabled: before.enabled,
          },
          after: {
            name: provider.name,
            baseUrl: provider.baseUrl,
            enabled: provider.enabled,
          },
          result: 'success',
          requestId: requestId(request),
        })
        return publicProvider(provider)
      })
    }),
  ),

  http.delete('/api/manage/ai-providers/:providerId', ({ params, request }) =>
    respond(() => {
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, null, () => {
        const index = db.providers.findIndex(
          (item) => item.id === params.providerId,
        )
        if (index < 0) {
          throw new MockApiError(404, ErrorCode.NotFound, '服务商不存在')
        }
        const provider = db.providers[index]
        const providerModels = db.models.filter(
          (item) => item.providerId === provider.id,
        )
        const boundIds = new Set([
          db.bindings.chatModelId,
          db.bindings.imageModelId,
        ])
        if (
          providerModels.length ||
          providerModels.some((item) => boundIds.has(item.id))
        ) {
          throw new MockApiError(
            409,
            ErrorCode.InvalidPayload,
            '服务商下仍有模型，请先解除平台绑定并删除全部模型',
          )
        }
        db.providers.splice(index, 1)
        appendAudit(db, admin, {
          action: 'provider.delete',
          resourceType: 'ai_provider',
          resourceId: provider.id,
          before: {
            name: provider.name,
            code: provider.code,
            baseUrl: provider.baseUrl,
            enabled: provider.enabled,
          },
          result: 'success',
          requestId: requestId(request),
        })
        return null
      })
    }),
  ),

  http.post('/api/manage/ai-providers/:providerId/test', ({ params, request }) =>
    respond(() => {
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, null, () => {
        const provider = db.providers.find(
          (item) => item.id === params.providerId,
        )
        if (!provider) {
          throw new MockApiError(404, ErrorCode.NotFound, '服务商不存在')
        }
        const success = !provider.baseUrl.includes('test2') &&
          !provider.baseUrl.includes('fail')
        const testedAt = new Date().toISOString()
        provider.connectionStatus = success ? 'healthy' : 'error'
        provider.lastTest = {
          testedAt,
          success,
          message: success
            ? '认证与响应结构正常'
            : '连接超时，请检查 Base URL',
        }
        provider.updatedAt = testedAt
        appendAudit(db, admin, {
          action: 'provider.test',
          resourceType: 'ai_provider',
          resourceId: provider.id,
          after: { connectionStatus: provider.connectionStatus },
          result: success ? 'success' : 'failure',
          requestId: requestId(request),
        })
        return publicProvider(provider)
      })
    }),
  ),

  http.post('/api/manage/ai-providers/:providerId/rotate-key', ({ params, request }) =>
    respond(async () => {
      const payload = await readJson<{ apiKey: string }>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, { hasApiKey: true }, () => {
        const provider = db.providers.find(
          (item) => item.id === params.providerId,
        )
        if (!provider) {
          throw new MockApiError(404, ErrorCode.NotFound, '服务商不存在')
        }
        if (!payload.apiKey?.trim()) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '请输入新的 API Key',
          )
        }
        provider.apiKey = payload.apiKey.trim()
        provider.maskedApiKey = maskKey(provider.apiKey)
        provider.connectionStatus = 'untested'
        provider.lastTest = undefined
        provider.updatedAt = new Date().toISOString()
        db.models
          .filter((item) => item.providerId === provider.id)
          .forEach((item) => {
            item.connectionStatus = 'untested'
            item.lastTestAt = undefined
            item.lastTest = undefined
          })
        appendAudit(db, admin, {
          action: 'provider.rotate_key',
          resourceType: 'ai_provider',
          resourceId: provider.id,
          after: { apiKey: provider.apiKey, connectionStatus: 'untested' },
          result: 'success',
          requestId: requestId(request),
        })
        return publicProvider(provider)
      })
    }),
  ),

  http.get('/api/manage/ai-models', ({ request }) =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      const providerId = new URL(request.url).searchParams.get('providerId')
      return db.models
        .filter((item) => !providerId || item.providerId === providerId)
        .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
        .map((item) => publicModel(item, db))
    }),
  ),

  http.post('/api/manage/ai-models', ({ request }) =>
    respond(async () => {
      const payload = await readJson<CreateAIModelPayload>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        const provider = db.providers.find(
          (item) => item.id === payload.providerId,
        )
        if (!provider) {
          throw new MockApiError(404, ErrorCode.NotFound, '服务商不存在')
        }
        if (
          !payload.displayName?.trim() ||
          !payload.modelId?.trim() ||
          !['chat', 'image'].includes(payload.type)
        ) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '请填写有效的模型名称、模型 ID 和类型',
          )
        }
        if (
          db.models.some(
            (item) =>
              item.providerId === provider.id &&
              item.modelId === payload.modelId.trim(),
          )
        ) {
          throw new MockApiError(
            409,
            ErrorCode.DuplicateRequest,
            '该服务商下的模型 ID 已存在',
          )
        }
        const now = new Date().toISOString()
        const model = {
          id: createId('model'),
          providerId: provider.id,
          displayName: payload.displayName.trim(),
          modelId: payload.modelId.trim(),
          type: payload.type,
          enabled: payload.enabled ?? true,
          connectionStatus: 'untested' as const,
          capabilities: { ...payload.capabilities },
          createdAt: now,
          updatedAt: now,
        }
        db.models.unshift(model)
        appendAudit(db, admin, {
          action: 'model.create',
          resourceType: 'ai_model',
          resourceId: model.id,
          after: {
            providerId: model.providerId,
            modelId: model.modelId,
            type: model.type,
            capabilities: model.capabilities,
          },
          result: 'success',
          requestId: requestId(request),
        })
        return publicModel(model, db)
      })
    }),
  ),

  http.patch('/api/manage/ai-models/:modelId', ({ params, request }) =>
    respond(async () => {
      const payload = await readJson<UpdateAIModelPayload>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        const model = db.models.find((item) => item.id === params.modelId)
        if (!model) {
          throw new MockApiError(404, ErrorCode.NotFound, '模型不存在')
        }
        const bindingKey =
          model.type === 'chat' ? 'chatModelId' : 'imageModelId'
        if (
          payload.enabled === false &&
          db.bindings[bindingKey] === model.id
        ) {
          throw new MockApiError(
            409,
            ErrorCode.InvalidPayload,
            '请先切换或解除平台模型，再停用当前模型',
          )
        }
        const before = publicModel(model, db)
        const configChanged =
          (payload.modelId !== undefined &&
            payload.modelId.trim() !== model.modelId) ||
          (payload.capabilities !== undefined &&
            !sameCapabilities(
              model.type,
              model.capabilities,
              payload.capabilities,
            ))
        if (payload.displayName !== undefined) {
          model.displayName = payload.displayName.trim()
        }
        if (payload.modelId !== undefined) {
          model.modelId = payload.modelId.trim()
        }
        if (payload.enabled !== undefined) model.enabled = payload.enabled
        if (payload.capabilities !== undefined) {
          model.capabilities = { ...payload.capabilities }
        }
        if (configChanged) {
          model.connectionStatus = 'untested'
          model.lastTestAt = undefined
          model.lastTest = undefined
        }
        model.updatedAt = new Date().toISOString()
        appendAudit(db, admin, {
          action: 'model.update',
          resourceType: 'ai_model',
          resourceId: model.id,
          before: {
            modelId: before.modelId,
            enabled: before.enabled,
            capabilities: before.capabilities,
          },
          after: {
            modelId: model.modelId,
            enabled: model.enabled,
            capabilities: model.capabilities,
          },
          result: 'success',
          requestId: requestId(request),
        })
        return publicModel(model, db)
      })
    }),
  ),

  http.delete('/api/manage/ai-models/:modelId', ({ params, request }) =>
    respond(() => {
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, null, () => {
        const index = db.models.findIndex((item) => item.id === params.modelId)
        if (index < 0) {
          throw new MockApiError(404, ErrorCode.NotFound, '模型不存在')
        }
        const model = db.models[index]
        if (
          db.bindings.chatModelId === model.id ||
          db.bindings.imageModelId === model.id
        ) {
          throw new MockApiError(
            409,
            ErrorCode.InvalidPayload,
            '当前平台模型不能删除，请先解除或切换平台绑定',
          )
        }
        db.models.splice(index, 1)
        appendAudit(db, admin, {
          action: 'model.delete',
          resourceType: 'ai_model',
          resourceId: model.id,
          before: {
            providerId: model.providerId,
            displayName: model.displayName,
            modelId: model.modelId,
            type: model.type,
            enabled: model.enabled,
          },
          result: 'success',
          requestId: requestId(request),
        })
        return null
      })
    }),
  ),

  http.post('/api/manage/ai-models/:modelId/test', ({ params, request }) =>
    respond(() => {
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, null, () => {
        const model = db.models.find((item) => item.id === params.modelId)
        if (!model) {
          throw new MockApiError(404, ErrorCode.NotFound, '模型不存在')
        }
        const provider = db.providers.find(
          (item) => item.id === model.providerId,
        )
        const success = Boolean(
          provider?.enabled && provider.connectionStatus === 'healthy',
        )
        model.connectionStatus = success ? 'healthy' : 'error'
        model.lastTestAt = new Date().toISOString()
        model.lastTest = {
          testedAt: model.lastTestAt,
          success,
          message: success
            ? '模型能力测试正常'
            : '模型测试失败：服务商连接未通过',
        }
        model.updatedAt = model.lastTestAt
        appendAudit(db, admin, {
          action: 'model.test',
          resourceType: 'ai_model',
          resourceId: model.id,
          after: { connectionStatus: model.connectionStatus },
          result: success ? 'success' : 'failure',
          requestId: requestId(request),
        })
        return publicModel(model, db)
      })
    }),
  ),

  http.get('/api/manage/platform-model-bindings', () =>
    respond(() => {
      const { db } = dbAndAdmin('platform:manage')
      return db.bindings
    }),
  ),

  http.post('/api/manage/platform-model-bindings', ({ request }) =>
    respond(async () => {
      const payload = await readJson<{
        type: ModelType
        modelId: string | null
      }>(request)
      const { db, admin } = dbAndAdmin('platform:manage')
      return idempotentMutation(db, admin, request, payload, () => {
        if (!['chat', 'image'].includes(payload.type)) {
          throw new MockApiError(
            422,
            ErrorCode.InvalidPayload,
            '模型类型不正确',
          )
        }
        const key =
          payload.type === 'chat' ? 'chatModelId' : 'imageModelId'
        const previous = db.bindings[key]
        if (payload.modelId) {
          const model = db.models.find((item) => item.id === payload.modelId)
          const provider = model
            ? db.providers.find((item) => item.id === model.providerId)
            : undefined
          if (!model || model.type !== payload.type) {
            throw new MockApiError(
              422,
              ErrorCode.InvalidPayload,
              '模型不存在或类型不匹配',
            )
          }
          if (
            !model.enabled ||
            model.connectionStatus !== 'healthy' ||
            !provider?.enabled ||
            provider.connectionStatus !== 'healthy'
          ) {
            throw new MockApiError(
              409,
              ErrorCode.InvalidPayload,
              '模型及所属服务商必须启用并通过测试',
            )
          }
          if (
            model.type === 'image' &&
            (!model.capabilities.textToImage ||
              !model.capabilities.imageToImage)
          ) {
            throw new MockApiError(
              409,
              ErrorCode.InvalidPayload,
              '平台生图模型必须同时支持文生图和图生图',
            )
          }
          if (
            model.type === 'chat' &&
            (!model.capabilities.promptOptimization ||
              !model.capabilities.visionInput)
          ) {
            throw new MockApiError(
              409,
              ErrorCode.InvalidPayload,
              '平台对话模型必须同时支持提示词优化和图片理解',
            )
          }
        }
        db.bindings[key] = payload.modelId
        appendAudit(db, admin, {
          action: 'model.bind_platform',
          resourceType: 'platform_model_binding',
          resourceId: payload.type,
          before: { modelId: previous },
          after: { modelId: payload.modelId },
          result: 'success',
          requestId: requestId(request),
        })
        return db.bindings
      })
    }),
  ),
]
