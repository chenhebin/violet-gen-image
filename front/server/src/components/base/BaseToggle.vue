<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    modelValue: boolean
    label: string
    description?: string
    disabled?: boolean
  }>(),
  {
    description: undefined,
    disabled: false,
  },
)

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

function toggle(): void {
  if (!props.disabled) emit('update:modelValue', !props.modelValue)
}
</script>

<template>
  <button
    class="toggle"
    :class="{ 'toggle--active': modelValue }"
    type="button"
    role="switch"
    :aria-checked="modelValue"
    :disabled="disabled"
    @click="toggle"
  >
    <span class="toggle__text">
      <span class="toggle__label">{{ label }}</span>
      <span v-if="description" class="toggle__description">{{ description }}</span>
    </span>
    <span class="toggle__track" aria-hidden="true">
      <span class="toggle__thumb"></span>
    </span>
  </button>
</template>

<style scoped>
.toggle {
  display: flex;
  width: 100%;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0;
  background: transparent;
  text-align: left;
}

.toggle__text {
  display: grid;
  gap: 2px;
}

.toggle__label {
  font-size: 13px;
  font-weight: 700;
}

.toggle__description {
  color: var(--ink-muted);
  font-size: 12px;
}

.toggle__track {
  position: relative;
  width: 38px;
  height: 22px;
  flex: 0 0 38px;
  border-radius: var(--radius-pill);
  background: #bdc5c1;
  transition: background var(--motion-fast);
}

.toggle__thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 1px 3px rgb(20 28 26 / 24%);
  transition: transform var(--motion-fast) var(--ease-out);
}

.toggle--active .toggle__track {
  background: var(--primary);
}

.toggle--active .toggle__thumb {
  transform: translateX(16px);
}

.toggle:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>
