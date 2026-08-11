<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string
    tone?: 'default' | 'dark' | 'danger'
    disabled?: boolean
  }>(),
  {
    tone: 'default',
    disabled: false,
  },
)
</script>

<template>
  <button
    class="icon-button"
    :class="`icon-button--${tone}`"
    type="button"
    :disabled="disabled"
    :aria-label="label"
    :title="label"
  >
    <slot></slot>
  </button>
</template>

<style scoped>
.icon-button {
  display: inline-flex;
  width: 44px;
  height: 44px;
  flex: 0 0 44px;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--ink-muted);
  transition:
    background var(--motion-fast),
    color var(--motion-fast),
    border-color var(--motion-fast),
    transform var(--motion-fast);
}

.icon-button:active:not(:disabled) {
  transform: scale(0.94);
}

.icon-button--default:hover:not(:disabled) {
  border-color: var(--border);
  background: var(--surface-soft);
  color: var(--ink);
}

.icon-button--dark {
  color: rgb(255 255 255 / 72%);
}

.icon-button--dark:hover:not(:disabled) {
  background: rgb(255 255 255 / 9%);
  color: #fff;
}

.icon-button--danger {
  color: var(--danger);
}

.icon-button--danger:hover:not(:disabled) {
  background: var(--danger-soft);
}

.icon-button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}
</style>
