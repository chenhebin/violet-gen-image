import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './styles/base.css'

async function enableMocks(): Promise<void> {
  const mocksEnabled = import.meta.env.VITE_ENABLE_MOCKS === 'true'
  document.documentElement.dataset.apiMode = mocksEnabled ? 'mock' : 'backend'

  if (import.meta.env.DEV) {
    console.info(`[映研管理端] API 模式：${mocksEnabled ? 'Mock' : '真实后端'}`)
  }

  if (!mocksEnabled) return
  const { worker } = await import('./mocks/browser')
  await worker.start({
    onUnhandledRequest: 'bypass',
    serviceWorker: {
      url: '/mockServiceWorker.js',
    },
  })
}

async function bootstrap(): Promise<void> {
  await enableMocks()

  const app = createApp(App)
  app.use(createPinia())
  app.use(router)
  app.mount('#app')
}

void bootstrap()
