<script setup lang="ts">
import { LoaderCircle } from '@lucide/vue'

withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
    size?: 'small' | 'medium' | 'sm' | 'md'
    type?: 'button' | 'submit' | 'reset'
    loading?: boolean
    disabled?: boolean
    block?: boolean
  }>(),
  {
    variant: 'primary',
    size: 'medium',
    type: 'button',
    loading: false,
    disabled: false,
    block: false,
  },
)
</script>

<template>
  <button
    class="base-button"
    :class="[
      `base-button--${variant}`,
      `base-button--${size}`,
      { 'base-button--block': block },
    ]"
    :type="type"
    :disabled="disabled || loading"
    :aria-busy="loading"
  >
    <LoaderCircle v-if="loading" class="base-button__spinner" :size="17" />
    <span v-else-if="$slots.icon" class="base-button__icon">
      <slot name="icon"></slot>
    </span>
    <span><slot></slot></span>
  </button>
</template>

<style scoped>
.base-button {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  font-weight: 700;
  line-height: 1;
  white-space: nowrap;
  transition:
    background var(--motion-fast),
    border-color var(--motion-fast),
    color var(--motion-fast),
    transform var(--motion-fast);
}

.base-button:not(:disabled):active {
  transform: translateY(1px) scale(0.985);
}

.base-button--medium {
  min-height: 44px;
  padding: 0 16px;
  font-size: 14px;
}

.base-button--small,
.base-button--sm {
  min-height: 36px;
  padding: 0 12px;
  font-size: 13px;
}

.base-button--md {
  min-height: 44px;
  padding: 0 16px;
  font-size: 14px;
}

.base-button--primary {
  background: var(--primary);
  color: #fff;
}

.base-button--primary:hover:not(:disabled) {
  background: var(--primary-hover);
}

.base-button--secondary {
  border-color: var(--border-strong);
  background: var(--surface);
  color: var(--ink);
}

.base-button--secondary:hover:not(:disabled),
.base-button--ghost:hover:not(:disabled) {
  background: var(--surface-soft);
}

.base-button--ghost {
  background: transparent;
  color: var(--ink-muted);
}

.base-button--danger {
  background: var(--danger);
  color: #fff;
}

.base-button--danger:hover:not(:disabled) {
  background: var(--danger-hover);
}

.base-button:disabled {
  cursor: not-allowed;
  opacity: 0.52;
}

.base-button--block {
  width: 100%;
}

.base-button__icon {
  display: inline-flex;
  align-items: center;
}

.base-button__spinner {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 640px) {
  .base-button--small,
  .base-button--sm {
    min-height: 44px;
  }
}
</style>
