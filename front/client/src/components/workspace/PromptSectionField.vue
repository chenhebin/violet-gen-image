<script setup lang="ts">
import { PROMPT_CONFIG } from '@/config'
import type { PromptSections } from '@/types/domain'

const props = defineProps<{
  fieldKey: keyof PromptSections
  label: string
  value: string
  unchanged: boolean
}>()

const emit = defineEmits<{
  'update:value': [value: string]
  'update:unchanged': [value: boolean]
}>()

const inputId = `prompt-section-${props.fieldKey}`
const toggleId = `${inputId}-unchanged`

function updateValue(event: Event): void {
  emit('update:value', (event.target as HTMLTextAreaElement).value)
}

function updateUnchanged(event: Event): void {
  emit('update:unchanged', (event.target as HTMLInputElement).checked)
}
</script>

<template>
  <div class="prompt-section-field" :class="{ unchanged }">
    <div class="field-header">
      <label class="field-label" :for="inputId">{{ label }}</label>
      <label class="unchanged-toggle" :for="toggleId">
        <span>保持不变</span>
        <input
          :id="toggleId"
          class="switch-input sr-only"
          type="checkbox"
          role="switch"
          :checked="unchanged"
          :aria-label="`${label}保持不变`"
          @change="updateUnchanged"
        />
        <span class="switch-track" aria-hidden="true"><i /></span>
      </label>
    </div>

    <textarea
      :id="inputId"
      :value="unchanged ? PROMPT_CONFIG.unchangedText : value"
      :disabled="unchanged"
      rows="2"
      @input="updateValue"
    />
  </div>
</template>

<style scoped>
.prompt-section-field {
  display: grid;
  min-width: 0;
  gap: 5px;
}

.field-header {
  display: flex;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.field-label {
  min-width: 0;
  color: var(--ink-muted);
  font-size: 10px;
  font-weight: 680;
}

.unchanged-toggle {
  position: relative;
  display: inline-flex;
  min-height: 44px;
  flex: 0 0 auto;
  align-items: center;
  gap: 7px;
  color: var(--ink-faint);
  font-size: 10px;
  font-weight: 650;
  user-select: none;
}

.switch-track {
  position: relative;
  width: 34px;
  height: 20px;
  flex: 0 0 auto;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-pill);
  background: var(--surface-soft);
  transition:
    border-color var(--motion-fast) var(--ease-out),
    background var(--motion-fast) var(--ease-out),
    box-shadow var(--motion-fast) var(--ease-out);
}

.switch-track i {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--surface);
  box-shadow: 0 1px 3px rgb(23 25 29 / 18%);
  transition: transform var(--motion-normal) var(--ease-out);
}

.unchanged-toggle:hover .switch-track {
  border-color: var(--primary);
}

.switch-input:focus-visible + .switch-track {
  outline: 3px solid rgb(20 108 99 / 18%);
  outline-offset: 2px;
}

.switch-input:checked + .switch-track {
  border-color: var(--primary);
  background: var(--primary);
  box-shadow: inset 0 0 0 1px rgb(255 255 255 / 10%);
}

.switch-input:checked + .switch-track i {
  transform: translateX(14px);
}

textarea {
  width: 100%;
  min-height: 58px;
  padding: 8px 9px;
  resize: vertical;
  border: 1px solid var(--border-strong);
  border-radius: 7px;
  outline: 0;
  background: var(--surface);
  font-size: 11px;
  line-height: 1.5;
  transition:
    border-color var(--motion-fast),
    background var(--motion-fast),
    box-shadow var(--motion-fast),
    color var(--motion-fast);
}

textarea:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px rgb(20 108 99 / 10%);
}

textarea:disabled {
  resize: none;
  border-color: rgb(20 108 99 / 22%);
  background: var(--primary-soft);
  color: var(--primary);
  font-weight: 650;
  opacity: 1;
  -webkit-text-fill-color: var(--primary);
}

.prompt-section-field.unchanged .unchanged-toggle {
  color: var(--primary);
}

@media (max-width: 760px) {
  .prompt-section-field {
    gap: 6px;
  }

  .field-label,
  .unchanged-toggle {
    font-size: 12px;
  }

  .switch-track {
    width: 38px;
    height: 22px;
  }

  .switch-track i {
    width: 16px;
    height: 16px;
  }

  .switch-input:checked + .switch-track i {
    transform: translateX(16px);
  }

  textarea {
    min-height: 72px;
    padding: 10px 11px;
    font-size: 13px;
  }
}

@media (max-width: 360px) {
  .field-header {
    align-items: flex-start;
    flex-direction: column;
    gap: 0;
  }

  .unchanged-toggle {
    width: 100%;
    justify-content: space-between;
  }
}
</style>
