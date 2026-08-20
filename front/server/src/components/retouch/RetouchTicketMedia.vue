<script setup lang="ts">
import { Maximize2 } from '@lucide/vue'
import type {
  GenerationResult,
  ManagedAsset,
} from '@/types/domain'

defineProps<{
  selectedResults: GenerationResult[]
  supplementalAssets: ManagedAsset[]
  deliverables: GenerationResult[]
}>()

const emit = defineEmits<{
  preview: [id: string]
  imageError: [id: string]
}>()
</script>

<template>
  <section class="detail-section">
    <header>
      <div>
        <span>待处理素材</span>
        <h3>用户选中的 AI 成片</h3>
      </div>
      <b>{{ selectedResults.length }} 张</b>
    </header>
    <div class="media-grid">
      <figure v-for="(result, index) in selectedResults" :key="result.id">
        <button
          type="button"
          class="media-preview-button"
          :aria-label="`放大查看用户选中的 AI 成片 ${index + 1}`"
          @click="emit('preview', `selected:${result.id}`)"
        >
          <img
            :src="result.url"
            :alt="`用户选中的 AI 成片 ${index + 1}`"
            @error="emit('imageError', `selected:${result.id}`)"
          />
          <span class="preview-indicator" aria-hidden="true">
            <Maximize2 :size="16" />
          </span>
        </button>
        <figcaption>{{ result.width }} × {{ result.height }}</figcaption>
      </figure>
    </div>

    <div v-if="supplementalAssets.length" class="supplemental">
      <strong>补充参考图</strong>
      <div class="media-grid media-grid--small">
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
              <Maximize2 :size="16" />
            </span>
          </button>
          <figcaption>{{ asset.name }}</figcaption>
        </figure>
      </div>
    </div>
  </section>

  <section v-if="deliverables.length" class="detail-section">
    <header>
      <div>
        <span>人工交付</span>
        <h3>已上传成片</h3>
      </div>
      <b>{{ deliverables.length }} 张</b>
    </header>
    <div class="media-grid">
      <figure v-for="(result, index) in deliverables" :key="result.id">
        <button
          type="button"
          class="media-preview-button"
          :aria-label="`放大查看人工修图交付成片 ${index + 1}`"
          @click="emit('preview', `deliverable:${result.id}`)"
        >
          <img
            :src="result.url"
            :alt="`人工修图交付成片 ${index + 1}`"
            @error="emit('imageError', `deliverable:${result.id}`)"
          />
          <span class="preview-indicator" aria-hidden="true">
            <Maximize2 :size="16" />
          </span>
        </button>
        <figcaption>{{ result.width }} × {{ result.height }}</figcaption>
      </figure>
    </div>
  </section>
</template>

<style scoped>
.detail-section {
  padding: 18px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
}

.detail-section > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

header span {
  color: var(--ink-muted);
  font-size: 10px;
  font-weight: 750;
}

header h3 {
  margin-top: 2px;
  font-size: 15px;
}

header b {
  color: var(--ink-muted);
  font-size: 11px;
  font-weight: 600;
}

.media-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 10px;
}

.media-grid figure {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface-soft);
}

.media-preview-button {
  position: relative;
  display: block;
  overflow: hidden;
  width: 100%;
  aspect-ratio: 4 / 3;
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
  top: 8px;
  right: 8px;
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border: 1px solid rgb(255 255 255 / 30%);
  border-radius: var(--radius-sm);
  background: rgb(17 24 23 / 72%);
  color: #fff;
  opacity: 0;
  transition: opacity var(--motion-fast);
}

.media-preview-button:hover .preview-indicator,
.media-preview-button:focus-visible .preview-indicator { opacity: 1; }

.media-grid figcaption {
  overflow: hidden;
  padding: 8px 10px;
  color: var(--ink-muted);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.media-grid--small {
  grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
  margin-top: 9px;
}

.supplemental {
  margin-top: 16px;
  font-size: 12px;
}

@media (max-width: 640px) {
  .media-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .media-grid--small { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .preview-indicator { opacity: 0.85; }
}

@media (max-width: 420px) {
  .media-grid,
  .media-grid--small { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (prefers-reduced-motion: reduce) {
  .media-preview-button img,
  .preview-indicator { transition: none; }
}
</style>
