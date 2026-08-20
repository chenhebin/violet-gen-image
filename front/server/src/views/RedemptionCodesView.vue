<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Download, Layers3, Plus, ShieldOff } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import ConfirmDialog from '@/components/base/ConfirmDialog.vue'
import ExtendRedemptionModal from '@/components/redemptions/ExtendRedemptionModal.vue'
import GenerateCodesModal from '@/components/redemptions/GenerateCodesModal.vue'
import RedemptionCodeDrawer from '@/components/redemptions/RedemptionCodeDrawer.vue'
import RedemptionCodesTable from '@/components/redemptions/RedemptionCodesTable.vue'
import RedemptionFilters, {
  type RedemptionFilterValue,
} from '@/components/redemptions/RedemptionFilters.vue'
import { useToast } from '@/composables/useToast'
import { useRedemptionStore } from '@/stores/redemption'
import type {
  CreateRedemptionBatchPayload,
  CreateRedemptionBatchResult,
  RedemptionCode,
} from '@/types'

type MutationTarget =
  | { kind: 'codes'; ids: string[]; label: string; count: number }
  | { kind: 'batch'; batchId: string; label: string; count: number }

const route = useRoute()
const router = useRouter()
const store = useRedemptionStore()
const toast = useToast()

const pageSize = 20
const page = ref(Number(route.query.page) || 1)
const filters = reactive<RedemptionFilterValue>({
  keyword: String(route.query.keyword || ''),
  status: (route.query.status || '') as RedemptionFilterValue['status'],
  batchId: String(route.query.batchId || ''),
  redeemedBy: String(route.query.redeemedBy || ''),
  expiringSoon: route.query.expiringSoon === 'true',
})
const selectedIds = ref<string[]>([])
const createOpen = ref(false)
const createResult = ref<CreateRedemptionBatchResult | null>(null)
const selectedCode = ref<RedemptionCode | null>(null)
const drawerOpen = ref(false)
const disableTarget = ref<MutationTarget | null>(null)
const extendTarget = ref<MutationTarget | null>(null)
const exportingBatchId = ref<string | null>(null)

const codes = computed(() => store.codes.items)
const batches = computed(() => store.batches.items)
const selectedBatch = computed(() =>
  batches.value.find((batch) => batch.id === filters.batchId),
)

function errorText(error: unknown) {
  return error instanceof Error ? error.message : '操作未完成，请稍后重试'
}

async function loadCodes() {
  await store.loadCodes({
    page: page.value,
    pageSize,
    keyword: filters.keyword.trim() || undefined,
    status: filters.status || undefined,
    batchId: filters.batchId || undefined,
    redeemedBy: filters.redeemedBy.trim() || undefined,
    expiringSoon: filters.expiringSoon || undefined,
  })
}

async function loadInitial() {
  try {
    await Promise.all([
      store.loadBatches({ page: 1, pageSize: 100 }),
      loadCodes(),
    ])
  } catch (error) {
    toast.error({ title: '兑换码台账加载失败', message: errorText(error) })
  }
}

async function syncAndSearch(resetPage = true) {
  if (resetPage) page.value = 1
  await router.replace({
    query: {
      ...(filters.keyword && { keyword: filters.keyword }),
      ...(filters.status && { status: filters.status }),
      ...(filters.batchId && { batchId: filters.batchId }),
      ...(filters.redeemedBy && { redeemedBy: filters.redeemedBy }),
      ...(filters.expiringSoon && { expiringSoon: 'true' }),
      ...(page.value > 1 && { page: String(page.value) }),
    },
  })
  selectedIds.value = []
  try {
    await loadCodes()
  } catch (error) {
    toast.error({ title: '查询失败', message: errorText(error) })
  }
}

function resetFilters() {
  Object.assign(filters, {
    keyword: '',
    status: '',
    batchId: '',
    redeemedBy: '',
    expiringSoon: false,
  })
  void syncAndSearch()
}

