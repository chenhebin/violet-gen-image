<script setup lang="ts">
import { nextTick, onMounted, onUpdated, ref } from 'vue'
import EmptyState from './EmptyState.vue'

export interface DataTableColumn {
  key: string
  label: string
  width?: string
  align?: 'left' | 'center' | 'right'
}

const props = withDefaults(
  defineProps<{
    columns: DataTableColumn[]
    loading?: boolean
    hasRows?: boolean
    emptyTitle?: string
    emptyDescription?: string
    minWidth?: number
  }>(),
  {
    loading: false,
    hasRows: false,
    emptyTitle: '没有匹配记录',
    emptyDescription: '调整筛选条件后再试。',
    minWidth: 840,
  },
)

const tableRoot = ref<HTMLElement | null>(null)

function syncMobileLabels(): void {
  tableRoot.value?.querySelectorAll('tbody tr').forEach((row) => {
    row.querySelectorAll('td').forEach((cell, index) => {
      cell.dataset.label = props.columns[index]?.label ?? ''
    })
  })
}

onMounted(() => void nextTick(syncMobileLabels))
onUpdated(() => void nextTick(syncMobileLabels))
</script>

<template>
  <div ref="tableRoot" class="data-table">
    <div class="data-table__scroll">
      <table :style="{ minWidth: `${minWidth}px` }">
        <thead>
          <tr>
            <th
              v-for="column in columns"
              :key="column.key"
              :style="{ width: column.width, textAlign: column.align ?? 'left' }"
            >
              {{ column.label }}
            </th>
          </tr>
        </thead>
        <tbody v-if="loading">
          <tr v-for="row in 5" :key="row">
            <td v-for="column in columns" :key="column.key">
              <span class="data-table__skeleton"></span>
            </td>
          </tr>
        </tbody>
        <tbody v-else-if="hasRows">
          <slot name="body"></slot>
        </tbody>
      </table>
    </div>
    <EmptyState
      v-if="!loading && !hasRows"
      :title="emptyTitle"
      :description="emptyDescription"
      compact
    >
      <template v-if="$slots.emptyAction" #action>
        <slot name="emptyAction"></slot>
      </template>
    </EmptyState>
  </div>
</template>

<style scoped>
.data-table {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.data-table__scroll {
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th {
  height: 42px;
  padding: 0 16px;
  border-bottom: 1px solid var(--border);
  background: var(--surface-soft);
  color: var(--ink-muted);
  font-size: 11px;
  font-weight: 750;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

:deep(td) {
  height: 60px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border);
  font-size: 13px;
  vertical-align: middle;
}

:deep(tbody tr:last-child td) {
  border-bottom: 0;
}

:deep(tbody tr) {
  transition: background var(--motion-fast);
}

:deep(tbody tr:hover) {
  background: #fafcfb;
}

:deep(tbody tr:active) {
  background: var(--primary-soft);
}

.data-table__skeleton {
  display: block;
  width: 76%;
  height: 11px;
  border-radius: var(--radius-pill);
  background: linear-gradient(
    90deg,
    var(--surface-soft) 0%,
    #f8faf9 50%,
    var(--surface-soft) 100%
  );
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

@media (max-width: 640px) {
  .data-table {
    overflow: visible;
    border: 0;
    background: transparent;
    box-shadow: none;
  }

  .data-table__scroll {
    overflow: visible;
  }

  table {
    display: block;
    min-width: 0 !important;
  }

  thead {
    display: none;
  }

  tbody {
    display: grid;
    gap: 10px;
  }

  :deep(tbody tr) {
    position: relative;
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 13px 16px;
    padding: 15px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    background: var(--surface);
    box-shadow: 0 5px 18px rgb(22 30 28 / 4%);
    animation: table-row-in var(--motion-normal) var(--ease-out) both;
  }

  :deep(tbody tr:nth-child(2)) {
    animation-delay: 40ms;
  }

  :deep(tbody tr:nth-child(3)) {
    animation-delay: 80ms;
  }

  :deep(tbody tr:nth-child(4)) {
    animation-delay: 120ms;
  }

  :deep(td) {
    display: grid;
    height: auto;
    min-height: 0;
    align-content: start;
    gap: 3px;
    padding: 0;
    border: 0;
    text-align: left !important;
  }

  :deep(td::before) {
    color: var(--ink-faint);
    content: attr(data-label);
    font-size: 9px;
    font-weight: 700;
  }

  :deep(td > *) {
    justify-self: start;
  }

  :deep(td.number-cell) {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    justify-content: flex-start;
    gap: 0;
  }

  :deep(td.number-cell::before) {
    width: 100%;
  }

  :deep(td:first-child) {
    grid-column: 1 / -1;
    padding-right: 48px;
    padding-bottom: 12px;
    border-bottom: 1px solid var(--border);
  }

  :deep(td:last-child) {
    position: absolute;
    top: 9px;
    right: 9px;
    display: block;
  }

  :deep(td:last-child::before) {
    display: none;
  }

  :deep(td:last-child button) {
    width: 44px;
    height: 44px;
  }

  .data-table__skeleton {
    min-height: 12px;
  }
}

@keyframes table-row-in {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
}
</style>
