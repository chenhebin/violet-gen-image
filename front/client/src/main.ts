import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import './styles/base.css'

async function bootstrap(): Promise<void> {
  const mocksEnabled = import.meta.env.VITE_ENABLE_MOCKS === 'true'
  document.documentElement.dataset.apiMode = mocksEnabled ? 'mock' : 'backend'

  if (import.meta.env.DEV) {
    console.info(`[映研 Client] API 模式：${mocksEnabled ? 'Mock' : '真实后端'}`)
  }

  if (mocksEnabled) {
    const { worker } = await import('./mocks/browser')
    await worker.start({ onUnhandledRequest: 'bypass' })
  }

  const app = createApp(App)
  app.use(createPinia())
  app.use(router)
  app.mount('#app')
}

void bootstrap()
