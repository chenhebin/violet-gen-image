<script setup lang="ts">
import { computed } from 'vue'
import { ArrowDownToLine, CheckCircle2, RotateCcw } from '@lucide/vue'
import {
  isRunningTaskStatus,
  TASK_STAGE_STATUS_LABELS,
  type StageView,
} from '@/config'
import type { GenerationTask } from '@/types/domain'
import { downloadAsset } from '@/utils/download'

const props = defineProps<{
  task: GenerationTask
  selectedResult: number
  stageView: StageView
}>()

defineEmits<{ select: [index: number] }>()

const selectedOutput = computed(
  () => props.task.results[props.selectedResult],
)

async function downloadSelected(): Promise<void> {
  if (!selectedOutput.value) return
  await downloadAsset({
    assetId: selectedOutput.value.id,
    currentUrl: selectedOutput.value.downloadUrl || selectedOutput.value.url,
    filename: `映研-${selectedOutput.value.id}.jpg`,
  })
}
</script>

<template>
  <div class="generation-results">
    <div
      v-if="!isRunningTaskStatus(task.status)"
      class="result-footer"
    >
      <div class="result-status">
        <CheckCircle2
          v-if="task.status === 'completed' || task.status === 'partial'"
          :size="18"
        />
        <RotateCcw v-else :size="18" />
        <div>
          <strong>{{ TASK_STAGE_STATUS_LABELS[task.status] }}</strong>
          <span v-if="task.refundedCredits">
            已退回 {{ task.refundedCredits }} 次
          </span>
          <span v-else>
            成功 {{ task.successfulCount }}/{{ task.requestedCount }} 张
          </span>
        </div>
      </div>
      <a
        v-if="selectedOutput"
        class="download-button"
        :href="selectedOutput.downloadUrl || selectedOutput.url"
        :download="
          selectedOutput.downloadUrl ? undefined : `映研-${selectedOutput.id}.jpg`
        "
        @click.prevent="downloadSelected"
      >
        <ArrowDownToLine :size="17" />
        下载当前图片
      </a>
    </div>

    <div v-if="task.results.length" class="result-strip" aria-label="生成结果">
      <button
        v-for="(result, index) in task.results"
        :key="result.id"
        :class="{ active: selectedResult === index && stageView === 'result' }"
        :aria-label="`查看第 ${index + 1} 张结果`"
        @click="$emit('select', index)"
      >
        <img :src="result.url" alt="" />
        <span>{{ String(index + 1).padStart(2, '0') }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.result-footer {
  display: flex;
  min-height: 68px;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 10px 16px;
  border-top: 1px solid var(--border);
  background: var(--surface);
}

.result-status {
  display: flex;
  align-items: center;
  gap: 9px;
  color: var(--primary);
}

.result-status strong,
.result-status span {
  display: block;
}

.result-status strong {
  color: var(--ink);
  font-size: 12px;
}

.result-status span {
  color: var(--ink-muted);
  font-size: 10px;
}

.download-button {
  display: inline-flex;
  min-height: 40px;
  align-items: center;
  gap: 7px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface);
  color: var(--ink);
  font-size: 11px;
  font-weight: 680;
}

.result-strip {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding: 10px 14px;
  border-top: 1px solid var(--border);
  background: var(--surface);
}

.result-strip button {
  position: relative;
  overflow: hidden;
  width: 58px;
  height: 58px;
  flex: 0 0 58px;
  border: 2px solid transparent;
  border-radius: 6px;
  background: var(--surface-soft);
}

.result-strip button.active {
  border-color: var(--primary);
}

.result-strip img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.result-strip span {
  position: absolute;
  right: 3px;
  bottom: 3px;
  padding: 1px 3px;
  border-radius: 3px;
  background: rgb(23 25 29 / 72%);
  color: white;
  font-size: 8px;
}

@media (max-width: 560px) {
  .result-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .download-button {
    justify-content: center;
  }
}
</style>
