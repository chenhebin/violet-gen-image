<script setup lang="ts">
import { ref } from 'vue'
import {
  FileImage,
  ImagePlus,
  LoaderCircle,
  Plus,
  Trash2,
} from '@lucide/vue'
import { ASSET_CONFIG, REFERENCE_ROLE_OPTIONS } from '@/config'
import { useWorkspaceStore } from '@/stores/workspace'
import { assetApi } from '@/services/api'
import type { Asset, AssetKind, ReferenceRole } from '@/types/domain'

const workspace = useWorkspaceStore()
const sourceInput = ref<HTMLInputElement | null>(null)
const referenceInput = ref<HTMLInputElement | null>(null)
const dragging = ref<AssetKind | null>(null)

function filesFrom(event: Event): File[] {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  return files
}

async function selectFiles(event: Event, kind: AssetKind): Promise<void> {
  await workspace.uploadFiles(filesFrom(event), kind, kind === 'reference' ? 'style' : undefined)
}

async function dropFiles(event: DragEvent, kind: AssetKind): Promise<void> {
  dragging.value = null
  const files = Array.from(event.dataTransfer?.files ?? [])
  await workspace.uploadFiles(files, kind, kind === 'reference' ? 'style' : undefined)
}

function previewLabel(asset: Asset): string {
  if (asset.kind === 'source') return '待修改原图'
  return (
    REFERENCE_ROLE_OPTIONS.find((item) => item.value === asset.role)?.label ??
    '风格'
  )
}

async function refreshPreview(asset: Asset): Promise<void> {
  try {
    const signed = await assetApi.getUrl(asset.id)
    asset.previewUrl = signed.url
    asset.previewUrlExpiresAt = signed.expiresAt
  } catch {
    // Keep the failed URL visible; the next hydration can retry.
  }
}
</script>

<template>
  <aside class="asset-rail" aria-label="影像素材">
    <header class="panel-heading">
      <div>
        <span>素材</span>
        <h2>影像校样条</h2>
      </div>
      <b>{{ workspace.draft.assets.length }}/{{ ASSET_CONFIG.maxCount }}</b>
    </header>

    <div class="proof-strip">
      <section class="asset-section">
        <div class="section-heading">
          <div>
            <h3>待修改原图</h3>
            <p>可选，人物、商品或场景原片</p>
          </div>
          <button
            class="add-button"
            aria-label="添加待修改原图"
            title="添加待修改原图"
            @click="sourceInput?.click()"
          >
            <Plus :size="18" />
          </button>
        </div>
        <input
          ref="sourceInput"
          class="sr-only"
          type="file"
          :accept="ASSET_CONFIG.acceptAttribute"
          multiple
          @change="selectFiles($event, 'source')"
        />

        <div
          v-if="!workspace.sourceAssets.length"
          class="drop-zone"
          :class="{ dragging: dragging === 'source' }"
          @click="sourceInput?.click()"
          @dragenter.prevent="dragging = 'source'"
          @dragover.prevent
          @dragleave.prevent="dragging = null"
          @drop.prevent="dropFiles($event, 'source')"
        >
          <FileImage :size="24" />
          <strong>上传原图</strong>
          <span>可多选，单张不超过 {{ ASSET_CONFIG.maxFileSizeLabel }}</span>
        </div>

        <article
          v-for="(asset, index) in workspace.sourceAssets"
          :key="asset.id"
          class="proof"
        >
          <div class="frame-index">{{ String(index + 1).padStart(2, '0') }}</div>
          <div class="proof-image">
            <img v-if="asset.previewUrl" :src="asset.previewUrl" :alt="asset.name" @error="refreshPreview(asset)" />
            <FileImage v-else :size="22" />
            <span>{{ previewLabel(asset) }}</span>
          </div>
          <div class="proof-meta">
            <p :title="asset.name">{{ asset.name }}</p>
            <button
              :aria-label="`删除 ${asset.name}`"
              title="删除图片"
              @click="workspace.removeAsset(asset)"
            >
              <Trash2 :size="16" />
            </button>
          </div>
        </article>
      </section>

      <section class="asset-section">
        <div class="section-heading">
          <div>
            <h3>参考图片</h3>
            <p>用于分析风格、构图和光影，不会作为生图原图上传</p>
          </div>
          <button
            class="add-button"
            aria-label="添加参考图片"
            title="添加参考图片"
            @click="referenceInput?.click()"
          >
            <Plus :size="18" />
          </button>
        </div>
        <input
          ref="referenceInput"
          class="sr-only"
          type="file"
          :accept="ASSET_CONFIG.acceptAttribute"
          multiple
          @change="selectFiles($event, 'reference')"
        />

        <div
          v-if="!workspace.referenceAssets.length"
          class="drop-zone"
          :class="{ dragging: dragging === 'reference' }"
          @click="referenceInput?.click()"
          @dragenter.prevent="dragging = 'reference'"
          @dragover.prevent
          @dragleave.prevent="dragging = null"
          @drop.prevent="dropFiles($event, 'reference')"
        >
          <ImagePlus :size="24" />
          <strong>添加参考图</strong>
          <span>上传后标记它的用途</span>
        </div>

        <article
          v-for="(asset, index) in workspace.referenceAssets"
          :key="asset.id"
          class="proof"
        >
          <div class="frame-index">R{{ index + 1 }}</div>
          <div class="proof-image">
            <img v-if="asset.previewUrl" :src="asset.previewUrl" :alt="asset.name" @error="refreshPreview(asset)" />
            <FileImage v-else :size="22" />
            <span>{{ previewLabel(asset) }}</span>
          </div>
          <div class="proof-meta">
            <select
              :value="asset.role ?? 'style'"
              :aria-label="`${asset.name} 的参考用途`"
              @change="
                workspace.setReferenceRole(
                  asset.id,
                  ($event.target as HTMLSelectElement).value as ReferenceRole,
                )
              "
            >
              <option
                v-for="option in REFERENCE_ROLE_OPTIONS"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </option>
            </select>
            <button
              :aria-label="`删除 ${asset.name}`"
              title="删除图片"
              @click="workspace.removeAsset(asset)"
            >
              <Trash2 :size="16" />
            </button>
          </div>
        </article>
      </section>
    </div>

    <div v-if="workspace.isUploading" class="upload-state">
      <LoaderCircle :size="17" />
      <span>正在整理素材…</span>
    </div>
    <p v-if="workspace.error" class="asset-error">{{ workspace.error }}</p>
  </aside>
