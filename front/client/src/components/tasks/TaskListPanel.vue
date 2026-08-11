<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  ArrowRight,
  Image as ImageIcon,
  ListFilter,
  LoaderCircle,
} from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import RetouchStatusBadge from '@/components/retouch/RetouchStatusBadge.vue'
import TaskStatusBadge from '@/components/tasks/TaskStatusBadge.vue'
import {
  isFinalTaskStatus,
  isRunningTaskStatus,
  TASK_FILTER_OPTIONS,
  type TaskFilter,
} from '@/config'
import type { GenerationTask } from '@/types/domain'

const props = defineProps<{
  tasks: GenerationTask[]
  loading: boolean
}>()

defineEmits<{
  open: [taskId: string]
  create: []
}>()

const filter = ref<TaskFilter>('all')
const filteredTasks = computed(() => {
  if (filter.value === 'running') {
    return props.tasks.filter((task) => isRunningTaskStatus(task.status))
  }
  if (filter.value === 'completed') {
    return props.tasks.filter((task) => isFinalTaskStatus(task.status))
  }
  return props.tasks
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
  <section class="task-surface">
    <header class="task-toolbar">
      <div class="filters" role="radiogroup" aria-label="任务筛选">
        <ListFilter :size="17" aria-hidden="true" />
        <button
          v-for="item in TASK_FILTER_OPTIONS"
          :key="item.value"
          :class="{ active: filter === item.value }"
          role="radio"
          :aria-checked="filter === item.value"
          @click="filter = item.value"
        >
          {{ item.label }}
        </button>
      </div>
      <span>共 {{ filteredTasks.length }} 个任务</span>
    </header>

    <div class="table-heading" aria-hidden="true">
      <span>任务</span>
      <span>状态</span>
      <span>结算</span>
      <span>创建时间</span>
      <span />
    </div>

    <div v-if="loading && !tasks.length" class="loading-state">
      <LoaderCircle :size="24" />
      <p>正在读取任务记录…</p>
    </div>

    <div v-else-if="filteredTasks.length" class="task-list">
      <button
        v-for="task in filteredTasks"
        :key="task.id"
        class="task-row"
        @click="$emit('open', task.id)"
      >
        <span class="task-main">
          <span class="task-thumb">
            <img
              v-if="task.results[0]?.url || task.assets[0]?.previewUrl"
              :src="task.results[0]?.url || task.assets[0]?.previewUrl"
              alt=""
            />
            <ImageIcon v-else :size="19" />
          </span>
          <span>
            <strong>{{ task.title }}</strong>
            <small>
              {{ task.mode === 'image-to-image' ? '图生图' : '文生图' }}
              · {{ task.settings.aspectRatio }}
            </small>
          </span>
        </span>
        <span class="status-cell">
          <TaskStatusBadge :status="task.status" />
          <RetouchStatusBadge
            v-if="task.retouchTicket"
            :status="task.retouchTicket.status"
          />
          <small v-if="isRunningTaskStatus(task.status)">
            {{ task.progress }}%
          </small>
        </span>
        <span class="settlement">
          <strong>{{ task.spentCredits }} 次</strong>
          <small v-if="task.refundedCredits">
            退回 {{ task.refundedCredits }} 次
          </small>
          <small v-else>{{ task.successfulCount }} 张结果</small>
        </span>
        <time>{{ formatDate(task.createdAt) }}</time>
        <ArrowRight :size="18" class="row-arrow" />
      </button>
    </div>

    <div v-else class="empty-state">
      <span><ImageIcon :size="26" /></span>
      <h2>{{ filter === 'all' ? '还没有创作任务' : '这个分类暂时为空' }}</h2>
      <p>完成素材与提示词确认后，提交的任务会出现在这里。</p>
      <BaseButton variant="secondary" @click="$emit('create')">
        前往工作台
      </BaseButton>
    </div>
  </section>
</template>

<style scoped>
.task-surface {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.task-toolbar {
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

.task-toolbar > span {
  color: var(--ink-faint);
  font-size: 11px;
}

.table-heading,
.task-row {
  display: grid;
  grid-template-columns: minmax(260px, 2.3fr) minmax(130px, 1fr) minmax(
      110px,
      0.8fr
    ) minmax(120px, 0.8fr) 28px;
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

.task-row {
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

.task-row:last-child {
  border-bottom: 0;
}

.task-row:hover {
  background: #fafbfc;
}

.task-row:active {
  transform: scale(0.995);
}

.task-main {
  display: grid;
  min-width: 0;
  grid-template-columns: 54px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
}

.task-thumb {
  display: grid;
  overflow: hidden;
  width: 54px;
  height: 54px;
  place-items: center;
  border-radius: 6px;
  background: var(--surface-soft);
  color: var(--ink-faint);
}

.task-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.task-main strong,
.task-main small,
.settlement strong,
.settlement small {
  display: block;
}

.task-main strong {
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-main small,
.settlement small,
.status-cell small,
.task-row time {
  color: var(--ink-faint);
  font-size: 10px;
}

.status-cell {
  display: grid;
  justify-items: start;
  gap: 3px;
}

.settlement strong {
  font-size: 12px;
}

.row-arrow {
  color: var(--ink-faint);
}

.loading-state,
.empty-state {
  display: grid;
  min-height: 360px;
  place-items: center;
  align-content: center;
  gap: 10px;
  color: var(--ink-muted);
  text-align: center;
}

.loading-state svg {
  animation: spin 800ms linear infinite;
}

.loading-state p,
.empty-state p {
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

.empty-state .base-button {
  margin-top: 8px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 760px) {
  .task-surface {
    overflow: visible;
    border: 0;
    background: transparent;
    box-shadow: none;
  }

  .task-toolbar {
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

  .task-list {
    display: grid;
    gap: 10px;
  }

  .table-heading {
    display: none;
  }

  .task-row {
    grid-template-columns: 1fr auto;
    gap: 10px;
    min-height: 116px;
    padding: 14px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    box-shadow: 0 5px 18px rgb(23 25 29 / 4%);
    animation: row-in var(--motion-normal) var(--ease-out) both;
  }

  .task-row:nth-child(2) {
    animation-delay: 40ms;
  }

  .task-row:nth-child(3) {
    animation-delay: 80ms;
  }

  .task-row:nth-child(4) {
    animation-delay: 120ms;
  }

  .task-main {
    grid-column: 1 / -1;
  }

  .status-cell,
  .settlement {
    align-self: start;
  }

  .task-row time {
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
