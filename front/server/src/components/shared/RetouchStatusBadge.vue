<script setup lang="ts">
import StatusBadge from '@/components/base/StatusBadge.vue'
import { RETOUCH_STATUS_LABELS } from '@/config'
import type { RetouchTicketStatus } from '@/types/domain'

defineProps<{ status: RetouchTicketStatus }>()

const toneMap: Record<
  RetouchTicketStatus,
  'neutral' | 'success' | 'warning' | 'danger' | 'info'
> = {
  submitted: 'warning',
  quote_pending: 'info',
  accepted: 'warning',
  processing: 'info',
  awaiting_confirmation: 'warning',
  delivered: 'success',
  rejected: 'danger',
  cancelled: 'neutral',
}
</script>

<template>
  <StatusBadge :tone="toneMap[status]">
    {{ RETOUCH_STATUS_LABELS[status] }}
  </StatusBadge>
</template>

