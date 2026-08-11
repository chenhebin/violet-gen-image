import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { dashboardApi } from '@/services/dashboard'
import { normalizeStoreError } from '@/stores/shared'
import type { AppError } from '@/types/api'
import type { DashboardData } from '@/types/domain'

export const useDashboardStore = defineStore('dashboard', () => {
  const data = ref<DashboardData | null>(null)
  const isLoading = ref(false)
  const error = ref<AppError | null>(null)
  const loading = computed(() => isLoading.value)

  async function loadDashboard(signal?: AbortSignal): Promise<DashboardData> {
    isLoading.value = true
    error.value = null
    try {
      data.value = await dashboardApi.get(signal)
      return data.value
    } catch (cause) {
      error.value = normalizeStoreError(cause)
      throw error.value
    } finally {
      isLoading.value = false
    }
  }

  return {
    data,
    isLoading,
    loading,
    error,
    loadDashboard,
    load: loadDashboard,
  }
})

