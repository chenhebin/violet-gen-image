<script setup lang="ts">
import {
  CalendarPlus,
  Copy,
  Eye,
  History,
  ShieldOff,
  TicketCheck,
} from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDrawer from '@/components/base/BaseDrawer.vue'
import type { RedemptionCode, RedemptionCodeDetail } from '@/types'
import RedemptionStatusBadge from './RedemptionStatusBadge.vue'
import { formatDateTime } from './formatters'

const props = defineProps<{
  open: boolean
  code: RedemptionCode | RedemptionCodeDetail | null
  revealedCode?: string
  loading?: boolean
}>()

const emit = defineEmits<{
  close: []
  reveal: [code: RedemptionCode]
  copy: [value: string]
  disable: [code: RedemptionCode]
  extend: [code: RedemptionCode]
}>()

function hasHistory(
  code: RedemptionCode | RedemptionCodeDetail,
): code is RedemptionCodeDetail {
  return 'operationHistory' in code
}
</script>

<template>
  <BaseDrawer
    :open="props.open"
    title="兑换码详情"
    description="核销状态、有效期与管理操作历史"
    size="large"
    @close="emit('close')"
  >
    <div v-if="props.loading && !props.code" class="drawer-state">
      正在读取兑换码详情...
    </div>

    <div v-else-if="props.code" class="drawer-content">
      <section class="identity-panel">
        <div class="ticket-mark" aria-hidden="true">
          <TicketCheck :size="25" />
        </div>
        <div class="identity">
          <span>兑换码</span>
          <strong class="data-mono">
            {{ props.revealedCode || props.code.maskedCode }}
          </strong>
          <small>编号 {{ props.code.id }}</small>
        </div>
        <RedemptionStatusBadge
          :status="props.code.status"
          :expiring-soon="props.code.expiringSoon"
        />
      </section>

      <div class="sensitive-actions">
        <BaseButton
          v-if="props.code.status === 'unused' && !props.revealedCode"
          variant="secondary"
          size="sm"
          @click="emit('reveal', props.code)"
        >
          <Eye :size="15" aria-hidden="true" />
          查看完整码
        </BaseButton>
        <BaseButton
          v-if="props.revealedCode"
          variant="secondary"
          size="sm"
          @click="emit('copy', props.revealedCode)"
        >
          <Copy :size="15" aria-hidden="true" />
          复制完整码
        </BaseButton>
        <p>查看与复制均会写入审计记录，完整码不会持久保存在浏览器。</p>
      </div>

      <section class="detail-section">
        <header>
          <span>01</span>
          <h3>发放配置</h3>
        </header>
        <dl class="detail-grid">
          <div>
            <dt>生成批次</dt>
            <dd>{{ props.code.batchName }}</dd>
          </div>
          <div>
            <dt>商品标识</dt>
            <dd class="data-mono">{{ props.code.productCode }}</dd>
          </div>
          <div>
            <dt>可兑换次数</dt>
            <dd>{{ props.code.credits }} 次</dd>
          </div>
          <div>
            <dt>有效期</dt>
            <dd>{{ formatDateTime(props.code.expiresAt) }}</dd>
          </div>
          <div>
            <dt>创建时间</dt>
            <dd>{{ formatDateTime(props.code.createdAt) }}</dd>
          </div>
        </dl>
      </section>

      <section class="detail-section">
        <header>
          <span>02</span>
          <h3>核销与状态</h3>
        </header>
        <dl class="detail-grid">
          <div>
            <dt>兑换用户</dt>
            <dd>{{ props.code.redeemedByEmail || '尚未兑换' }}</dd>
          </div>
          <div>
            <dt>兑换时间</dt>
            <dd>
              {{
                props.code.redeemedAt
                  ? formatDateTime(props.code.redeemedAt)
                  : '—'
              }}
            </dd>
          </div>
          <div>
            <dt>失效时间</dt>
            <dd>
              {{
                props.code.disabledAt
                  ? formatDateTime(props.code.disabledAt)
                  : '—'
              }}
            </dd>
          </div>
          <div class="detail-grid__wide">
            <dt>失效原因</dt>
            <dd>{{ props.code.disabledReason || '—' }}</dd>
          </div>
        </dl>
      </section>

      <section class="detail-section">
        <header>
          <History :size="16" aria-hidden="true" />
          <h3>操作历史</h3>
        </header>
        <ol
          v-if="hasHistory(props.code) && props.code.operationHistory.length"
          class="timeline"
        >
          <li
            v-for="(entry, index) in props.code.operationHistory"
            :key="`${entry.action}-${entry.createdAt}-${index}`"
          >
            <i aria-hidden="true" />
            <div>
              <strong>{{ entry.action }}</strong>
              <span>{{ entry.operator }} · {{ formatDateTime(entry.createdAt) }}</span>
              <p v-if="entry.reason">{{ entry.reason }}</p>
            </div>
          </li>
        </ol>
        <p v-else class="empty-history">暂无额外管理操作。</p>
      </section>

      <div
        v-if="
          props.code.status === 'unused' || props.code.status === 'expired'
        "
        class="drawer-actions"
      >
        <BaseButton
          variant="secondary"
          @click="emit('extend', props.code)"
        >
          <CalendarPlus :size="16" aria-hidden="true" />
          延长有效期
        </BaseButton>
        <BaseButton
          v-if="props.code.status === 'unused'"
          variant="danger"
          @click="emit('disable', props.code)"
        >
          <ShieldOff :size="16" aria-hidden="true" />
          失效兑换码
        </BaseButton>
      </div>
    </div>
  </BaseDrawer>
