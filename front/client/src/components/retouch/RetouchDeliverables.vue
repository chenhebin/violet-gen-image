<script setup lang="ts">
import { Download, Image as ImageIcon, Maximize2 } from '@lucide/vue'
import type { GenerationResult } from '@/types/domain'
import { downloadAsset } from '@/utils/download'

defineProps<{
  deliverables: GenerationResult[]
}>()

const emit = defineEmits<{
  preview: [id: string]
  imageError: [id: string]
}>()

async function downloadResult(item: GenerationResult, index: number): Promise<void> {
  await downloadAsset({
    assetId: item.id,
    currentUrl: item.downloadUrl || item.url,
    filename: `精修成片-${index + 1}.jpg`,
  })
}
</script>

<template>
  <section
    v-if="deliverables.length"
    class="deliverables"
    aria-labelledby="retouch-deliverables-heading"
  >
    <header>
      <div>
        <p>成片交付</p>
        <h3 id="retouch-deliverables-heading">可下载文件</h3>
      </div>
      <span>{{ deliverables.length }} 张</span>
    </header>

    <div class="delivery-grid">
      <figure v-for="(item, index) in deliverables" :key="item.id">
        <button
          type="button"
          class="image-frame"
          :aria-label="`放大查看精修交付成片 ${index + 1}`"
          @click="emit('preview', `deliverable:${item.id}`)"
        >
          <img
            :src="item.url"
            :alt="`精修交付成片 ${index + 1}`"
            @error="emit('imageError', `deliverable:${item.id}`)"
          />
          <span class="preview-indicator" aria-hidden="true">
            <Maximize2 :size="15" />
          </span>
        </button>
        <figcaption>
          <span>
            <ImageIcon :size="14" />
            成片 {{ index + 1 }}
          </span>
          <a
            :href="item.downloadUrl || item.url"
            :download="item.downloadUrl ? undefined : `精修成片-${index + 1}`"
            @click.prevent="downloadResult(item, index)"
          >
            <Download :size="15" />
            下载
          </a>
        </figcaption>
      </figure>
    </div>
  </section>
</template>

<style scoped>
.deliverables {
  padding: 24px 0;
  border-top: 1px solid var(--border);
}

header {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
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

.delivery-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

figure {
  overflow: hidden;
  margin: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
}

.image-frame {
  position: relative;
  display: block;
  overflow: hidden;
  width: 100%;
  aspect-ratio: 4 / 3;
  background: var(--surface-soft);
  cursor: zoom-in;
}

.image-frame img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform var(--motion-normal) var(--ease-out);
}

.image-frame:hover img { transform: scale(1.025); }

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

.image-frame:hover .preview-indicator,
.image-frame:focus-visible .preview-indicator { opacity: 1; }

figcaption {
  display: flex;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 10px;
}

figcaption span,
figcaption a {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 650;
}

figcaption span {
  min-width: 0;
  color: var(--ink-muted);
}

figcaption a {
  min-height: 30px;
  padding: 0 8px;
  border-radius: 5px;
  color: var(--primary);
  white-space: nowrap;
}

figcaption a:hover {
  background: var(--primary-soft);
}

@media (max-width: 520px) {
  .delivery-grid {
    grid-template-columns: 1fr;
  }

  .preview-indicator { opacity: 0.85; }
}

@media (prefers-reduced-motion: reduce) {
  .image-frame img,
  .preview-indicator { transition: none; }
}
</style>
