import {
  apiRequest,
  mutationHeaders,
} from '@/services/http'
import type { PageResult } from '@/types/api'
import type {
  AssetQuery,
  ManagedAsset,
} from '@/types/domain'

export const assetApi = {
  list(query: AssetQuery = {}, signal?: AbortSignal) {
    return apiRequest<PageResult<ManagedAsset>>({
      method: 'GET',
      url: '/manage/assets',
      params: query,
      signal,
    })
  },

  get(assetId: string, signal?: AbortSignal) {
    return apiRequest<ManagedAsset>({
      method: 'GET',
      url: `/manage/assets/${assetId}`,
      signal,
    })
  },

  getSignedUrl(assetId: string) {
    return apiRequest<{ url: string; expiresAt: string }>({
      method: 'POST',
      url: `/manage/assets/${assetId}/signed-url`,
      headers: mutationHeaders('sign_asset_url'),
    })
  },

	getUrl(assetId: string, purpose: 'preview' | 'download' = 'preview') {
		return apiRequest<{ url: string; expiresAt: string }>({
			method: 'GET',
			url: `/manage/assets/${assetId}/url`,
			params: { purpose },
		})
	},

  setRetained(assetId: string, retained: boolean, reason: string) {
    return apiRequest<ManagedAsset>({
      method: 'POST',
      url: `/manage/assets/${assetId}/retain`,
      data: { retained, reason },
      headers: mutationHeaders('retain_asset'),
    })
  },

  cleanup(assetId: string, reason: string) {
    return apiRequest<ManagedAsset>({
      method: 'POST',
      url: `/manage/assets/${assetId}/cleanup`,
      data: { reason },
      headers: mutationHeaders('cleanup_asset'),
    })
  },
}
