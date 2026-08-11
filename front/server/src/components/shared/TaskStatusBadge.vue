<script setup lang="ts">
import StatusBadge from '@/components/base/StatusBadge.vue'
import { TASK_STATUS_LABELS } from '@/config'
import type { TaskStatus } from '@/types/domain'

defineProps<{ status: TaskStatus }>()

const toneMap: Record<
  TaskStatus,
  'neutral' | 'success' | 'warning' | 'danger' | 'info'
> = {
  queued: 'neutral',
  processing: 'info',
  completed: 'success',
  partial: 'warning',
  failed: 'danger',
  cancelled: 'neutral',
}
</script>

<template>
  <StatusBadge :tone="toneMap[status]">
    {{ TASK_STATUS_LABELS[status] }}
  </StatusBadge>
</template>

