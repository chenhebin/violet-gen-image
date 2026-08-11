<script setup lang="ts">
import {
  CheckCircle2,
  CircleDollarSign,
  Clock3,
  XCircle,
} from '@lucide/vue'
import { RETOUCH_TICKET_STATUS_LABELS } from '@/config'
import type { RetouchTicketStatus } from '@/types/domain'

defineProps<{ status: RetouchTicketStatus }>()
</script>

<template>
  <span class="retouch-status" :class="status">
    <CheckCircle2
      v-if="status === 'delivered' || status === 'awaiting_confirmation'"
      :size="13"
      aria-hidden="true"
    />
    <XCircle
      v-else-if="status === 'rejected' || status === 'cancelled'"
      :size="13"
      aria-hidden="true"
    />
    <CircleDollarSign
      v-else-if="status === 'quote_pending'"
      :size="13"
      aria-hidden="true"
    />
    <Clock3 v-else :size="13" aria-hidden="true" />
    {{ RETOUCH_TICKET_STATUS_LABELS[status] }}
  </span>
</template>

<style scoped>
.retouch-status {
  display: inline-flex;
  min-height: 26px;
  align-items: center;
  gap: 5px;
  padding: 0 8px;
  border-radius: 5px;
  background: var(--surface-soft);
  color: var(--ink-muted);
  font-size: 10px;
  font-weight: 730;
  white-space: nowrap;
}

.retouch-status.quote_pending,
.retouch-status.awaiting_confirmation {
  background: #fff3de;
  color: var(--warning);
}

.retouch-status.accepted,
.retouch-status.processing,
.retouch-status.delivered {
  background: var(--primary-soft);
  color: var(--primary);
}

.retouch-status.rejected,
.retouch-status.cancelled {
  background: var(--coral-soft);
  color: var(--coral);
}
</style>
