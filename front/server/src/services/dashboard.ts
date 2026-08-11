import { apiRequest } from '@/services/http'
import type { DashboardData } from '@/types/domain'

export const dashboardApi = {
  get(signal?: AbortSignal) {
    return apiRequest<DashboardData>({
      method: 'GET',
      url: '/manage/dashboard',
      signal,
    })
  },
}

