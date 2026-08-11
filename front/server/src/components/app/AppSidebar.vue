<script setup lang="ts">
import {
  Bot,
  Images,
  LayoutDashboard,
  Layers3,
  Paintbrush,
  ScrollText,
  TicketCheck,
  Users,
  X,
} from '@lucide/vue'
import { computed, type Component } from 'vue'
import { RouterLink } from 'vue-router'
import IconButton from '@/components/base/IconButton.vue'
import { useAuthStore } from '@/stores/auth'
import type { AdminPermission } from '@/types/domain'

interface NavigationItem {
  label: string
  to: string
  icon: Component
  permission?: AdminPermission
}

interface NavigationGroup {
  label: string
  items: NavigationItem[]
}

defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()
const auth = useAuthStore()

const groups: NavigationGroup[] = [
  {
    label: '运营',
    items: [
      { label: '运营概览', to: '/manage/dashboard', icon: LayoutDashboard },
      {
        label: '兑换码',
        to: '/manage/redemption-codes',
        icon: TicketCheck,
        permission: 'platform:manage',
      },
      {
        label: '生成批次',
        to: '/manage/redemption-batches',
        icon: Layers3,
        permission: 'platform:manage',
      },
      {
        label: 'AI 服务',
        to: '/manage/ai-providers',
        icon: Bot,
        permission: 'platform:manage',
      },
      {
        label: '人工工单',
        to: '/manage/retouch-tickets',
        icon: Paintbrush,
        permission: 'retouch:manage',
      },
    ],
  },
  {
    label: '平台数据',
    items: [
      {
        label: '用户与次数',
        to: '/manage/users',
        icon: Users,
        permission: 'platform:manage',
      },
      {
        label: '生成任务',
        to: '/manage/generation-tasks',
        icon: Images,
        permission: 'platform:manage',
      },
      {
        label: '图片资产',
        to: '/manage/assets',
        icon: Images,
        permission: 'platform:manage',
      },
      {
        label: '审计日志',
        to: '/manage/audit-logs',
        icon: ScrollText,
        permission: 'platform:manage',
      },
    ],
  },
]

const visibleGroups = computed(() =>
  groups
    .map((group) => ({
      ...group,
      items: group.items.filter(
        (item) => !item.permission || auth.hasPermission(item.permission),
      ),
    }))
    .filter((group) => group.items.length > 0),
)
</script>

<template>
  <Transition name="sidebar-scrim">
    <button
      v-if="open"
      class="sidebar-scrim"
      type="button"
      aria-label="关闭导航"
      @click="emit('close')"
    ></button>
  </Transition>
  <aside class="app-sidebar" :class="{ 'app-sidebar--open': open }">
    <header class="app-sidebar__brand">
      <RouterLink to="/manage/dashboard" class="brand" @click="emit('close')">
        <span class="brand__mark" aria-hidden="true">映</span>
        <span>
          <strong>映研</strong>
          <small>OPERATIONS</small>
        </span>
      </RouterLink>
      <IconButton
        class="app-sidebar__close"
        label="关闭导航"
        tone="dark"
        @click="emit('close')"
      >
        <X :size="20" />
      </IconButton>
    </header>

    <nav class="app-sidebar__nav" aria-label="管理端主导航">
      <section v-for="group in visibleGroups" :key="group.label">
        <h2>{{ group.label }}</h2>
        <RouterLink
          v-for="item in group.items"
          :key="item.to"
          :to="item.to"
          class="nav-item"
          @click="emit('close')"
        >
          <component :is="item.icon" :size="18" aria-hidden="true" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </section>
    </nav>

    <footer class="app-sidebar__footer">
      <span class="app-sidebar__status-dot" aria-hidden="true"></span>
      <span>
        <strong>管理服务已连接</strong>
        <small>所有操作将写入审计记录</small>
      </span>
    </footer>
  </aside>
</template>

<style scoped>
.app-sidebar {
  position: fixed;
  z-index: 50;
  inset: 0 auto 0 0;
  display: grid;
  width: var(--sidebar-width);
  height: 100dvh;
  color: rgb(255 255 255 / 78%);
  background: var(--sidebar);
  grid-template-rows: auto minmax(0, 1fr) auto;
}

.app-sidebar__brand {
  display: flex;
  height: var(--topbar-height);
  align-items: center;
  justify-content: space-between;
  padding: 0 16px 0 18px;
  border-bottom: 1px solid var(--sidebar-line);
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 11px;
}

