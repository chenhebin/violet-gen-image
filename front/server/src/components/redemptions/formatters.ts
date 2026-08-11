import { REDEMPTION_STATUS_LABELS } from '@/config'
import type { RedemptionCodeStatus } from '@/types'

export function formatDateTime(value?: string | null) {
  if (!value) return '永久有效'

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(new Date(value))
}

export function formatDate(value?: string | null) {
  if (!value) return '永久有效'

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date(value))
}

export function redemptionStatusLabel(status: RedemptionCodeStatus) {
  return REDEMPTION_STATUS_LABELS[status]
}

export function formatPercent(value: number) {
  const normalized = value > 1 ? value : value * 100
  return `${Math.round(normalized)}%`
}
