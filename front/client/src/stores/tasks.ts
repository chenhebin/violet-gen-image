import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { isFinalTaskStatus, TASK_TIMING } from '@/config'
import { taskApi } from '@/services/api'
import { AppError, ErrorCode } from '@/types/api'
import type { GenerationTask } from '@/types/domain'

function isCanceledRequest(caught: unknown): boolean {
  return (
    typeof caught === 'object' &&
    caught !== null &&
    'code' in caught &&
    caught.code === 'ERR_CANCELED'
  )
}

export const useTaskStore = defineStore('tasks', () => {
  const tasks = ref<GenerationTask[]>([])
  const activeTask = ref<GenerationTask | null>(null)
  const loading = ref(false)
  const error = ref('')
  const syncError = ref('')
  const page = ref(1)
  const pageSize = ref(20)
  const total = ref(0)
  const hasMore = ref(false)
  let listController: AbortController | null = null
  let taskController: AbortController | null = null
  let pollTimer: number | null = null

  const hasRunningTasks = computed(() =>
    tasks.value.some((task) => !isFinalTaskStatus(task.status)),
  )

  function upsert(task: GenerationTask): void {
    const index = tasks.value.findIndex((item) => item.id === task.id)
    if (index >= 0) tasks.value[index] = task
    else tasks.value.unshift(task)
    if (activeTask.value?.id === task.id) activeTask.value = task
  }

  async function load(options: { page?: number; silent?: boolean } = {}): Promise<void> {
    listController?.abort()
    listController = new AbortController()
    const targetPage = options.page ?? page.value
    loading.value = !options.silent
    error.value = ''
    try {
      const result = await taskApi.list(
        { page: targetPage, pageSize: pageSize.value },
        listController.signal,
      )
      tasks.value = result.items
      page.value = result.page
      pageSize.value = result.pageSize
      total.value = result.total
      hasMore.value = result.hasMore
    } catch (caught) {
      if (
        typeof caught === 'object' &&
        caught !== null &&
        'code' in caught &&
        caught.code === 'ERR_CANCELED'
      ) {
        return
      }
      error.value = caught instanceof Error ? caught.message : '任务加载失败'
      throw caught
    } finally {
      if (!options.silent) loading.value = false
    }
  }

  async function open(taskId: string): Promise<GenerationTask> {
    taskController?.abort()
    taskController = new AbortController()
    loading.value = true
    try {
      const task = await taskApi.get(taskId, taskController.signal)
      activeTask.value = task
      upsert(task)
      return task
    } finally {
      loading.value = false
    }
  }

  function close(): void {
    taskController?.abort()
    activeTask.value = null
  }

  async function cancel(taskId: string): Promise<void> {
    const task = await taskApi.cancel(taskId)
    upsert(task)
  }

  function monitor(
    taskId: string,
    onUpdate?: (task: GenerationTask) => void,
    onSyncError?: (message: string) => void,
  ): void {
    stopMonitoring()
    syncError.value = ''
    let failedPolls = 0
    const tick = async () => {
      try {
        const task = await taskApi.get(taskId)
        failedPolls = 0
        syncError.value = ''
        upsert(task)
        onUpdate?.(task)
        if (isFinalTaskStatus(task.status)) {
          stopMonitoring()
          return
        }
      } catch (caught) {
        if (isCanceledRequest(caught)) return
        if (
          caught instanceof AppError &&
          (caught.code === ErrorCode.AuthRequired || caught.code === ErrorCode.AccountDisabled)
        ) {
          stopMonitoring()
          syncError.value = '登录状态已失效，请重新登录'
          onSyncError?.(syncError.value)
          return
        }
        failedPolls += 1
        if (failedPolls >= TASK_TIMING.monitorRetryLimit) {
          const message = '任务状态同步失败，请手动刷新重试'
          syncError.value = message
          stopMonitoring()
          onSyncError?.(message)
          return
        }
        const retryDelay = Math.min(
          TASK_TIMING.monitorRetryBaseMs * 2 ** (failedPolls - 1),
          TASK_TIMING.monitorRetryMaxMs,
        )
        pollTimer = window.setTimeout(tick, retryDelay)
        return
      }
      pollTimer = window.setTimeout(tick, TASK_TIMING.monitorPollMs)
    }
    void tick()
  }

  function stopMonitoring(): void {
    if (pollTimer !== null) window.clearTimeout(pollTimer)
    pollTimer = null
  }

  return {
    tasks,
    activeTask,
    loading,
    error,
    hasRunningTasks,
    syncError,
    page,
    pageSize,
    total,
    hasMore,
    load,
    open,
    close,
    cancel,
    monitor,
    stopMonitoring,
    upsert,
  }
})
