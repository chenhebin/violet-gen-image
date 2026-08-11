<script setup lang="ts">
import { CircleOff, Coins, KeyRound, ShieldCheck } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDrawer from '@/components/base/BaseDrawer.vue'
import RetouchStatusBadge from '@/components/shared/RetouchStatusBadge.vue'
import TaskStatusBadge from '@/components/shared/TaskStatusBadge.vue'
import UserStatusBadge from '@/components/shared/UserStatusBadge.vue'
import { REDEMPTION_STATUS_LABELS } from '@/config'
import type { ManagedUserDetail } from '@/types/domain'
import { formatDateTime, formatSignedNumber } from '@/utils/format'
import type { UserAction } from './UserActionModal.vue'

defineProps<{
  open: boolean
  user: ManagedUserDetail | null
  loading: boolean
}>()

defineEmits<{
  close: []
  action: [action: UserAction]
}>()
</script>

<template>
  <BaseDrawer
    :open="open"
    :title="user?.email ?? '用户详情'"
    :description="user ? `用户编号 ${user.id}` : '正在读取用户数据'"
    size="large"
    @close="$emit('close')"
  >
    <div v-if="loading && !user" class="drawer-loading">正在读取用户详情…</div>
    <div v-else-if="user" class="user-detail">
      <section class="account-spine">
        <div>
          <span>账号状态</span>
          <UserStatusBadge :status="user.status" />
        </div>
        <dl>
          <div>
            <dt>可用次数</dt>
            <dd>{{ user.balance }}</dd>
          </div>
          <div>
            <dt>累计兑换</dt>
            <dd>{{ user.totalRedeemed }}</dd>
          </div>
          <div>
            <dt>累计消耗</dt>
            <dd>{{ user.totalConsumed }}</dd>
          </div>
          <div>
            <dt>任务 / 工单</dt>
            <dd>{{ user.taskCount }} / {{ user.ticketCount }}</dd>
          </div>
        </dl>
      </section>

      <section class="detail-section">
        <header>
          <div>
            <span>Credits ledger</span>
            <h3>次数流水</h3>
          </div>
          <b>{{ user.ledger.length }} 条</b>
        </header>
        <div class="ledger-list">
          <article v-for="entry in user.ledger" :key="entry.id">
            <div>
              <strong>{{ entry.description }}</strong>
              <span>{{ formatDateTime(entry.createdAt) }}</span>
            </div>
            <div class="ledger-amount">
              <strong
                class="mono"
                :class="entry.amount > 0 ? 'positive' : 'negative'"
              >
                {{ formatSignedNumber(entry.amount) }}
              </strong>
              <span class="mono">{{ entry.balanceBefore }} → {{ entry.balanceAfter }}</span>
            </div>
            <p v-if="entry.reason">{{ entry.reason }}</p>
          </article>
        </div>
      </section>

      <section class="detail-section">
        <header>
          <div>
            <span>Redemption history</span>
            <h3>兑换记录</h3>
          </div>
          <b>{{ user.redemptionCodes.length }} 个</b>
        </header>
        <div class="compact-list">
          <article v-for="code in user.redemptionCodes" :key="code.id">
            <div>
              <strong class="mono">{{ code.maskedCode }}</strong>
              <span>{{ code.batchName }}</span>
            </div>
            <div>
              <strong>{{ code.credits }} 次</strong>
              <span>{{ REDEMPTION_STATUS_LABELS[code.status] }}</span>
            </div>
          </article>
          <p v-if="!user.redemptionCodes.length" class="empty-copy">尚无兑换记录</p>
        </div>
      </section>

      <section class="detail-section">
        <header>
          <div>
            <span>Generation tasks</span>
            <h3>最近生成任务</h3>
          </div>
          <b>{{ user.tasks.length }} 个</b>
        </header>
        <div class="compact-list">
          <article v-for="task in user.tasks" :key="task.id">
            <div>
              <strong>{{ task.title }}</strong>
              <span>{{ formatDateTime(task.createdAt) }}</span>
            </div>
            <TaskStatusBadge :status="task.status" />
          </article>
          <p v-if="!user.tasks.length" class="empty-copy">尚无生成任务</p>
        </div>
      </section>

      <section class="detail-section">
        <header>
          <div>
            <span>Retouch tickets</span>
            <h3>人工修图记录</h3>
          </div>
          <b>{{ user.tickets.length }} 个</b>
        </header>
        <div class="compact-list">
          <article v-for="ticket in user.tickets" :key="ticket.id">
            <div>
              <strong class="mono">{{ ticket.ticketNo }}</strong>
              <span>{{ ticket.taskTitle }}</span>
            </div>
            <RetouchStatusBadge :status="ticket.status" />
          </article>
          <p v-if="!user.tickets.length" class="empty-copy">尚无人工修图工单</p>
        </div>
      </section>
    </div>

    <template v-if="user" #footer>
      <BaseButton variant="secondary" @click="$emit('action', 'reset')">
        <template #icon><KeyRound :size="16" /></template>
        重置密码
      </BaseButton>
      <BaseButton
        :variant="user.status === 'active' ? 'danger' : 'secondary'"
        @click="$emit('action', user.status === 'active' ? 'disable' : 'enable')"
      >
        <template #icon>
          <CircleOff v-if="user.status === 'active'" :size="16" />
          <ShieldCheck v-else :size="16" />
        </template>
        {{ user.status === 'active' ? '停用账号' : '恢复账号' }}
      </BaseButton>
      <BaseButton @click="$emit('action', 'adjust')">
        <template #icon><Coins :size="16" /></template>
        调整次数
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

