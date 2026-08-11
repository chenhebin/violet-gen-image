<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import {
  Coins,
  ImagePlus,
  ListChecks,
  Menu,
  Scissors,
  Ticket,
} from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'
import { useEntitlementStore } from '@/stores/entitlement'

defineEmits<{
  redeem: []
  account: []
}>()

const auth = useAuthStore()
const entitlement = useEntitlementStore()

const initial = computed(() => auth.user?.email.slice(0, 1).toUpperCase() ?? 'Y')
</script>

<template>
  <header class="app-header">
    <div class="header-inner">
      <RouterLink class="brand" to="/app/create" aria-label="映研开始创作">
        <span class="brand-seal">映</span>
        <span class="brand-name">映研</span>
      </RouterLink>

      <nav aria-label="主要导航">
        <RouterLink to="/app/create">
          <ImagePlus :size="20" aria-hidden="true" />
          <span>开始创作</span>
        </RouterLink>
        <RouterLink to="/app/tasks">
          <ListChecks :size="20" aria-hidden="true" />
          <span>任务记录</span>
        </RouterLink>
        <RouterLink to="/app/retouch">
          <Scissors :size="20" aria-hidden="true" />
          <span>人工修图</span>
        </RouterLink>
      </nav>

      <div class="header-actions">
        <button
          class="balance-button"
          aria-label="查看账户与剩余次数"
          @click="$emit('account')"
        >
          <Coins :size="18" />
          <span class="balance-label">剩余次数</span>
          <strong>{{ entitlement.balance }}</strong>
        </button>
        <button
          class="redeem-button"
          aria-label="兑换码"
          @click="$emit('redeem')"
        >
          <Ticket :size="18" />
          <span>兑换码</span>
        </button>
        <button
          class="user-button"
          :aria-label="`打开 ${auth.user?.email ?? ''} 的账户菜单`"
          @click="$emit('account')"
        >
          <span>{{ initial }}</span>
          <Menu :size="18" />
        </button>
      </div>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  position: fixed;
  z-index: 50;
  top: 0;
  right: 0;
  left: 0;
  height: var(--header-height);
  border-bottom: 1px solid var(--border);
  background: rgb(255 255 255 / 94%);
  backdrop-filter: blur(14px);
}

.header-inner {
  display: grid;
  width: 100%;
  height: 100%;
  grid-template-columns: minmax(150px, 1fr) auto minmax(150px, 1fr);
  align-items: center;
  gap: 24px;
  padding: 0 20px;
}

.brand {
  display: inline-flex;
  width: fit-content;
  align-items: center;
  gap: 10px;
  font-weight: 760;
}

.brand-seal {
  display: grid;
  width: 30px;
  height: 34px;
  place-items: center;
  border-radius: 5px 5px 7px 7px;
  background: var(--ink);
  color: white;
  font-family: 'Songti SC', 'STSong', serif;
  font-size: 17px;
}

.brand-name {
  font-size: 18px;
}

nav {
  display: flex;
  height: 100%;
  align-items: center;
  gap: 28px;
}

nav a {
  position: relative;
  display: grid;
  height: 100%;
  place-items: center;
  color: var(--ink-muted);
  font-size: 14px;
  font-weight: 620;
  transition:
    color var(--motion-fast),
    transform var(--motion-fast);
}

nav a > svg {
  display: none;
}

nav a::after {
  position: absolute;
  right: 0;
  bottom: -1px;
  left: 0;
  height: 2px;
  background: transparent;
  content: '';
}

nav a.router-link-active {
  color: var(--ink);
}

nav a.router-link-active::after {
  background: var(--primary);
}

nav a:active {
  transform: translateY(1px);
}

.header-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.balance-button,
.redeem-button,
.user-button {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  gap: 8px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--ink-muted);
  font-size: 13px;
}

.balance-button,
.redeem-button {
  padding: 0 12px;
}

.balance-button {
  background: var(--primary-soft);
  color: var(--primary);
}

.balance-button strong {
  font-size: 15px;
}

.redeem-button:hover,
.user-button:hover {
  background: var(--surface-soft);
  color: var(--ink);
}

.user-button {
  padding: 0 8px;
}

.user-button > span {
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border-radius: 50%;
  background: var(--ink);
  color: white;
  font-size: 12px;
  font-weight: 700;
}

@media (max-width: 760px) {
  .app-header {
    backdrop-filter: none;
  }

  .header-inner {
    grid-template-columns: auto 1fr;
    gap: 10px;
    padding-inline: 12px;
  }

  .brand-name,
  .balance-label,
  .redeem-button span {
    display: none;
  }

  nav {
    position: fixed;
    z-index: 49;
    top: auto;
    right: 12px;
    bottom: max(8px, env(safe-area-inset-bottom));
    left: 12px;
    box-sizing: border-box;
    isolation: isolate;
    max-width: calc(100vw - 24px);
    height: var(--mobile-nav-height);
    justify-content: space-evenly;
    overflow: clip;
    gap: 0;
    padding-inline: 4px;
    border: 1px solid rgb(220 225 231 / 88%);
    border-radius: var(--radius-md);
    background: rgb(255 255 255 / 94%);
    box-shadow: 0 10px 30px rgb(23 25 29 / 13%);
    backdrop-filter: blur(18px) saturate(1.15);
  }

  nav a {
    z-index: 0;
    display: flex;
    min-width: 0;
    flex: 1;
    height: var(--mobile-nav-height);
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
    font-size: 10px;
    line-height: 1.2;
  }

  nav a > svg {
    display: block;
    transition: transform var(--motion-normal) var(--ease-out);
  }

  nav a::after {
    top: 5px;
    right: calc(50% - 22px);
    bottom: 5px;
    left: calc(50% - 22px);
    height: auto;
    border-radius: 8px;
    background: transparent;
    content: '';
    z-index: -1;
  }

  nav a.router-link-active {
    color: var(--primary);
  }

  nav a.router-link-active::after {
    background: var(--primary-soft);
  }

  nav a.router-link-active > svg {
    transform: translateY(-1px);
  }

  nav a:active > svg {
    transform: scale(0.9);
  }

  .balance-button,
  .redeem-button {
    width: 44px;
    justify-content: center;
    padding: 0;
  }

  .balance-button strong {
    font-size: 12px;
  }

  .balance-button:active,
  .redeem-button:active,
  .user-button:active {
    transform: scale(0.96);
  }
}
</style>
