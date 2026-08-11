<script setup lang="ts">
import { ChevronLeft, ChevronRight } from '@lucide/vue'
import BaseButton from './BaseButton.vue'

const props = defineProps<{
  page: number
  pageSize: number
  total: number
  hasMore: boolean
  loading?: boolean
}>()

const emit = defineEmits<{
  change: [page: number]
}>()

function go(page: number): void {
  if (page < 1 || props.loading) return
  emit('change', page)
}
</script>

<template>
  <footer v-if="total > 0" class="pagination" aria-label="分页">
    <p>
      第 {{ page }} 页
      <span>共 {{ total }} 条</span>
    </p>
    <div>
      <BaseButton
        variant="secondary"
        size="small"
        :disabled="page <= 1 || loading"
        @click="go(page - 1)"
      >
        <template #icon><ChevronLeft :size="15" /></template>
        上一页
      </BaseButton>
      <BaseButton
        variant="secondary"
        size="small"
        :disabled="!hasMore || loading"
        @click="go(page + 1)"
      >
        下一页
        <template #icon><ChevronRight :size="15" /></template>
      </BaseButton>
    </div>
  </footer>
</template>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding-top: 14px;
  color: var(--ink-muted);
  font-size: 12px;
}

.pagination p,
.pagination div {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pagination span {
  color: var(--ink-faint);
}

@media (max-width: 520px) {
  .pagination {
    align-items: stretch;
    flex-direction: column;
  }

  .pagination div {
    justify-content: space-between;
  }

  .pagination div :deep(.base-button) {
    flex: 1;
  }
}
</style>
