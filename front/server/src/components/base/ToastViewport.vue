<script setup lang="ts">
import {
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  Info,
  X,
} from '@lucide/vue'
import IconButton from './IconButton.vue'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const icons = {
  success: CheckCircle2,
  danger: AlertCircle,
  warning: AlertTriangle,
  info: Info,
} as const
</script>

<template>
  <Teleport to="body">
    <div class="toast-viewport" aria-live="polite" aria-label="操作通知">
      <TransitionGroup name="toast">
        <article
          v-for="item in toast.items.value"
          :key="item.id"
          class="toast"
          :class="`toast--${item.tone}`"
          role="status"
        >
          <component :is="icons[item.tone]" class="toast__icon" :size="19" />
          <div class="toast__content">
            <p class="toast__title">{{ item.title }}</p>
            <p v-if="item.message" class="toast__message">{{ item.message }}</p>
          </div>
          <IconButton label="关闭通知" @click="toast.dismiss(item.id)">
            <X :size="17" />
          </IconButton>
        </article>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-viewport {
  position: fixed;
  z-index: 120;
  top: 18px;
  right: 18px;
  display: grid;
  width: min(390px, calc(100vw - 28px));
  gap: 9px;
  pointer-events: none;
}

.toast {
  display: grid;
  min-height: 68px;
  align-items: start;
  padding: 11px 10px 11px 14px;
  border: 1px solid var(--border);
  border-left: 3px solid currentColor;
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-md);
  color: var(--info);
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 10px;
  pointer-events: auto;
}

.toast--success {
  color: var(--success);
}

.toast--warning {
  color: var(--warning);
}

.toast--danger {
  color: var(--danger);
}

.toast__icon {
  margin-top: 3px;
}

.toast__content {
  min-width: 0;
}

.toast__title {
  color: var(--ink);
  font-size: 13px;
  font-weight: 750;
}

.toast__message {
  margin-top: 3px;
  color: var(--ink-muted);
  font-size: 12px;
}

.toast-enter-active,
.toast-leave-active {
  transition:
    opacity var(--motion-normal) var(--ease-out),
    transform var(--motion-normal) var(--ease-out);
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateX(12px);
}
</style>
