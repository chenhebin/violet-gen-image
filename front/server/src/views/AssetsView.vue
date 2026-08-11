<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  Archive,
  Eye,
  ImageOff,
  Images,
  RefreshCw,
  Search,
} from '@lucide/vue'
import AssetActionModal, {
  type AssetAction,
} from '@/components/assets/AssetActionModal.vue'
import AssetDetailDrawer from '@/components/assets/AssetDetailDrawer.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import EmptyState from '@/components/base/EmptyState.vue'
import FilterBar from '@/components/base/FilterBar.vue'
import FormField from '@/components/base/FormField.vue'
import PaginationBar from '@/components/base/PaginationBar.vue'
import StatusBadge from '@/components/base/StatusBadge.vue'
import { ASSET_KIND_LABELS } from '@/config'
import { useToast } from '@/composables/useToast'
import { useAssetStore } from '@/stores/assets'
import type {
  AssetKind,
  AssetQuery,
  ManagedAsset,
} from '@/types/domain'
import { formatDateTime, formatFileSize } from '@/utils/format'

const store = useAssetStore()
const toast = useToast()
const keyword = ref('')
const kind = ref<AssetKind | ''>('')
const retained = ref<'' | 'true' | 'false'>('')
const selectedId = ref('')
const action = ref<AssetAction | null>(null)
const signedUrl = ref('')
const signedUrlExpiresAt = ref('')

function currentQuery(page = 1): AssetQuery {
  return {
    page,
    pageSize: store.assets.pageSize,
    keyword: keyword.value.trim() || undefined,
    kind: kind.value || undefined,
    retained: retained.value === '' ? undefined : retained.value === 'true',
  }
}

async function load(page = 1): Promise<void> {
  try {
    await store.loadAssets(currentQuery(page))
  } catch (error) {
    toast.error({
      title: '图片资产加载失败',
      message: error instanceof Error ? error.message : '请稍后重试',
    })
  }
}

async function openAsset(asset: ManagedAsset): Promise<void> {
  selectedId.value = asset.id
  signedUrl.value = ''
  signedUrlExpiresAt.value = ''
  try {
    await store.loadAsset(asset.id)
  } catch (error) {
    selectedId.value = ''
    toast.error({
      title: '图片详情加载失败',
      message: error instanceof Error ? error.message : '请稍后重试',
    })
  }
}

async function requestSignedUrl(): Promise<void> {
  const asset = store.currentAsset
  if (!asset) return
  try {
    const result = await store.getSignedUrl(asset.id)
    signedUrl.value = result.url
    signedUrlExpiresAt.value = result.expiresAt
    toast.success('短期签名地址已生成')
  } catch (error) {
    toast.error({
      title: '签名地址生成失败',
      message: error instanceof Error ? error.message : '请稍后重试',
    })
  }
}

async function submitAction(reason: string): Promise<void> {
  const asset = store.currentAsset
  if (!asset || !action.value) return
  try {
    const result =
      action.value === 'cleanup'
        ? await store.cleanup(asset.id, reason)
        : await store.setRetained(asset.id, action.value === 'retain', reason)
    store.currentAsset = result
    toast.success(
      action.value === 'cleanup'
        ? '图片对象已清理'
        : action.value === 'retain'
          ? '图片已设为长期保留'
          : '已解除长期保留',
    )
    action.value = null
  } catch (error) {
    toast.error({
      title: '图片操作未完成',
      message: error instanceof Error ? error.message : '请稍后重试',
    })
  }
}

function closeDrawer(): void {
  if (store.isMutating) return
  selectedId.value = ''
  store.currentAsset = null
  signedUrl.value = ''
  signedUrlExpiresAt.value = ''
  action.value = null
}

onMounted(() => void load())
</script>

