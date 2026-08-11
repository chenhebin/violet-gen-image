import { apiRequest } from '@/services/http'
import type { PageResult } from '@/types/api'
import type {
  ManagedGenerationTask,
  ManagedGenerationTaskSummary,
  ManagedTaskQuery,
} from '@/types/domain'

export const taskApi = {
  list(query: ManagedTaskQuery = {}, signal?: AbortSignal) {
    return apiRequest<PageResult<ManagedGenerationTaskSummary>>({
      method: 'GET',
      url: '/manage/generation-tasks',
      params: query,
      signal,
    })
  },

  get(taskId: string, signal?: AbortSignal) {
    return apiRequest<ManagedGenerationTask>({
      method: 'GET',
      url: `/manage/generation-tasks/${taskId}`,
      signal,
    })
  },
}

