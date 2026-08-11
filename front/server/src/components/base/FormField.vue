<script setup lang="ts">
defineProps<{
  label: string
  forId?: string
  hint?: string
  error?: string
  required?: boolean
}>()
</script>

<template>
  <div class="form-field" :class="{ 'form-field--error': error }">
    <label class="form-field__label" :for="forId">
      {{ label }}
      <span v-if="required" class="form-field__required" aria-hidden="true">*</span>
    </label>
    <slot></slot>
    <p v-if="error" class="form-field__message form-field__message--error">
      {{ error }}
    </p>
    <p v-else-if="hint" class="form-field__message">{{ hint }}</p>
  </div>
</template>

<style scoped>
.form-field {
  display: grid;
  gap: 7px;
}

.form-field__label {
  color: var(--ink);
  font-size: 13px;
  font-weight: 700;
}

.form-field__required {
  margin-left: 3px;
  color: var(--danger);
}

.form-field__message {
  color: var(--ink-muted);
  font-size: 12px;
}

.form-field__message--error {
  color: var(--danger);
}

.form-field--error :deep(.form-control) {
  border-color: var(--danger);
}
</style>
