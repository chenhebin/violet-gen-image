<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  ArrowDownToLine,
  CheckCircle2,
  Image as ImageIcon,
} from '@lucide/vue'
import { PROMPT_SECTION_OPTIONS } from '@/config'
import type { GenerationTask } from '@/types/domain'

const props = defineProps<{ task: GenerationTask }>()
const selectedResult = ref(0)
const selectedUrl = computed(
  () => props.task.results[selectedResult.value]?.url ?? '',
)

watch(
  () => props.task.id,
  () => {
    selectedResult.value = 0
  },
)

function sectionLabel(key: string): string {
  return (
    PROMPT_SECTION_OPTIONS.find((section) => section.key === key)?.label ?? key
  )
}
</script>

<template>
  <div class="task-media-sections">
    <section v-if="task.results.length" class="result-section">
      <div class="section-heading">
        <h3>生成结果</h3>
        <span>{{ task.results.length }} 张</span>
      </div>
      <div class="result-preview">
        <img :src="selectedUrl" alt="任务生成结果" />
        <a
          :href="task.results[selectedResult]?.downloadUrl || selectedUrl"
          :download="
            task.results[selectedResult]?.downloadUrl
              ? undefined
              : `映研-${task.results[selectedResult]?.id}.jpg`
          "
          aria-label="下载当前结果"
          title="下载当前结果"
        >
          <ArrowDownToLine :size="18" />
        </a>
      </div>
      <div class="result-thumbs">
        <button
          v-for="(result, index) in task.results"
          :key="result.id"
          :class="{ active: selectedResult === index }"
          :aria-label="`查看第 ${index + 1} 张结果`"
          @click="selectedResult = index"
        >
          <img :src="result.url" alt="" />
        </button>
      </div>
    </section>

    <section v-if="task.assets.length">
      <div class="section-heading">
        <h3>任务素材</h3>
        <span>{{ task.assets.length }} 张</span>
      </div>
      <div class="asset-grid">
        <div v-for="asset in task.assets" :key="asset.id">
          <img v-if="asset.previewUrl" :src="asset.previewUrl" :alt="asset.name" />
          <span v-else><ImageIcon :size="20" /></span>
          <p>{{ asset.kind === 'source' ? '原图' : '参考图' }}</p>
        </div>
      </div>
    </section>

    <section>
      <div class="section-heading">
        <h3>已确认提示词</h3>
        <CheckCircle2 :size="17" />
      </div>
      <p class="source-copy">{{ task.prompt.source }}</p>
      <dl class="prompt-list">
        <div v-for="(value, key) in task.prompt.sections" :key="key">
          <dt>{{ sectionLabel(key) }}</dt>
          <dd>{{ value }}</dd>
        </div>
      </dl>
    </section>
  </div>
</template>

<style scoped>
.task-media-sections > section {
  padding-top: 24px;
  margin-top: 24px;
  border-top: 1px solid var(--border);
}

.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.section-heading h3 {
  font-size: 14px;
}

.section-heading span {
  color: var(--ink-faint);
  font-size: 10px;
}

.section-heading svg {
  color: var(--primary);
}

.result-preview {
  position: relative;
  overflow: hidden;
  margin-top: 12px;
  border-radius: var(--radius-md);
  background: var(--surface-soft);
}

.result-preview img {
  width: 100%;
  max-height: 430px;
  object-fit: contain;
}

.result-preview a {
  position: absolute;
  right: 10px;
  bottom: 10px;
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  border-radius: 6px;
  background: rgb(255 255 255 / 92%);
  box-shadow: var(--shadow-sm);
}

.result-thumbs {
  display: flex;
  gap: 8px;
  margin-top: 9px;
}

.result-thumbs button {
  overflow: hidden;
  width: 54px;
  height: 54px;
  border: 2px solid transparent;
  border-radius: 6px;
  background: var(--surface-soft);
}

.result-thumbs button.active {
  border-color: var(--primary);
}

.result-thumbs img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.asset-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  margin-top: 12px;
}

.asset-grid > div {
  position: relative;
  overflow: hidden;
  aspect-ratio: 1;
  border-radius: 6px;
  background: var(--surface-soft);
}

.asset-grid img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.asset-grid > div > span {
  display: grid;
  height: 100%;
  place-items: center;
  color: var(--ink-faint);
}

.asset-grid p {
  position: absolute;
  right: 4px;
  bottom: 4px;
  padding: 2px 5px;
  border-radius: 3px;
  background: rgb(255 255 255 / 88%);
  font-size: 8px;
}

.source-copy {
  padding: 12px;
  margin-top: 12px;
  border-left: 3px solid var(--primary);
  background: var(--surface-soft);
  font-size: 12px;
  line-height: 1.65;
}

.prompt-list {
  margin: 8px 0 0;
}

.prompt-list div {
  display: grid;
  grid-template-columns: 66px 1fr;
  gap: 10px;
  padding: 9px 0;
  border-bottom: 1px solid var(--border);
}

.prompt-list dt {
  color: var(--ink-faint);
  font-size: 9px;
}

.prompt-list dd {
  margin: 0;
  color: var(--ink-muted);
  font-size: 11px;
  font-weight: 450;
  line-height: 1.55;
}

@media (max-width: 560px) {
  .asset-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}
</style>
