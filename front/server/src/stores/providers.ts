import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { providerApi } from '@/services/providers'
import { normalizeStoreError } from '@/stores/shared'
import type { AppError } from '@/types/api'
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

export const useProviderStore = defineStore('providers', () => {
  const providers = ref<AIProvider[]>([])
  const models = ref<AIModel[]>([])
  const bindings = ref<PlatformModelBindings>({
    chatModelId: null,
    imageModelId: null,
  })
  const isLoading = ref(false)
  const isMutating = ref(false)
  const error = ref<AppError | null>(null)
  const loading = computed(() => isLoading.value)
  const currentChatModel = computed(
    () => models.value.find((item) => item.id === bindings.value.chatModelId),
  )
  const currentImageModel = computed(
    () => models.value.find((item) => item.id === bindings.value.imageModelId),
  )

  async function run<T>(
    operation: () => Promise<T>,
    mutating = false,
  ): Promise<T> {
    const target = mutating ? isMutating : isLoading
    target.value = true
    error.value = null
    try {
      return await operation()
    } catch (cause) {
      error.value = normalizeStoreError(cause)
      throw error.value
    } finally {
      target.value = false
    }
  }

  async function loadAll(signal?: AbortSignal) {
    const [providerResult, modelResult, bindingResult] = await run(() =>
      Promise.all([
        providerApi.listProviders(signal),
        providerApi.listModels(undefined, signal),
        providerApi.getBindings(signal),
      ]),
    )
    providers.value = providerResult
    models.value = modelResult
    bindings.value = bindingResult
  }

  async function refresh(): Promise<void> {
    await loadAll()
  }

  async function createProvider(payload: CreateAIProviderPayload) {
    const result = await run(
      () => providerApi.createProvider(payload),
      true,
    )
    providers.value.unshift(result)
    return result
  }

  async function updateProvider(
    providerId: string,
    payload: UpdateAIProviderPayload,
  ) {
    const result = await run(
      () => providerApi.updateProvider(providerId, payload),
      true,
    )
    replaceProvider(result)
    await loadModels()
    return result
  }

  async function deleteProvider(providerId: string) {
    await run(() => providerApi.deleteProvider(providerId), true)
    providers.value = providers.value.filter((item) => item.id !== providerId)
    models.value = models.value.filter((item) => item.providerId !== providerId)
  }

  async function testProvider(providerId: string) {
    const result = await run(
      () => providerApi.testProvider(providerId),
      true,
    )
    replaceProvider(result)
    return result
  }

  async function rotateKey(providerId: string, apiKey: string) {
    const result = await run(
      () => providerApi.rotateKey(providerId, apiKey),
      true,
    )
    replaceProvider(result)
    await loadModels()
    return result
  }

  async function loadModels(providerId?: string) {
    models.value = await run(() => providerApi.listModels(providerId))
    if (providerId) {
      const untouched = models.value.filter(
        (item) => item.providerId !== providerId,
      )
      const scoped = await providerApi.listModels(providerId)
      models.value = [...untouched, ...scoped]
    }
    return models.value
  }

  async function createModel(payload: CreateAIModelPayload) {
    const result = await run(() => providerApi.createModel(payload), true)
    models.value.unshift(result)
    return result
  }

  async function updateModel(
    modelId: string,
    payload: UpdateAIModelPayload,
  ) {
    const result = await run(
      () => providerApi.updateModel(modelId, payload),
      true,
    )
    replaceModel(result)
    bindings.value = await providerApi.getBindings()
    return result
  }

  async function deleteModel(modelId: string) {
    await run(() => providerApi.deleteModel(modelId), true)
    models.value = models.value.filter((item) => item.id !== modelId)
  }

  async function testModel(modelId: string) {
    const result = await run(() => providerApi.testModel(modelId), true)
    replaceModel(result)
    return result
  }

  async function bindModel(type: ModelType, modelId: string | null) {
    bindings.value = await run(
      () => providerApi.bindModel(type, modelId),
      true,
    )
    models.value = await providerApi.listModels()
    return bindings.value
  }

  function replaceProvider(provider: AIProvider): void {
    const index = providers.value.findIndex((item) => item.id === provider.id)
    if (index >= 0) providers.value[index] = provider
    else providers.value.unshift(provider)
  }

  function replaceModel(model: AIModel): void {
    const index = models.value.findIndex((item) => item.id === model.id)
    if (index >= 0) models.value[index] = model
    else models.value.unshift(model)
  }

  return {
    providers,
    models,
    bindings,
    isLoading,
    loading,
    isMutating,
    error,
    currentChatModel,
    currentImageModel,
    loadAll,
    fetchAll: loadAll,
    refresh,
    loadModels,
    createProvider,
    updateProvider,
    deleteProvider,
    testProvider,
    rotateKey,
    createModel,
    updateModel,
    deleteModel,
    testModel,
    bindModel,
  }
})
