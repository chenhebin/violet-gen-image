<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Plus, Search, TicketCheck, X } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import ConfirmDialog from '@/components/base/ConfirmDialog.vue'
import ExtendRedemptionModal from '@/components/redemptions/ExtendRedemptionModal.vue'
import GenerateCodesModal from '@/components/redemptions/GenerateCodesModal.vue'
import RedemptionBatchDrawer from '@/components/redemptions/RedemptionBatchDrawer.vue'
import RedemptionBatchesTable from '@/components/redemptions/RedemptionBatchesTable.vue'
import RenameRedemptionBatchModal from '@/components/redemptions/RenameRedemptionBatchModal.vue'
import { APP_CONFIG } from '@/config'
import { useToast } from '@/composables/useToast'
import { useRedemptionStore } from '@/stores/redemption'
import type {
  CreateRedemptionBatchPayload,
  CreateRedemptionBatchResult,
  RedemptionBatch,
  RedemptionCode,
  UpdateRedemptionBatchPayload,
} from '@/types'

const route = useRoute()
const router = useRouter()
const store = useRedemptionStore()
const toast = useToast()
const pageSize = 20

const page = ref(Number(route.query.page) || 1)
const filters = reactive({
  keyword: String(route.query.keyword || ''),
  productCode: String(route.query.productCode || ''),
})
const createOpen = ref(false)
const createResult = ref<CreateRedemptionBatchResult | null>(null)
const drawerOpen = ref(false)
const selectedBatch = ref<RedemptionBatch | null>(null)
const disableBatch = ref<RedemptionBatch | null>(null)
const extendBatch = ref<RedemptionBatch | null>(null)
const renameBatch = ref<RedemptionBatch | null>(null)
const exportingId = ref<string | null>(null)
const revealingBatch = ref(false)

const batches = computed(() => store.batches.items)
const drawerCodes = computed<RedemptionCode[]>(() =>
  store.codes.items.filter(
    (code) => code.batchId === selectedBatch.value?.id,
  ),
)

function errorText(error: unknown) {
  return error instanceof Error ? error.message : '操作未完成，请稍后重试'
}

async function loadBatches() {
  await store.loadBatches({
    page: page.value,
    pageSize,
    keyword: filters.keyword.trim() || undefined,
    productCode: filters.productCode || undefined,
  })
}

async function search(resetPage = true) {
  if (resetPage) page.value = 1
  await router.replace({
    query: {
      ...(filters.keyword && { keyword: filters.keyword }),
      ...(filters.productCode && { productCode: filters.productCode }),
      ...(page.value > 1 && { page: String(page.value) }),
    },
  })
  try {
    await loadBatches()
  } catch (error) {
    toast.error({ title: '批次查询失败', message: errorText(error) })
  }
}

function resetFilters() {
  filters.keyword = ''
  filters.productCode = ''
  void search()
}

async function changePage(nextPage: number) {
  page.value = nextPage
  await search(false)
}

async function createBatch(payload: CreateRedemptionBatchPayload) {
  try {
    createResult.value = await store.createBatch(payload)
    toast.success(`已生成 ${createResult.value.batch.quantity} 个兑换码`)
  } catch (error) {
    toast.error({ title: '兑换码生成失败', message: errorText(error) })
  }
}

function openCreate() {
  createResult.value = null
  createOpen.value = true
}

function closeCreate() {
  createOpen.value = false
  createResult.value = null
  store.clearSensitiveValues()
}

async function openDetail(batch: RedemptionBatch) {
  selectedBatch.value = batch
  drawerOpen.value = true
  try {
    const [detail] = await Promise.all([
      store.loadBatch(batch.id),
      store.loadCodes({ page: 1, pageSize: 100, batchId: batch.id }),
    ])
    selectedBatch.value = detail
  } catch (error) {
    toast.error({ title: '批次详情加载失败', message: errorText(error) })
  }
}

async function revealCode(code: RedemptionCode) {
  if (store.revealedCodes[code.id]) {
    delete store.revealedCodes[code.id]
    return
  }
  try {
    await store.revealCode(code.id)
    toast.info('完整兑换码已显示，本次查看已记录审计')
  } catch (error) {
    toast.error({ title: '无法查看完整码', message: errorText(error) })
  }
}

