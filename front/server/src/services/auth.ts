import { apiRequest } from '@/services/http'
import type { AdminLoginPayload, AdminSession } from '@/types/domain'

export const authApi = {
  login(payload: AdminLoginPayload, signal?: AbortSignal) {
    return apiRequest<AdminSession>({
      method: 'POST',
      url: '/manage/auth/login',
      data: payload,
      signal,
    })
  },

  session(signal?: AbortSignal) {
    return apiRequest<AdminSession>({
      method: 'GET',
      url: '/manage/auth/session',
      signal,
    })
  },

  logout() {
    return apiRequest<null>({
      method: 'POST',
      url: '/manage/auth/logout',
    })
  },
}
