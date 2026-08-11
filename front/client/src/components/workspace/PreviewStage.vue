<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import SegmentedControl from '@/components/base/SegmentedControl.vue'
import GenerationResultBar from '@/components/workspace/GenerationResultBar.vue'
import PreviewCanvas from '@/components/workspace/PreviewCanvas.vue'
import {
  STAGE_VIEW_OPTIONS,
  type StageView,
} from '@/config'
import { useWorkspaceStore } from '@/stores/workspace'

const workspace = useWorkspaceStore()
const stageView = ref<StageView>('source')
const selectedResult = ref(0)
const lightboxOpen = ref(false)

const task = computed(() => workspace.currentTask)
const results = computed(() => task.value?.results ?? [])
const selectedSource = computed(() => workspace.sourceAssets[0]?.previewUrl)
const selectedOutput = computed(() => results.value[selectedResult.value])
const displayUrl = computed(() =>
  stageView.value === 'result'
    ? selectedOutput.value?.url
    : selectedSource.value ?? workspace.referenceAssets[0]?.previewUrl,
)
const hasResults = computed(() => results.value.length > 0)

watch(
  () => task.value?.id,
  () => {
    selectedResult.value = 0
    stageView.value = task.value?.results.length ? 'result' : 'source'
  },
)

watch(
  () => task.value?.results.length,
  (count) => {
    if (count) stageView.value = 'result'
  },
)

function selectResult(index: number): void {
  selectedResult.value = index
  stageView.value = 'result'
}
</script>

<template>
  <section class="preview-stage" aria-label="图片预览与生成结果">
    <header class="stage-heading">
      <div>
        <span>预览</span>
        <h2>成片校样台</h2>
      </div>
      <SegmentedControl
        v-if="hasResults && selectedSource"
        v-model="stageView"
        class="view-switch"
        label="查看原图或结果"
        :options="STAGE_VIEW_OPTIONS"
      />
      <span v-else class="ratio-label">
        {{ workspace.draft.settings.aspectRatio }}
      </span>
    </header>

    <PreviewCanvas
      :task="task"
      :display-url="displayUrl"
      :stage-view="stageView"
      :aspect-ratio="workspace.draft.settings.aspectRatio"
      @expand="lightboxOpen = true"
    />

    <GenerationResultBar
      v-if="task"
      :task="task"
      :selected-result="selectedResult"
      :stage-view="stageView"
      @select="selectResult"
    />

    <BaseModal
      :open="lightboxOpen"
      title="图片预览"
      size="wide"
      @close="lightboxOpen = false"
    >
      <img
        v-if="displayUrl"
        class="lightbox-image"
        :src="displayUrl"
        alt="放大预览"
      />
    </BaseModal>
  </section>
</template>

<style scoped>
.preview-stage {
  display: grid;
  min-width: 0;
  min-height: 0;
  grid-template-rows: auto minmax(0, 1fr) auto;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: #eef0f2;
  box-shadow: var(--shadow-sm);
}

.stage-heading {
  display: flex;
  min-height: 70px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px 12px 18px;
  border-bottom: 1px solid var(--border);
  background: var(--surface);
}

.stage-heading > div > span {
  color: var(--primary);
  font-size: 10px;
  font-weight: 800;
}

.stage-heading h2 {
  margin-top: 2px;
  font-size: 16px;
}

.ratio-label {
  padding: 5px 8px;
  border-radius: 5px;
  background: var(--surface-soft);
  color: var(--ink-muted);
  font-size: 10px;
  font-weight: 750;
}

.view-switch {
  min-height: 38px;
  grid-auto-columns: minmax(64px, 1fr);
}

.lightbox-image {
  width: 100%;
  max-height: 70dvh;
  object-fit: contain;
}

@media (max-width: 900px) {
  .preview-stage {
    min-height: 600px;
    grid-template-rows: auto minmax(420px, 1fr) auto;
  }
}

@media (max-width: 560px) {
  .preview-stage {
    min-height: 480px;
    grid-template-rows: auto minmax(310px, 1fr) auto;
  }

  .view-switch {
    width: auto;
  }
}
</style>