.brand__mark {
  display: grid;
  width: 32px;
  height: 32px;
  border: 1px solid rgb(255 255 255 / 28%);
  border-radius: var(--radius-sm);
  color: #fff;
  font-family: var(--font-display);
  font-size: 17px;
  place-items: center;
}

.brand strong,
.brand small {
  display: block;
}

.brand strong {
  color: #fff;
  font-family: var(--font-display);
  font-size: 17px;
  line-height: 1.15;
}

.brand small {
  margin-top: 3px;
  color: rgb(255 255 255 / 42%);
  font-size: 8px;
  font-weight: 750;
  letter-spacing: 0.13em;
}

.app-sidebar__close {
  display: none;
}

.app-sidebar__nav {
  min-height: 0;
  overflow-y: auto;
  padding: 22px 12px;
}

.app-sidebar__nav section + section {
  margin-top: 24px;
}

.app-sidebar__nav h2 {
  padding: 0 10px;
  margin-bottom: 7px;
  color: rgb(255 255 255 / 36%);
  font-size: 10px;
  font-weight: 750;
  letter-spacing: 0.1em;
}

.nav-item {
  position: relative;
  display: flex;
  min-height: 42px;
  align-items: center;
  gap: 11px;
  padding: 0 10px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 600;
  transition:
    background var(--motion-fast),
    color var(--motion-fast),
    transform var(--motion-fast);
}

.nav-item:hover {
  background: rgb(255 255 255 / 6%);
  color: #fff;
}

.nav-item.router-link-active {
  background: rgb(255 255 255 / 10%);
  color: #fff;
}

.nav-item.router-link-active::before {
  position: absolute;
  left: -12px;
  width: 3px;
  height: 22px;
  border-radius: 0 var(--radius-pill) var(--radius-pill) 0;
  background: #75b8a9;
  content: '';
}

.nav-item:active {
  transform: translateX(2px);
}

.app-sidebar__footer {
  display: flex;
  min-height: 72px;
  align-items: flex-start;
  gap: 9px;
  padding: 15px 18px;
  border-top: 1px solid var(--sidebar-line);
}

.app-sidebar__status-dot {
  width: 7px;
  height: 7px;
  flex: 0 0 7px;
  margin-top: 5px;
  border-radius: 50%;
  background: #75b89d;
  box-shadow: 0 0 0 4px rgb(117 184 157 / 10%);
}

.app-sidebar__footer strong,
.app-sidebar__footer small {
  display: block;
}

.app-sidebar__footer strong {
  color: rgb(255 255 255 / 76%);
  font-size: 11px;
}

.app-sidebar__footer small {
  margin-top: 3px;
  color: rgb(255 255 255 / 38%);
  font-size: 9px;
  line-height: 1.4;
}

.sidebar-scrim {
  display: none;
}

@media (max-width: 960px) {
  .app-sidebar {
    width: min(304px, calc(100vw - 24px));
    box-shadow: 24px 0 60px rgb(10 15 14 / 24%);
    transform: translateX(-100%);
    transition: transform 320ms var(--ease-out);
  }

  .app-sidebar--open {
    transform: translateX(0);
  }

  .app-sidebar__close {
    display: inline-flex;
  }

  .app-sidebar__nav {
    padding: 16px 12px;
    overscroll-behavior: contain;
  }

  .nav-item {
    min-height: 48px;
    padding-inline: 12px;
    font-size: 14px;
  }

  .app-sidebar__footer {
    padding-bottom: calc(15px + env(safe-area-inset-bottom));
  }

  .app-sidebar--open .nav-item {
    animation: nav-item-in var(--motion-normal) var(--ease-out) both;
  }

  .app-sidebar--open section:nth-child(2) .nav-item {
    animation-delay: 60ms;
  }

  .sidebar-scrim {
    position: fixed;
    z-index: 49;
    inset: 0;
    display: block;
    width: 100%;
    height: 100%;
    background: var(--scrim);
  }

  .sidebar-scrim-enter-active,
  .sidebar-scrim-leave-active {
    transition: opacity var(--motion-normal);
  }

  .sidebar-scrim-enter-from,
  .sidebar-scrim-leave-to {
    opacity: 0;
  }
}

@keyframes nav-item-in {
  from {
    opacity: 0;
    transform: translateX(-8px);
  }
}
</style>
