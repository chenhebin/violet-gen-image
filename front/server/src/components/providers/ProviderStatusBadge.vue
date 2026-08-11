<script setup lang="ts">
import StatusBadge from '@/components/base/StatusBadge.vue'
import { CONNECTION_STATUS_LABELS } from '@/config'
import type { ConnectionStatus } from '@/types'

const props = defineProps<{
  status: ConnectionStatus
}>()

const tones: Record<
  ConnectionStatus,
  'neutral' | 'success' | 'danger'
> = {
  untested: 'neutral',
  healthy: 'success',
  error: 'danger',
}
</script>

<template>
  <StatusBadge :tone="tones[props.status]">
    {{ CONNECTION_STATUS_LABELS[props.status] }}
  </StatusBadge>
</template>