.user-detail {
  display: grid;
  gap: 18px;
}

.account-spine,
.detail-section {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
}

.account-spine {
  display: grid;
  grid-template-columns: 190px minmax(0, 1fr);
}

.account-spine > div {
  display: grid;
  align-content: center;
  justify-items: start;
  gap: 9px;
  padding: 20px;
  border-right: 1px solid var(--border);
  background: var(--surface-soft);
}

.account-spine span,
.detail-section header span {
  color: var(--ink-muted);
  font-size: 10px;
  font-weight: 750;
}

.account-spine dl {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
}

.account-spine dl > div {
  display: grid;
  align-content: center;
  gap: 4px;
  padding: 18px;
  border-right: 1px solid var(--border);
}

.account-spine dl > div:last-child {
  border-right: 0;
}

dt {
  color: var(--ink-muted);
  font-size: 11px;
}

dd {
  margin: 0;
  font-family: var(--font-mono);
  font-size: 16px;
  font-weight: 700;
}

.detail-section > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  min-height: 58px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--surface-soft);
}

.detail-section h3 {
  margin-top: 2px;
  font-size: 14px;
}

.detail-section header b {
  color: var(--ink-muted);
  font-family: var(--font-mono);
  font-size: 11px;
}

.ledger-list,
.compact-list {
  display: grid;
}

.ledger-list article,
.compact-list article {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  padding: 13px 16px;
  border-bottom: 1px solid var(--border);
}

.ledger-list article:last-child,
.compact-list article:last-child {
  border-bottom: 0;
}

.ledger-list article > div:first-child,
.compact-list article > div {
  display: grid;
  gap: 3px;
}

.ledger-list strong,
.compact-list strong {
  font-size: 12px;
}

.ledger-list span,
.compact-list span,
.ledger-list p {
  color: var(--ink-muted);
  font-size: 10px;
}

.ledger-list p {
  grid-column: 1 / -1;
}

.ledger-amount {
  display: grid;
  justify-items: end;
  gap: 3px;
}

.ledger-amount .positive {
  color: var(--success);
}

.ledger-amount .negative {
  color: var(--danger);
}

.compact-list article {
  align-items: center;
}

.compact-list article > div:last-child {
  justify-items: end;
}

.empty-copy {
  padding: 24px 16px;
  color: var(--ink-muted);
  font-size: 12px;
  text-align: center;
}

@media (max-width: 780px) {
  .account-spine {
    grid-template-columns: 1fr;
  }

  .account-spine > div {
    border-right: 0;
    border-bottom: 1px solid var(--border);
  }

  .account-spine dl {
    grid-template-columns: repeat(2, 1fr);
  }

  .account-spine dl > div:nth-child(2) {
    border-right: 0;
  }

  .account-spine dl > div:nth-child(-n + 2) {
    border-bottom: 1px solid var(--border);
  }
}
</style>
