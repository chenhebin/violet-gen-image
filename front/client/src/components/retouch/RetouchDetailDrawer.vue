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
import RetouchActionModal, {
  type RetouchAction,
} from './RetouchActionModal.vue'
import RetouchDeliverables from './RetouchDeliverables.vue'
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
const canCancel = computed(() =>
  props.ticket
    ? ['submitted', 'quote_pending', 'accepted'].includes(props.ticket.status)
    : false,
)

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
            </section>

            <section class="request-section">
              <header>
                <div>
                  <p>精修需求</p>
                  <h3>提交内容</h3>
                </div>
                <span>{{ ticket.selectedResults.length }} 张原结果</span>
              </header>

              <div class="media-strip">
                <figure v-for="(result, index) in ticket.selectedResults" :key="result.id">
                  <img :src="result.url" :alt="`待精修原结果 ${index + 1}`" />
                  <figcaption>原结果 {{ index + 1 }}</figcaption>
                </figure>
              </div>

              <div class="requirement-copy">
                <strong>处理要求</strong>
                <p>{{ ticket.requirement }}</p>
              </div>

              <div v-if="ticket.supplementalAssets.length" class="supplemental">
                <strong>补充参考</strong>
                <div class="asset-row">
                  <figure v-for="asset in ticket.supplementalAssets" :key="asset.id">
                    <img v-if="asset.previewUrl" :src="asset.previewUrl" :alt="asset.name" />
                    <span>{{ asset.name }}</span>
                  </figure>
                </div>
              </div>

              <div v-if="ticket.revision" class="revision-note">
                <span><RotateCcw :size="15" />已提交返修要求</span>
                <p>{{ ticket.revision.message }}</p>
              </div>
            </section>

            <RetouchDeliverables :deliverables="ticket.deliverables" />
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
  min-width: 720px;
  max-width: 100%;
  height: 100%;
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

.drawer-header p,
.request-section header p {
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

.overview-heading,
.request-section > header {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 16px;
}

.overview-heading time,
.request-section header > span {
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

.request-section {
  padding: 24px 0;
  margin-top: 24px;
  border-top: 1px solid var(--border);
}

.request-section h3 {
  margin-top: 2px;
  font-size: 16px;
}

.media-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  margin-top: 16px;
}

figure {
  overflow: hidden;
  margin: 0;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-soft);
}

.media-strip img {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
}

.media-strip figcaption,
.asset-row span {
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
.revision-note {
  margin-top: 18px;
}

.requirement-copy > strong,
.supplemental > strong {
  font-size: 11px;
}

.requirement-copy p,
.revision-note p {
  margin-top: 6px;
  color: var(--ink-muted);
  font-size: 12px;
  line-height: 1.7;
  white-space: pre-wrap;
}

.asset-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  margin-top: 8px;
}

.asset-row img {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
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

  .media-strip,
  .asset-row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  footer {
    align-items: stretch;
    flex-direction: column;
  }

  .footer-spacer {
    display: none;
  }
}

@media (max-width: 760px) {
  aside {
    width: 100%;
    min-width: 0;
  }
}
</style>
