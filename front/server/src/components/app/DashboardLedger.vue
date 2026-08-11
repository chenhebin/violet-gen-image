<script setup lang="ts">
import { AlertTriangle, ArrowUpRight, Clock3 } from '@lucide/vue'
import StatusBadge from '@/components/base/StatusBadge.vue'
import {
  RETOUCH_STATUS_LABELS,
} from '@/config'
import type {
  DashboardAlert,
  ManageRetouchTicketSummary,
  RedemptionBatch,
  RetouchTicketStatus,
} from '@/types/domain'

defineProps<{
  alerts: DashboardAlert[]
  pendingTickets: ManageRetouchTicketSummary[]
  recentBatches: RedemptionBatch[]
  showPlatformData: boolean
}>()

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

function ticketTone(
  status: RetouchTicketStatus,
): 'neutral' | 'warning' | 'info' | 'success' {
  if (status === 'processing') return 'info'
  if (status === 'delivered') return 'success'
  if (status === 'submitted' || status === 'accepted') return 'warning'
  return 'neutral'
}
</script>

<template>
  <div class="dashboard-ledger">
    <section class="ledger-panel" aria-labelledby="pending-title">
      <header>
        <div>
          <h2 id="pending-title">人工工单待办</h2>
          <p>按最近更新时间排列</p>
        </div>
        <RouterLink to="/manage/retouch-tickets">
          查看全部 <ArrowUpRight :size="14" />
        </RouterLink>
      </header>
      <div v-if="pendingTickets.length" class="ledger-list">
        <RouterLink
          v-for="ticket in pendingTickets.slice(0, 6)"
          :key="ticket.id"
          :to="{
            path: '/manage/retouch-tickets',
            query: { ticketId: ticket.id },
          }"
          class="ledger-row"
        >
          <span class="ledger-row__id mono">{{ ticket.ticketNo }}</span>
          <span class="ledger-row__subject">
            <strong>{{ ticket.user.email }}</strong>
            <small>{{ ticket.taskTitle }}</small>
          </span>
          <StatusBadge :tone="ticketTone(ticket.status)">
            {{ RETOUCH_STATUS_LABELS[ticket.status] }}
          </StatusBadge>
          <span class="ledger-row__time">
            <Clock3 :size="13" />
            {{ formatDate(ticket.updatedAt) }}
          </span>
        </RouterLink>
      </div>
      <p v-else class="ledger-empty">当前没有需要处理的人工工单。</p>
    </section>

    <aside class="dashboard-side">
      <section class="attention-panel" aria-labelledby="attention-title">
        <header>
          <h2 id="attention-title">需要关注</h2>
          <span>{{ alerts.length }}</span>
        </header>
        <div v-if="alerts.length" class="attention-list">
          <RouterLink
            v-for="alert in alerts.slice(0, 4)"
            :key="alert.id"
            :to="alert.href"
          >
            <AlertTriangle :size="16" />
            <span>
              <strong>{{ alert.title }}</strong>
              <small>{{ alert.description }}</small>
            </span>
          </RouterLink>
        </div>
        <p v-else class="ledger-empty">没有需要立即处理的异常。</p>
      </section>

      <section
        v-if="showPlatformData"
        class="batch-panel"
        aria-labelledby="batch-title"
      >
        <header>
          <h2 id="batch-title">最近生成批次</h2>
          <RouterLink to="/manage/redemption-batches">全部批次</RouterLink>
        </header>
        <RouterLink
          v-for="batch in recentBatches.slice(0, 3)"
          :key="batch.id"
          :to="{
            path: '/manage/redemption-batches',
            query: { batchId: batch.id },
          }"
          class="batch-row"
        >
          <span>
            <strong>{{ batch.name }}</strong>
            <small>{{ batch.quantity }} 个 · 每码 {{ batch.creditsPerCode }} 次</small>
          </span>
          <span class="batch-row__rate mono">
            {{ Math.round(batch.usageRate * 100) }}%
          </span>
        </RouterLink>
      </section>
    </aside>
  </div>
</template>

<style scoped>
.dashboard-ledger {
  display: grid;
  align-items: start;
  grid-template-columns: minmax(0, 1.65fr) minmax(300px, 0.75fr);
  gap: 18px;
}

.ledger-panel,
.attention-panel,
.batch-panel {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.ledger-panel > header,
.attention-panel > header,
.batch-panel > header {
  display: flex;
  min-height: 58px;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
}

h2 {
  font-size: 14px;
}

header p,
header a {
  color: var(--ink-muted);
  font-size: 11px;
}

header a {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-weight: 700;
}

.ledger-row {
  display: grid;
  min-height: 67px;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border);
  grid-template-columns: 125px minmax(160px, 1fr) auto 110px;
  gap: 14px;
  transition: background var(--motion-fast);
}

.ledger-row:last-child {
  border-bottom: 0;
}

.ledger-row:hover {
  background: var(--surface-raised);
}

.ledger-row__id {
  color: var(--primary);
  font-size: 11px;
}

.ledger-row__subject {
  min-width: 0;
}

.ledger-row__subject strong,
.ledger-row__subject small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ledger-row__subject strong {
  font-size: 12px;
}

.ledger-row__subject small {
  margin-top: 3px;
  color: var(--ink-muted);
  font-size: 10px;
}

.ledger-row__time {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--ink-faint);
  font-size: 10px;
}

.dashboard-side {
  display: grid;
  gap: 18px;
}

.attention-panel header > span {
  display: grid;
  width: 24px;
  height: 24px;
  border-radius: var(--radius-pill);
  background: var(--warning-soft);
  color: var(--warning);
  font-family: var(--font-mono);
  font-size: 10px;
  place-items: center;
}

.attention-list a {
  display: grid;
  padding: 13px 15px;
  border-bottom: 1px solid var(--border);
  color: var(--warning);
  grid-template-columns: auto 1fr;
  gap: 9px;
}

.attention-list a:last-child {
  border-bottom: 0;
}

.attention-list strong,
.attention-list small {
  display: block;
}

.attention-list strong {
  color: var(--ink);
  font-size: 12px;
}

.attention-list small {
  margin-top: 3px;
  color: var(--ink-muted);
  font-size: 10px;
  line-height: 1.45;
}

.batch-row {
  display: flex;
  min-height: 60px;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 10px 15px;
  border-bottom: 1px solid var(--border);
}

.batch-row:last-child {
  border-bottom: 0;
}

.batch-row strong,
.batch-row small {
  display: block;
}

.batch-row strong {
  font-size: 12px;
}

.batch-row small {
  margin-top: 3px;
  color: var(--ink-muted);
  font-size: 10px;
}

.batch-row__rate {
  color: var(--primary);
  font-size: 12px;
}

.ledger-empty {
  padding: 28px 16px;
  color: var(--ink-muted);
  font-size: 12px;
  text-align: center;
}

@media (max-width: 1100px) {
  .dashboard-ledger {
    grid-template-columns: 1fr;
  }

  .dashboard-side {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 700px) {
  .ledger-row {
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .ledger-row__id,
  .ledger-row__time {
    display: none;
  }

  .dashboard-side {
    grid-template-columns: 1fr;
  }
}
</style>