async function revealAll(batch: RedemptionBatch) {
  revealingBatch.value = true
  try {
    const result = await store.revealBatch(batch.id)
    toast.info(`已显示 ${result.length} 个未使用兑换码`)
  } catch (error) {
    toast.error({ title: '批量查看失败', message: errorText(error) })
  } finally {
    revealingBatch.value = false
  }
}

async function copy(value: string, label = '完整兑换码') {
  try {
    await navigator.clipboard.writeText(value)
    toast.success(`${label}已复制`)
  } catch {
    toast.error('浏览器未授权剪贴板，请手动选择内容')
  }
}

async function downloadBatch(
  batch: RedemptionBatch | string,
  format: 'csv' | 'xianyu',
) {
  const batchId = typeof batch === 'string' ? batch : batch.id
  exportingId.value = batchId
  try {
    const file = await store.exportBatch(batchId, format)
    const url = URL.createObjectURL(
      new Blob([file.content || file.csv || ''], { type: file.mediaType }),
    )
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = file.filename
    anchor.click()
    URL.revokeObjectURL(url)
    toast.success(format === 'xianyu' ? '闲鱼库存已生成并开始下载' : 'CSV 已生成并开始下载')
  } catch (error) {
    toast.error({ title: '导出失败', message: errorText(error) })
  } finally {
    exportingId.value = null
  }
}

function exportBatch(batch: RedemptionBatch | string) {
  return downloadBatch(batch, 'csv')
}

function exportXianyu(batch: RedemptionBatch | string) {
  return downloadBatch(batch, 'xianyu')
}

async function confirmDisable(reason: string) {
  if (!disableBatch.value) return
  try {
    const result = await store.disableCodes({
      batchId: disableBatch.value.id,
      reason,
    })
    toast.success({
      title: `已失效 ${result.affected} 个兑换码`,
      message: result.skipped ? `另有 ${result.skipped} 个被跳过` : undefined,
    })
    disableBatch.value = null
    drawerOpen.value = false
  } catch (error) {
    toast.error({ title: '批量失效失败', message: errorText(error) })
  }
}

async function confirmExtend(payload: {
  expiresAt: string | null
  reason: string
}) {
  if (!extendBatch.value) return
  try {
    const result = await store.extendCodes({
      batchId: extendBatch.value.id,
      ...payload,
    })
    toast.success({
      title: `已更新 ${result.affected} 个兑换码`,
      message: result.skipped ? `跳过 ${result.skipped} 个不可延期记录` : undefined,
    })
    extendBatch.value = null
    drawerOpen.value = false
  } catch (error) {
    toast.error({ title: '批量延期失败', message: errorText(error) })
  }
}

async function confirmRename(payload: UpdateRedemptionBatchPayload) {
  if (!renameBatch.value) return
  try {
    const updated = await store.updateBatch(renameBatch.value.id, payload)
    if (selectedBatch.value?.id === updated.id) selectedBatch.value = updated
    renameBatch.value = null
    toast.success('批次名称已更新')
  } catch (error) {
    toast.error({ title: '批次名称更新失败', message: errorText(error) })
  }
}

onMounted(() => {
  void search(false)
})
onBeforeUnmount(store.clearSensitiveValues)
</script>

