import {
  apiRequest,
  mutationHeaders,
} from '@/services/http'
import { NETWORK_CONFIG } from '@/config'
import type {
  AIModel,
  AIProvider,
  CreateAIModelPayload,
  CreateAIProviderPayload,
  ModelType,
  PlatformModelBindings,
  UpdateAIModelPayload,
  UpdateAIProviderPayload,
} from '@/types/domain'

export const providerApi = {
  listProviders(signal?: AbortSignal) {
    return apiRequest<AIProvider[]>({
      method: 'GET',
      url: '/manage/ai-providers',
      signal,
    })
  },

  createProvider(payload: CreateAIProviderPayload) {
    return apiRequest<AIProvider>({
      method: 'POST',
      url: '/manage/ai-providers',
      data: payload,
      headers: mutationHeaders('create_ai_provider'),
    })
  },

  updateProvider(providerId: string, payload: UpdateAIProviderPayload) {
    return apiRequest<AIProvider>({
      method: 'PATCH',
      url: `/manage/ai-providers/${providerId}`,
      data: payload,
      headers: mutationHeaders('update_ai_provider'),
    })
  },

  deleteProvider(providerId: string) {
    return apiRequest<null>({
      method: 'DELETE',
      url: `/manage/ai-providers/${providerId}`,
      headers: mutationHeaders('delete_ai_provider'),
    })
  },

  testProvider(providerId: string) {
    return apiRequest<AIProvider>({
      method: 'POST',
      url: `/manage/ai-providers/${providerId}/test`,
      headers: mutationHeaders('test_ai_provider'),
    })
  },

  rotateKey(providerId: string, apiKey: string) {
    return apiRequest<AIProvider>({
      method: 'POST',
      url: `/manage/ai-providers/${providerId}/rotate-key`,
      data: { apiKey },
      headers: mutationHeaders('rotate_provider_key'),
    })
  },

  listModels(providerId?: string, signal?: AbortSignal) {
    return apiRequest<AIModel[]>({
      method: 'GET',
      url: '/manage/ai-models',
      params: { providerId },
      signal,
    })
  },

  createModel(payload: CreateAIModelPayload) {
    return apiRequest<AIModel>({
      method: 'POST',
      url: '/manage/ai-models',
      data: payload,
      headers: mutationHeaders('create_ai_model'),
    })
  },

  updateModel(modelId: string, payload: UpdateAIModelPayload) {
    return apiRequest<AIModel>({
      method: 'PATCH',
      url: `/manage/ai-models/${modelId}`,
      data: payload,
      headers: mutationHeaders('update_ai_model'),
    })
  },

  deleteModel(modelId: string) {
    return apiRequest<null>({
      method: 'DELETE',
      url: `/manage/ai-models/${modelId}`,
      headers: mutationHeaders('delete_ai_model'),
    })
  },

  testModel(modelId: string) {
    return apiRequest<AIModel>({
      method: 'POST',
      url: `/manage/ai-models/${modelId}/test`,
      headers: mutationHeaders('test_ai_model'),
      timeout: NETWORK_CONFIG.aiModelTestTimeoutMs,
    })
  },

  getBindings(signal?: AbortSignal) {
    return apiRequest<PlatformModelBindings>({
      method: 'GET',
      url: '/manage/platform-model-bindings',
      signal,
    })
  },

  bindModel(type: ModelType, modelId: string | null) {
    return apiRequest<PlatformModelBindings>({
      method: 'POST',
      url: '/manage/platform-model-bindings',
      data: { type, modelId },
      headers: mutationHeaders('bind_platform_model'),
    })
  },
}
