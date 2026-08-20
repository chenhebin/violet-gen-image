import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { isFinalRetouchTicketStatus } from '@/config'
import {
  assetApi,
  retouchTicketApi,
  type CreateRetouchTicketPayload,
} from '@/services/api'
import { useEntitlementStore } from '@/stores/entitlement'
import type { Asset, RetouchTicket } from '@/types/domain'

function isCanceledRequest(caught: unknown): boolean {
  return (
    typeof caught === 'object' &&
    caught !== null &&
    'code' in caught &&
    caught.code === 'ERR_CANCELED'
  )
}

export const useRetouchStore = defineStore('retouch', () => {
  const tickets = ref<RetouchTicket[]>([])
  const activeTicket = ref<RetouchTicket | null>(null)
  const listLoading = ref(false)
  const detailLoading = ref(false)
  const actionLoading = ref(false)
  const error = ref('')
  const page = ref(1)
  const pageSize = ref(20)
  const total = ref(0)
  const hasMore = ref(false)
  let listController: AbortController | null = null
  let detailController: AbortController | null = null
  let actionController: AbortController | null = null

  const loading = computed(
    () => listLoading.value || detailLoading.value,
  )
  const hasActiveTickets = computed(() =>
    tickets.value.some(
      (ticket) => !isFinalRetouchTicketStatus(ticket.status),
    ),
  )

  function upsert(ticket: RetouchTicket): void {
    const index = tickets.value.findIndex((item) => item.id === ticket.id)
    if (index >= 0) tickets.value[index] = ticket
    else tickets.value.unshift(ticket)
    tickets.value.sort(
      (left, right) =>
        new Date(right.updatedAt).getTime() -
        new Date(left.updatedAt).getTime(),
    )
    if (activeTicket.value?.id === ticket.id) activeTicket.value = ticket
  }

  async function load(options: { page?: number; silent?: boolean } = {}): Promise<void> {
    listController?.abort()
    const controller = new AbortController()
    listController = controller
    listLoading.value = true
    error.value = ''
    try {
      const result = await retouchTicketApi.list(
        { page: options.page ?? page.value, pageSize: pageSize.value },
        controller.signal,
      )
      tickets.value = result.items
      page.value = result.page
      pageSize.value = result.pageSize
      total.value = result.total
      hasMore.value = result.hasMore
      if (
        activeTicket.value &&
        tickets.value.some((item) => item.id === activeTicket.value?.id)
      ) {
        activeTicket.value =
          tickets.value.find((item) => item.id === activeTicket.value?.id) ??
          activeTicket.value
      }
    } catch (caught) {
      if (isCanceledRequest(caught)) return
      error.value =
        caught instanceof Error ? caught.message : '人工修图记录加载失败'
      throw caught
    } finally {
      if (listController === controller) {
        listController = null
        listLoading.value = false
      }
    }
  }

  async function open(ticketId: string): Promise<RetouchTicket | null> {
    detailController?.abort()
    const controller = new AbortController()
    detailController = controller
    detailLoading.value = true
    error.value = ''
    try {
      const ticket = await retouchTicketApi.get(ticketId, controller.signal)
      activeTicket.value = ticket
      upsert(ticket)
      return ticket
    } catch (caught) {
      if (isCanceledRequest(caught)) return null
      error.value =
        caught instanceof Error ? caught.message : '人工修图工单加载失败'
      throw caught
    } finally {
      if (detailController === controller) {
        detailController = null
        detailLoading.value = false
      }
    }
  }

  function close(): void {
    detailController?.abort()
    detailController = null
    detailLoading.value = false
    activeTicket.value = null
  }

  async function runAction(
    request: (signal: AbortSignal) => Promise<RetouchTicket>,
  ): Promise<RetouchTicket | null> {
    actionController?.abort()
    const controller = new AbortController()
    actionController = controller
    actionLoading.value = true
    error.value = ''
    try {
      const ticket = await request(controller.signal)
      upsert(ticket)
      return ticket
    } catch (caught) {
      if (isCanceledRequest(caught)) return null
      error.value =
        caught instanceof Error ? caught.message : '人工修图操作失败'
      throw caught
    } finally {
      if (actionController === controller) {
        actionController = null
        actionLoading.value = false
      }
    }
  }

  function create(
    taskId: string,
    payload: CreateRetouchTicketPayload,
  ): Promise<RetouchTicket | null> {
    return runAction((signal) =>
      retouchTicketApi.create(taskId, payload, undefined, signal),
    )
  }

  function createWithFiles(
    taskId: string,
    selectedResultIds: string[],
    requirement: string,
    supplementalFiles: File[],
  ): Promise<RetouchTicket | null> {
    return runAction(async (signal) => {
      const uploadedAssets: Asset[] = []
      try {
        for (const file of supplementalFiles) {
          uploadedAssets.push(
            await assetApi.upload(
              file,
              'retouch-reference',
              undefined,
              undefined,
              signal,
            ),
          )
        }

        const ticket = await retouchTicketApi.create(
          taskId,
          {
            selectedResultIds,
            requirement,
            supplementalAssetIds: uploadedAssets.map((asset) => asset.id),
          },
          undefined,
          signal,
        )
        return {
          ...ticket,
          supplementalAssets: ticket.supplementalAssets.map((asset) => ({
            ...asset,
            previewUrl:
              uploadedAssets.find((uploaded) => uploaded.id === asset.id)
                ?.previewUrl ?? asset.previewUrl,
          })),
        }
      } catch (caught) {
        await Promise.allSettled(
          uploadedAssets.map((asset) => assetApi.remove(asset.id)),
        )
        throw caught
      }
    })
  }

  function acceptQuote(
    ticketId: string,
    quoteId: string,
  ): Promise<RetouchTicket | null> {
    const entitlementStore = useEntitlementStore()
    return runAction(async (signal) => {
      const result = await retouchTicketApi.acceptQuote(
        ticketId,
        quoteId,
        undefined,
        signal,
      )
      entitlementStore.setFromServer(result.entitlement)
      await entitlementStore.refreshLedger().catch(() => undefined)
      return result.ticket
    })
  }

  function cancel(ticketId: string): Promise<RetouchTicket | null> {
    const entitlementStore = useEntitlementStore()
    return runAction(async (signal) => {
      const result = await retouchTicketApi.cancel(
        ticketId,
        undefined,
        signal,
      )
      entitlementStore.setFromServer(result.entitlement)
      await entitlementStore.refreshLedger().catch(() => undefined)
      return result.ticket
    })
  }

  function confirm(ticketId: string): Promise<RetouchTicket | null> {
    return runAction((signal) =>
      retouchTicketApi.confirm(ticketId, undefined, signal),
    )
  }

  function requestRevision(
    ticketId: string,
    message: string,
  ): Promise<RetouchTicket | null> {
    return runAction((signal) =>
      retouchTicketApi.requestRevision(
        ticketId,
        message,
        undefined,
        signal,
      ),
    )
  }

  function reset(): void {
    listController?.abort()
    detailController?.abort()
    actionController?.abort()
    listController = null
    detailController = null
    actionController = null
    tickets.value = []
    activeTicket.value = null
    listLoading.value = false
    detailLoading.value = false
    actionLoading.value = false
    error.value = ''
  }

  return {
    tickets,
    activeTicket,
    loading,
    hasActiveTickets,
    page,
    pageSize,
    total,
    hasMore,
    actionLoading,
    error,
    load,
    open,
    close,
    create,
    createWithFiles,
    acceptQuote,
    cancel,
    confirm,
    requestRevision,
    reset,
    upsert,
  }
})
