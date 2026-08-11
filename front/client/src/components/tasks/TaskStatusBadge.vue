<script setup lang="ts">
import { Clock3 } from '@lucide/vue'
import {
  isRunningTaskStatus,
  TASK_STATUS_LABELS,
} from '@/config'
import type { TaskStatus } from '@/types/domain'

defineProps<{ status: TaskStatus }>()
</script>

<template>
  <span class="task-status" :class="status">
    <Clock3 v-if="isRunningTaskStatus(status)" :size="13" aria-hidden="true" />
    {{ TASK_STATUS_LABELS[status] }}
  </span>
</template>

<style scoped>
.task-status {
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
}

.task-status.completed,
.task-status.partial {
  background: var(--primary-soft);
  color: var(--primary);
}

.task-status.failed-refunded,
.task-status.cancelled {
  background: var(--coral-soft);
  color: var(--coral);
}
</style>
