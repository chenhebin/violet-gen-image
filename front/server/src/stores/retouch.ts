import { ref } from 'vue'
import { defineStore } from 'pinia'
import { retouchApi } from '@/services/retouch'
import { normalizeStoreError } from '@/stores/shared'
import type { AppError, PageResult } from '@/types/api'
import type {
  ManageRetouchTicket,
  ManageRetouchTicketSummary,
  RetouchTicketQuery,
} from '@/types/domain'

export const useRetouchStore = defineStore('retouch', () => {
  const tickets = ref<PageResult<ManageRetouchTicketSummary>>({
    items: [],
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: false,
  })
  const currentTicket = ref<ManageRetouchTicket | null>(null)
  const query = ref<RetouchTicketQuery>({})
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

  async function loadTickets(
    nextQuery: RetouchTicketQuery = query.value,
    signal?: AbortSignal,
  ) {
    query.value = { ...nextQuery }
    tickets.value = await execute(() => retouchApi.list(nextQuery, signal))
    return tickets.value
  }

  async function loadTicket(ticketId: string) {
    currentTicket.value = await execute(() => retouchApi.get(ticketId))
    return currentTicket.value
  }

  async function mutate(
    ticketId: string,
    operation: () => Promise<ManageRetouchTicket>,
  ) {
    const result = await execute(operation, true)
    currentTicket.value = result
    await loadTickets()
    return result
  }

  function quote(ticketId: string, credits: number, note?: string) {
    return mutate(ticketId, () => retouchApi.quote(ticketId, credits, note))
  }

  function start(ticketId: string) {
    return mutate(ticketId, () => retouchApi.start(ticketId))
  }

  function deliver(ticketId: string, files: File[], note?: string) {
    return mutate(ticketId, () => retouchApi.deliver(ticketId, files, note))
  }

  function reject(ticketId: string, reason: string) {
    return mutate(ticketId, () => retouchApi.reject(ticketId, reason))
  }

  function fail(ticketId: string, reason: string) {
    return mutate(ticketId, () => retouchApi.fail(ticketId, reason))
  }

  return {
    tickets,
    currentTicket,
    query,
    isLoading,
    isMutating,
    error,
    loadTickets,
    fetchTickets: loadTickets,
    loadTicket,
    quote,
    start,
    deliver,
    reject,
    fail,
  }
})