</template>

<style scoped>
.drawer-state {
  min-height: 360px;
  color: var(--color-text-muted, #68716f);
  place-content: center;
}

.drawer-content {
  display: grid;
  gap: 22px;
  min-height: 100%;
}

.identity-panel {
  display: grid;
  grid-template-columns: 50px minmax(0, 1fr) auto;
  gap: 14px;
  align-items: center;
  padding: 18px;
  background: #f2f6f5;
  border-left: 3px solid var(--color-primary, #236c62);
  border-radius: 5px 8px 8px 5px;
}

.ticket-mark {
  display: grid;
  width: 50px;
  height: 50px;
  color: var(--color-primary, #236c62);
  background: #fff;
  border: 1px solid #d8e6e2;
  border-radius: 7px;
  place-items: center;
}

.identity {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.identity span,
.identity small {
  color: var(--color-text-muted, #68716f);
  font-size: 11px;
}

.identity strong {
  overflow-wrap: anywhere;
  font-size: 18px;
}

.sensitive-actions {
  display: flex;
  gap: 10px;
  align-items: center;
  min-height: 48px;
  padding: 9px 12px;
  border: 1px dashed var(--color-border, #dce1df);
  border-radius: 7px;
}

.sensitive-actions p {
  margin: 0 0 0 auto;
  color: var(--color-text-muted, #68716f);
  font-size: 11px;
}

.detail-section {
  padding-top: 4px;
}

.detail-section header {
  display: flex;
  gap: 9px;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 9px;
  border-bottom: 1px solid var(--color-border-soft, #edf0ef);
}

.detail-section header span,
.detail-section header svg {
  color: var(--color-primary, #236c62);
  font-family: var(--font-mono, monospace);
  font-size: 11px;
  font-weight: 800;
}

.detail-section h3 {
  margin: 0;
  font-size: 13px;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  background: var(--color-border-soft, #edf0ef);
  border: 1px solid var(--color-border-soft, #edf0ef);
  border-radius: 7px;
}

.detail-grid > div {
  display: grid;
  gap: 6px;
  min-height: 64px;
  padding: 12px 14px;
  background: #fff;
}

.detail-grid__wide {
  grid-column: 1 / -1;
}

dt {
  color: var(--color-text-muted, #68716f);
  font-size: 10px;
}

dd {
  margin: 0;
  color: var(--color-text, #1b1f1f);
  font-size: 12px;
  overflow-wrap: anywhere;
}

.timeline {
  display: grid;
  gap: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}

.timeline li {
  position: relative;
  display: grid;
  grid-template-columns: 18px 1fr;
  gap: 9px;
  min-height: 62px;
}

.timeline li:not(:last-child)::before {
  position: absolute;
  top: 14px;
  bottom: -3px;
  left: 5px;
  width: 1px;
  background: #dbe2e0;
  content: '';
}

.timeline i {
  z-index: 1;
  width: 11px;
  height: 11px;
  margin-top: 3px;
  background: var(--color-primary, #236c62);
  border: 3px solid #e2eeeb;
  border-radius: 50%;
  box-sizing: content-box;
}

.timeline li > div {
  display: grid;
  gap: 3px;
  align-content: start;
}

.timeline strong {
  font-size: 12px;
}

.timeline span,
.empty-history {
  color: var(--color-text-muted, #68716f);
  font-size: 10px;
}

.timeline p {
  margin: 2px 0 0;
  color: var(--color-text, #1b1f1f);
  font-size: 11px;
}

.drawer-actions {
  position: sticky;
  bottom: -20px;
  display: flex;
  justify-content: flex-end;
  gap: 9px;
  margin-top: auto;
  padding: 14px 0 2px;
  background: #fff;
  border-top: 1px solid var(--color-border-soft, #edf0ef);
}

@media (max-width: 600px) {
  .identity-panel {
    grid-template-columns: 42px 1fr;
  }

  .identity-panel > :last-child {
    grid-column: 1 / -1;
  }

  .ticket-mark {
    width: 42px;
    height: 42px;
  }

  .sensitive-actions {
    align-items: flex-start;
    flex-direction: column;
  }

  .sensitive-actions p {
    margin-left: 0;
  }

  .detail-grid {
    grid-template-columns: 1fr;
  }

  .detail-grid__wide {
    grid-column: auto;
  }
}
</style>
