<script setup lang="ts">
import {
  AlertTriangle,
  ChevronDown,
  LogOut,
  Menu,
  RotateCcw,
  Search,
  UserRound,
} from '@lucide/vue'
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import IconButton from '@/components/base/IconButton.vue'
import { ROLE_LABELS } from '@/config'
import { useToast } from '@/composables/useToast'
import { demoApi } from '@/services/demo'
import { useAuthStore } from '@/stores/auth'

const emit = defineEmits<{ openNavigation: [] }>()
const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const toast = useToast()
const search = ref('')
const isLoggingOut = ref(false)
const isResettingDemo = ref(false)
const mockEnabled = import.meta.env.VITE_ENABLE_MOCKS === 'true'

const pageTitle = computed(() => String(route.meta.title ?? '映研管理端'))
const initials = computed(() => auth.session?.name.slice(0, 1) ?? '管')

function submitSearch(): void {
  const keyword = search.value.trim()
  if (!keyword) return

  const destination = /^(RT|TICKET|工单)/i.test(keyword)
    ? '/manage/retouch-tickets'
    : /^(GT|TASK|任务)/i.test(keyword)
      ? '/manage/generation-tasks'
      : '/manage/users'

  void router.push({ path: destination, query: { keyword } })
}

async function logout(): Promise<void> {
  if (isLoggingOut.value) return
  isLoggingOut.value = true
  try {
    await auth.logout()
    await router.replace('/manage/login')
  } catch {
    toast.error({
      title: '退出登录未完成',
      message: '请检查网络后重试。',
    })
  } finally {
    isLoggingOut.value = false
  }
}

async function resetDemo(): Promise<void> {
  if (isResettingDemo.value) return
  isResettingDemo.value = true
  try {
    await demoApi.reset()
    window.location.assign('/manage/login?reset=1')
  } catch {
    toast.error({
      title: '演示数据重置失败',
      message: '请检查本地 Mock 服务后重试。',
    })
    isResettingDemo.value = false
  }
}
</script>

<template>
  <header class="app-topbar">
    <div class="app-topbar__leading">
      <IconButton
        class="mobile-only"
        label="打开导航"
        @click="emit('openNavigation')"
      >
        <Menu :size="21" />
      </IconButton>
      <h1>{{ pageTitle }}</h1>
    </div>

    <form
      v-if="auth.hasPermission('platform:manage')"
      class="global-search desktop-only"
      role="search"
      @submit.prevent="submitSearch"
    >
      <Search :size="16" aria-hidden="true" />
      <label class="sr-only" for="global-search">全局搜索</label>
      <input
        id="global-search"
        v-model="search"
        type="search"
        placeholder="搜索用户、任务或工单"
      />
    </form>

    <div class="app-topbar__actions">
      <RouterLink
        v-if="auth.hasPermission('platform:manage')"
        class="health-alert desktop-only"
        to="/manage/ai-providers"
        aria-label="查看 AI 服务状态"
      >
        <AlertTriangle :size="16" />
        <span>服务状态</span>
      </RouterLink>

      <details class="user-menu">
        <summary>
          <span class="user-menu__avatar">{{ initials }}</span>
          <span class="user-menu__identity desktop-only">
            <strong>{{ auth.session?.name }}</strong>
            <small v-if="auth.session">{{ ROLE_LABELS[auth.session.role] }}</small>
          </span>
          <ChevronDown class="desktop-only" :size="15" />
        </summary>
        <div class="user-menu__popover">
          <div class="user-menu__summary">
            <UserRound :size="18" />
            <span>
              <strong>{{ auth.session?.name }}</strong>
              <small>{{ auth.session?.email }}</small>
            </span>
          </div>
          <button type="button" :disabled="isLoggingOut" @click="logout">
            <LogOut :size="17" />
            {{ isLoggingOut ? '正在退出…' : '退出登录' }}
          </button>
          <button
            v-if="mockEnabled"
            class="reset-demo"
            type="button"
            :disabled="isResettingDemo"
            @click="resetDemo"
          >
            <RotateCcw :size="17" />
            {{ isResettingDemo ? '正在重置…' : '重置演示数据' }}
          </button>
        </div>
      </details>
    </div>
  </header>