<template>
  <main class="page batches-page">
    <header class="page__header">
      <div>
        <p class="page__eyebrow">Issuance ledger</p>
        <h1 class="page__title">生成批次</h1>
        <p class="page__description">
          每次生成都形成可追溯批次，集中查看未使用库存、兑换进度、过期风险和发放配置。
        </p>
      </div>
      <div class="page__actions">
        <BaseButton
          variant="secondary"
          @click="router.push('/manage/redemption-codes')"
        >
          <TicketCheck :size="16" aria-hidden="true" />
          查看兑换码
        </BaseButton>
        <BaseButton @click="openCreate">
          <Plus :size="16" aria-hidden="true" />
          生成兑换码
        </BaseButton>
      </div>
    </header>

    <section class="batch-filter">
      <div class="filter-title">
        <span>批次查询</span>
        <strong>按运营用途追踪发放</strong>
      </div>
      <label>
        <span>批次名称或编号</span>
        <input
          v-model="filters.keyword"
          class="form-control"
          placeholder="搜索批次"
          @keydown.enter="search()"
        />
      </label>
      <label>
        <span>商品标识</span>
        <select v-model="filters.productCode" class="form-control">
          <option value="">全部商品</option>
          <option :value="APP_CONFIG.productCode">
            {{ APP_CONFIG.productCode }}
          </option>
        </select>
      </label>
      <div class="filter-actions">
        <BaseButton
          variant="ghost"
          size="sm"
          :disabled="store.isLoading"
          @click="resetFilters"
        >
          <X :size="15" aria-hidden="true" />
          清空
        </BaseButton>
        <BaseButton
          size="sm"
          :loading="store.isLoading"
          @click="search()"
        >
          <Search :size="15" aria-hidden="true" />
          查询
        </BaseButton>
      </div>
    </section>

    <RedemptionBatchesTable
      :items="batches"
      :page="store.batches.page"
      :page-size="store.batches.pageSize"
      :total="store.batches.total"
      :loading="store.isLoading"
      :exporting-id="exportingId"
      @update:page="changePage"
      @detail="openDetail"
      @export="exportBatch"
      @export-xianyu="exportXianyu"
      @disable="disableBatch = $event"
      @extend="extendBatch = $event"
      @rename="renameBatch = $event"
    />
  </main>

  <GenerateCodesModal
    :open="createOpen"
    :loading="store.isMutating"
    :exporting="Boolean(exportingId)"
    :result="createResult"
    @close="closeCreate"
    @submit="createBatch"
    @copy="copy"
    @export="exportBatch"
    @export-xianyu="exportXianyu"
  />

  <RedemptionBatchDrawer
    :open="drawerOpen"
    :batch="selectedBatch"
    :codes="drawerCodes"
    :revealed-codes="store.revealedCodes"
    :loading="store.isLoading"
    :revealing="revealingBatch"
    :exporting="exportingId === selectedBatch?.id"
    @close="drawerOpen = false"
    @reveal="revealCode"
    @reveal-all="revealAll"
    @copy="copy"
    @export="exportBatch"
    @export-xianyu="exportXianyu"
    @disable="disableBatch = $event"
    @extend="extendBatch = $event"
  />

  <ConfirmDialog
    :open="Boolean(disableBatch)"
    title="失效批次中的未使用兑换码"
    :description="`${disableBatch?.name || ''} 当前有 ${disableBatch?.counts.unused || 0} 个未使用兑换码。已兑换、已过期和已失效记录不会改变。`"
    confirm-label="确认批量失效"
    danger
    :loading="store.isMutating"
    reason-label="失效原因"
    reason-placeholder="说明该批次停止发放的原因"
    reason-required
    @close="disableBatch = null"
    @confirm="confirmDisable"
  />

  <ExtendRedemptionModal
    :open="Boolean(extendBatch)"
    :target-label="extendBatch?.name || ''"
    :count="
      (extendBatch?.counts.unused || 0) + (extendBatch?.counts.expired || 0)
    "
    :loading="store.isMutating"
    @close="extendBatch = null"
    @confirm="confirmExtend"
  />

  <RenameRedemptionBatchModal
    :open="Boolean(renameBatch)"
    :batch="renameBatch"
    :loading="store.isMutating"
    @close="renameBatch = null"
    @submit="confirmRename"
  />
</template>

<style scoped>
.batches-page {
  display: grid;
  align-content: start;
  gap: 16px;
}

.batches-page .page__header {
  margin-bottom: 2px;
}

.batch-filter {
  display: grid;
  grid-template-columns: 190px minmax(220px, 1.4fr) minmax(180px, 0.8fr) auto;
  gap: 14px;
  align-items: end;
  padding: 16px 18px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.filter-title {
  align-self: center;
  display: grid;
  gap: 3px;
}

.filter-title span,
.batch-filter label > span {
  color: var(--ink-muted);
  font-size: 10px;
}

.filter-title strong {
  font-size: 12px;
}

.batch-filter label {
  display: grid;
  gap: 6px;
}

.batch-filter .form-control {
  min-height: 38px;
  padding-block: 7px;
  font-size: 12px;
}

.filter-actions {
  display: flex;
  gap: 8px;
}

@media (max-width: 1000px) {
  .batch-filter {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 600px) {
  .batch-filter {
    grid-template-columns: 1fr;
  }

  .filter-actions {
    justify-content: flex-end;
  }
}
</style>
