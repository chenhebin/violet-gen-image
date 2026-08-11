<script setup lang="ts" generic="T extends string">
defineProps<{
  modelValue: T
  options: ReadonlyArray<{ value: T; label: string }>
  label: string
}>()

defineEmits<{
  'update:modelValue': [value: T]
}>()
</script>

<template>
  <div class="segmented" role="radiogroup" :aria-label="label">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      role="radio"
      :aria-checked="modelValue === option.value"
      :class="{ active: modelValue === option.value }"
      @click="$emit('update:modelValue', option.value)"
    >
      {{ option.label }}
    </button>
  </div>
</template>

<style scoped>
.segmented {
  display: inline-grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(88px, 1fr);
  gap: 2px;
  min-height: 44px;
  padding: 3px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-soft);
}

button {
  min-height: 36px;
  padding: 0 14px;
  border-radius: 5px;
  background: transparent;
  color: var(--ink-muted);
  font-size: 14px;
  font-weight: 650;
  transition:
    background var(--motion-fast),
    color var(--motion-fast),
    box-shadow var(--motion-fast);
}

button.active {
  background: var(--surface);
  color: var(--ink);
  box-shadow: var(--shadow-sm);
}
</style>
