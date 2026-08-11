import { apiRequest, mutationHeaders } from '@/services/http'

export const demoApi = {
  reset(): Promise<null> {
    return apiRequest<null>({
      method: 'POST',
      url: '/manage/mock/reset',
      headers: mutationHeaders('reset_demo_data'),
    })
  },
}

