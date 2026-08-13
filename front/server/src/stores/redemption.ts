import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { redemptionApi } from '@/services/redemption'
import { normalizeStoreError } from '@/stores/shared'
import type { AppError, PageResult } from '@/types/api'
import type {
  BulkMutationResult,
  CreateRedemptionBatchPayload,
  CreateRedemptionBatchResult,
  DisableRedemptionPayload,
  ExtendRedemptionPayload,
  RedemptionBatch,
  RedemptionBatchQuery,
  RedemptionCode,
  RedemptionCodeDetail,
  RedemptionCodeQuery,
  UpdateRedemptionBatchPayload,
} from '@/types/domain'

function emptyPage<T>(): PageResult<T> {
  return { items: [], page: 1, pageSize: 20, total: 0, hasMore: false }
}

export const useRedemptionStore = defineStore('redemption', () => {
  const codes = ref<PageResult<RedemptionCode>>(emptyPage())
  const batches = ref<PageResult<RedemptionBatch>>(emptyPage())
  const currentCode = ref<RedemptionCodeDetail | null>(null)
  const currentBatch = ref<RedemptionBatch | null>(null)
  const latestCreated = ref<CreateRedemptionBatchResult | null>(null)
  const revealedCodes = ref<Record<string, string>>({})
  const codeQuery = ref<RedemptionCodeQuery>({})
  const batchQuery = ref<RedemptionBatchQuery>({})
  const isLoading = ref(false)
  const isMutating = ref(false)
  const error = ref<AppError | null>(null)
  const loading = computed(() => isLoading.value)

  async function runLoad<T>(operation: () => Promise<T>): Promise<T> {
    isLoading.value = true
    error.value = null
    try {
      return await operation()
    } catch (cause) {
      error.value = normalizeStoreError(cause)
      throw error.value
    } finally {
      isLoading.value = false
    }
  }

  async function runMutation<T>(operation: () => Promise<T>): Promise<T> {
    isMutating.value = true
    error.value = null
    try {
      return await operation()
    } catch (cause) {
      error.value = normalizeStoreError(cause)
      throw error.value
    } finally {
      isMutating.value = false
    }
  }

  async function loadCodes(
    query: RedemptionCodeQuery = codeQuery.value,
    signal?: AbortSignal,
  ) {
    codeQuery.value = { ...query }
    codes.value = await runLoad(() => redemptionApi.listCodes(query, signal))
    return codes.value
  }

  async function loadBatches(
    query: RedemptionBatchQuery = batchQuery.value,
    signal?: AbortSignal,
  ) {
    batchQuery.value = { ...query }
    batches.value = await runLoad(() =>
      redemptionApi.listBatches(query, signal),
    )
    return batches.value
  }

  async function loadCode(codeId: string) {
    currentCode.value = await runLoad(() => redemptionApi.getCode(codeId))
    return currentCode.value
  }

  async function loadBatch(batchId: string) {
    currentBatch.value = await runLoad(() =>
      redemptionApi.getBatch(batchId),
    )
    return currentBatch.value
  }

  async function createBatch(payload: CreateRedemptionBatchPayload) {
    latestCreated.value = await runMutation(() =>
      redemptionApi.createBatch(payload),
    )
    await Promise.all([loadBatches(), loadCodes()])
    return latestCreated.value
  }

  async function updateBatch(
    batchId: string,
    payload: UpdateRedemptionBatchPayload,
  ) {
    const updated = await runMutation(() =>
      redemptionApi.updateBatch(batchId, payload),
    )
    const index = batches.value.items.findIndex((item) => item.id === batchId)
    if (index >= 0) batches.value.items[index] = updated
    if (currentBatch.value?.id === batchId) currentBatch.value = updated
    return updated
  }

  async function revealCode(codeId: string): Promise<string> {
    const result = await runMutation(() => redemptionApi.revealCode(codeId))
    revealedCodes.value[codeId] = result.fullCode
    return result.fullCode
  }

  async function revealBatch(batchId: string) {
    const result = await runMutation(() =>
      redemptionApi.revealBatch(batchId),
    )
    result.forEach((item) => {
      revealedCodes.value[item.id] = item.fullCode
    })
    return result
  }

  async function exportBatch(batchId: string) {
    return runMutation(() => redemptionApi.exportBatch(batchId))
  }

  async function disableCodes(
    payload: DisableRedemptionPayload,
  ): Promise<BulkMutationResult> {
    const result = await runMutation(() => redemptionApi.disable(payload))
    await Promise.all([loadCodes(), loadBatches()])
    return result
  }

  async function extendCodes(
    payload: ExtendRedemptionPayload,
  ): Promise<BulkMutationResult> {
    const result = await runMutation(() => redemptionApi.extend(payload))
    await Promise.all([loadCodes(), loadBatches()])
    return result
  }

  function clearSensitiveValues(): void {
    revealedCodes.value = {}
    latestCreated.value = null
  }

  return {
    codes,
    batches,
    currentCode,
    currentBatch,
    latestCreated,
    revealedCodes,
    codeQuery,
    batchQuery,
    isLoading,
    loading,
    isMutating,
    error,
    loadCodes,
    fetchCodes: loadCodes,
    loadBatches,
    fetchBatches: loadBatches,
    loadCode,
    loadBatch,
    createBatch,
    updateBatch,
    revealCode,
    revealBatch,
    exportBatch,
    disableCodes,
    extendCodes,
    clearSensitiveValues,
  }
})
