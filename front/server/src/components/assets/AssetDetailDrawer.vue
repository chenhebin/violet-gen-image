<script setup lang="ts">
import {
  Archive,
  ArchiveRestore,
  ExternalLink,
  Link2,
  Trash2,
} from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDrawer from '@/components/base/BaseDrawer.vue'
import StatusBadge from '@/components/base/StatusBadge.vue'
import { ASSET_KIND_LABELS } from '@/config'
import type { ManagedAsset } from '@/types/domain'
import { formatDateTime, formatFileSize } from '@/utils/format'
import type { AssetAction } from './AssetActionModal.vue'

defineProps<{
  open: boolean
  asset: ManagedAsset | null
  loading: boolean
  signedUrl?: string
  signedUrlExpiresAt?: string
}>()

defineEmits<{
  close: []
  requestSignedUrl: []
  action: [action: AssetAction]
}>()
</script>

<template>
  <BaseDrawer
    :open="open"
    :title="asset?.name ?? '图片资产详情'"
    :description="asset ? `${asset.ownerEmail} · ${asset.id}` : '正在读取图片元数据'"
    size="large"
    @close="$emit('close')"
  >
    <div v-if="loading && !asset" class="drawer-loading">正在读取图片详情…</div>
    <div v-else-if="asset" class="asset-detail">
      <section class="preview-stage" :class="{ 'preview-stage--deleted': asset.deletedAt }">
        <img v-if="asset.previewUrl && !asset.deletedAt" :src="asset.previewUrl" :alt="asset.name" />
        <div v-else class="deleted-placeholder">
          <Trash2 :size="28" />
          <strong>对象文件已清理</strong>
          <span>元数据和审计记录仍然保留</span>
        </div>
        <div class="preview-caption">
          <div>
            <StatusBadge :tone="asset.deletedAt ? 'danger' : 'info'">
              {{ ASSET_KIND_LABELS[asset.kind] }}
            </StatusBadge>
            <StatusBadge v-if="asset.retained" tone="success">长期保留</StatusBadge>
          </div>
          <span>{{ asset.width }} × {{ asset.height }}</span>
        </div>
      </section>

      <section v-if="signedUrl" class="signed-link">
        <Link2 :size="18" />
        <div>
          <strong>短期签名地址已生成</strong>
          <span>有效期至 {{ formatDateTime(signedUrlExpiresAt) }}</span>
        </div>
        <a :href="signedUrl" target="_blank" rel="noopener noreferrer">
          打开预览
          <ExternalLink :size="14" />
        </a>
      </section>

      <section class="detail-section metadata">
        <header>
          <span>Asset metadata</span>
          <h3>图片元数据</h3>
        </header>
        <dl>
          <div>
            <dt>文件类型</dt>
            <dd>{{ asset.mimeType }}</dd>
          </div>
          <div>
            <dt>文件大小</dt>
            <dd>{{ formatFileSize(asset.size) }}</dd>
          </div>
          <div>
            <dt>资源角色</dt>
            <dd>{{ asset.role ?? '-' }}</dd>
          </div>
          <div>
            <dt>上传时间</dt>
            <dd>{{ formatDateTime(asset.createdAt) }}</dd>
          </div>
          <div>
            <dt>留存到期</dt>
            <dd>{{ asset.retained ? '长期保留' : formatDateTime(asset.retentionExpiresAt) }}</dd>
          </div>
          <div>
            <dt>清理时间</dt>
            <dd>{{ formatDateTime(asset.deletedAt) }}</dd>
          </div>
        </dl>
      </section>

      <section class="detail-section relations">
        <header>
          <span>Business relations</span>
          <h3>业务关联</h3>
        </header>
        <dl>
          <div>
            <dt>所属用户</dt>
            <dd>{{ asset.ownerEmail }}</dd>
          </div>
          <div>
            <dt>用户编号</dt>
            <dd class="mono">{{ asset.ownerId }}</dd>
          </div>
          <div>
            <dt>任务编号</dt>
            <dd class="mono">{{ asset.taskId ?? '-' }}</dd>
          </div>
          <div>
            <dt>工单编号</dt>
            <dd class="mono">{{ asset.ticketId ?? '-' }}</dd>
          </div>
        </dl>
      </section>
    </div>

    <template v-if="asset && !asset.deletedAt" #footer>
      <BaseButton variant="secondary" @click="$emit('requestSignedUrl')">
        <template #icon><Link2 :size="16" /></template>
        生成签名预览
      </BaseButton>
      <BaseButton
        variant="secondary"
        @click="$emit('action', asset.retained ? 'release' : 'retain')"
      >
        <template #icon>
          <ArchiveRestore v-if="asset.retained" :size="16" />
          <Archive v-else :size="16" />
        </template>
        {{ asset.retained ? '解除长期保留' : '设为长期保留' }}
      </BaseButton>
      <BaseButton variant="danger" @click="$emit('action', 'cleanup')">
        <template #icon><Trash2 :size="16" /></template>
        提前清理
      </BaseButton>
    </template>
  </BaseDrawer>
