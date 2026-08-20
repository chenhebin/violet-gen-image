<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Eye, RefreshCw, Search } from '@lucide/vue'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDataTable, {
  type DataTableColumn,
} from '@/components/base/BaseDataTable.vue'
import FilterBar from '@/components/base/FilterBar.vue'
import FormField from '@/components/base/FormField.vue'
import PaginationBar from '@/components/base/PaginationBar.vue'
import RetouchActionModal, {
  type RetouchAction,
  type RetouchActionPayload,
} from '@/components/retouch/RetouchActionModal.vue'
import RetouchTicketDrawer from '@/components/retouch/RetouchTicketDrawer.vue'
import RetouchStatusBadge from '@/components/shared/RetouchStatusBadge.vue'
import UserStatusBadge from '@/components/shared/UserStatusBadge.vue'
import { RETOUCH_STATUS_LABELS } from '@/config'
import { useToast } from '@/composables/useToast'
import { useRetouchStore } from '@/stores/retouch'
import type {
  ManageRetouchTicketSummary,
  RetouchTicketQuery,
  RetouchTicketStatus,
} from '@/types/domain'
import { formatDateTime } from '@/utils/format'

const store = useRetouchStore()
const toast = useToast()
const route = useRoute()
const router = useRouter()
const keyword = ref(
  typeof route.query.keyword === 'string' ? route.query.keyword : '',
)
const status = ref<RetouchTicketStatus | ''>(
  typeof route.query.status === 'string'
    ? (route.query.status as RetouchTicketStatus)
    : '',
)
const slaFilter = ref<'' | 'overdue' | 'due-soon'>(
  route.query.sla === 'overdue' || route.query.sla === 'due-soon'
    ? (route.query.sla as 'overdue' | 'due-soon')
    : '',
)
const selectedId = ref('')
const action = ref<RetouchAction | null>(null)

const columns: DataTableColumn[] = [
  { key: 'ticket', label: '工单 / 任务', width: '30%' },
  { key: 'user', label: '用户', width: '22%' },
  { key: 'status', label: '状态', width: '16%' },
  { key: 'quote', label: '报价', width: '10%' },
  { key: 'updated', label: '最近更新', width: '16%' },
  { key: 'actions', label: '', width: '72px', align: 'right' },
]

function currentQuery(page = 1): RetouchTicketQuery {
  return {
    page,
    pageSize: store.tickets.pageSize,
    keyword: keyword.value.trim() || undefined,
    status: status.value || undefined,
    sla: slaFilter.value || undefined,
  }
}

async function load(page = 1): Promise<void> {
  try {
    await store.loadTickets(currentQuery(page))
  } catch (error) {
    toast.error({
      title: '工单列表加载失败',
      message: error instanceof Error ? error.message : '请稍后重试',
    })
  }
}

async function openTicket(ticket: ManageRetouchTicketSummary): Promise<void> {
  selectedId.value = ticket.id
  try {
    await store.loadTicket(ticket.id)
  } catch (error) {
    selectedId.value = ''
    toast.error({
      title: '工单详情加载失败',
      message: error instanceof Error ? error.message : '请稍后重试',
    })
  }
}

function closeDrawer(): void {
  if (store.isMutating) return
  selectedId.value = ''
  store.currentTicket = null
  action.value = null
  if (route.query.ticketId) {
    void router.replace({
      query: { ...route.query, ticketId: undefined },
    })
  }
}

async function submitAction(payload: RetouchActionPayload): Promise<void> {
  const ticket = store.currentTicket
  if (!ticket || !action.value) return
  try {
    if (action.value === 'quote') {
      await store.quote(ticket.id, payload.credits ?? 1, payload.note)
    } else if (action.value === 'start') {
      await store.start(ticket.id)
    } else if (action.value === 'deliver') {
      await store.deliver(ticket.id, payload.files ?? [], payload.note)
    } else if (action.value === 'reject') {
      await store.reject(ticket.id, payload.reason ?? '')
    } else {
      await store.fail(ticket.id, payload.reason ?? '')
    }
    toast.success('工单状态已更新')
    action.value = null
  } catch (error) {
    toast.error({
      title: '工单操作未完成',
      message: error instanceof Error ? error.message : '工单状态可能已变化',
    })
  }
}

onMounted(async () => {
  await load()
  if (typeof route.query.ticketId === 'string') {
    selectedId.value = route.query.ticketId
    try {
      await store.loadTicket(route.query.ticketId)
    } catch (error) {
      selectedId.value = ''
      toast.error({
        title: '关联工单无法打开',
        message: error instanceof Error ? error.message : '工单可能已不存在',
      })
    }
  }
})
</script>