async function changePage(nextPage: number) {
  page.value = nextPage
  await syncAndSearch(false)
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

async function reveal(code: RedemptionCode) {
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

async function copy(value: string, label = '完整兑换码') {
  try {
    await navigator.clipboard.writeText(value)
    toast.success(`${label}已复制`)
  } catch {
    toast.error('浏览器未授权剪贴板，请手动选择内容')
  }
}

async function openDetail(code: RedemptionCode) {
  selectedCode.value = code
  drawerOpen.value = true
  try {
    selectedCode.value = await store.loadCode(code.id)
  } catch (error) {
    toast.error({ title: '详情加载失败', message: errorText(error) })
  }
}

function targetForCode(code: RedemptionCode): MutationTarget {
  return {
    kind: 'codes',
    ids: [code.id],
    label: code.maskedCode,
    count: 1,
  }
}

function openBulkDisable() {
  if (!selectedIds.value.length) return
  disableTarget.value = {
    kind: 'codes',
    ids: [...selectedIds.value],
    label: '已选兑换码',
    count: selectedIds.value.length,
  }
}

async function confirmDisable(reason: string) {
  if (!disableTarget.value) return
  const target = disableTarget.value
  try {
    const result = await store.disableCodes({
      ...(target.kind === 'codes'
        ? { codeIds: target.ids }
        : { batchId: target.batchId }),
      reason,
    })
    toast.success({
      title: `已失效 ${result.affected} 个兑换码`,
      message: result.skipped ? `另有 ${result.skipped} 个因状态变化被跳过` : undefined,
    })
    selectedIds.value = []
    disableTarget.value = null
    drawerOpen.value = false
  } catch (error) {
    toast.error({ title: '失效操作失败', message: errorText(error) })
  }
}

async function confirmExtend(payload: {
  expiresAt: string | null
  reason: string
}) {
  if (!extendTarget.value) return
  const target = extendTarget.value
  try {
    const result = await store.extendCodes({
      ...(target.kind === 'codes'
        ? { codeIds: target.ids }
        : { batchId: target.batchId }),
      ...payload,
    })
    toast.success({
      title: `已更新 ${result.affected} 个兑换码`,
      message: result.skipped ? `跳过 ${result.skipped} 个不可延期记录` : undefined,
    })
    extendTarget.value = null
    drawerOpen.value = false
  } catch (error) {
    toast.error({ title: '延期失败', message: errorText(error) })
  }
}

async function exportBatch(batchId: string) {
  exportingBatchId.value = batchId
  try {
    const file = await store.exportBatch(batchId)
    const url = URL.createObjectURL(
      new Blob([file.content || file.csv || ''], { type: file.mediaType }),
    )
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = file.filename
    anchor.click()
    URL.revokeObjectURL(url)
    toast.success('CSV 已生成并开始下载')
  } catch (error) {
    toast.error({ title: '导出失败', message: errorText(error) })
  } finally {
    exportingBatchId.value = null
  }
}

onMounted(loadInitial)
onBeforeUnmount(store.clearSensitiveValues)
</script>

<template>
  <main class="page redemption-page">
    <header class="page__header">
      <div>
        <p class="page__eyebrow">Redemption inventory</p>
        <h1 class="page__title">兑换码</h1>
        <p class="page__description">
          管理咸鱼商品对应的发放凭证。完整码仅在授权操作中短暂展示，核销历史保持只读。
        </p>
      </div>
      <div class="page__actions">
        <BaseButton
          v-if="selectedBatch"
          variant="secondary"
          :loading="exportingBatchId === selectedBatch.id"
          @click="exportBatch(selectedBatch.id)"
        >
          <Download :size="16" aria-hidden="true" />
          导出当前批次
        </BaseButton>
        <BaseButton
          variant="secondary"
          @click="router.push('/manage/redemption-batches')"
        >
          <Layers3 :size="16" aria-hidden="true" />
          生成批次
        </BaseButton>
        <BaseButton @click="openCreate">
          <Plus :size="16" aria-hidden="true" />
          生成兑换码
        </BaseButton>
      </div>
    </header>

    <RedemptionFilters
      :model-value="filters"
      :batches="batches"
      :loading="store.isLoading"
      @update:model-value="Object.assign(filters, $event)"
      @search="syncAndSearch()"
      @reset="resetFilters"
    />

    <div v-if="selectedIds.length" class="bulk-bar">
      <div>
        <ShieldOff :size="17" aria-hidden="true" />
        <strong>已选择 {{ selectedIds.length }} 个未使用兑换码</strong>
        <span>批量操作提交时会再次校验最新状态</span>
      </div>
      <BaseButton
        variant="secondary"
        size="sm"
        @click="
          extendTarget = {
            kind: 'codes',
            ids: [...selectedIds],
            label: '已选兑换码',
            count: selectedIds.length,
          }
        "
      >
        批量延期
      </BaseButton>
      <BaseButton variant="danger" size="sm" @click="openBulkDisable">
        批量失效
      </BaseButton>
    </div>

    <RedemptionCodesTable
      :items="codes"
      :page="store.codes.page"
      :page-size="store.codes.pageSize"
      :total="store.codes.total"
      :loading="store.isLoading"
      :selected-ids="selectedIds"
      :revealed-codes="store.revealedCodes"
      @update:page="changePage"
      @update:selected-ids="selectedIds = $event"
      @reveal="reveal"
      @copy="copy"
      @detail="openDetail"
      @disable="disableTarget = targetForCode($event)"
      @extend="extendTarget = targetForCode($event)"
    />
  </main>

  <GenerateCodesModal
    :open="createOpen"
    :loading="store.isMutating"
    :exporting="Boolean(exportingBatchId)"
    :result="createResult"
    @close="closeCreate"
    @submit="createBatch"
    @copy="copy"
    @export="exportBatch"
  />

  <RedemptionCodeDrawer
    :open="drawerOpen"
    :code="selectedCode"
    :revealed-code="
      selectedCode ? store.revealedCodes[selectedCode.id] : undefined
    "
    :loading="store.isLoading"
    @close="drawerOpen = false"
    @reveal="reveal"
    @copy="copy"
    @disable="disableTarget = targetForCode($event)"
    @extend="extendTarget = targetForCode($event)"
  />

  <ConfirmDialog
    :open="Boolean(disableTarget)"
    title="失效兑换码"
    :description="`将处理 ${disableTarget?.label || ''} 中的 ${disableTarget?.count || 0} 个目标。已兑换、过期和已失效记录会被跳过，用户次数不会被收回。`"
    confirm-label="确认失效"
    danger
    :loading="store.isMutating"
    reason-label="失效原因"
    reason-placeholder="说明为什么需要失效这些兑换码"
    reason-required
    @close="disableTarget = null"
    @confirm="confirmDisable"
  />

  <ExtendRedemptionModal
    :open="Boolean(extendTarget)"
    :target-label="extendTarget?.label || ''"
    :count="extendTarget?.count || 0"
    :loading="store.isMutating"
    @close="extendTarget = null"
    @confirm="confirmExtend"
  />
</template>

<style scoped>
.redemption-page {
  display: grid;
  align-content: start;
  gap: 16px;
}

.redemption-page .page__header {
  margin-bottom: 2px;
}

.bulk-bar {
  display: flex;
  gap: 9px;
  align-items: center;
  min-height: 54px;
  padding: 8px 10px 8px 15px;
  color: var(--primary);
  background: var(--primary-soft);
  border: 1px solid #cde0db;
  border-radius: var(--radius-md);
}

.bulk-bar > div {
  display: flex;
  flex: 1;
  gap: 9px;
  align-items: center;
  min-width: 0;
}

.bulk-bar strong {
  color: var(--ink);
  font-size: 12px;
}

.bulk-bar span {
  color: var(--ink-muted);
  font-size: 10px;
}

@media (max-width: 760px) {
  .bulk-bar,
  .bulk-bar > div {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
