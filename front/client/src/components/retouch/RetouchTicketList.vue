<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  ArrowRight,
  Image as ImageIcon,
  ListFilter,
  LoaderCircle,
  Scissors,
} from '@lucide/vue'
import RetouchStatusBadge from './RetouchStatusBadge.vue'
import type { RetouchTicket } from '@/types/domain'

type TicketFilter = 'all' | 'attention' | 'active' | 'finished'

const props = defineProps<{
  tickets: RetouchTicket[]
  loading: boolean
}>()

defineEmits<{
  open: [ticketId: string]
}>()

const filter = ref<TicketFilter>('all')
const filterOptions: ReadonlyArray<{ value: TicketFilter; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'attention', label: '待我处理' },
  { value: 'active', label: '进行中' },
  { value: 'finished', label: '已结束' },
]

const filteredTickets = computed(() => {
  if (filter.value === 'attention') {
    return props.tickets.filter((ticket) =>
      ['quote_pending', 'awaiting_confirmation'].includes(ticket.status),
    )
  }
  if (filter.value === 'active') {
    return props.tickets.filter((ticket) =>
      ['submitted', 'accepted', 'processing'].includes(ticket.status),
    )
  }
  if (filter.value === 'finished') {
    return props.tickets.filter((ticket) =>
      ['delivered', 'rejected', 'cancelled'].includes(ticket.status),
    )
  }
  return props.tickets
})

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
</script>

<template>
  <section class="ticket-surface">
    <header class="ticket-toolbar">
      <div class="filters" role="radiogroup" aria-label="精修记录筛选">
        <ListFilter :size="17" aria-hidden="true" />
        <button
          v-for="item in filterOptions"
          :key="item.value"
          :class="{ active: filter === item.value }"
          role="radio"
          :aria-checked="filter === item.value"
          @click="filter = item.value"
        >
          {{ item.label }}
        </button>
      </div>
      <span>共 {{ filteredTickets.length }} 个工单</span>
    </header>

    <div class="table-heading" aria-hidden="true">
      <span>精修项目</span>
      <span>状态</span>
      <span>报价 / 结算</span>
      <span>更新时间</span>
      <span />
    </div>

    <div v-if="loading && !tickets.length" class="state-panel">
      <LoaderCircle :size="24" class="spinner" />
      <p>正在读取精修记录...</p>
    </div>

    <div v-else-if="filteredTickets.length" class="ticket-list">
      <button
        v-for="ticket in filteredTickets"
        :key="ticket.id"
        class="ticket-row"
        @click="$emit('open', ticket.id)"
      >
        <span class="ticket-main">
          <span class="ticket-thumb">
            <img
              v-if="ticket.deliverables[0]?.url || ticket.selectedResults[0]?.url"
              :src="ticket.deliverables[0]?.url || ticket.selectedResults[0]?.url"
              alt=""
            />
            <ImageIcon v-else :size="19" />
          </span>
          <span class="ticket-title">
            <strong>{{ ticket.taskTitle }}</strong>
            <small>{{ ticket.ticketNo }} · {{ ticket.selectedResults.length }} 张</small>
          </span>
        </span>

        <span class="status-cell">
          <RetouchStatusBadge :status="ticket.status" />
          <small v-if="ticket.revision">已使用 1 次返修</small>
        </span>

        <span class="settlement">
          <strong v-if="ticket.quote">{{ ticket.quote.credits }} 次</strong>
          <strong v-else>等待报价</strong>
          <small v-if="ticket.refundedCredits">
            已退回 {{ ticket.refundedCredits }} 次
          </small>
          <small v-else-if="ticket.spentCredits">
            已结算 {{ ticket.spentCredits }} 次
          </small>
          <small v-else>尚未结算</small>
        </span>

        <time :datetime="ticket.updatedAt">{{ formatDate(ticket.updatedAt) }}</time>
        <ArrowRight :size="18" class="row-arrow" />
      </button>
    </div>

    <div v-else class="state-panel empty-state">
      <span><Scissors :size="26" /></span>
      <h2>{{ filter === 'all' ? '还没有人工精修记录' : '这个分类暂时为空' }}</h2>
      <p>从已完成的生成任务发起人工精修后，工单会出现在这里。</p>
    </div>
  </section>
</template>

<style scoped>
.ticket-surface {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.ticket-toolbar {
  display: flex;
  min-height: 64px;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border);
}

.filters {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--ink-faint);
}

.filters > svg {
  margin-right: 6px;
}

