import {
  apiRequest,
  mutationHeaders,
} from '@/services/http'
import type { PageResult } from '@/types/api'
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

export type RedemptionExportFormat = 'csv' | 'xianyu'

export interface RedemptionBatchExport {
  filename: string
  content: string
  csv?: string
  mediaType: string
  format: RedemptionExportFormat
  count: number
}

export const redemptionApi = {
  listCodes(query: RedemptionCodeQuery = {}, signal?: AbortSignal) {
    return apiRequest<PageResult<RedemptionCode>>({
      method: 'GET',
      url: '/manage/redemption-codes',
      params: query,
      signal,
    })
  },

  getCode(codeId: string, signal?: AbortSignal) {
    return apiRequest<RedemptionCodeDetail>({
      method: 'GET',
      url: `/manage/redemption-codes/${codeId}`,
      signal,
    })
  },

  listBatches(query: RedemptionBatchQuery = {}, signal?: AbortSignal) {
    return apiRequest<PageResult<RedemptionBatch>>({
      method: 'GET',
      url: '/manage/redemption-batches',
      params: query,
      signal,
    })
  },

  getBatch(batchId: string, signal?: AbortSignal) {
    return apiRequest<RedemptionBatch>({
      method: 'GET',
      url: `/manage/redemption-batches/${batchId}`,
      signal,
    })
  },

  createBatch(payload: CreateRedemptionBatchPayload) {
    return apiRequest<CreateRedemptionBatchResult>({
      method: 'POST',
      url: '/manage/redemption-batches',
      data: payload,
      headers: mutationHeaders('create_redemption_batch'),
    })
  },

  updateBatch(batchId: string, payload: UpdateRedemptionBatchPayload) {
    return apiRequest<RedemptionBatch>({
      method: 'PATCH',
      url: `/manage/redemption-batches/${batchId}`,
      data: payload,
      headers: mutationHeaders('rename_redemption_batch'),
    })
  },

  revealCode(codeId: string) {
    return apiRequest<{ id: string; fullCode: string }>({
      method: 'POST',
      url: `/manage/redemption-codes/${codeId}/reveal`,
      headers: mutationHeaders('reveal_redemption_code'),
    })
  },

  revealBatch(batchId: string) {
    return apiRequest<Array<{ id: string; fullCode: string }>>({
      method: 'POST',
      url: `/manage/redemption-batches/${batchId}/reveal`,
      headers: mutationHeaders('reveal_redemption_batch'),
    })
  },

  exportBatch(batchId: string, format: RedemptionExportFormat = 'csv') {
    return apiRequest<RedemptionBatchExport>({
      method: 'POST',
      url: `/manage/redemption-batches/${batchId}/export`,
      params: { format },
      headers: mutationHeaders('export_redemption_batch'),
    })
  },

  disable(payload: DisableRedemptionPayload) {
    return apiRequest<BulkMutationResult>({
      method: 'POST',
      url: '/manage/redemption-codes/disable',
      data: payload,
      headers: mutationHeaders('disable_redemption_codes'),
    })
  },

  extend(payload: ExtendRedemptionPayload) {
    return apiRequest<BulkMutationResult>({
      method: 'POST',
      url: '/manage/redemption-codes/extend',
      data: payload,
      headers: mutationHeaders('extend_redemption_codes'),
    })
  },
}
