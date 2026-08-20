<script setup lang="ts">
import { Maximize2, RotateCcw } from '@lucide/vue'
import type {
  Asset,
  GenerationResult,
  RetouchRevision,
} from '@/types/domain'

defineProps<{
  selectedResults: GenerationResult[]
  supplementalAssets: Asset[]
  requirement: string
  revision?: RetouchRevision
}>()

const emit = defineEmits<{
  preview: [id: string]
  imageError: [id: string]
}>()
</script>

<template>
  <section class="request-section">
    <header>
      <div>
        <p>精修需求</p>
        <h3>提交内容</h3>
      </div>
      <span>{{ selectedResults.length }} 张原结果</span>
    </header>

    <div class="media-strip">
      <figure v-for="(result, index) in selectedResults" :key="result.id">
        <button
          type="button"
          class="media-preview-button"
          :aria-label="`放大查看待精修原结果 ${index + 1}`"
          @click="emit('preview', `selected:${result.id}`)"
        >
          <img
            :src="result.url"
            :alt="`待精修原结果 ${index + 1}`"
            @error="emit('imageError', `selected:${result.id}`)"
          />
          <span class="preview-indicator" aria-hidden="true">
            <Maximize2 :size="15" />
          </span>
        </button>
        <figcaption>原结果 {{ index + 1 }}</figcaption>
      </figure>
    </div>

    <div class="requirement-copy">
      <strong>处理要求</strong>
      <p>{{ requirement }}</p>
    </div>

    <div v-if="supplementalAssets.length" class="supplemental">
      <strong>补充参考</strong>
      <div class="asset-row">
        <figure v-for="asset in supplementalAssets" :key="asset.id">
          <button
            v-if="asset.previewUrl"
            type="button"
            class="media-preview-button"
            :aria-label="`放大查看补充参考图：${asset.name}`"
            @click="emit('preview', `supplemental:${asset.id}`)"
          >
            <img
              :src="asset.previewUrl"
              :alt="asset.name"
              @error="emit('imageError', `supplemental:${asset.id}`)"
            />
            <span class="preview-indicator" aria-hidden="true">
              <Maximize2 :size="15" />
            </span>
          </button>
          <span>{{ asset.name }}</span>
        </figure>
      </div>
    </div>

    <div v-if="revision" class="revision-note">
      <span><RotateCcw :size="15" />已提交返修要求</span>
      <p>{{ revision.message }}</p>
    </div>
  </section>
</template>

<style scoped>
.request-section {
  padding: 24px 0;
  margin-top: 24px;
  border-top: 1px solid var(--border);
}

header {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 16px;
}

header p {
  color: var(--primary);
  font-size: 9px;
  font-weight: 800;
}

header h3 {
  margin-top: 2px;
  font-size: 16px;
}

header > span {
  color: var(--ink-faint);
  font-size: 10px;
}

.media-strip,
.asset-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.media-strip { margin-top: 16px; }
.asset-row { margin-top: 8px; }

figure {
  overflow: hidden;
  margin: 0;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-soft);
}

.media-preview-button {
  position: relative;
  display: block;
  overflow: hidden;
  width: 100%;
  aspect-ratio: 1;
  background: transparent;
  cursor: zoom-in;
}

.media-preview-button img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform var(--motion-normal) var(--ease-out);
}

.media-preview-button:hover img { transform: scale(1.025); }

.preview-indicator {
  position: absolute;
  top: 7px;
  right: 7px;
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border: 1px solid rgb(255 255 255 / 30%);
  border-radius: 6px;
  background: rgb(17 24 25 / 72%);
  color: #fff;
  opacity: 0;
  transition: opacity var(--motion-fast);
}

.media-preview-button:hover .preview-indicator,
.media-preview-button:focus-visible .preview-indicator { opacity: 1; }

figcaption,
.asset-row figure > span {
  display: block;
  overflow: hidden;
  padding: 6px 8px;
  color: var(--ink-muted);
  font-size: 9px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.requirement-copy,
.supplemental,
.revision-note { margin-top: 18px; }

.requirement-copy > strong,
.supplemental > strong { font-size: 11px; }

.requirement-copy p,
.revision-note p {
  margin-top: 6px;
  color: var(--ink-muted);
  font-size: 12px;
  line-height: 1.7;
  white-space: pre-wrap;
}

.revision-note {
  padding: 14px;
  border-left: 3px solid var(--warning);
  background: #fffaf0;
}

.revision-note span {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--warning);
  font-size: 11px;
  font-weight: 700;
}

@media (max-width: 600px) {
  .media-strip,
  .asset-row { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .preview-indicator { opacity: 0.85; }
}

@media (prefers-reduced-motion: reduce) {
  .media-preview-button img,
  .preview-indicator { transition: none; }
}
</style>
