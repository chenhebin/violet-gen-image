import { ref } from 'vue'
import { defineStore } from 'pinia'
import { taskApi } from '@/services/tasks'
import { normalizeStoreError } from '@/stores/shared'
import type { AppError, PageResult } from '@/types/api'
import type {
  ManagedGenerationTask,
  ManagedGenerationTaskSummary,
  ManagedTaskQuery,
} from '@/types/domain'

export const useTaskStore = defineStore('tasks', () => {
  const tasks = ref<PageResult<ManagedGenerationTaskSummary>>({
    items: [],
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: false,
  })
  const currentTask = ref<ManagedGenerationTask | null>(null)
  const query = ref<ManagedTaskQuery>({})
  const isLoading = ref(false)
  const error = ref<AppError | null>(null)

  async function execute<T>(operation: () => Promise<T>): Promise<T> {
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

  async function loadTasks(
    nextQuery: ManagedTaskQuery = query.value,
    signal?: AbortSignal,
  ) {
    query.value = { ...nextQuery }
    tasks.value = await execute(() => taskApi.list(nextQuery, signal))
    return tasks.value
  }

  async function loadTask(taskId: string) {
    currentTask.value = await execute(() => taskApi.get(taskId))
    return currentTask.value
  }

  return {
    tasks,
    currentTask,
    query,
    isLoading,
    error,
    loadTasks,
    fetchTasks: loadTasks,
    loadTask,
  }
})

