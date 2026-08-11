<script setup lang="ts">
import {
  CircleDollarSign,
  ImageUp,
  MessageSquareText,
  Play,
  ShieldX,
} from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDrawer from '@/components/base/BaseDrawer.vue'
import RetouchStatusBadge from '@/components/shared/RetouchStatusBadge.vue'
import TaskStatusBadge from '@/components/shared/TaskStatusBadge.vue'
import { formatDateTime } from '@/utils/format'
import type { ManageRetouchTicket } from '@/types/domain'
import type { RetouchAction } from './RetouchActionModal.vue'

defineProps<{
  open: boolean
  ticket: ManageRetouchTicket | null
  loading: boolean
}>()

defineEmits<{
  close: []
  action: [action: RetouchAction]
}>()
</script>

<template>
  <BaseDrawer
    :open="open"
    :title="ticket?.ticketNo ?? '人工修图工单'"
    :description="ticket ? `${ticket.taskTitle} · ${ticket.user.email}` : '正在读取工单详情'"
    size="large"
    @close="$emit('close')"
  >
    <div v-if="loading && !ticket" class="drawer-loading">正在读取工单详情…</div>
    <div v-else-if="ticket" class="ticket-detail">
      <section class="ticket-ledger">
        <div class="ticket-ledger__status">
          <span>当前状态</span>
          <RetouchStatusBadge :status="ticket.status" />
        </div>
        <dl>
          <div>
            <dt>报价</dt>
            <dd>{{ ticket.quote?.credits ?? '-' }} 次</dd>
          </div>
          <div>
            <dt>预占</dt>
            <dd>{{ ticket.reservedCredits }} 次</dd>
          </div>
          <div>
            <dt>已结算</dt>
            <dd>{{ ticket.spentCredits }} 次</dd>
          </div>
          <div>
            <dt>已退款</dt>
            <dd>{{ ticket.refundedCredits }} 次</dd>
          </div>
        </dl>
      </section>

      <section class="detail-section">
        <header>
          <div>
            <span>用户要求</span>
            <h3>人工修图说明</h3>
          </div>
          <time :datetime="ticket.updatedAt">{{ formatDateTime(ticket.updatedAt) }}</time>
        </header>
        <p class="requirement">{{ ticket.requirement }}</p>
        <div v-if="ticket.revision" class="revision">
          <MessageSquareText :size="17" />
          <div>
            <strong>用户返修要求</strong>
            <p>{{ ticket.revision.message }}</p>
          </div>
        </div>
      </section>

      <section class="detail-section">
        <header>
          <div>
            <span>待处理素材</span>
            <h3>用户选中的 AI 成片</h3>
          </div>
          <b>{{ ticket.selectedResults.length }} 张</b>
        </header>
        <div class="media-grid">
          <figure v-for="result in ticket.selectedResults" :key="result.id">
            <img :src="result.url" alt="用户选中的 AI 成片" />
            <figcaption>{{ result.width }} × {{ result.height }}</figcaption>
          </figure>
        </div>
        <div v-if="ticket.supplementalAssets.length" class="supplemental">
          <strong>补充参考图</strong>
          <div class="media-grid media-grid--small">
            <figure v-for="asset in ticket.supplementalAssets" :key="asset.id">
              <img :src="asset.previewUrl" :alt="asset.name" />
              <figcaption>{{ asset.name }}</figcaption>
            </figure>
          </div>
        </div>
      </section>

      <section v-if="ticket.deliverables.length" class="detail-section">
        <header>
          <div>
            <span>人工交付</span>
            <h3>已上传成片</h3>
          </div>
          <b>{{ ticket.deliverables.length }} 张</b>
        </header>
        <div class="media-grid">
          <figure v-for="result in ticket.deliverables" :key="result.id">
            <img :src="result.url" alt="人工修图交付成片" />
            <figcaption>{{ result.width }} × {{ result.height }}</figcaption>
          </figure>
        </div>
      </section>

      <section class="detail-section source-task">
        <header>
          <div>
            <span>来源任务</span>
            <h3>{{ ticket.sourceTaskDetail.title }}</h3>
          </div>
          <TaskStatusBadge :status="ticket.sourceTaskDetail.status" />
        </header>
        <dl>
          <div>
            <dt>模式</dt>
            <dd>
              {{
                ticket.sourceTaskDetail.mode === 'text-to-image'
                  ? '文生图'
                  : '图生图'
              }}
            </dd>
          </div>
          <div>
            <dt>模型快照</dt>
            <dd>{{ ticket.sourceTaskDetail.modelName }}</dd>
          </div>
          <div>
            <dt>原始需求</dt>
            <dd>{{ ticket.sourceTaskDetail.sourceRequirement }}</dd>
          </div>
        </dl>
      </section>

      <section class="detail-section">
        <header>
          <div>
            <span>处理记录</span>
            <h3>工单时间线</h3>
          </div>
        </header>
        <ol class="timeline">
          <li v-for="entry in ticket.timeline" :key="`${entry.action}-${entry.createdAt}`">
            <i aria-hidden="true"></i>
            <div>
              <strong>{{ entry.action }}</strong>
              <p v-if="entry.note">{{ entry.note }}</p>
              <time :datetime="entry.createdAt">{{ formatDateTime(entry.createdAt) }}</time>
            </div>
          </li>
        </ol>
      </section>
    </div>

    <template v-if="ticket" #footer>
      <BaseButton
        v-if="['submitted', 'quote_pending'].includes(ticket.status)"
        variant="danger"
        @click="$emit('action', 'reject')"
      >
        <template #icon><ShieldX :size="16" /></template>
        拒绝需求
      </BaseButton>
      <BaseButton
        v-if="['accepted', 'processing', 'awaiting_confirmation'].includes(ticket.status)"
        variant="danger"
        @click="$emit('action', 'fail')"
      >
        <template #icon><ShieldX :size="16" /></template>
        履约失败
      </BaseButton>
      <BaseButton
        v-if="['submitted', 'quote_pending'].includes(ticket.status)"
        @click="$emit('action', 'quote')"
      >
        <template #icon><CircleDollarSign :size="16" /></template>
        {{ ticket.quote ? '调整报价' : '给出报价' }}
      </BaseButton>
      <BaseButton v-if="ticket.status === 'accepted'" @click="$emit('action', 'start')">
        <template #icon><Play :size="16" /></template>
        确认开工
      </BaseButton>
      <BaseButton v-if="ticket.status === 'processing'" @click="$emit('action', 'deliver')">
        <template #icon><ImageUp :size="16" /></template>
        上传并交付
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

