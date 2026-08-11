<script setup lang="ts">
import { RotateCcw } from '@lucide/vue'
import TaskStatusBadge from '@/components/tasks/TaskStatusBadge.vue'
import { isRunningTaskStatus } from '@/config'
import type { GenerationTask } from '@/types/domain'

defineProps<{ task: GenerationTask }>()

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
</script>

<template>
  <section class="task-overview">
    <div class="status-row">
      <TaskStatusBadge :status="task.status" />
      <time>{{ formatDate(task.createdAt) }}</time>
    </div>

    <div v-if="isRunningTaskStatus(task.status)" class="progress">
      <div>
        <span>生成进度</span>
        <strong>{{ task.progress }}%</strong>
      </div>
      <i><b :style="{ width: `${task.progress}%` }" /></i>
    </div>

    <dl>
      <div>
        <dt>创作模式</dt>
        <dd>{{ task.mode === 'image-to-image' ? '图生图' : '文生图' }}</dd>
      </div>
      <div>
        <dt>画面比例</dt>
        <dd>{{ task.settings.aspectRatio }}</dd>
      </div>
      <div>
        <dt>本次结算</dt>
        <dd>{{ task.spentCredits }} 次</dd>
      </div>
      <div>
        <dt>结果</dt>
        <dd>{{ task.successfulCount }}/{{ task.requestedCount }} 张</dd>
      </div>
    </dl>

    <p v-if="task.refundedCredits" class="refund-note">
      <RotateCcw :size="16" />
      已自动退回 {{ task.refundedCredits }} 次
    </p>
  </section>
</template>

<style scoped>
.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.status-row time {
  color: var(--ink-faint);
  font-size: 10px;
}

.progress {
  padding: 14px;
  margin-top: 14px;
  border-radius: var(--radius-md);
  background: var(--primary-soft);
}

.progress div {
  display: flex;
  justify-content: space-between;
  color: var(--primary);
  font-size: 11px;
}

.progress i {
  display: block;
  height: 5px;
  overflow: hidden;
  margin-top: 8px;
  border-radius: 99px;
  background: rgb(20 108 99 / 14%);
}

.progress b {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--primary);
}

dl {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  margin: 14px 0 0;
}

dl div {
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
}

dt {
  color: var(--ink-faint);
  font-size: 9px;
}

dd {
  margin: 4px 0 0;
  font-size: 11px;
  font-weight: 650;
}

.refund-note {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 10px 12px;
  margin-top: 10px;
  border-left: 3px solid var(--primary);
  background: var(--primary-soft);
  color: var(--primary);
  font-size: 11px;
}

@media (max-width: 560px) {
  dl {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
