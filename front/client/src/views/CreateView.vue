<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from 'vue'
import {
  AlignLeft,
  Eye,
  FileImage,
} from '@lucide/vue'
import AssetRail from '@/components/workspace/AssetRail.vue'
import AIProcessingNoticeModal from '@/components/workspace/AIProcessingNoticeModal.vue'
import PreviewStage from '@/components/workspace/PreviewStage.vue'
import PromptPanel from '@/components/workspace/PromptPanel.vue'
import QuoteBar from '@/components/workspace/QuoteBar.vue'
import { useToast } from '@/composables/useToast'
import { isFinalTaskStatus } from '@/config'
import { useEntitlementStore } from '@/stores/entitlement'
import { useTaskStore } from '@/stores/tasks'
import { useWorkspaceStore } from '@/stores/workspace'
import { aiNoticeApi, type AIProcessingNotice } from '@/services/api'
import type { GenerationTask } from '@/types/domain'

const workspace = useWorkspaceStore()
const entitlement = useEntitlementStore()
const tasks = useTaskStore()
const toast = useToast()
const activeSection = ref('assets')
const aiNotice = ref<AIProcessingNotice | null>(null)
const aiNoticeLoading = ref(false)
const aiNoticeOpen = computed(() => Boolean(aiNotice.value && !aiNotice.value.acknowledged))
let sectionFrame: number | null = null

const workspaceSections = [
  { id: 'assets', label: '素材', icon: FileImage },
  { id: 'prompt', label: '方案', icon: AlignLeft },
  { id: 'preview', label: '预览', icon: Eye },
] as const

const ready = computed(() => workspace.canSubmit)

watch(
  [
    () => workspace.draft.settings.outputCount,
    () => entitlement.balance,
  ],
  ([count]) => {
    void entitlement.requestQuote(count).catch((caught) => {
      toast.error(
        '无法获取报价',
        caught instanceof Error ? caught.message : '请稍后重试',
      )
    })
  },
  { immediate: true },
)

onMounted(() => {
  if (!entitlement.entitlement) void entitlement.load()
  void loadAINotice()
  window.addEventListener('scroll', scheduleActiveSection, { passive: true })
  window.addEventListener('resize', scheduleActiveSection)
  void nextTick(scheduleActiveSection)
})

async function loadAINotice(): Promise<void> {
  try {
    aiNotice.value = await aiNoticeApi.get()
  } catch (caught) {
    toast.error('无法加载服务告知', caught instanceof Error ? caught.message : '请稍后重试')
  }
}

async function acknowledgeAINotice(): Promise<void> {
  if (!aiNotice.value) return
  aiNoticeLoading.value = true
  try {
    aiNotice.value = await aiNoticeApi.acknowledge(aiNotice.value.version)
  } catch (caught) {
    toast.error('确认未完成', caught instanceof Error ? caught.message : '请稍后重试')
  } finally {
    aiNoticeLoading.value = false
  }
}

onBeforeUnmount(() => {
  window.removeEventListener('scroll', scheduleActiveSection)
  window.removeEventListener('resize', scheduleActiveSection)
  if (sectionFrame !== null) window.cancelAnimationFrame(sectionFrame)
  tasks.stopMonitoring()
})

function scheduleActiveSection(): void {
  if (sectionFrame !== null) return
  sectionFrame = window.requestAnimationFrame(() => {
    sectionFrame = null
    const anchor = window.innerHeight * 0.48
    const sections = workspaceSections
      .map((section) => ({
        id: section.id,
        top:
          document
            .getElementById(`workspace-${section.id}`)
            ?.getBoundingClientRect().top ?? Number.POSITIVE_INFINITY,
      }))
      .filter((section) => Number.isFinite(section.top))
    const current =
      sections.filter((section) => section.top <= anchor).at(-1) ?? sections[0]
    if (current) activeSection.value = current.id
  })
}

