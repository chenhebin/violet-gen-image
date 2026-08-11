import { createRouter, createWebHistory } from 'vue-router'
import { onAuthenticationFailure } from '@/services/http'
import { useAuthStore } from '@/stores/auth'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/app/create',
    },
    {
      path: '/auth',
      name: 'auth',
      component: () => import('@/views/AuthView.vue'),
      meta: { public: true },
    },
    {
      path: '/app',
      component: () => import('@/layouts/AppShell.vue'),
      children: [
        {
          path: '',
          redirect: '/app/create',
        },
        {
          path: 'create',
          name: 'create',
          component: () => import('@/views/CreateView.vue'),
        },
        {
          path: 'tasks',
          name: 'tasks',
          component: () => import('@/views/TasksView.vue'),
        },
        {
          path: 'tasks/:taskId',
          name: 'task-detail',
          component: () => import('@/views/TasksView.vue'),
        },
        {
          path: 'retouch',
          name: 'retouch-records',
          component: () => import('@/views/RetouchRecordsView.vue'),
        },
        {
          path: 'retouch/:ticketId',
          name: 'retouch-detail',
          component: () => import('@/views/RetouchRecordsView.vue'),
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/app/create',
    },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

onAuthenticationFailure(() => {
  const auth = useAuthStore()
  auth.invalidateSession()
  const current = router.currentRoute.value
  if (current.name === 'auth') return
  void router.replace({
    name: 'auth',
    query: { redirect: current.fullPath },
  })
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  await auth.restore()

  if (!to.meta.public && !auth.isAuthenticated) {
    return {
      name: 'auth',
      query: { redirect: to.fullPath },
    }
  }

  if (to.name === 'auth' && auth.isAuthenticated) {
    return { name: 'create' }
  }

  return true
})
