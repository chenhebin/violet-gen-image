<script setup lang="ts">
import { Fingerprint, ShieldCheck, ShieldX } from '@lucide/vue'
import BaseDrawer from '@/components/base/BaseDrawer.vue'
import StatusBadge from '@/components/base/StatusBadge.vue'
import { AUDIT_RESULT_LABELS, ROLE_LABELS } from '@/config'
import type { AuditEvent } from '@/types/domain'
import { formatAuditSnapshot } from '@/utils/audit'
import { formatDateTime } from '@/utils/format'

defineProps<{
  open: boolean
  event: AuditEvent | null
}>()

defineEmits<{ close: [] }>()
</script>

<template>
  <BaseDrawer
    :open="open"
    :title="event?.action ?? '审计详情'"
    :description="event ? `${event.resourceType} · ${event.resourceId}` : '管理操作追踪'"
    size="large"
    @close="$emit('close')"
  >
    <div v-if="event" class="audit-detail">
      <section class="audit-spine">
        <div>
          <span>执行结果</span>
          <StatusBadge :tone="event.result === 'success' ? 'success' : 'danger'">
            <ShieldCheck v-if="event.result === 'success'" :size="13" />
            <ShieldX v-else :size="13" />
            {{ AUDIT_RESULT_LABELS[event.result] }}
          </StatusBadge>
        </div>
        <dl>
          <div>
            <dt>操作者</dt>
            <dd>{{ event.operatorEmail }}</dd>
          </div>
          <div>
            <dt>角色</dt>
            <dd>{{ ROLE_LABELS[event.operatorRole] }}</dd>
          </div>
          <div>
            <dt>操作时间</dt>
            <dd>{{ formatDateTime(event.createdAt) }}</dd>
          </div>
        </dl>
      </section>

      <section class="request-trace">
        <Fingerprint :size="19" />
        <div>
          <span>请求追踪 ID</span>
          <strong class="mono">{{ event.requestId }}</strong>
        </div>
        <div>
          <span>来源</span>
          <strong>{{ event.ip ?? '-' }} · {{ event.device ?? '-' }}</strong>
        </div>
      </section>

      <section class="detail-section">
        <header>
          <span>Action context</span>
          <h3>操作上下文</h3>
        </header>
        <dl>
          <div>
            <dt>动作</dt>
            <dd class="mono">{{ event.action }}</dd>
          </div>
          <div>
            <dt>资源类型</dt>
            <dd>{{ event.resourceType }}</dd>
          </div>
          <div>
            <dt>资源编号</dt>
            <dd class="mono">{{ event.resourceId }}</dd>
          </div>
          <div>
            <dt>操作原因</dt>
            <dd>{{ event.reason ?? '-' }}</dd>
          </div>
        </dl>
      </section>

      <section class="snapshot-grid">
        <article>
          <header>
            <span>Before</span>
            <h3>操作前</h3>
          </header>
          <pre>{{ formatAuditSnapshot(event.before) }}</pre>
        </article>
        <article>
          <header>
            <span>After</span>
            <h3>操作后</h3>
          </header>
          <pre>{{ formatAuditSnapshot(event.after) }}</pre>
        </article>
      </section>

      <p class="security-note">
        完整兑换码、API Key、临时密码与图片签名地址不会写入审计详情。
      </p>
    </div>
  </BaseDrawer>
</template>

<style scoped>
.audit-detail {
  display: grid;
  gap: 18px;
}

.audit-spine,
.detail-section,
.request-trace,
.snapshot-grid article {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
}

.audit-spine {
  display: grid;
  grid-template-columns: 190px minmax(0, 1fr);
}

.audit-spine > div {
  display: grid;
  align-content: center;
  justify-items: start;
  gap: 9px;
  padding: 20px;
  border-right: 1px solid var(--border);
  background: var(--surface-soft);
}

.audit-spine > div > span,
.detail-section header span,
.snapshot-grid header span,
.request-trace span {
  color: var(--ink-muted);
  font-size: 10px;
  font-weight: 750;
}

.audit-spine dl {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
}

.audit-spine dl > div {
  display: grid;
  align-content: center;
  gap: 4px;
  padding: 18px;
  border-right: 1px solid var(--border);
}

.audit-spine dl > div:last-child {
  border-right: 0;
}

dt {
  color: var(--ink-muted);
  font-size: 11px;
}

dd {
  overflow: hidden;
  margin: 0;
  font-size: 12px;
  font-weight: 650;
  text-overflow: ellipsis;
}

.request-trace {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: var(--primary-soft);
}

.request-trace > svg {
  color: var(--primary);
}

.request-trace > div {
  display: grid;
  gap: 2px;
}

.request-trace strong {
  font-size: 11px;
}

.detail-section {
  padding: 18px;
}

.detail-section header,
.snapshot-grid header {
  margin-bottom: 14px;
}

.detail-section h3,
.snapshot-grid h3 {
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
  grid-template-columns: 105px minmax(0, 1fr);
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

.snapshot-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.snapshot-grid article {
  min-width: 0;
}

.snapshot-grid header {
  padding: 14px 16px;
  margin: 0;
  border-bottom: 1px solid var(--border);
  background: var(--surface-soft);
}

.snapshot-grid pre {
  min-height: 180px;
  max-height: 420px;
  padding: 14px 16px;
  margin: 0;
  overflow: auto;
  color: var(--ink-muted);
  font-family: var(--font-mono);
  font-size: 10px;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

.security-note {
  color: var(--ink-muted);
  font-size: 11px;
  text-align: center;
}

@media (max-width: 760px) {
  .audit-spine {
    grid-template-columns: 1fr;
  }

  .audit-spine > div {
    border-right: 0;
    border-bottom: 1px solid var(--border);
  }

  .audit-spine dl,
  .snapshot-grid,
  .detail-section dl {
    grid-template-columns: 1fr;
  }

  .audit-spine dl > div,
  .detail-section dl > div,
  .detail-section dl > div:nth-child(odd) {
    border-right: 0;
    border-bottom: 1px solid var(--border);
  }

  .audit-spine dl > div:last-child,
  .detail-section dl > div:last-child {
    border-bottom: 0;
  }

  .request-trace {
    grid-template-columns: auto 1fr;
  }

  .request-trace > div:last-child {
    grid-column: 2;
  }
}
</style>
