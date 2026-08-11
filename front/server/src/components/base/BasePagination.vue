<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from '@lucide/vue'
import IconButton from './IconButton.vue'

const props = withDefaults(
  defineProps<{
    page: number
    pageSize?: number
    total: number
  }>(),
  {
    pageSize: 20,
  },
)

const emit = defineEmits<{ 'update:page': [page: number] }>()

const pageCount = computed(() =>
  Math.max(1, Math.ceil(props.total / props.pageSize)),
)
const rangeStart = computed(() =>
  props.total === 0 ? 0 : (props.page - 1) * props.pageSize + 1,
)
const rangeEnd = computed(() =>
  Math.min(props.page * props.pageSize, props.total),
)
const visiblePages = computed(() => {
  const start = Math.max(1, Math.min(props.page - 2, pageCount.value - 4))
  const end = Math.min(pageCount.value, start + 4)
  return Array.from({ length: end - start + 1 }, (_, index) => start + index)
})

function goTo(page: number): void {
  const target = Math.min(Math.max(page, 1), pageCount.value)
  if (target !== props.page) emit('update:page', target)
}
</script>

<template>
  <nav class="pagination" aria-label="分页">
    <p class="pagination__summary">
      第 {{ rangeStart }}–{{ rangeEnd }} 条，共 {{ total }} 条
    </p>
    <div class="pagination__controls">
      <IconButton
        label="上一页"
        :disabled="page <= 1"
        @click="goTo(page - 1)"
      >
        <ChevronLeft :size="18" />
      </IconButton>
      <button
        v-for="item in visiblePages"
        :key="item"
        class="pagination__page"
        :class="{ 'pagination__page--active': item === page }"
        type="button"
        :aria-current="item === page ? 'page' : undefined"
        :aria-label="`第 ${item} 页`"
        @click="goTo(item)"
      >
        {{ item }}
      </button>
      <IconButton
        label="下一页"
        :disabled="page >= pageCount"
        @click="goTo(page + 1)"
      >
        <ChevronRight :size="18" />
      </IconButton>
    </div>
  </nav>
</template>

<style scoped>
.pagination {
  display: flex;
  min-height: 60px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 8px 2px;
}

.pagination__summary {
  color: var(--ink-muted);
  font-size: 12px;
}

.pagination__controls {
  display: flex;
  align-items: center;
  gap: 3px;
}

.pagination__page {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--ink-muted);
  font-size: 13px;
  font-weight: 700;
}

.pagination__page:hover {
  background: var(--surface-soft);
}

.pagination__page--active {
  background: var(--primary-soft);
  color: var(--primary);
}

@media (max-width: 560px) {
  .pagination {
    align-items: flex-start;
    flex-direction: column;
  }

  .pagination__controls {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