<template>
  <main class="page">
    <header class="page__header">
      <div>
        <p class="page__eyebrow">Retouch ledger</p>
        <h1 class="page__title">人工修图工单</h1>
        <p class="page__description">
          从报价、开工到交付统一留痕。所有操作遵循工单状态和次数结算规则。
        </p>
      </div>
      <div class="page__actions">
        <BaseButton variant="secondary" :loading="store.isLoading" @click="load(store.tickets.page)">
          <template #icon><RefreshCw :size="16" /></template>
          刷新
        </BaseButton>
      </div>
    </header>

    <FilterBar>
      <FormField label="检索工单" for-id="retouch-search">
        <div class="search-control">
          <Search :size="16" />
          <input
            id="retouch-search"
            v-model="keyword"
            class="form-control"
            placeholder="工单号、任务或用户邮箱"
            @keyup.enter="load()"
          />
        </div>
      </FormField>
      <FormField label="处理状态" for-id="retouch-status">
        <select id="retouch-status" v-model="status" class="form-control" @change="load()">
          <option value="">全部状态</option>
          <option v-for="(label, value) in RETOUCH_STATUS_LABELS" :key="value" :value="value">
            {{ label }}
          </option>
        </select>
      </FormField>
      <FormField label="SLA 状态" for-id="retouch-sla">
        <select id="retouch-sla" v-model="slaFilter" class="form-control" @change="load()">
          <option value="">全部 SLA</option>
          <option value="due-soon">即将逾期（24 小时内）</option>
          <option value="overdue">已逾期</option>
        </select>
      </FormField>
      <template #actions>
        <BaseButton @click="load()">
          <template #icon><Search :size="16" /></template>
          查询
        </BaseButton>
      </template>
    </FilterBar>

    <section class="table-section" aria-label="人工工单列表">
      <div class="table-summary">
        <span>共享处理队列</span>
        <strong>{{ store.tickets.total }} 个工单</strong>
      </div>
      <BaseDataTable
        :columns="columns"
        :loading="store.isLoading"
        :has-rows="store.tickets.items.length > 0"
        empty-title="没有匹配的人工工单"
        empty-description="新提交的人工修图需求会出现在这里。"
        :min-width="940"
      >
        <template #body>
          <tr
            v-for="ticket in store.tickets.items"
            :key="ticket.id"
            tabindex="0"
            @click="openTicket(ticket)"
            @keyup.enter="openTicket(ticket)"
          >
            <td>
              <div class="primary-cell">
                <strong class="mono">{{ ticket.ticketNo }}</strong>
                <span>{{ ticket.taskTitle }}</span>
              </div>
            </td>
            <td>
              <div class="user-cell">
                <span>{{ ticket.user.email }}</span>
                <UserStatusBadge :status="ticket.user.status" />
              </div>
            </td>
            <td>
              <div class="status-cell">
                <RetouchStatusBadge :status="ticket.status" />
                <small v-if="ticket.sla.overdue" class="sla-overdue">已逾期</small>
                <small v-else-if="ticket.sla.remainingSeconds !== null && ticket.sla.remainingSeconds <= 24 * 60 * 60" class="sla-soon">即将逾期</small>
              </div>
            </td>
            <td class="mono">{{ ticket.quoteCredits ? `${ticket.quoteCredits} 次` : '待定' }}</td>
            <td class="muted-cell">{{ formatDateTime(ticket.updatedAt) }}</td>
            <td class="row-action">
              <button aria-label="查看工单详情" @click.stop="openTicket(ticket)">
                <Eye :size="17" />
              </button>
            </td>
          </tr>
        </template>
      </BaseDataTable>
      <PaginationBar
        :page="store.tickets.page"
        :page-size="store.tickets.pageSize"
        :total="store.tickets.total"
        :has-more="store.tickets.hasMore"
        :loading="store.isLoading"
        @change="load"
      />
    </section>
  </main>

  <RetouchTicketDrawer
    :open="Boolean(selectedId)"
    :ticket="store.currentTicket"
    :loading="store.isLoading"
    @close="closeDrawer"
    @action="action = $event"
  />
  <RetouchActionModal
    :action="action"
    :ticket="store.currentTicket"
    :loading="store.isMutating"
    @close="action = null"
    @submit="submitAction"
  />
</template>

<style scoped>
.search-control {
  position: relative;
  width: min(360px, 100%);
}

.search-control svg {
  position: absolute;
  z-index: 1;
  top: 14px;
  left: 12px;
  color: var(--ink-faint);
}

.search-control input {
  padding-left: 37px;
}

select.form-control {
  min-width: 190px;
}

.table-section {
  margin-top: 18px;
}

.table-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 2px 10px;
  color: var(--ink-muted);
  font-size: 12px;
}

.table-summary strong {
  color: var(--ink);
  font-family: var(--font-mono);
  font-weight: 650;
}

tbody tr {
  cursor: pointer;
}

.primary-cell,
.user-cell {
  display: grid;
  gap: 4px;
}

.primary-cell strong {
  font-size: 12px;
}

.primary-cell span,
.user-cell > span {
  overflow: hidden;
  color: var(--ink-muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-cell {
  justify-items: start;
}

.muted-cell {
  color: var(--ink-muted);
  font-size: 11px;
}

.status-cell {
  display: grid;
  justify-items: start;
  gap: 4px;
}

.status-cell small {
  font-size: 10px;
  font-weight: 650;
}

.sla-overdue { color: var(--danger); }
.sla-soon { color: var(--warning); }

.row-action {
  text-align: right;
}

.row-action button {
  display: inline-grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--ink-muted);
}

.row-action button:hover {
  background: var(--primary-soft);
  color: var(--primary);
}
</style>
