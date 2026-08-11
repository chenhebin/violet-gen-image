<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { Plus, RefreshCw } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import EmptyState from '@/components/base/EmptyState.vue'
import DashboardLedger from '@/components/app/DashboardLedger.vue'
import DashboardStatusRail from '@/components/app/DashboardStatusRail.vue'
import { useAuthStore } from '@/stores/auth'
import { useDashboardStore } from '@/stores/dashboard'

const auth = useAuthStore()
const dashboard = useDashboardStore()
let controller: AbortController | null = null

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 11) return '上午好'
  if (hour < 18) return '下午好'
  return '晚上好'
})

async function load(): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  try {
    await dashboard.loadDashboard(controller.signal)
  } catch {
    // The inline error state exposes retry without duplicating store errors.
  }
}

onMounted(() => void load())
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <div class="page dashboard">
    <header class="page__header">
      <div>
        <p class="page__eyebrow">今日运营</p>
        <h1 class="page__title">
          {{ greeting }}，{{ auth.session?.name ?? '管理员' }}
        </h1>
        <p class="page__description">
          优先处理异常、临期兑换码与等待中的人工交付。
        </p>
      </div>
      <div class="page__actions">
        <BaseButton
          variant="secondary"
          :loading="dashboard.isLoading"
          @click="load"
        >
          <template #icon><RefreshCw :size="16" /></template>
          刷新状态
        </BaseButton>
        <RouterLink
          v-if="auth.hasPermission('platform:manage')"
          to="/manage/redemption-codes?create=1"
        >
          <BaseButton>
            <template #icon><Plus :size="16" /></template>
            生成兑换码
          </BaseButton>
        </RouterLink>
      </div>
    </header>

    <div v-if="dashboard.isLoading && !dashboard.data" class="dashboard-loading">
      <span></span>
      <span></span>
      <span></span>
    </div>

    <EmptyState
      v-else-if="dashboard.error && !dashboard.data"
      title="运营数据加载失败"
      :description="`${dashboard.error.message} 请检查服务状态后重试。`"
    >
      <template #action>
        <BaseButton @click="load">重新加载</BaseButton>
      </template>
    </EmptyState>

    <template v-else-if="dashboard.data">
      <DashboardStatusRail
        :metrics="dashboard.data.metrics"
        :current-models="dashboard.data.currentModels"
        :show-models="auth.hasPermission('platform:manage')"
      />
      <DashboardLedger
        :alerts="dashboard.data.alerts"
        :pending-tickets="dashboard.data.pendingTickets"
        :recent-batches="dashboard.data.recentBatches"
        :show-platform-data="auth.hasPermission('platform:manage')"
      />
    </template>
  </div>
</template>

<style scoped>
.dashboard {
  display: grid;
  align-content: start;
  gap: 18px;
}

.dashboard > .page__header {
  margin-bottom: 0;
}

.dashboard-loading {
  display: grid;
  padding: 24px;
  border-radius: var(--radius-md);
  background: var(--sidebar);
  grid-template-columns: repeat(3, 1fr);
  gap: 22px;
}

.dashboard-loading span {
  height: 90px;
  border-radius: var(--radius-sm);
  background: linear-gradient(
    90deg,
    rgb(255 255 255 / 4%),
    rgb(255 255 255 / 10%),
    rgb(255 255 255 / 4%)
  );
  background-size: 200% 100%;
  animation: dashboard-shimmer 1.2s linear infinite;
}

@keyframes dashboard-shimmer {
  to {
    background-position: -200% 0;
  }
}
</style>
