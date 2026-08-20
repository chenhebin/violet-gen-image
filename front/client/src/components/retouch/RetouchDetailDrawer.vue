<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  Ban,
  Check,
  Clock3,
  Coins,
  RotateCcw,
  Sparkles,
  X,
} from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import ImageLightbox, {
  type LightboxImage,
} from '@/components/base/ImageLightbox.vue'
import { assetApi } from '@/services/api'
import RetouchActionModal, {
  type RetouchAction,
} from './RetouchActionModal.vue'
import RetouchDeliverables from './RetouchDeliverables.vue'
import RetouchRequestSection from './RetouchRequestSection.vue'
import RetouchStatusBadge from './RetouchStatusBadge.vue'
import RetouchTimeline from './RetouchTimeline.vue'
import type { RetouchTicket } from '@/types/domain'
const props = defineProps<{
  open: boolean
  ticket: RetouchTicket | null
  loading: boolean
  submitting: boolean
}>()
const emit = defineEmits<{
  close: []
  acceptQuote: [ticketId: string, quoteId: string]
  cancel: [ticketId: string]
  confirm: [ticketId: string]
  requestRevision: [ticketId: string, message: string]
}>()
const pendingAction = ref<RetouchAction | null>(null)
const selectedPreviewId = ref('')
const previewUrlOverrides = ref<Record<string, string>>({})
const canCancel = computed(() =>
  props.ticket
    ? ['submitted', 'quote_pending', 'accepted'].includes(props.ticket.status)
    : false,
)
const selectedResults = computed(() =>
  (props.ticket?.selectedResults ?? []).map((result) => ({
    ...result,
    url: previewUrlOverrides.value[`selected:${result.id}`] ?? result.url,
  })),
)
const supplementalAssets = computed(() =>
  (props.ticket?.supplementalAssets ?? []).map((asset) => ({
    ...asset,
    previewUrl:
      previewUrlOverrides.value[`supplemental:${asset.id}`] ?? asset.previewUrl,
  })),
)
const deliverables = computed(() =>
  (props.ticket?.deliverables ?? []).map((result) => ({
    ...result,
    url: previewUrlOverrides.value[`deliverable:${result.id}`] ?? result.url,
  })),
)
const lightboxImages = computed<LightboxImage[]>(() => [
  ...selectedResults.value.map((result, index) => ({
    id: `selected:${result.id}`,
    src: result.url,
    alt: `待精修原结果 ${index + 1}`,
    label: `待精修原结果 ${index + 1}`,
    meta: `${result.width} × ${result.height}`,
  })),
  ...supplementalAssets.value.flatMap((asset) =>
    asset.previewUrl
      ? [{
          id: `supplemental:${asset.id}`,
          src: asset.previewUrl,
          alt: asset.name,
          label: asset.name,
        }]
      : [],
  ),
  ...deliverables.value.map((result, index) => ({
    id: `deliverable:${result.id}`,
    src: result.url,
    alt: `人工精修成片 ${index + 1}`,
    label: `人工精修成片 ${index + 1}`,
    meta: `${result.width} × ${result.height}`,
  })),
])

function openPreview(id: string): void {
  selectedPreviewId.value = id
}

