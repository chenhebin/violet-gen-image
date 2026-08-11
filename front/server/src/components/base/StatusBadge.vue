<script setup lang="ts">
withDefaults(
  defineProps<{
    tone?: 'neutral' | 'success' | 'warning' | 'danger' | 'info'
    dot?: boolean
  }>(),
  {
    tone: 'neutral',
    dot: true,
  },
)
</script>

<template>
  <span class="status-badge" :class="`status-badge--${tone}`">
    <span v-if="dot" class="status-badge__dot" aria-hidden="true"></span>
    <slot></slot>
  </span>
</template>

<style scoped>
.status-badge {
  display: inline-flex;
  min-height: 26px;
  align-items: center;
  gap: 6px;
  padding: 3px 9px;
  border: 1px solid transparent;
  border-radius: var(--radius-pill);
  font-size: 12px;
  font-weight: 650;
  line-height: 1.2;
  white-space: nowrap;
}

.status-badge__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.status-badge--neutral {
  border-color: var(--border);
  background: var(--surface-soft);
  color: var(--ink-muted);
}

.status-badge--success {
  background: var(--success-soft);
  color: var(--success);
}

.status-badge--warning {
  background: var(--warning-soft);
  color: var(--warning);
}

.status-badge--danger {
  background: var(--danger-soft);
  color: var(--danger);
}

.status-badge--info {
  background: var(--info-soft);
  color: var(--info);
}
</style>
