import { describe, expect, it } from 'vitest'
import { isFinalRetouchTicketStatus } from '@/config'
import type { RetouchTicketStatus } from '@/types/domain'

describe('retouch ticket refresh rules', () => {
  it.each<RetouchTicketStatus>([
    'submitted',
    'quote_pending',
    'accepted',
    'processing',
    'awaiting_confirmation',
  ])('keeps refreshing non-final status %s', (status) => {
    expect(isFinalRetouchTicketStatus(status)).toBe(false)
  })

  it.each<RetouchTicketStatus>([
    'delivered',
    'rejected',
    'cancelled',
  ])('stops refreshing final status %s', (status) => {
    expect(isFinalRetouchTicketStatus(status)).toBe(true)
  })
})