async function refreshPreview(id: string): Promise<void> {
  const assetId = id.slice(id.indexOf(':') + 1)
  try {
    const signed = await assetApi.getUrl(assetId)
    previewUrlOverrides.value = {
      ...previewUrlOverrides.value,
      [id]: signed.url,
    }
  } catch {
    // Keep the current thumbnail visible when refreshing is unavailable.
  }
}
function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape' && props.open && !pendingAction.value) emit('close')
}
function submitAction(message?: string): void {
  const ticket = props.ticket
  if (!ticket) return

  if (pendingAction.value === 'accept' && ticket.quote) {
    emit('acceptQuote', ticket.id, ticket.quote.id)
  } else if (pendingAction.value === 'cancel') {
    emit('cancel', ticket.id)
  } else if (pendingAction.value === 'confirm') {
    emit('confirm', ticket.id)
  } else if (pendingAction.value === 'revision' && message) {
    emit('requestRevision', ticket.id, message)
  }
}
function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
watch(
  () => props.ticket?.status,
  () => {
    pendingAction.value = null
  },
)
watch(
  [() => props.ticket?.id, () => props.open],
  ([ticketId, open], [previousTicketId]) => {
    selectedPreviewId.value = ''
    if (!open || ticketId !== previousTicketId) previewUrlOverrides.value = {}
  },
)
onMounted(() => document.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => document.removeEventListener('keydown', onKeydown))
</script>
<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="open" class="drawer-layer">
        <button
          class="drawer-scrim"
          aria-label="关闭精修详情"
          @click="$emit('close')"
        />
        <aside role="dialog" aria-modal="true" aria-labelledby="retouch-heading">
          <header class="drawer-header">
            <div>
              <p>{{ ticket?.ticketNo ?? '精修工单' }}</p>
              <h2 id="retouch-heading">{{ ticket?.taskTitle ?? '正在加载' }}</h2>
            </div>
            <button class="icon-button" aria-label="关闭" @click="$emit('close')">
              <X :size="20" />
            </button>
          </header>

          <div v-if="loading && !ticket" class="loading-state">
            <Clock3 :size="25" />
            <p>正在读取工单详情...</p>
          </div>

          <div v-else-if="ticket" class="drawer-content">
            <section class="ticket-overview">
              <div class="overview-heading">
                <RetouchStatusBadge :status="ticket.status" />
                <time :datetime="ticket.updatedAt">
                  更新于 {{ formatDate(ticket.updatedAt) }}
                </time>
              </div>

              <div class="settlement-grid">
                <div>
                  <span><Coins :size="15" />报价</span>
                  <strong>{{ ticket.quote?.credits ?? '待定' }} 次</strong>
                  <small v-if="ticket.quote?.status === 'active'">剩余 {{ Math.ceil(ticket.quote.remainingSeconds / 3600) }} 小时</small>
                  <small v-else-if="ticket.quote?.status === 'expired'" class="warning-text">报价已过期</small>
                </div>
                <div>
                  <span>已结算</span>
                  <strong>{{ ticket.spentCredits }} 次</strong>
                </div>
                <div>
                  <span>已退回</span>
                  <strong>{{ ticket.refundedCredits }} 次</strong>
                </div>
                <div>
                  <span>返修机会</span>
                  <strong>{{ ticket.revision ? '已使用' : '1 次' }}</strong>
                </div>
              </div>
              <p v-if="ticket.sla.stage !== 'completed'" class="sla-note" :class="{ overdue: ticket.sla.overdue }">
                {{ ticket.sla.overdue ? '当前阶段已逾期' : '当前阶段剩余 ' + Math.ceil((ticket.sla.remainingSeconds ?? 0) / 3600) + ' 小时' }}
              </p>
            </section>

            <RetouchRequestSection
              :selected-results="selectedResults"
              :supplemental-assets="supplementalAssets"
              :requirement="ticket.requirement"
              :revision="ticket.revision"
              @preview="openPreview"
              @image-error="refreshPreview"
            />

            <RetouchDeliverables
              :deliverables="deliverables"
              @preview="openPreview"
              @image-error="refreshPreview"
            />
            <RetouchTimeline :entries="ticket.timeline" />
          </div>

          <footer v-if="ticket && (canCancel || ticket.status === 'quote_pending' || ticket.status === 'awaiting_confirmation')">
            <BaseButton
              v-if="canCancel"
              variant="danger"
              @click="pendingAction = 'cancel'"
            >
              <template #icon><Ban :size="17" /></template>
              取消工单
            </BaseButton>
            <span class="footer-spacer" />
            <BaseButton
              v-if="ticket.status === 'awaiting_confirmation' && !ticket.revision"
              variant="secondary"
              @click="pendingAction = 'revision'"
            >
              <template #icon><RotateCcw :size="17" /></template>
              申请返修
            </BaseButton>
            <BaseButton
              v-if="ticket.status === 'quote_pending'"
              @click="pendingAction = 'accept'"
            >
              <template #icon><Sparkles :size="17" /></template>
              接受 {{ ticket.quote?.credits ?? 0 }} 次报价
            </BaseButton>
            <BaseButton
              v-if="ticket.status === 'awaiting_confirmation'"
              @click="pendingAction = 'confirm'"
            >
              <template #icon><Check :size="17" /></template>
              确认交付
            </BaseButton>
          </footer>
        </aside>
      </div>
    </Transition>
  </Teleport>

  <ImageLightbox
    :open="Boolean(selectedPreviewId)"
    :images="lightboxImages"
    :selected-id="selectedPreviewId"
    @close="selectedPreviewId = ''"
    @select="selectedPreviewId = $event"
    @image-error="refreshPreview"
  />

  <RetouchActionModal
    v-if="ticket"
    :action="pendingAction"
    :ticket="ticket"
    :busy="submitting"
    @close="pendingAction = null"
    @confirm="submitAction"
  />