.filters button {
  min-height: 38px;
  padding: 0 12px;
  border-radius: 6px;
  background: transparent;
  color: var(--ink-muted);
  font-size: 12px;
  font-weight: 650;
}

.filters button.active {
  background: var(--surface-soft);
  color: var(--ink);
}

.ticket-toolbar > span {
  color: var(--ink-faint);
  font-size: 11px;
}

.table-heading,
.ticket-row {
  display: grid;
  grid-template-columns:
    minmax(260px, 2.3fr) minmax(140px, 1fr) minmax(120px, 0.8fr)
    minmax(120px, 0.8fr) 28px;
  gap: 18px;
  align-items: center;
}

.table-heading {
  min-height: 38px;
  padding: 0 18px;
  border-bottom: 1px solid var(--border);
  background: var(--surface-soft);
  color: var(--ink-faint);
  font-size: 9px;
  font-weight: 750;
}

.ticket-row {
  width: 100%;
  min-height: 84px;
  padding: 12px 18px;
  border-bottom: 1px solid var(--border);
  background: var(--surface);
  text-align: left;
  transition:
    background var(--motion-fast),
    border-color var(--motion-fast),
    transform var(--motion-fast);
}

.ticket-row:last-child {
  border-bottom: 0;
}

.ticket-row:hover {
  background: #fafbfc;
}

.ticket-row:active {
  transform: scale(0.995);
}

.ticket-main {
  display: grid;
  min-width: 0;
  grid-template-columns: 54px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
}

.ticket-thumb {
  display: grid;
  overflow: hidden;
  width: 54px;
  height: 54px;
  place-items: center;
  border-radius: 6px;
  background: var(--surface-soft);
  color: var(--ink-faint);
}

.ticket-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.ticket-title {
  min-width: 0;
}

.ticket-title strong,
.ticket-title small,
.settlement strong,
.settlement small {
  display: block;
}

.ticket-title strong {
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ticket-title small,
.settlement small,
.status-cell small,
.ticket-row time {
  color: var(--ink-faint);
  font-size: 10px;
}

.status-cell {
  display: grid;
  justify-items: start;
  gap: 4px;
}

.settlement strong {
  font-size: 12px;
}

.row-arrow {
  color: var(--ink-faint);
}

.state-panel {
  display: grid;
  min-height: 360px;
  place-items: center;
  align-content: center;
  gap: 10px;
  color: var(--ink-muted);
  text-align: center;
}

.spinner {
  animation: spin 800ms linear infinite;
}

.state-panel p {
  max-width: 340px;
  font-size: 12px;
}

.empty-state > span {
  display: grid;
  width: 58px;
  height: 58px;
  place-items: center;
  border-radius: 50%;
  background: var(--surface-soft);
}

.empty-state h2 {
  font-family: 'Songti SC', serif;
  font-size: 18px;
  font-weight: 600;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 760px) {
  .ticket-surface {
    overflow: visible;
    border: 0;
    background: transparent;
    box-shadow: none;
  }

  .ticket-toolbar {
    align-items: flex-start;
    flex-direction: column;
    gap: 6px;
    padding: 8px;
    margin-bottom: 10px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    background: var(--surface);
  }

  .filters {
    width: 100%;
    overflow-x: auto;
    scrollbar-width: none;
  }

  .filters::-webkit-scrollbar {
    display: none;
  }

  .filters > svg {
    flex: 0 0 auto;
    margin-left: 6px;
  }

  .filters button {
    min-height: 44px;
    flex: 1 0 auto;
  }

  .ticket-list {
    display: grid;
    gap: 10px;
  }

  .table-heading {
    display: none;
  }

  .ticket-row {
    grid-template-columns: 1fr auto;
    gap: 10px;
    min-height: 116px;
    padding: 14px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    box-shadow: 0 5px 18px rgb(23 25 29 / 4%);
    animation: row-in var(--motion-normal) var(--ease-out) both;
  }

  .ticket-row:nth-child(2) {
    animation-delay: 40ms;
  }

  .ticket-row:nth-child(3) {
    animation-delay: 80ms;
  }

  .ticket-row:nth-child(4) {
    animation-delay: 120ms;
  }

  .ticket-main {
    grid-column: 1 / -1;
  }

  .status-cell,
  .settlement {
    align-self: start;
  }

  .ticket-row time {
    display: none;
  }

  .row-arrow {
    align-self: center;
    grid-row: 2;
    grid-column: 2;
  }
}

@keyframes row-in {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
}
</style>
