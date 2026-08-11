import { ref } from 'vue'
import { defineStore } from 'pinia'
import { auditApi } from '@/services/audits'
import { normalizeStoreError } from '@/stores/shared'
import type { AppError, PageResult } from '@/types/api'
import type { AuditEvent, AuditQuery } from '@/types/domain'

export const useAuditStore = defineStore('audits', () => {
  const events = ref<PageResult<AuditEvent>>({
    items: [],
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: false,
  })
  const query = ref<AuditQuery>({})
  const isLoading = ref(false)
  const error = ref<AppError | null>(null)

  async function loadAudits(
    nextQuery: AuditQuery = query.value,
    signal?: AbortSignal,
  ) {
    isLoading.value = true
    error.value = null
    query.value = { ...nextQuery }
    try {
      events.value = await auditApi.list(nextQuery, signal)
      return events.value
    } catch (cause) {
      error.value = normalizeStoreError(cause)
      throw error.value
    } finally {
      isLoading.value = false
    }
  }

  return {
    events,
    audits: events,
    query,
    isLoading,
    error,
    loadAudits,
    fetchAudits: loadAudits,
  }
})

