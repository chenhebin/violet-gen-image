<script setup lang="ts">
import StatusBadge from '@/components/base/StatusBadge.vue'
import { USER_STATUS_LABELS } from '@/config'
import type { UserStatus } from '@/types/domain'

defineProps<{ status: UserStatus }>()
</script>

<template>
  <StatusBadge :tone="status === 'active' ? 'success' : 'danger'">
    {{ USER_STATUS_LABELS[status] }}
  </StatusBadge>
</template>
