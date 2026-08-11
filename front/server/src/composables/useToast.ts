import { readonly, ref } from 'vue'

export type ToastTone = 'success' | 'danger' | 'warning' | 'info'

export interface ToastMessage {
  id: number
  tone: ToastTone
  title: string
  message?: string
  duration: number
}

interface ToastInput {
  title: string
  message?: string
  duration?: number
}

const DEFAULT_DURATION = 4200
const items = ref<ToastMessage[]>([])
let nextId = 1

function dismiss(id: number): void {
  items.value = items.value.filter((item) => item.id !== id)
}

function push(tone: ToastTone, input: ToastInput | string): number {
  const normalized = typeof input === 'string' ? { title: input } : input
  const id = nextId++
  const duration = normalized.duration ?? DEFAULT_DURATION

  items.value.push({ id, tone, duration, ...normalized })
  if (duration > 0) window.setTimeout(() => dismiss(id), duration)
  return id
}

export function useToast() {
  return {
    items: readonly(items),
    dismiss,
    success: (input: ToastInput | string) => push('success', input),
    error: (input: ToastInput | string) => push('danger', input),
    warning: (input: ToastInput | string) => push('warning', input),
    info: (input: ToastInput | string) => push('info', input),
  }
}