</template>

<style scoped>
.drawer-loading {
  display: grid;
  min-height: 320px;
  place-items: center;
  color: var(--ink-muted);
}

.asset-detail {
  display: grid;
  gap: 18px;
}

.preview-stage,
.detail-section {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
}

.preview-stage {
  position: relative;
  min-height: 360px;
  background:
    linear-gradient(45deg, #e8ecea 25%, transparent 25%),
    linear-gradient(-45deg, #e8ecea 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, #e8ecea 75%),
    linear-gradient(-45deg, transparent 75%, #e8ecea 75%);
  background-color: #f7f9f8;
  background-position: 0 0, 0 8px, 8px -8px, -8px 0;
  background-size: 16px 16px;
}

.preview-stage > img {
  width: 100%;
  height: min(58vh, 620px);
  object-fit: contain;
}

.preview-stage--deleted {
  display: grid;
  place-items: center;
}

.deleted-placeholder {
  display: grid;
  justify-items: center;
  gap: 6px;
  color: var(--ink-muted);
}

.deleted-placeholder strong {
  color: var(--ink);
  font-size: 14px;
}

.deleted-placeholder span {
  font-size: 11px;
}

.preview-caption {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  background: rgb(255 255 255 / 92%);
  backdrop-filter: blur(8px);
}

.preview-caption > div {
  display: flex;
  gap: 7px;
}

.preview-caption > span {
  color: var(--ink-muted);
  font-family: var(--font-mono);
  font-size: 11px;
}

.signed-link {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border: 1px solid var(--primary);
  border-radius: var(--radius-md);
  background: var(--primary-soft);
}

.signed-link > svg {
  color: var(--primary);
}

.signed-link div {
  display: grid;
  gap: 2px;
}

.signed-link strong {
  font-size: 12px;
}

.signed-link span {
  color: var(--ink-muted);
  font-size: 10px;
}

.signed-link a {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--primary);
  font-size: 12px;
  font-weight: 700;
}

.detail-section {
  padding: 18px;
}

.detail-section header {
  margin-bottom: 14px;
}

.detail-section header span {
  color: var(--ink-muted);
  font-size: 10px;
  font-weight: 750;
}

.detail-section h3 {
  margin-top: 2px;
  font-size: 15px;
}

.detail-section dl {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.detail-section dl > div {
  display: grid;
  grid-template-columns: 110px minmax(0, 1fr);
  gap: 10px;
  padding: 12px;
  border-bottom: 1px solid var(--border);
}

.detail-section dl > div:nth-child(odd) {
  border-right: 1px solid var(--border);
}

.detail-section dl > div:nth-last-child(-n + 2) {
  border-bottom: 0;
}

dt {
  color: var(--ink-muted);
  font-size: 11px;
}

dd {
  overflow: hidden;
  margin: 0;
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
}

@media (max-width: 700px) {
  .preview-stage {
    min-height: 260px;
  }

  .detail-section dl {
    grid-template-columns: 1fr;
  }

  .detail-section dl > div,
  .detail-section dl > div:nth-child(odd) {
    border-right: 0;
    border-bottom: 1px solid var(--border);
  }

  .detail-section dl > div:last-child {
    border-bottom: 0;
  }

  .signed-link {
    grid-template-columns: auto 1fr;
  }

  .signed-link a {
    grid-column: 2;
  }
}
</style>
