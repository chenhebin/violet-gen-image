<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Eye, Fingerprint, RefreshCw, Search } from '@lucide/vue'
import AuditDetailDrawer from '@/components/audits/AuditDetailDrawer.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDataTable, {
  type DataTableColumn,
} from '@/components/base/BaseDataTable.vue'
import FilterBar from '@/components/base/FilterBar.vue'
import FormField from '@/components/base/FormField.vue'
import PaginationBar from '@/components/base/PaginationBar.vue'
import StatusBadge from '@/components/base/StatusBadge.vue'
import { AUDIT_RESULT_LABELS, ROLE_LABELS } from '@/config'
import { useToast } from '@/composables/useToast'
import { useAuditStore } from '@/stores/audits'
import type {
  AuditEvent,
  AuditQuery,
  AuditResult,
} from '@/types/domain'
import { formatDateTime } from '@/utils/format'

const store = useAuditStore()
const toast = useToast()
const keyword = ref('')
const result = ref<AuditResult | ''>('')
const resourceType = ref('')
const startAt = ref('')
const endAt = ref('')
const selectedEvent = ref<AuditEvent | null>(null)

const columns: DataTableColumn[] = [
  { key: 'time', label: '时间', width: '16%' },
  { key: 'operator', label: '操作者', width: '21%' },
  { key: 'action', label: '动作', width: '19%' },
  { key: 'resource', label: '资源', width: '20%' },
  { key: 'result', label: '结果', width: '10%' },
  { key: 'request', label: '请求 ID', width: '18%' },
  { key: 'actions', label: '', width: '64px', align: 'right' },
]

function currentQuery(page = 1): AuditQuery {
  return {
    page,
    pageSize: store.events.pageSize,
    keyword: keyword.value.trim() || undefined,
    result: result.value || undefined,
    resourceType: resourceType.value.trim() || undefined,
    startAt: startAt.value ? new Date(`${startAt.value}T00:00:00`).toISOString() : undefined,
    endAt: endAt.value ? new Date(`${endAt.value}T23:59:59`).toISOString() : undefined,
  }
}

async function load(page = 1): Promise<void> {
  try {
    await store.loadAudits(currentQuery(page))
  } catch (error) {
    toast.error({
      title: '审计日志加载失败',
      message: error instanceof Error ? error.message : '请稍后重试',
    })
  }
}

onMounted(() => void load())
</script>

<template>
  <main class="page">
    <header class="page__header">
      <div>
        <p class="page__eyebrow">Immutable audit trail</p>
        <h1 class="page__title">操作审计</h1>
        <p class="page__description">
          追踪管理写操作和敏感读取。完整码、密钥、密码与签名地址始终脱敏。
        </p>
      </div>
      <div class="page__actions">
        <BaseButton variant="secondary" :loading="store.isLoading" @click="load(store.events.page)">
          <template #icon><RefreshCw :size="16" /></template>
          刷新
        </BaseButton>
      </div>
    </header>

    <FilterBar>
      <FormField label="检索日志" for-id="audit-search">
        <div class="search-control">
          <Search :size="16" />
          <input
            id="audit-search"
            v-model="keyword"
            class="form-control"
            type="search"
            placeholder="操作者、动作或资源编号"
            @keyup.enter="load()"
          />
        </div>
      </FormField>
      <FormField label="执行结果" for-id="audit-result">
        <select id="audit-result" v-model="result" class="form-control" @change="load()">
          <option value="">全部结果</option>
          <option v-for="(label, value) in AUDIT_RESULT_LABELS" :key="value" :value="value">
            {{ label }}
          </option>
        </select>
      </FormField>
      <FormField label="资源类型" for-id="audit-resource">
        <input
          id="audit-resource"
          v-model="resourceType"
          class="form-control"
          placeholder="例如 redemption_code"
        />
      </FormField>
      <FormField label="开始日期" for-id="audit-start">
        <input id="audit-start" v-model="startAt" class="form-control" type="date" />
      </FormField>
      <FormField label="结束日期" for-id="audit-end">
        <input id="audit-end" v-model="endAt" class="form-control" type="date" />
      </FormField>
      <template #actions>
        <BaseButton @click="load()">
          <template #icon><Search :size="16" /></template>
          查询
        </BaseButton>
      </template>
    </FilterBar>

    <section class="table-section" aria-label="审计日志列表">
      <div class="table-summary">
        <span><Fingerprint :size="15" />不可修改的操作记录</span>
        <strong>{{ store.events.total }} 条</strong>
      </div>
      <BaseDataTable
        :columns="columns"
        :loading="store.isLoading"
        :has-rows="store.events.items.length > 0"
        empty-title="没有匹配的审计记录"
        empty-description="调整操作者、资源、结果或时间范围后再试。"
        :min-width="1080"
      >
        <template #body>
          <tr
            v-for="event in store.events.items"
            :key="event.id"
            tabindex="0"
            @click="selectedEvent = event"
            @keyup.enter="selectedEvent = event"
          >
            <td class="muted-cell">{{ formatDateTime(event.createdAt) }}</td>
            <td>
              <div class="primary-cell">
                <strong>{{ event.operatorEmail }}</strong>
                <span>{{ ROLE_LABELS[event.operatorRole] }}</span>
              </div>
            </td>
            <td class="mono action-cell">{{ event.action }}</td>
            <td>
              <div class="primary-cell">
                <strong>{{ event.resourceType }}</strong>
                <span class="mono">{{ event.resourceId }}</span>
              </div>
            </td>
            <td>
              <StatusBadge :tone="event.result === 'success' ? 'success' : 'danger'">
                {{ AUDIT_RESULT_LABELS[event.result] }}
              </StatusBadge>
            </td>
            <td class="mono request-cell">{{ event.requestId }}</td>
            <td class="row-action">
              <button aria-label="查看审计详情" @click.stop="selectedEvent = event">
                <Eye :size="17" />
              </button>
            </td>
          </tr>
        </template>
      </BaseDataTable>
      <PaginationBar
        :page="store.events.page"
        :page-size="store.events.pageSize"
        :total="store.events.total"
        :has-more="store.events.hasMore"
        :loading="store.isLoading"
        @change="load"
      />
    </section>
  </main>

  <AuditDetailDrawer
    :open="Boolean(selectedEvent)"
    :event="selectedEvent"
    @close="selectedEvent = null"
  />
</template>

<style scoped>
.search-control {
  position: relative;
  width: min(320px, 100%);
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
  min-width: 130px;
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

.table-summary span {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

.table-summary strong {
  color: var(--ink);
  font-family: var(--font-mono);
}

tbody tr {
  cursor: pointer;
}

.primary-cell {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.primary-cell strong,
.primary-cell span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.primary-cell strong {
  font-size: 11px;
}

.primary-cell span,
.muted-cell {
  color: var(--ink-muted);
  font-size: 10px;
}

.action-cell,
.request-cell {
  overflow: hidden;
  max-width: 180px;
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.request-cell {
  color: var(--ink-muted);
}

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
