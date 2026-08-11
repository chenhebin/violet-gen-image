<script setup lang="ts">
import { computed } from 'vue'
import { DoorOpen, Laptop, Minus, Plus, RotateCcw, X } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { useAuthStore } from '@/stores/auth'
import { useEntitlementStore } from '@/stores/entitlement'
import type { LedgerEntry } from '@/types/domain'

defineProps<{ open: boolean }>()
defineEmits<{ close: []; logout: [] }>()
const auth = useAuthStore()
const entitlement = useEntitlementStore()

const rows = computed(() => entitlement.ledger.slice(0, 16))
const devicePlatform = navigator.platform || '此设备'

function label(entry: LedgerEntry): string {
  return {
    redemption: '兑换',
    reserve: '次数预占',
    release: '释放预占',
    refund: '次数退回',
    adjustment: '人工调整',
  }[entry.type]
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="open" class="drawer-layer">
        <button class="drawer-scrim" aria-label="关闭账户中心" @click="$emit('close')" />
        <aside role="dialog" aria-modal="true" aria-labelledby="account-heading">
          <header>
            <div>
              <p>账户中心</p>
              <h2 id="account-heading">{{ auth.user?.email }}</h2>
            </div>
            <button class="icon-button" aria-label="关闭" @click="$emit('close')">
              <X :size="20" />
            </button>
          </header>

          <div class="drawer-content">
            <section class="balance-panel">
              <span>当前可用</span>
              <strong>{{ entitlement.balance }} <small>次</small></strong>
              <p>生成 1 张图片消耗 1 次，失败部分会自动退回。</p>
            </section>

            <section>
              <div class="section-heading">
                <h3>次数流水</h3>
                <span>{{ rows.length }} 条</span>
              </div>
              <div v-if="rows.length" class="ledger">
                <article v-for="entry in rows" :key="entry.id">
                  <span class="ledger-icon" :class="entry.type">
                    <Plus v-if="entry.amount > 0" :size="16" />
                    <Minus v-else-if="entry.amount < 0" :size="16" />
                    <RotateCcw v-else :size="16" />
                  </span>
                  <div>
                    <strong>{{ label(entry) }}</strong>
                    <p>{{ entry.description }}</p>
                    <time>{{ formatDate(entry.createdAt) }}</time>
                  </div>
                  <b :class="{ positive: entry.amount > 0 }">
                    {{ entry.amount > 0 ? '+' : '' }}{{ entry.amount }}
                  </b>
                </article>
              </div>
              <p v-else class="empty">还没有次数记录。</p>
            </section>

            <section class="device">
              <Laptop :size="19" />
              <div>
                <h3>当前设备</h3>
                <p>浏览器会话 · {{ devicePlatform }}</p>
              </div>
            </section>
          </div>

          <footer>
            <p>素材仅用于当前创作演示。接入正式服务前请配置清理周期与隐私说明。</p>
            <BaseButton variant="secondary" @click="$emit('logout')">
              <template #icon><DoorOpen :size="18" /></template>
              退出登录
            </BaseButton>
          </footer>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.drawer-layer {
  position: fixed;
  z-index: 75;
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
  width: min(460px, 100%);
  height: 100%;
  grid-template-rows: auto 1fr auto;
  border-left: 1px solid var(--border);
  background: var(--surface);
  box-shadow: var(--shadow-md);
}

header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 22px 24px;
  border-bottom: 1px solid var(--border);
}

header p {
  color: var(--ink-muted);
  font-size: 12px;
}

header h2 {
  margin-top: 2px;
  overflow: hidden;
  font-size: 17px;
  text-overflow: ellipsis;
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
  overflow: auto;
  padding: 22px 24px;
}

.balance-panel {
  padding: 22px;
  border: 1px solid #cfe1dd;
  border-radius: var(--radius-md);
  background: var(--primary-soft);
}

.balance-panel span {
  color: var(--primary);
  font-size: 12px;
  font-weight: 700;
}

.balance-panel strong {
  display: block;
  margin: 5px 0;
  color: var(--ink);
  font-size: 34px;
}

.balance-panel small {
  font-size: 14px;
}

.balance-panel p,
.device p,
footer p {
  color: var(--ink-muted);
  font-size: 12px;
}

.drawer-content > section + section {
  margin-top: 28px;
}

.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-heading h3,
.device h3 {
  font-size: 14px;
}

.section-heading span {
  color: var(--ink-faint);
  font-size: 12px;
}

.ledger {
  margin-top: 10px;
  border-top: 1px solid var(--border);
}

.ledger article {
  display: grid;
  grid-template-columns: 32px 1fr auto;
  gap: 10px;
  align-items: start;
  padding: 14px 0;
  border-bottom: 1px solid var(--border);
}

.ledger-icon {
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border-radius: 50%;
  background: var(--surface-soft);
}

.ledger-icon.redemption,
.ledger-icon.release,
.ledger-icon.refund {
  background: var(--primary-soft);
  color: var(--primary);
}

.ledger article strong {
  font-size: 13px;
}

.ledger article p,
.ledger article time {
  display: block;
  color: var(--ink-muted);
  font-size: 11px;
}

.ledger article b {
  color: var(--ink-muted);
  font-size: 13px;
}

.ledger article b.positive {
  color: var(--success);
}

.device {
  display: grid;
  grid-template-columns: 24px 1fr;
  gap: 10px;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.empty {
  padding: 24px 0;
  color: var(--ink-muted);
  font-size: 13px;
}

footer {
  display: grid;
  gap: 14px;
  padding: 18px 24px;
  border-top: 1px solid var(--border);
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
</style>
