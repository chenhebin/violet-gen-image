<script setup lang="ts">
import { LoaderCircle } from '@lucide/vue'

withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
    size?: 'small' | 'medium'
    loading?: boolean
    disabled?: boolean
    type?: 'button' | 'submit' | 'reset'
  }>(),
  {
    variant: 'primary',
    size: 'medium',
    loading: false,
    disabled: false,
    type: 'button',
  },
)
</script>

<template>
  <button
    class="base-button"
    :class="[`is-${variant}`, `is-${size}`]"
    :type="type"
    :disabled="disabled || loading"
  >
    <LoaderCircle v-if="loading" :size="17" class="spinner" aria-hidden="true" />
    <slot name="icon" />
    <span><slot /></span>
  </button>
</template>

<style scoped>
.base-button {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 18px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  font-weight: 650;
  transition:
    background var(--motion-fast),
    border-color var(--motion-fast),
    color var(--motion-fast),
    transform var(--motion-fast);
}

.base-button:not(:disabled):active {
  transform: translateY(1px);
}

.base-button:disabled {
  cursor: not-allowed;
  opacity: 0.54;
}

.is-primary {
  background: var(--primary);
  color: white;
  box-shadow: 0 1px 1px rgb(0 0 0 / 8%);
}

.is-primary:not(:disabled):hover {
  background: var(--primary-hover);
}

.is-secondary {
  border-color: var(--border);
  background: var(--surface);
  color: var(--ink);
}

.is-secondary:not(:disabled):hover {
  border-color: var(--border-strong);
  background: var(--surface-soft);
}

.is-ghost {
  background: transparent;
  color: var(--ink-muted);
}

.is-ghost:not(:disabled):hover {
  background: var(--surface-soft);
  color: var(--ink);
}

.is-danger {
  background: var(--coral-soft);
  color: var(--danger);
}

.is-small {
  min-height: 36px;
  padding-inline: 12px;
  font-size: 13px;
}

.spinner {
  animation: spin 700ms linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
