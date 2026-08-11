import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { isFinalTaskStatus, TASK_TIMING } from '@/config'
import { taskApi } from '@/services/api'
import type { GenerationTask } from '@/types/domain'

export const useTaskStore = defineStore('tasks', () => {
  const tasks = ref<GenerationTask[]>([])
  const activeTask = ref<GenerationTask | null>(null)
  const loading = ref(false)
  const error = ref('')
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

  async function load(): Promise<void> {
    listController?.abort()
    listController = new AbortController()
    loading.value = true
    error.value = ''
    try {
      tasks.value = await taskApi.list(listController.signal)
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
      loading.value = false
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
  ): void {
    stopMonitoring()
    const tick = async () => {
      try {
        const task = await taskApi.get(taskId)
        upsert(task)
        onUpdate?.(task)
        if (isFinalTaskStatus(task.status)) {
          stopMonitoring()
          return
        }
      } catch {
        stopMonitoring()
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
    load,
    open,
    close,
    cancel,
    monitor,
    stopMonitoring,
    upsert,
  }
})