.ticket-detail {
  display: grid;
  gap: 18px;
}

.ticket-ledger,
.detail-section {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
}

.ticket-ledger {
  display: grid;
  grid-template-columns: 210px minmax(0, 1fr);
  overflow: hidden;
}

.ticket-ledger__status {
  display: grid;
  align-content: center;
  gap: 9px;
  padding: 20px;
  border-right: 1px solid var(--border);
  background: var(--surface-soft);
}

.ticket-ledger__status > span,
.detail-section header span {
  color: var(--ink-muted);
  font-size: 10px;
  font-weight: 750;
}

.ticket-ledger dl {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
}

.ticket-ledger dl > div {
  display: grid;
  align-content: center;
  gap: 4px;
  padding: 18px;
  border-right: 1px solid var(--border);
}

.ticket-ledger dl > div:last-child {
  border-right: 0;
}

dt {
  color: var(--ink-muted);
  font-size: 11px;
}

dd {
  margin: 0;
  font-size: 13px;
  font-weight: 650;
}

.ticket-ledger dd {
  font-family: var(--font-mono);
  font-size: 16px;
}

.detail-section {
  padding: 18px;
}

.detail-section > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.detail-section h3 {
  margin-top: 2px;
  font-size: 15px;
}

.detail-section header time,
.detail-section header b {
  color: var(--ink-muted);
  font-size: 11px;
  font-weight: 600;
}

.requirement {
  color: var(--ink);
  font-size: 14px;
  line-height: 1.75;
  white-space: pre-wrap;
}

.revision {
  display: flex;
  gap: 10px;
  padding: 12px;
  margin-top: 14px;
  border-left: 3px solid var(--warning);
  border-radius: var(--radius-sm);
  background: var(--warning-soft);
}

.revision svg {
  flex: 0 0 auto;
  color: var(--warning);
}

.revision strong {
  font-size: 12px;
}

.revision p {
  margin-top: 3px;
  color: var(--ink-muted);
  font-size: 12px;
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

.media-grid img {
  width: 100%;
  aspect-ratio: 4 / 3;
  object-fit: cover;
}

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

.source-task dl {
  display: grid;
  gap: 10px;
}

.source-task dl > div {
  display: grid;
  grid-template-columns: 110px minmax(0, 1fr);
  gap: 12px;
}

.source-task dd {
  line-height: 1.65;
}

.timeline {
  display: grid;
  gap: 0;
  padding: 0;
  margin: 0;
  list-style: none;
}

.timeline li {
  position: relative;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  gap: 10px;
  padding-bottom: 16px;
}

.timeline li:not(:last-child)::before {
  position: absolute;
  top: 8px;
  bottom: -2px;
  left: 4px;
  width: 1px;
  background: var(--border);
  content: '';
}

.timeline i {
  position: relative;
  z-index: 1;
  width: 9px;
  height: 9px;
  margin-top: 5px;
  border: 2px solid var(--surface);
  border-radius: 50%;
  background: var(--primary);
  box-shadow: 0 0 0 1px var(--primary);
}

.timeline strong {
  font-size: 12px;
}

.timeline p,
.timeline time {
  display: block;
  margin-top: 3px;
  color: var(--ink-muted);
  font-size: 11px;
}

@media (max-width: 780px) {
  .ticket-ledger {
    grid-template-columns: 1fr;
  }

  .ticket-ledger__status {
    border-right: 0;
    border-bottom: 1px solid var(--border);
  }

  .ticket-ledger dl {
    grid-template-columns: repeat(2, 1fr);
  }

  .ticket-ledger dl > div:nth-child(2) {
    border-right: 0;
  }

  .ticket-ledger dl > div:nth-child(-n + 2) {
    border-bottom: 1px solid var(--border);
  }
}
</style>