<template>
  <main class="page">
    <header class="page__header">
      <div>
        <p class="page__eyebrow">Visual asset registry</p>
        <h1 class="page__title">图片资产</h1>
        <p class="page__description">
          检索原图、参考图、AI 结果和人工成片，管理签名预览与 90 天留存策略。
        </p>
      </div>
      <div class="page__actions">
        <BaseButton variant="secondary" :loading="store.isLoading" @click="load(store.assets.page)">
          <template #icon><RefreshCw :size="16" /></template>
          刷新
        </BaseButton>
      </div>
    </header>

    <FilterBar>
      <FormField label="检索图片" for-id="asset-search">
        <div class="search-control">
          <Search :size="16" />
          <input
            id="asset-search"
            v-model="keyword"
            class="form-control"
            type="search"
            placeholder="文件名、用户或资源编号"
            @keyup.enter="load()"
          />
        </div>
      </FormField>
      <FormField label="图片类型" for-id="asset-kind">
        <select id="asset-kind" v-model="kind" class="form-control" @change="load()">
          <option value="">全部类型</option>
          <option v-for="(label, value) in ASSET_KIND_LABELS" :key="value" :value="value">
            {{ label }}
          </option>
        </select>
      </FormField>
      <FormField label="留存状态" for-id="asset-retained">
        <select id="asset-retained" v-model="retained" class="form-control" @change="load()">
          <option value="">全部状态</option>
          <option value="true">长期保留</option>
          <option value="false">按期清理</option>
        </select>
      </FormField>
      <template #actions>
        <BaseButton @click="load()">
          <template #icon><Search :size="16" /></template>
          查询
        </BaseButton>
      </template>
    </FilterBar>

    <section class="asset-section" aria-label="图片资产列表">
      <div class="asset-summary">
        <span><Images :size="15" />平台图片档案</span>
        <strong>{{ store.assets.total }} 个文件</strong>
      </div>

      <div v-if="store.isLoading" class="asset-grid">
        <div v-for="index in 8" :key="index" class="asset-skeleton"></div>
      </div>
      <div v-else-if="store.assets.items.length" class="asset-grid">
        <article
          v-for="asset in store.assets.items"
          :key="asset.id"
          class="asset-card"
          tabindex="0"
          @click="openAsset(asset)"
          @keyup.enter="openAsset(asset)"
        >
          <div class="asset-card__preview">
            <img v-if="asset.previewUrl && !asset.deletedAt" :src="asset.previewUrl" :alt="asset.name" />
            <div v-else class="asset-card__missing">
              <ImageOff :size="24" />
              <span>对象已清理</span>
            </div>
            <div class="asset-card__badges">
              <StatusBadge :tone="asset.deletedAt ? 'danger' : 'info'" :dot="false">
                {{ ASSET_KIND_LABELS[asset.kind] }}
              </StatusBadge>
              <StatusBadge v-if="asset.retained" tone="success" :dot="false">
                <Archive :size="12" />
                长期保留
              </StatusBadge>
            </div>
            <button aria-label="查看图片详情" @click.stop="openAsset(asset)">
              <Eye :size="17" />
            </button>
          </div>
          <div class="asset-card__body">
            <strong>{{ asset.name }}</strong>
            <span>{{ asset.ownerEmail }}</span>
            <div>
              <small>{{ asset.width }} × {{ asset.height }}</small>
              <small>{{ formatFileSize(asset.size) }}</small>
            </div>
            <time :datetime="asset.createdAt">{{ formatDateTime(asset.createdAt) }}</time>
          </div>
        </article>
      </div>
      <EmptyState
        v-else
        title="没有匹配的图片资产"
        description="调整图片类型、留存状态或关键词后再试。"
      />

      <PaginationBar
        :page="store.assets.page"
        :page-size="store.assets.pageSize"
        :total="store.assets.total"
        :has-more="store.assets.hasMore"
        :loading="store.isLoading"
        @change="load"
      />
    </section>
  </main>

  <AssetDetailDrawer
    :open="Boolean(selectedId)"
    :asset="store.currentAsset"
    :loading="store.isLoading"
    :signed-url="signedUrl"
    :signed-url-expires-at="signedUrlExpiresAt"
    @close="closeDrawer"
    @request-signed-url="requestSignedUrl"
    @action="action = $event"
  />
  <AssetActionModal
    :action="action"
    :asset="store.currentAsset"
    :loading="store.isMutating"
    @close="action = null"
    @submit="submitAction"
  />
</template>

<style scoped>
.search-control {
  position: relative;
  width: min(350px, 100%);
}

.search-control svg {
  position: absolute;
  z-index: 1;
  top: 14px;
  left: 12px;
  color: var(--ink-faint);
}

.search-control input {
  padding-left: 37px;
}

select.form-control {
  min-width: 170px;
}

.asset-section {
  margin-top: 18px;
}

.asset-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 2px 10px;
  color: var(--ink-muted);
  font-size: 12px;
}

.asset-summary span {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

.asset-summary strong {
  color: var(--ink);
  font-family: var(--font-mono);
}

.asset-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(225px, 1fr));
  gap: 14px;
}

.asset-card {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
  cursor: pointer;
  transition:
    transform var(--motion-fast),
    box-shadow var(--motion-fast);
}

.asset-card:hover {
  box-shadow: 0 8px 24px rgb(22 30 28 / 9%);
  transform: translateY(-2px);
}

.asset-card__preview {
  position: relative;
  overflow: hidden;
  aspect-ratio: 4 / 3;
  background: var(--surface-soft);
}

.asset-card__preview > img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.asset-card__missing {
  display: grid;
  height: 100%;
  place-items: center;
  align-content: center;
  gap: 7px;
  color: var(--ink-muted);
  font-size: 11px;
}

.asset-card__badges {
  position: absolute;
  top: 9px;
  left: 9px;
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.asset-card__preview > button {
  position: absolute;
  right: 9px;
  bottom: 9px;
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border-radius: var(--radius-sm);
  background: rgb(255 255 255 / 90%);
  color: var(--ink);
  box-shadow: var(--shadow-sm);
}

.asset-card__body {
  display: grid;
  gap: 3px;
  padding: 12px;
}

.asset-card__body strong,
.asset-card__body > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-card__body strong {
  font-size: 12px;
}

.asset-card__body > span,
.asset-card__body small,
.asset-card__body time {
  color: var(--ink-muted);
  font-size: 10px;
}

.asset-card__body > div {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding-top: 5px;
}

.asset-skeleton {
  min-height: 270px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: linear-gradient(
    90deg,
    var(--surface-soft),
    var(--surface),
    var(--surface-soft)
  );
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

@media (max-width: 560px) {
  .asset-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  .asset-card {
    animation: asset-in var(--motion-normal) var(--ease-out) both;
  }

  .asset-card:nth-child(2) {
    animation-delay: 45ms;
  }

  .asset-card:nth-child(3) {
    animation-delay: 90ms;
  }

  .asset-card__preview > button {
    right: 6px;
    bottom: 6px;
    width: 44px;
    height: 44px;
  }

  .asset-card__badges {
    top: 6px;
    left: 6px;
  }

  .asset-card__body {
    padding: 10px;
  }

  .asset-card__body > div {
    align-items: flex-start;
    flex-direction: column;
    gap: 1px;
  }

  .asset-skeleton {
    min-height: 220px;
  }
}

@keyframes asset-in {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
}
</style>
