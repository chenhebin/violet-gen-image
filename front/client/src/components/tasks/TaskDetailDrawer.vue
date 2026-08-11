<script setup lang="ts">
import {
  ArrowRight,
  Ban,
  Clock3,
  CopyPlus,
  Sparkles,
  X,
} from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import TaskMediaSections from '@/components/tasks/TaskMediaSections.vue'
import TaskOverview from '@/components/tasks/TaskOverview.vue'
import type { GenerationTask } from '@/types/domain'

defineProps<{
  open: boolean
  task: GenerationTask | null
  loading: boolean
}>()

defineEmits<{
  close: []
  cancel: [taskId: string]
  reuse: [task: GenerationTask]
  retouch: [task: GenerationTask]
  viewRetouch: [ticketId: string]
}>()
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="open" class="drawer-layer">
        <button
          class="drawer-scrim"
          aria-label="关闭任务详情"
          @click="$emit('close')"
        />
        <aside role="dialog" aria-modal="true" aria-labelledby="task-detail-heading">
          <header>
            <div>
              <p>任务详情</p>
              <h2 id="task-detail-heading">{{ task?.title ?? '正在加载' }}</h2>
            </div>
            <button class="icon-button" aria-label="关闭" @click="$emit('close')">
              <X :size="20" />
            </button>
          </header>

          <div v-if="loading && !task" class="loading-state">
            <Clock3 :size="25" />
            <p>正在读取任务…</p>
          </div>

          <div v-else-if="task" class="drawer-content">
            <TaskOverview :task="task" />
            <TaskMediaSections :task="task" />
          </div>

          <footer v-if="task">
            <BaseButton
              v-if="task.status === 'queued'"
              variant="danger"
              @click="$emit('cancel', task.id)"
            >
              <template #icon><Ban :size="17" /></template>
              取消并退回次数
            </BaseButton>
            <BaseButton
              v-if="
                task.retouchTicket &&
                  task.retouchTicket.status !== 'cancelled' &&
                  task.retouchTicket.status !== 'rejected'
              "
              @click="$emit('viewRetouch', task.retouchTicket.id)"
            >
              <template #icon><ArrowRight :size="17" /></template>
              查看人工修图记录
            </BaseButton>
            <BaseButton
              v-else-if="
                (!task.retouchTicket ||
                  task.retouchTicket.status === 'cancelled' ||
                  task.retouchTicket.status === 'rejected') &&
                  (task.status === 'completed' || task.status === 'partial') &&
                  task.results.length > 0
              "
              @click="$emit('retouch', task)"
            >
              <template #icon><Sparkles :size="17" /></template>
              {{ task.retouchTicket ? '重新申请人工精修' : '申请人工精修' }}
            </BaseButton>
            <BaseButton
              v-if="
                task.retouchTicket &&
                  (task.retouchTicket.status === 'cancelled' ||
                    task.retouchTicket.status === 'rejected')
              "
              variant="secondary"
              @click="$emit('viewRetouch', task.retouchTicket.id)"
            >
              <template #icon><ArrowRight :size="17" /></template>
              查看历史工单
            </BaseButton>
            <BaseButton variant="secondary" @click="$emit('reuse', task)">
              <template #icon><CopyPlus :size="17" /></template>
              复用参数创建新任务
            </BaseButton>
          </footer>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.drawer-layer {
  position: fixed;
  z-index: 76;
  inset: 0;
}

.drawer-scrim {
  position: absolute;
  inset: 0;
  width: 100%;
  background: var(--scrim);
  cursor: default;
}

aside {
  position: absolute;
  top: 0;
  right: 0;
  display: grid;
  width: 66.6667vw;
  min-width: 720px;
  max-width: 100%;
  height: 100%;
  grid-template-rows: auto minmax(0, 1fr) auto;
  border-left: 1px solid var(--border);
  background: var(--surface);
  box-shadow: var(--shadow-md);
}

header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
}

header p {
  color: var(--primary);
  font-size: 10px;
  font-weight: 750;
}

header h2 {
  margin-top: 2px;
  font-size: 18px;
}

.icon-button {
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--ink-muted);
}

.icon-button:hover {
  background: var(--surface-soft);
}

.drawer-content {
  min-height: 0;
  overflow: auto;
  padding: 22px 24px 36px;
}

.loading-state {
  display: grid;
  place-items: center;
  align-content: center;
  gap: 12px;
  color: var(--ink-muted);
}

footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 16px 24px;
  border-top: 1px solid var(--border);
  background: var(--surface);
}

.drawer-enter-active,
.drawer-leave-active {
  transition: opacity var(--motion-normal) var(--ease-out);
}

.drawer-enter-active aside,
.drawer-leave-active aside {
  transition: transform var(--motion-normal) var(--ease-out);
}

.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}

.drawer-enter-from aside,
.drawer-leave-to aside {
  transform: translateX(100%);
}

@media (max-width: 560px) {
  footer {
    align-items: stretch;
    flex-direction: column;
  }
}

@media (max-width: 760px) {
  aside {
    width: 100%;
    min-width: 0;
  }
}
</style>
