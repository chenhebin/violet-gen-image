<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Eye, Images, RefreshCw, Search } from '@lucide/vue'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDataTable, {
  type DataTableColumn,
} from '@/components/base/BaseDataTable.vue'
import FilterBar from '@/components/base/FilterBar.vue'
import FormField from '@/components/base/FormField.vue'
import PaginationBar from '@/components/base/PaginationBar.vue'
import TaskStatusBadge from '@/components/shared/TaskStatusBadge.vue'
import ManagedTaskDrawer from '@/components/tasks/ManagedTaskDrawer.vue'
import {
  TASK_STATUS_LABELS,
  WORKSPACE_MODE_LABELS,
} from '@/config'
import { useToast } from '@/composables/useToast'
import { useTaskStore } from '@/stores/tasks'
import type {
  ManagedGenerationTaskSummary,
  ManagedTaskQuery,
  TaskStatus,
  WorkspaceMode,
} from '@/types/domain'
import { formatDateTime } from '@/utils/format'

const store = useTaskStore()
const toast = useToast()
const route = useRoute()
const router = useRouter()
const keyword = ref(
  typeof route.query.keyword === 'string' ? route.query.keyword : '',
)
const status = ref<TaskStatus | ''>(
  typeof route.query.status === 'string'
    ? (route.query.status as TaskStatus)
    : '',
)
const mode = ref<WorkspaceMode | ''>(
  typeof route.query.mode === 'string'
    ? (route.query.mode as WorkspaceMode)
    : '',
)
const hasRetouchTicket = ref<'' | 'true' | 'false'>('')
const selectedId = ref('')

const columns: DataTableColumn[] = [
  { key: 'task', label: '任务 / 用户', width: '30%' },
  { key: 'status', label: '状态', width: '14%' },
  { key: 'mode', label: '模式', width: '10%' },
  { key: 'output', label: '成片', width: '10%', align: 'right' },
  { key: 'settlement', label: '消耗 / 退款', width: '14%', align: 'right' },
  { key: 'model', label: '执行模型', width: '15%' },
  { key: 'created', label: '创建时间', width: '15%' },
  { key: 'actions', label: '', width: '64px', align: 'right' },
]

function currentQuery(page = 1): ManagedTaskQuery {
  return {
    page,
    pageSize: store.tasks.pageSize,
    keyword: keyword.value.trim() || undefined,
    status: status.value || undefined,
    mode: mode.value || undefined,
    hasRetouchTicket:
      hasRetouchTicket.value === '' ? undefined : hasRetouchTicket.value === 'true',
  }
}

async function load(page = 1): Promise<void> {
  try {
    await store.loadTasks(currentQuery(page))
  } catch (error) {
    toast.error({
      title: '任务列表加载失败',
      message: error instanceof Error ? error.message : '请稍后重试',
    })
  }
}

async function openTask(task: ManagedGenerationTaskSummary): Promise<void> {
  selectedId.value = task.id
  try {
    await store.loadTask(task.id)
  } catch (error) {
    selectedId.value = ''
    toast.error({
      title: '任务详情加载失败',
      message: error instanceof Error ? error.message : '请稍后重试',
    })
  }
}

function closeDrawer(): void {
  selectedId.value = ''
  store.currentTask = null
}

async function openRetouch(ticketId: string): Promise<void> {
  await router.push({
    path: '/manage/retouch-tickets',
    query: { ticketId },
  })
}

onMounted(() => void load())
</script>

