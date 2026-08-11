<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AccountDrawer from '@/components/app/AccountDrawer.vue'
import AppHeader from '@/components/app/AppHeader.vue'
import RedeemModal from '@/components/app/RedeemModal.vue'
import ToastViewport from '@/components/base/ToastViewport.vue'
import { useToast } from '@/composables/useToast'
import { useAuthStore } from '@/stores/auth'
import { useEntitlementStore } from '@/stores/entitlement'
import { useRetouchStore } from '@/stores/retouch'
import { useWorkspaceStore } from '@/stores/workspace'

const auth = useAuthStore()
const entitlement = useEntitlementStore()
const retouch = useRetouchStore()
const workspace = useWorkspaceStore()
const route = useRoute()
const router = useRouter()
const toast = useToast()
const redeemOpen = ref(false)
const accountOpen = ref(false)
const viewKey = computed(() => {
  if (route.path.startsWith('/app/tasks')) return 'tasks'
  if (route.path.startsWith('/app/retouch')) return 'retouch'
  return String(route.name ?? route.path)
})

onMounted(async () => {
  try {
    await Promise.all([entitlement.load(), workspace.hydrateAssets()])
  } catch (caught) {
    toast.error(
      '账户信息加载失败',
      caught instanceof Error ? caught.message : '请刷新页面重试',
    )
  }
})

async function logout(): Promise<void> {
  await auth.logout()
  entitlement.reset()
  retouch.reset()
  accountOpen.value = false
  toast.info('已退出登录')
  await router.replace('/auth')
}
</script>

<template>
  <div class="app-shell">
    <AppHeader
      @redeem="redeemOpen = true"
      @account="accountOpen = true"
    />
    <main id="main-content">
      <RouterView v-slot="{ Component }">
        <Transition name="page" mode="out-in">
          <component :is="Component" :key="viewKey" />
        </Transition>
      </RouterView>
    </main>
    <RedeemModal :open="redeemOpen" @close="redeemOpen = false" />
    <AccountDrawer
      :open="accountOpen"
      @close="accountOpen = false"
      @logout="logout"
    />
    <ToastViewport />
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100dvh;
}

main {
  min-height: 100dvh;
  padding-top: var(--header-height);
}

.page-enter-active,
.page-leave-active {
  transition:
    opacity var(--motion-fast) ease,
    transform var(--motion-normal) var(--ease-out);
}

.page-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-3px);
}

@media (max-width: 760px) {
  main {
    padding-bottom: calc(
      var(--mobile-nav-height) +
        max(8px, env(safe-area-inset-bottom)) +
        12px
    );
  }

  .page-enter-from {
    transform: translateX(10px);
  }

  .page-leave-to {
    transform: translateX(-5px);
  }
}
</style>