function scrollToSection(section: string): void {
  activeSection.value = section
  document
    .getElementById(`workspace-${section}`)
    ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

async function submit(): Promise<void> {
  try {
    if (aiNoticeOpen.value) {
      toast.info('请先确认第三方 AI 处理告知', '确认后才能使用提示词优化和生图')
      return
    }
    if (workspace.referenceOptimizationRequired) {
      await workspace.submit()
      toast.success('参考图提示词已生成', '请检查并确认优化方案，再生成图片')
      await nextTick()
      scrollToSection('prompt')
      return
    }
    if (workspace.promptNeedsConfirmation) {
      toast.info('请先确认提示词方案', '确认后才能提交生成')
      scrollToSection('prompt')
      return
    }
    const latestQuote = await entitlement.requestQuote(
      workspace.draft.settings.outputCount,
    )
    if (!latestQuote?.canSubmit) {
      toast.error('剩余次数不足', '请使用顶部的兑换码入口补充次数')
      return
    }

    const task = await workspace.submit()
    if (!task) return
    tasks.upsert(task)
    if (task.entitlement) {
      entitlement.setFromServer(task.entitlement)
      await entitlement.refreshLedger()
    } else {
      // Keep test doubles and older API deployments safe while the server
      // response migrates to the authoritative task + entitlement shape.
      await entitlement.load()
    }
    toast.success('任务已提交', '生成进度会直接显示在校样台')
    tasks.monitor(task.id, handleTaskUpdate, (message) => {
      toast.error('任务状态同步失败', message)
    })
    await nextTick()
    scrollToSection('preview')
  } catch (caught) {
    toast.error(
      '任务未能提交',
      caught instanceof Error ? caught.message : '请检查当前方案',
    )
    await entitlement.load().catch(() => undefined)
  }
}

function handleTaskUpdate(task: GenerationTask): void {
  workspace.currentTask = task
  if (isFinalTaskStatus(task.status)) {
    void entitlement.load()
    if (task.status === 'completed') {
      toast.success('生成完成', `${task.successfulCount} 张成片已就绪`)
    } else if (task.refundedCredits) {
      toast.info(
        '任务已结算',
        `成功 ${task.successfulCount} 张，退回 ${task.refundedCredits} 次`,
      )
    }
  }
}
</script>

<template>
  <div class="create-page">
    <header class="workspace-toolbar">
      <div>
        <span>映研工作台</span>
        <h1>完成一次影像创作</h1>
      </div>
    </header>

    <nav class="mobile-workspace-nav" aria-label="创作步骤定位">
      <button
        v-for="section in workspaceSections"
        :key="section.id"
        :class="{ active: activeSection === section.id }"
        :aria-current="activeSection === section.id ? 'step' : undefined"
        @click="scrollToSection(section.id)"
      >
        <component :is="section.icon" :size="17" aria-hidden="true" />
        <span>{{ section.label }}</span>
      </button>
    </nav>

    <div class="workspace-grid">
      <div
        id="workspace-assets"
        class="workspace-panel workspace-assets"
        data-workspace-section
      >
        <AssetRail />
      </div>
      <div
        id="workspace-preview"
        class="workspace-panel workspace-preview"
        data-workspace-section
      >
        <PreviewStage />
      </div>
      <div
        id="workspace-prompt"
        class="workspace-panel workspace-prompt"
        data-workspace-section
      >
        <PromptPanel>
          <template #quote>
            <QuoteBar
              :quote="entitlement.quote"
              :quoting="entitlement.quoting"
              :submitting="workspace.submitting"
              :ready="ready"
              :output-count="workspace.draft.settings.outputCount"
              :reference-optimization-required="workspace.referenceOptimizationRequired"
              :prompt-needs-confirmation="workspace.promptNeedsConfirmation"
              @submit="submit"
            />
          </template>
        </PromptPanel>
      </div>
    </div>

    <AIProcessingNoticeModal
      :open="aiNoticeOpen"
      :notice="aiNotice"
      :loading="aiNoticeLoading"
      @acknowledge="acknowledgeAINotice"
    />
  </div>
</template>

<style scoped>
.create-page {
  display: grid;
  width: 100%;
  height: calc(100dvh - var(--header-height));
  min-height: 0;
  grid-template-rows: auto minmax(0, 1fr);
  overflow: hidden;
  padding: 20px;
}

.workspace-toolbar {
  display: flex;
  align-items: center;
  max-width: 1600px;
  min-height: 62px;
  margin: 0 auto 16px;
}

.workspace-toolbar > div:first-child > span {
  color: var(--primary);
  font-size: 10px;
  font-weight: 800;
}

.workspace-toolbar h1 {
  margin-top: 1px;
  font-family: 'Songti SC', 'STSong', serif;
  font-size: 22px;
  font-weight: 600;
}

.workspace-grid {
  display: grid;
  width: 100%;
  height: 100%;
  max-width: 1600px;
  min-height: 0;
  grid-template-columns: minmax(220px, 0.72fr) minmax(380px, 1.7fr) minmax(
      315px,
      0.95fr
    );
  align-items: stretch;
  gap: 14px;
  margin: 0 auto;
}

.workspace-panel {
  min-width: 0;
  min-height: 0;
}

.workspace-panel > :deep(*) {
  height: 100%;
}

.mobile-workspace-nav {
  display: none;
}

@media (min-width: 901px) {
  .workspace-panel {
    height: 100%;
    min-height: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .workspace-preview {
    overflow: hidden;
  }

  .workspace-prompt {
    overflow: hidden;
  }
}

@media (max-width: 1100px) {
  .create-page {
    padding: 16px;
  }

  .workspace-grid {
    grid-template-columns: minmax(210px, 0.72fr) minmax(330px, 1.4fr) minmax(
        285px,
        0.9fr
      );
    gap: 10px;
  }

}

@media (max-width: 900px) {
  .create-page {
    display: block;
    height: auto;
    min-height: 0;
    overflow: visible;
  }

  .workspace-grid {
    height: auto;
    grid-template-columns: 1fr;
  }

  .workspace-panel {
    height: auto;
    overflow: visible;
    scroll-margin-top: calc(var(--header-height) + 58px);
  }

  .workspace-panel > :deep(*) {
    height: auto;
  }

  .workspace-assets {
    order: 1;
  }

  .workspace-prompt {
    order: 2;
  }

  .workspace-preview {
    order: 3;
  }
}

@media (max-width: 560px) {
  .create-page {
    padding: 12px;
  }

  .workspace-toolbar {
    margin-bottom: 12px;
  }

  .workspace-toolbar h1 {
    font-size: 20px;
  }

  .mobile-workspace-nav {
    position: sticky;
    z-index: 12;
    top: calc(var(--header-height) + 8px);
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 4px;
    padding: 4px;
    margin-bottom: 12px;
    border: 1px solid rgb(220 225 231 / 85%);
    border-radius: 8px;
    background: rgb(255 255 255 / 90%);
    box-shadow: 0 10px 28px rgb(23 25 29 / 8%);
    backdrop-filter: blur(16px) saturate(1.1);
  }

  .mobile-workspace-nav button {
    display: inline-flex;
    min-height: 44px;
    align-items: center;
    justify-content: center;
    gap: 6px;
    border-radius: 6px;
    background: transparent;
    color: var(--ink-muted);
    font-size: 11px;
    font-weight: 700;
    transition:
      background var(--motion-fast),
      color var(--motion-fast),
      transform var(--motion-fast);
  }

  .mobile-workspace-nav button.active {
    background: var(--primary-soft);
    color: var(--primary);
  }

  .mobile-workspace-nav button:active {
    transform: scale(0.97);
  }

  .workspace-grid {
    gap: 12px;
  }
}
</style>
