import { readonly, ref } from 'vue'
import { UI_TIMING } from '@/config'
import { createId } from '@/utils/id'

export interface ToastItem {
  id: string
  title: string
  message?: string
  tone: 'success' | 'error' | 'info'
}

const items = ref<ToastItem[]>([])

function remove(id: string): void {
  items.value = items.value.filter((item) => item.id !== id)
}

function clear(): void {
  items.value = []
}

function show(toast: Omit<ToastItem, 'id'>): void {
  const id = createId('toast')
  items.value.push({ id, ...toast })
  window.setTimeout(() => remove(id), UI_TIMING.toastDurationMs)
}

export function useToast() {
  return {
    items: readonly(items),
    success(title: string, message?: string) {
      show({ title, message, tone: 'success' })
    },
    error(title: string, message?: string) {
      show({ title, message, tone: 'error' })
    },
    info(title: string, message?: string) {
      show({ title, message, tone: 'info' })
    },
    remove,
    clear,
  }
}