</template>

<style scoped>
.asset-rail {
  display: flex;
  min-width: 0;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.panel-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px;
  border-bottom: 1px solid var(--border);
}

.panel-heading span {
  color: var(--primary);
  font-size: 10px;
  font-weight: 800;
}

.panel-heading h2 {
  margin-top: 2px;
  font-size: 16px;
}

.panel-heading b {
  color: var(--ink-faint);
  font-size: 11px;
}

.proof-strip {
  flex: 1;
  padding: 8px 14px 18px;
  background:
    linear-gradient(90deg, transparent 20px, rgb(23 25 29 / 5%) 21px, transparent 22px)
    var(--surface);
}

.asset-section {
  padding-top: 16px;
}

.asset-section + .asset-section {
  padding-top: 20px;
  margin-top: 18px;
  border-top: 1px dashed var(--border-strong);
}

.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.section-heading h3 {
  font-size: 13px;
}

.section-heading p {
  color: var(--ink-faint);
  font-size: 10px;
}

.add-button,
.proof-meta button {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border-radius: 6px;
  background: var(--surface-soft);
  color: var(--ink-muted);
}

.add-button:hover,
.proof-meta button:hover {
  color: var(--primary);
}

.drop-zone {
  display: grid;
  min-height: 132px;
  place-items: center;
  align-content: center;
  gap: 5px;
  padding: 14px;
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius-md);
  background: rgb(246 247 249 / 72%);
  color: var(--ink-muted);
  text-align: center;
  transition:
    border-color var(--motion-fast),
    background var(--motion-fast);
  cursor: pointer;
}

.drop-zone.dragging,
.drop-zone:hover {
  border-color: var(--primary);
  background: var(--primary-soft);
}

.drop-zone strong {
  margin-top: 4px;
  color: var(--ink);
  font-size: 12px;
}

.drop-zone span {
  font-size: 9px;
}

.proof {
  position: relative;
  margin-top: 10px;
}

.frame-index {
  position: absolute;
  z-index: 1;
  top: 7px;
  left: 7px;
  display: grid;
  min-width: 27px;
  height: 22px;
  place-items: center;
  border-radius: 4px;
  background: rgb(23 25 29 / 78%);
  color: white;
  font-size: 9px;
  font-weight: 750;
}

.proof-image {
  position: relative;
  display: grid;
  overflow: hidden;
  aspect-ratio: 4 / 3;
  place-items: center;
  border-radius: 6px 6px 0 0;
  background: var(--surface-soft);
  color: var(--ink-faint);
}

.proof-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.proof-image span {
  position: absolute;
  right: 6px;
  bottom: 6px;
  padding: 3px 6px;
  border-radius: 4px;
  background: rgb(255 255 255 / 90%);
  color: var(--ink);
  font-size: 9px;
  font-weight: 700;
}

.proof-meta {
  display: grid;
  min-height: 42px;
  grid-template-columns: 1fr 36px;
  align-items: center;
  gap: 6px;
  padding: 3px 3px 3px 9px;
  border: 1px solid var(--border);
  border-top: 0;
  border-radius: 0 0 6px 6px;
}

.proof-meta p {
  overflow: hidden;
  color: var(--ink-muted);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.proof-meta select {
  min-width: 0;
  height: 34px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--ink-muted);
  font-size: 10px;
}

.proof-meta button {
  width: 36px;
  height: 36px;
  background: transparent;
}

.upload-state,
.asset-error {
  margin: 0 14px 14px;
  font-size: 11px;
}

.upload-state {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--primary);
}

.upload-state svg {
  animation: spin 700ms linear infinite;
}

.asset-error {
  color: var(--danger);
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 900px) {
  .proof-strip {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }

  .asset-section + .asset-section {
    padding-top: 16px;
    margin-top: 0;
    border-top: 0;
  }
}

</style>
<style scoped src="./AssetRail.mobile.css"></style>