</template>

<style scoped>
.drawer-layer {
  position: fixed;
  z-index: 76;
  inset: 0;
}
.warning-text,
.sla-note.overdue { color: var(--danger); }
.sla-note { margin: 12px 0 0; color: var(--primary); font-size: 12px; }

.drawer-scrim {
  position: absolute;
  inset: 0;
  width: 100%;
  background: var(--scrim);
  cursor: default;
}
aside {
  position: absolute;
  top: 0;
  right: 0;
  display: grid;
  width: 66.6667vw;
  min-width: 720px; max-width: 100%; height: 100%;
  grid-template-rows: auto minmax(0, 1fr) auto;
  border-left: 1px solid var(--border);
  background: var(--surface);
  box-shadow: var(--shadow-md);
}
.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
}
.drawer-header p {
  color: var(--primary);
  font-size: 9px;
  font-weight: 800;
}
.drawer-header h2 {
  margin-top: 2px;
  font-size: 18px;
}
.icon-button {
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--ink-muted);
}
.icon-button:hover {
  background: var(--surface-soft);
}
.drawer-content {
  min-height: 0;
  overflow: auto;
  padding: 22px 24px 36px;
}
.loading-state {
  display: grid;
  place-items: center;
  align-content: center;
  gap: 12px;
  color: var(--ink-muted);
}
.overview-heading {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 16px;
}
.overview-heading time {
  color: var(--ink-faint);
  font-size: 10px;
}
.settlement-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-top: 18px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}
.settlement-grid > div {
  min-width: 0;
  padding: 14px;
  border-right: 1px solid var(--border);
}
.settlement-grid > div:last-child {
  border-right: 0;
}

.settlement-grid span,
.settlement-grid strong {
  display: flex;
  align-items: center;
  gap: 5px;
}

.settlement-grid span {
  color: var(--ink-faint);
  font-size: 9px;
}

.settlement-grid strong {
  margin-top: 5px;
  font-size: 13px;
}

footer {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 24px;
  border-top: 1px solid var(--border);
  background: var(--surface);
}

.footer-spacer {
  flex: 1;
}

.drawer-enter-active,
.drawer-leave-active {
  transition: opacity var(--motion-normal) var(--ease-out);
}

.drawer-enter-active aside,
.drawer-leave-active aside {
  transition: transform var(--motion-normal) var(--ease-out);
}

.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}

.drawer-enter-from aside,
.drawer-leave-to aside {
  transform: translateX(100%);
}

@media (max-width: 560px) {
  .settlement-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .settlement-grid > div:nth-child(2) {
    border-right: 0;
  }

  .settlement-grid > div:nth-child(-n + 2) {
    border-bottom: 1px solid var(--border);
  }

  footer {
    align-items: stretch;
    flex-direction: column;
  }

  .footer-spacer {
    display: none;
  }
}
@media (max-width: 760px) { aside { width: 100%; min-width: 0; } }
</style>
