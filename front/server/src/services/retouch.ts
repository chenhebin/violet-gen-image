import {
  apiRequest,
  mutationHeaders,
} from '@/services/http'
import type { PageResult } from '@/types/api'
import type {
  ManageRetouchTicket,
  ManageRetouchTicketSummary,
  RetouchTicketQuery,
} from '@/types/domain'

export const retouchApi = {
  list(query: RetouchTicketQuery = {}, signal?: AbortSignal) {
    return apiRequest<PageResult<ManageRetouchTicketSummary>>({
      method: 'GET',
      url: '/manage/retouch-tickets',
      params: query,
      signal,
    })
  },

  get(ticketId: string, signal?: AbortSignal) {
    return apiRequest<ManageRetouchTicket>({
      method: 'GET',
      url: `/manage/retouch-tickets/${ticketId}`,
      signal,
    })
  },

  quote(ticketId: string, credits: number, note?: string) {
    return apiRequest<ManageRetouchTicket>({
      method: 'POST',
      url: `/manage/retouch-tickets/${ticketId}/quote`,
      data: { credits, note },
      headers: mutationHeaders('quote_retouch_ticket'),
    })
  },

  start(ticketId: string) {
    return apiRequest<ManageRetouchTicket>({
      method: 'POST',
      url: `/manage/retouch-tickets/${ticketId}/start`,
      headers: mutationHeaders('start_retouch_ticket'),
    })
  },

  deliver(ticketId: string, files: File[], note?: string) {
    const data = new FormData()
    files.forEach((file) => data.append('files', file))
    if (note) data.append('note', note)
    return apiRequest<ManageRetouchTicket>({
      method: 'POST',
      url: `/manage/retouch-tickets/${ticketId}/deliver`,
      data,
      headers: mutationHeaders('deliver_retouch_ticket'),
    })
  },

  reject(ticketId: string, reason: string) {
    return apiRequest<ManageRetouchTicket>({
      method: 'POST',
      url: `/manage/retouch-tickets/${ticketId}/reject`,
      data: { reason },
      headers: mutationHeaders('reject_retouch_ticket'),
    })
  },

  fail(ticketId: string, reason: string) {
    return apiRequest<ManageRetouchTicket>({
      method: 'POST',
      url: `/manage/retouch-tickets/${ticketId}/fail`,
      data: { reason },
      headers: mutationHeaders('fail_retouch_ticket'),
    })
  },
}