</template>

<style scoped>
.app-topbar {
  position: sticky;
  z-index: 30;
  top: 0;
  display: grid;
  height: var(--topbar-height);
  align-items: center;
  padding: 0 24px;
  border-bottom: 1px solid var(--border);
  background: rgb(255 255 255 / 92%);
  backdrop-filter: blur(14px);
  grid-template-columns: minmax(160px, 1fr) minmax(260px, 440px) minmax(160px, 1fr);
}

.app-topbar__leading {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
}

.app-topbar h1 {
  overflow: hidden;
  font-size: 15px;
  font-weight: 750;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.global-search {
  display: flex;
  height: 38px;
  align-items: center;
  gap: 8px;
  padding: 0 11px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface-soft);
  color: var(--ink-muted);
}

.global-search:focus-within {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px rgb(35 108 98 / 10%);
}

.global-search input {
  width: 100%;
  border: 0;
  outline: 0;
  background: transparent;
  font-size: 13px;
}

.app-topbar__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}

.health-alert {
  display: inline-flex;
  height: 36px;
  align-items: center;
  gap: 7px;
  padding: 0 10px;
  border-radius: var(--radius-sm);
  color: var(--warning);
  font-size: 12px;
  font-weight: 700;
}

.health-alert:hover {
  background: var(--warning-soft);
}

.user-menu {
  position: relative;
}

.user-menu summary {
  display: flex;
  min-height: 44px;
  align-items: center;
  gap: 9px;
  border-radius: var(--radius-sm);
  list-style: none;
  transition:
    background var(--motion-fast),
    transform var(--motion-fast);
}

.user-menu summary::-webkit-details-marker {
  display: none;
}

.user-menu__avatar {
  display: grid;
  width: 34px;
  height: 34px;
  border-radius: var(--radius-sm);
  background: var(--sidebar);
  color: #fff;
  font-family: var(--font-display);
  place-items: center;
}

.user-menu__identity strong,
.user-menu__identity small {
  display: block;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-menu__identity strong {
  font-size: 12px;
}

.user-menu__identity small {
  color: var(--ink-muted);
  font-size: 10px;
}

.user-menu__popover {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  width: 250px;
  padding: 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-md);
  transform-origin: top right;
}

.user-menu[open] .user-menu__popover {
  animation: menu-in var(--motion-normal) var(--ease-out) both;
}

.user-menu__summary {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px;
  border-bottom: 1px solid var(--border);
}

.user-menu__summary strong,
.user-menu__summary small {
  display: block;
}

.user-menu__summary strong {
  font-size: 13px;
}

.user-menu__summary small {
  margin-top: 2px;
  color: var(--ink-muted);
  font-size: 11px;
}

.user-menu__popover button {
  display: flex;
  width: 100%;
  min-height: 44px;
  align-items: center;
  gap: 9px;
  padding: 0 10px;
  margin-top: 6px;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--danger);
  font-size: 13px;
  font-weight: 700;
}

.user-menu__popover button:hover {
  background: var(--danger-soft);
}

.user-menu__popover .reset-demo {
  color: var(--ink-muted);
}

.user-menu__popover .reset-demo:hover {
  background: var(--surface-soft);
}

@media (max-width: 960px) {
  .app-topbar {
    padding: 0 14px;
    box-shadow: 0 8px 24px rgb(22 30 28 / 5%);
    grid-template-columns: 1fr auto;
  }

  .app-topbar h1 {
    font-size: 14px;
  }

  .user-menu summary:active {
    background: var(--surface-soft);
    transform: scale(0.96);
  }

  .user-menu__popover {
    position: fixed;
    top: calc(var(--topbar-height) + 8px);
    right: 14px;
    width: min(300px, calc(100vw - 28px));
  }
}

@keyframes menu-in {
  from {
    opacity: 0;
    transform: translateY(-6px) scale(0.98);
  }
}
</style>
