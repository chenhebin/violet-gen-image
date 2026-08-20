import { ref } from 'vue'
import { defineStore } from 'pinia'
import { assetApi } from '@/services/assets'
import { normalizeStoreError } from '@/stores/shared'
import type { AppError, PageResult } from '@/types/api'
import type { AssetQuery, ManagedAsset } from '@/types/domain'

export const useAssetStore = defineStore('assets', () => {
  const assets = ref<PageResult<ManagedAsset>>({
    items: [],
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: false,
  })
  const currentAsset = ref<ManagedAsset | null>(null)
  const query = ref<AssetQuery>({})
  const isLoading = ref(false)
  const isMutating = ref(false)
  const error = ref<AppError | null>(null)

  async function execute<T>(
    operation: () => Promise<T>,
    mutating = false,
  ): Promise<T> {
    ;(mutating ? isMutating : isLoading).value = true
    error.value = null
    try {
      return await operation()
    } catch (cause) {
      error.value = normalizeStoreError(cause)
      throw error.value
    } finally {
      ;(mutating ? isMutating : isLoading).value = false
    }
  }

  async function loadAssets(
    nextQuery: AssetQuery = query.value,
    signal?: AbortSignal,
  ) {
    query.value = { ...nextQuery }
    assets.value = await execute(() => assetApi.list(nextQuery, signal))
    return assets.value
  }

  async function loadAsset(assetId: string) {
    currentAsset.value = await execute(() => assetApi.get(assetId))
    return currentAsset.value
  }

  async function getSignedUrl(assetId: string) {
    return execute(() => assetApi.getSignedUrl(assetId), true)
  }

  async function refreshPreview(asset: ManagedAsset) {
    const result = await execute(() => assetApi.getUrl(asset.id))
    asset.previewUrl = result.url
    asset.previewUrlExpiresAt = result.expiresAt
    if (currentAsset.value?.id === asset.id) {
      currentAsset.value.previewUrl = result.url
      currentAsset.value.previewUrlExpiresAt = result.expiresAt
    }
    return result
  }

  async function setRetained(
    assetId: string,
    retained: boolean,
    reason: string,
  ) {
    const result = await execute(
      () => assetApi.setRetained(assetId, retained, reason),
      true,
    )
    await loadAssets()
    return result
  }

  async function cleanup(assetId: string, reason: string) {
    const result = await execute(
      () => assetApi.cleanup(assetId, reason),
      true,
    )
    await loadAssets()
    return result
  }

  return {
    assets,
    currentAsset,
    query,
    isLoading,
    isMutating,
    error,
    loadAssets,
    fetchAssets: loadAssets,
    loadAsset,
    getSignedUrl,
    refreshPreview,
    setRetained,
    cleanup,
  }
})
