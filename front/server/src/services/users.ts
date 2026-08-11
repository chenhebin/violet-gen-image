import {
  apiRequest,
  mutationHeaders,
} from '@/services/http'
import type { PageResult } from '@/types/api'
import type {
  AdjustCreditsPayload,
  AdjustmentLedger,
  ManagedUser,
  ManagedUserDetail,
  ManagedUserQuery,
  ResetPasswordResult,
  UserStatus,
} from '@/types/domain'

export const userApi = {
  list(query: ManagedUserQuery = {}, signal?: AbortSignal) {
    return apiRequest<PageResult<ManagedUser>>({
      method: 'GET',
      url: '/manage/users',
      params: query,
      signal,
    })
  },

  get(userId: string, signal?: AbortSignal) {
    return apiRequest<ManagedUserDetail>({
      method: 'GET',
      url: `/manage/users/${userId}`,
      signal,
    })
  },

  setStatus(userId: string, status: UserStatus, reason: string) {
    return apiRequest<ManagedUser>({
      method: 'POST',
      url: `/manage/users/${userId}/status`,
      data: { status, reason },
      headers: mutationHeaders('set_user_status'),
    })
  },

  resetPassword(userId: string) {
    return apiRequest<ResetPasswordResult>({
      method: 'POST',
      url: `/manage/users/${userId}/reset-password`,
      headers: mutationHeaders('reset_user_password'),
    })
  },

  adjustCredits(userId: string, payload: AdjustCreditsPayload) {
    return apiRequest<{
      user: ManagedUser
      ledger: AdjustmentLedger
    }>({
      method: 'POST',
      url: `/manage/users/${userId}/adjust-credits`,
      data: payload,
      headers: mutationHeaders('adjust_user_credits'),
    })
  },

  listLedger(
    userId?: string,
    params: { page?: number; pageSize?: number } = {},
    signal?: AbortSignal,
  ) {
    return apiRequest<PageResult<AdjustmentLedger>>({
      method: 'GET',
      url: '/manage/usage-ledger',
      params: { ...params, userId },
      signal,
    })
  },
}

