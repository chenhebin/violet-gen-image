import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { authApi } from '@/services/api'
import { AppError, ErrorCode } from '@/types/api'
import type { AuthPayload, RegisterPayload, User } from '@/types/domain'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const initialized = ref(false)
  const loading = ref(false)
  const error = ref('')

  const isAuthenticated = computed(() => Boolean(user.value))

  async function restore(): Promise<void> {
    if (initialized.value) return
    loading.value = true
    try {
      user.value = await authApi.session()
    } catch (caught) {
      if (!(caught instanceof AppError) || caught.code !== ErrorCode.AuthRequired) {
        error.value = caught instanceof Error ? caught.message : '无法恢复登录状态'
      }
      user.value = null
    } finally {
      initialized.value = true
      loading.value = false
    }
  }

  async function login(payload: AuthPayload): Promise<void> {
    error.value = ''
    loading.value = true
    try {
      user.value = await authApi.login(payload)
      initialized.value = true
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : '登录失败'
      throw caught
    } finally {
      loading.value = false
    }
  }

  async function register(payload: RegisterPayload): Promise<void> {
    error.value = ''
    loading.value = true
    try {
      user.value = await authApi.register(payload)
      initialized.value = true
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : '注册失败'
      throw caught
    } finally {
      loading.value = false
    }
  }

  async function logout(): Promise<void> {
    loading.value = true
    try {
      await authApi.logout()
    } finally {
      user.value = null
      initialized.value = true
      loading.value = false
    }
  }

  function invalidateSession(): void {
    user.value = null
    initialized.value = true
    loading.value = false
  }

  function clearError(): void {
    error.value = ''
  }

  return {
    user,
    initialized,
    loading,
    error,
    isAuthenticated,
    restore,
    login,
    register,
    logout,
    invalidateSession,
    clearError,
  }
})