<template>
  <main class="page">
    <header class="page__header">
      <div>
        <p class="page__eyebrow">Generation archive</p>
        <h1 class="page__title">生成任务</h1>
        <p class="page__description">
          追溯用户原始需求、确认提示词、模型快照、次数结算与最终成片。
        </p>
      </div>
      <div class="page__actions">
        <BaseButton variant="secondary" :loading="store.isLoading" @click="load(store.tasks.page)">
          <template #icon><RefreshCw :size="16" /></template>
          刷新
        </BaseButton>
      </div>
    </header>

    <FilterBar>
      <FormField label="检索任务" for-id="task-search">
        <div class="search-control">
          <Search :size="16" />
          <input
            id="task-search"
            v-model="keyword"
            class="form-control"
            type="search"
            placeholder="任务标题、用户或任务编号"
            @keyup.enter="load()"
          />
        </div>
      </FormField>
      <FormField label="任务状态" for-id="task-status">
        <select id="task-status" v-model="status" class="form-control" @change="load()">
          <option value="">全部状态</option>
          <option v-for="(label, value) in TASK_STATUS_LABELS" :key="value" :value="value">
            {{ label }}
          </option>
        </select>
      </FormField>
      <FormField label="创作模式" for-id="task-mode">
        <select id="task-mode" v-model="mode" class="form-control" @change="load()">
          <option value="">全部模式</option>
          <option v-for="(label, value) in WORKSPACE_MODE_LABELS" :key="value" :value="value">
            {{ label }}
          </option>
        </select>
      </FormField>
      <FormField label="人工工单" for-id="task-retouch">
        <select
          id="task-retouch"
          v-model="hasRetouchTicket"
          class="form-control"
          @change="load()"
        >
          <option value="">全部任务</option>
          <option value="true">已关联</option>
          <option value="false">未关联</option>
        </select>
      </FormField>
      <template #actions>
        <BaseButton @click="load()">
          <template #icon><Search :size="16" /></template>
          查询
        </BaseButton>
      </template>
    </FilterBar>

    <section class="table-section" aria-label="生成任务列表">
      <div class="table-summary">
        <span><Images :size="15" />平台任务档案</span>
        <strong>{{ store.tasks.total }} 个任务</strong>
      </div>
      <BaseDataTable
        :columns="columns"
        :loading="store.isLoading"
        :has-rows="store.tasks.items.length > 0"
        empty-title="没有匹配的生成任务"
        empty-description="调整任务状态、模式或关键词后再试。"
        :min-width="1120"
      >
        <template #body>
          <tr
            v-for="task in store.tasks.items"
            :key="task.id"
            tabindex="0"
            @click="openTask(task)"
            @keyup.enter="openTask(task)"
          >
            <td>
              <div class="primary-cell">
                <strong>{{ task.title }}</strong>
                <span>{{ task.ownerEmail }}</span>
                <small class="mono">{{ task.id }}</small>
              </div>
            </td>
            <td><TaskStatusBadge :status="task.status" /></td>
            <td>{{ WORKSPACE_MODE_LABELS[task.mode] }}</td>
            <td class="number-cell">
              {{ task.successfulCount }} / {{ task.requestedCount }}
            </td>
            <td class="number-cell">
              {{ task.spentCredits }} / {{ task.refundedCredits }}
            </td>
            <td>
              <div class="model-cell">
                <strong>{{ task.modelName }}</strong>
                <span>{{ task.providerName }}</span>
              </div>
            </td>
            <td class="muted-cell">{{ formatDateTime(task.createdAt) }}</td>
            <td class="row-action">
              <button aria-label="查看任务详情" @click.stop="openTask(task)">
                <Eye :size="17" />
              </button>
            </td>
          </tr>
        </template>
      </BaseDataTable>
      <PaginationBar
        :page="store.tasks.page"
        :page-size="store.tasks.pageSize"
        :total="store.tasks.total"
        :has-more="store.tasks.hasMore"
        :loading="store.isLoading"
        @change="load"
      />
    </section>
  </main>

  <ManagedTaskDrawer
    :open="Boolean(selectedId)"
    :task="store.currentTask"
    :loading="store.isLoading"
    @close="closeDrawer"
    @open-retouch="openRetouch"
  />
</template>

<style scoped>
.search-control {
  position: relative;
  width: min(330px, 100%);
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
  min-width: 146px;
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

.primary-cell,
.model-cell {
  display: grid;
  gap: 2px;
}

.primary-cell strong,
.model-cell strong {
  font-size: 12px;
}

.primary-cell span,
.model-cell span,
.primary-cell small,
.muted-cell {
  color: var(--ink-muted);
  font-size: 10px;
}

.number-cell {
  font-family: var(--font-mono);
  text-align: right;
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
