import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { taskApi } from '@/services/api'
import { useTaskStore } from '@/stores/tasks'

describe('task monitor recovery', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('retries transient polling failures and reports a terminal sync error', async () => {
    const get = vi.spyOn(taskApi, 'get').mockRejectedValue(new Error('offline'))
    const onSyncError = vi.fn()
    const store = useTaskStore()

    store.monitor('task-1', undefined, onSyncError)
    await vi.runOnlyPendingTimersAsync()
    await vi.runOnlyPendingTimersAsync()
    await vi.runOnlyPendingTimersAsync()

    expect(get).toHaveBeenCalledTimes(3)
    expect(store.syncError).toBe('任务状态同步失败，请手动刷新重试')
    expect(onSyncError).toHaveBeenCalledOnce()
  })
})
