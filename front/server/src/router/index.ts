import { createRouter, createWebHistory } from 'vue-router'
import { onAuthenticationFailure } from '@/services/http'
import type { AdminPermission } from '@/types/domain'
import { useAuthStore } from '@/stores/auth'
import AdminLayout from '@/layouts/AdminLayout.vue'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    public?: boolean
    requiresAuth?: boolean
    permission?: AdminPermission
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    {
      path: '/',
      redirect: '/manage/dashboard',
    },
    {
      path: '/manage/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { title: '管理员登录', public: true },
    },
    {
      path: '/manage',
      component: AdminLayout,
      meta: { requiresAuth: true },
      children: [
        { path: '', redirect: '/manage/dashboard' },
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue'),
          meta: { title: '运营概览' },
        },
        {
          path: 'redemption-codes',
          name: 'redemption-codes',
          component: () => import('@/views/RedemptionCodesView.vue'),
          meta: { title: '兑换码', permission: 'platform:manage' },
        },
        {
          path: 'redemption-batches',
          name: 'redemption-batches',
          component: () => import('@/views/RedemptionBatchesView.vue'),
          meta: { title: '生成批次', permission: 'platform:manage' },
        },
        {
          path: 'ai-providers',
          name: 'ai-providers',
          component: () => import('@/views/AiProvidersView.vue'),
          meta: { title: 'AI 服务', permission: 'platform:manage' },
        },
        {
          path: 'retouch-tickets',
          name: 'retouch-tickets',
          component: () => import('@/views/RetouchTicketsView.vue'),
          meta: { title: '人工工单', permission: 'retouch:manage' },
        },
        {
          path: 'users',
          name: 'users',
          component: () => import('@/views/UsersView.vue'),
          meta: { title: '用户与次数', permission: 'platform:manage' },
        },
        {
          path: 'generation-tasks',
          name: 'generation-tasks',
          component: () => import('@/views/GenerationTasksView.vue'),
          meta: { title: '生成任务', permission: 'platform:manage' },
        },
        {
          path: 'assets',
          name: 'assets',
          component: () => import('@/views/AssetsView.vue'),
          meta: { title: '图片资产', permission: 'platform:manage' },
        },
        {
          path: 'audit-logs',
          name: 'audit-logs',
          component: () => import('@/views/AuditLogsView.vue'),
          meta: { title: '审计日志', permission: 'platform:manage' },
        },
        {
          path: 'forbidden',
          name: 'forbidden',
          component: () => import('@/views/ForbiddenView.vue'),
          meta: { title: '无权访问' },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFoundView.vue'),
      meta: { title: '页面不存在' },
    },
  ],
})

let restoreAttempted = false

onAuthenticationFailure(() => {
  const auth = useAuthStore()
  auth.invalidateSession()
  const current = router.currentRoute.value
  if (current.name === 'login') return
  void router.replace({
    name: 'login',
    query: { redirect: current.fullPath },
  })
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()

  if (!restoreAttempted) {
    restoreAttempted = true
    await auth.restoreSession()
  }

  if (to.name === 'login') {
    return auth.isAuthenticated ? { name: 'dashboard' } : true
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return {
      name: 'login',
      query: { redirect: to.fullPath },
    }
  }

  if (to.meta.permission && !auth.hasPermission(to.meta.permission)) {
    return { name: 'forbidden' }
  }

  return true
})

router.afterEach((to) => {
  document.title = `${String(to.meta.title ?? '管理端')} · 映研`
})

export default router
