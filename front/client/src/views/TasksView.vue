<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Plus } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import RetouchSubmitModal from '@/components/retouch/RetouchSubmitModal.vue'
import TaskDetailDrawer from '@/components/tasks/TaskDetailDrawer.vue'
import TaskListPanel from '@/components/tasks/TaskListPanel.vue'
import { useToast } from '@/composables/useToast'
import { TASK_TIMING } from '@/config'
import { useEntitlementStore } from '@/stores/entitlement'
import { useRetouchStore } from '@/stores/retouch'
import { useTaskStore } from '@/stores/tasks'
import { useWorkspaceStore } from '@/stores/workspace'
import type { GenerationTask } from '@/types/domain'

const route = useRoute()
const router = useRouter()
const tasks = useTaskStore()
const workspace = useWorkspaceStore()
const entitlement = useEntitlementStore()
const retouch = useRetouchStore()
const toast = useToast()
const retouchTask = ref<GenerationTask | null>(null)
let refreshTimer: number | null = null

const taskId = computed(() =>
  typeof route.params.taskId === 'string' ? route.params.taskId : '',
)

onMounted(async () => {
  await loadTasks()
  refreshTimer = window.setInterval(() => {
    if (tasks.hasRunningTasks) void loadTasks(false)
  }, TASK_TIMING.listRefreshMs)
})

onBeforeUnmount(() => {
  if (refreshTimer !== null) window.clearInterval(refreshTimer)
  tasks.close()
})

watch(
  taskId,
  async (id) => {
    if (!id) {
      tasks.close()
      return
    }
    try {
      await tasks.open(id)
    } catch (caught) {
      toast.error(
        '任务无法打开',
        caught instanceof Error ? caught.message : '请稍后重试',
      )
      await router.replace('/app/tasks')
    }
  },
  { immediate: true },
)

async function loadTasks(reportError = true): Promise<void> {
  try {
    await tasks.load()
  } catch (caught) {
    if (reportError) {
      toast.error(
        '任务记录加载失败',
        caught instanceof Error ? caught.message : '请稍后重试',
      )
    }
  }
}

async function closeDetail(): Promise<void> {
  tasks.close()
  await router.push('/app/tasks')
}

async function cancelTask(id: string): Promise<void> {
  try {
    await tasks.cancel(id)
    await entitlement.load()
    toast.success('任务已取消', '预占次数已全部退回')
  } catch (caught) {
    toast.error(
      '无法取消任务',
      caught instanceof Error ? caught.message : '任务状态可能已经变化',
    )
  }
}

async function reuseTask(task: GenerationTask): Promise<void> {
  workspace.reuseTask(task)
  toast.info('参数已复用', '请检查并重新确认提示词方案')
  await router.push('/app/create')
}

function openRetouchRequest(task: GenerationTask): void {
  toast.clear()
  retouchTask.value = task
}

function closeRetouchRequest(): void {
  if (!retouch.actionLoading) retouchTask.value = null
}

async function submitRetouchRequest(
  selectedResultIds: string[],
  requirement: string,
  supplementalFiles: File[],
): Promise<void> {
  const task = retouchTask.value
  if (!task) return

  try {
    const ticket = await retouch.createWithFiles(
      task.id,
      selectedResultIds,
      requirement,
      supplementalFiles,
    )
    if (!ticket) return
    tasks.upsert({
      ...task,
      retouchTicket: {
        id: ticket.id,
        ticketNo: ticket.ticketNo,
        status: ticket.status,
        updatedAt: ticket.updatedAt,
        quoteCredits: ticket.quote?.credits,
      },
    })
    retouchTask.value = null
    toast.success('人工精修需求已提交', `工单编号 ${ticket.ticketNo}`)
  } catch (caught) {
    toast.error(
      '人工精修需求提交失败',
      caught instanceof Error ? caught.message : '请检查内容后重试',
    )
  }
}

async function viewRetouchRecord(ticketId: string): Promise<void> {
  await router.push(`/app/retouch/${ticketId}`)
}
</script>

<template>
  <div class="tasks-page">
    <header class="page-heading">
      <div>
        <span>创作档案</span>
        <h1>任务记录</h1>
        <p>查看生成进度、结果和每次次数结算。</p>
      </div>
      <BaseButton @click="router.push('/app/create')">
        <template #icon><Plus :size="18" /></template>
        创建新任务
      </BaseButton>
    </header>

    <TaskListPanel
      :tasks="tasks.tasks"
      :loading="tasks.loading"
      @open="router.push(`/app/tasks/${$event}`)"
      @create="router.push('/app/create')"
    />

    <TaskDetailDrawer
      :open="Boolean(taskId)"
      :task="tasks.activeTask"
      :loading="tasks.loading"
      @close="closeDetail"
      @cancel="cancelTask"
      @reuse="reuseTask"
      @retouch="openRetouchRequest"
      @view-retouch="viewRetouchRecord"
    />

    <RetouchSubmitModal
      :open="Boolean(retouchTask)"
      :task="retouchTask"
      :loading="retouch.actionLoading"
      @close="closeRetouchRequest"
      @submit="submitRetouchRequest"
    />
  </div>
</template>

<style scoped>
.tasks-page {
  width: min(1180px, calc(100% - 40px));
  padding: 36px 0 60px;
  margin: 0 auto;
}

.page-heading {
  display: flex;
  min-height: 100px;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 26px;
}

.page-heading span {
  color: var(--primary);
  font-size: 10px;
  font-weight: 800;
}

.page-heading h1 {
  margin-top: 3px;
  font-family: 'Songti SC', 'STSong', serif;
  font-size: 32px;
  font-weight: 600;
}

.page-heading p {
  margin-top: 5px;
  color: var(--ink-muted);
  font-size: 13px;
}

@media (max-width: 760px) {
  .tasks-page {
    width: calc(100% - 24px);
    padding: 18px 0 28px;
  }

  .page-heading {
    min-height: 0;
    align-items: stretch;
    flex-direction: column;
    gap: 14px;
    margin-bottom: 18px;
  }

  .page-heading h1 {
    font-size: 28px;
  }

  .page-heading p {
    max-width: 300px;
    line-height: 1.65;
  }

  .page-heading .base-button {
    align-self: flex-start;
  }
}
</style>
