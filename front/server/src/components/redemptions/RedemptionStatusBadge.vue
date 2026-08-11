<script setup lang="ts">
import StatusBadge from '@/components/base/StatusBadge.vue'
import { REDEMPTION_STATUS_LABELS } from '@/config'
import type { RedemptionCodeStatus } from '@/types'

const props = defineProps<{
  status: RedemptionCodeStatus
  expiringSoon?: boolean
}>()

const tones: Record<
  RedemptionCodeStatus,
  'neutral' | 'success' | 'warning' | 'danger'
> = {
  unused: 'success',
  redeemed: 'neutral',
  expired: 'warning',
  disabled: 'danger',
}
</script>

<template>
  <div class="status-line">
    <StatusBadge :tone="tones[props.status]">
      {{ REDEMPTION_STATUS_LABELS[props.status] }}
    </StatusBadge>
    <span
      v-if="props.expiringSoon && props.status === 'unused'"
      class="expiring"
    >
      即将过期
    </span>
  </div>
</template>

<style scoped>
.status-line {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.expiring {
  color: var(--color-warning, #966d22);
  font-size: 12px;
}
</style>
