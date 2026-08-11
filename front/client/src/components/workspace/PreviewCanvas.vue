<script setup lang="ts">
import { Expand, Images, LoaderCircle } from '@lucide/vue'
import {
  isRunningTaskStatus,
  TASK_STAGE_STATUS_LABELS,
  type StageView,
} from '@/config'
import type {
  GenerationSettings,
  GenerationTask,
} from '@/types/domain'

defineProps<{
  task: GenerationTask | null
  displayUrl?: string
  stageView: StageView
  aspectRatio: GenerationSettings['aspectRatio']
}>()

defineEmits<{ expand: [] }>()
</script>

<template>
  <div class="stage-canvas">
    <div
      v-if="task && isRunningTaskStatus(task.status)"
      class="task-progress"
    >
      <div class="progress-icon">
        <LoaderCircle :size="26" />
      </div>
      <p>{{ TASK_STAGE_STATUS_LABELS[task.status] }}</p>
      <strong>{{ task.progress }}%</strong>
      <div class="progress-track" aria-label="预计任务进度">
        <i :style="{ width: `${task.progress}%` }" />
      </div>
      <span>进度为预计值；刷新或离开页面后任务仍会继续。</span>
    </div>

    <div
      v-else-if="displayUrl"
      class="image-proof"
      :class="`ratio-${aspectRatio.replace(':', '-')}`"
    >
      <img
        :src="displayUrl"
        :alt="stageView === 'result' ? '生成结果预览' : '原图预览'"
      />
      <div class="proof-label">
        <span>{{ stageView === 'result' ? '生成成片' : '原始校样' }}</span>
        <button
          aria-label="放大预览"
          title="放大预览"
          @click="$emit('expand')"
        >
          <Expand :size="18" />
        </button>
      </div>
    </div>

    <div v-else class="empty-stage">
      <span><Images :size="30" /></span>
      <h3>校样区等待创作</h3>
      <p>上传的图片与生成结果会在这里展示。</p>
    </div>
  </div>
</template>

<style scoped>
.stage-canvas {
  position: relative;
  display: grid;
  min-height: 0;
  overflow: hidden;
  place-items: center;
  padding: 28px;
  background:
    linear-gradient(45deg, rgb(23 25 29 / 2%) 25%, transparent 25%),
    linear-gradient(-45deg, rgb(23 25 29 / 2%) 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, rgb(23 25 29 / 2%) 75%),
    linear-gradient(-45deg, transparent 75%, rgb(23 25 29 / 2%) 75%);
  background-position:
    0 0,
    0 8px,
    8px -8px,
    -8px 0;
  background-size: 16px 16px;
}

.image-proof {
  position: relative;
  overflow: hidden;
  width: auto;
  max-width: min(100%, 520px);
  height: min(100%, 570px);
  max-height: 100%;
  border: 8px solid white;
  border-bottom-width: 38px;
  border-radius: 3px;
  background: white;
  box-shadow: 0 18px 45px rgb(23 25 29 / 18%);
}

.ratio-1-1 {
  aspect-ratio: 1;
}

.ratio-3-4 {
  aspect-ratio: 3 / 4;
}

.ratio-4-3 {
  aspect-ratio: 4 / 3;
}

.ratio-9-16 {
  max-width: min(74%, 350px);
  aspect-ratio: 9 / 16;
}

.ratio-16-9 {
  aspect-ratio: 16 / 9;
}

.image-proof > img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.proof-label {
  position: absolute;
  right: 0;
  bottom: -34px;
  left: 0;
  display: flex;
  height: 30px;
  align-items: center;
  justify-content: space-between;
  color: var(--ink-muted);
  font-size: 9px;
  font-weight: 750;
  letter-spacing: 0.08em;
}

.proof-label button {
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border-radius: 5px;
  background: transparent;
  color: var(--ink-muted);
}

.empty-stage,
.task-progress {
  display: grid;
  justify-items: center;
  text-align: center;
}

.empty-stage > span,
.progress-icon {
  display: grid;
  width: 64px;
  height: 64px;
  place-items: center;
  border: 1px solid var(--border-strong);
  border-radius: 50%;
  background: rgb(255 255 255 / 75%);
  color: var(--ink-muted);
}

.empty-stage h3 {
  margin-top: 16px;
  font-family: 'Songti SC', serif;
  font-size: 18px;
}

.empty-stage p,
.task-progress span {
  max-width: 290px;
  margin-top: 5px;
  color: var(--ink-muted);
  font-size: 11px;
}

.progress-icon {
  color: var(--primary);
}

.progress-icon svg {
  animation: spin 850ms linear infinite;
}

.task-progress p {
  margin-top: 16px;
  color: var(--ink-muted);
  font-size: 12px;
}

.task-progress strong {
  margin-top: 2px;
  font-size: 32px;
}

.progress-track {
  width: min(260px, 70vw);
  height: 5px;
  overflow: hidden;
  margin-top: 12px;
  border-radius: 99px;
  background: var(--border);
}

.progress-track i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--primary);
  transition: width 500ms var(--ease-out);
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 560px) {
  .stage-canvas {
    padding: 18px;
  }
}
</style>
