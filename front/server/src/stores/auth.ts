import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { authApi } from '@/services/auth'
import { clearCsrfToken } from '@/services/http'
import type { AppError } from '@/types/api'
import type {
  AdminLoginPayload,
  AdminPermission,
  AdminSession,
} from '@/types/domain'
import { hasPermission as sessionHasPermission } from '@/utils/permissions'
import { normalizeStoreError } from '@/stores/shared'

export const useAuthStore = defineStore('auth', () => {
  const session = ref<AdminSession | null>(null)
  const isLoading = ref(false)
  const error = ref<AppError | null>(null)

  const isAuthenticated = computed(() => Boolean(session.value))
  const isPlatformAdmin = computed(
    () => session.value?.role === 'platform_admin',
  )
  const permissions = computed(() => session.value?.permissions ?? [])
  const loading = computed(() => isLoading.value)

  async function login(payload: AdminLoginPayload): Promise<AdminSession> {
    clearCsrfToken()
    isLoading.value = true
    error.value = null
    try {
      session.value = await authApi.login(payload)
      return session.value
    } catch (cause) {
      clearCsrfToken()
      error.value = normalizeStoreError(cause)
      throw error.value
    } finally {
      isLoading.value = false
    }
  }

  async function restoreSession(): Promise<AdminSession | null> {
    isLoading.value = true
    error.value = null
    try {
      session.value = await authApi.session()
      return session.value
    } catch (cause) {
      const normalized = normalizeStoreError(cause)
      session.value = null
      clearCsrfToken()
      if (normalized.code !== 1001) error.value = normalized
      return null
    } finally {
      isLoading.value = false
    }
  }

  async function logout(): Promise<void> {
    isLoading.value = true
    try {
      await authApi.logout()
    } finally {
      session.value = null
      clearCsrfToken()
      isLoading.value = false
    }
  }

  function hasPermission(permission: AdminPermission): boolean {
    return sessionHasPermission(session.value, permission)
  }

  function clearError(): void {
    error.value = null
  }

  function invalidateSession(): void {
    session.value = null
    error.value = null
    clearCsrfToken()
  }

  return {
    session,
    isLoading,
    loading,
    error,
    isAuthenticated,
    isPlatformAdmin,
    permissions,
    login,
    restoreSession,
    restore: restoreSession,
    logout,
    hasPermission,
    clearError,
    invalidateSession,
  }
})
