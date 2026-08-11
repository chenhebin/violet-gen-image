import { ref } from 'vue'
import { defineStore } from 'pinia'
import { userApi } from '@/services/users'
import { normalizeStoreError } from '@/stores/shared'
import type { AppError, PageResult } from '@/types/api'
import type {
  AdjustCreditsPayload,
  ManagedUser,
  ManagedUserDetail,
  ManagedUserQuery,
  UserStatus,
} from '@/types/domain'

const emptyPage = (): PageResult<ManagedUser> => ({
  items: [],
  page: 1,
  pageSize: 20,
  total: 0,
  hasMore: false,
})

export const useUserStore = defineStore('users', () => {
  const users = ref(emptyPage())
  const currentUser = ref<ManagedUserDetail | null>(null)
  const query = ref<ManagedUserQuery>({})
  const isLoading = ref(false)
  const isMutating = ref(false)
  const error = ref<AppError | null>(null)

  async function execute<T>(
    operation: () => Promise<T>,
    mutating = false,
  ): Promise<T> {
    ;(mutating ? isMutating : isLoading).value = true
    error.value = null
    try {
      return await operation()
    } catch (cause) {
      error.value = normalizeStoreError(cause)
      throw error.value
    } finally {
      ;(mutating ? isMutating : isLoading).value = false
    }
  }

  async function loadUsers(
    nextQuery: ManagedUserQuery = query.value,
    signal?: AbortSignal,
  ) {
    query.value = { ...nextQuery }
    users.value = await execute(() => userApi.list(nextQuery, signal))
    return users.value
  }

  async function loadUser(userId: string) {
    currentUser.value = await execute(() => userApi.get(userId))
    return currentUser.value
  }

  async function setStatus(
    userId: string,
    status: UserStatus,
    reason: string,
  ) {
    const user = await execute(
      () => userApi.setStatus(userId, status, reason),
      true,
    )
    await loadUsers()
    if (currentUser.value?.id === userId) await loadUser(userId)
    return user
  }

  async function resetPassword(userId: string) {
    return execute(() => userApi.resetPassword(userId), true)
  }

  async function adjustCredits(
    userId: string,
    payload: AdjustCreditsPayload,
  ) {
    const result = await execute(
      () => userApi.adjustCredits(userId, payload),
      true,
    )
    await loadUsers()
    if (currentUser.value?.id === userId) await loadUser(userId)
    return result
  }

  return {
    users,
    currentUser,
    query,
    isLoading,
    isMutating,
    error,
    loadUsers,
    fetchUsers: loadUsers,
    loadUser,
    setStatus,
    resetPassword,
    adjustCredits,
  }
})

