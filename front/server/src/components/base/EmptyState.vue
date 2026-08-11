<script setup lang="ts">
import { Inbox } from '@lucide/vue'

withDefaults(
  defineProps<{
    title: string
    description?: string
    compact?: boolean
  }>(),
  {
    description: undefined,
    compact: false,
  },
)
</script>

<template>
  <div class="empty-state" :class="{ 'empty-state--compact': compact }">
    <div class="empty-state__icon" aria-hidden="true">
      <slot name="icon"><Inbox :size="22" /></slot>
    </div>
    <h3 class="empty-state__title">{{ title }}</h3>
    <p v-if="description" class="empty-state__description">{{ description }}</p>
    <div v-if="$slots.action" class="empty-state__action">
      <slot name="action"></slot>
    </div>
  </div>
</template>

<style scoped>
.empty-state {
  display: grid;
  min-height: 280px;
  align-content: center;
  justify-items: center;
  padding: 36px 24px;
  text-align: center;
}

.empty-state--compact {
  min-height: 180px;
}

.empty-state__icon {
  display: grid;
  width: 44px;
  height: 44px;
  margin-bottom: 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-soft);
  color: var(--ink-muted);
  place-items: center;
}

.empty-state__title {
  font-size: 15px;
}

.empty-state__description {
  max-width: 420px;
  margin-top: 6px;
  color: var(--ink-muted);
  font-size: 13px;
}

.empty-state__action {
  margin-top: 16px;
}
</style>
