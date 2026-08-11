<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseButton from './BaseButton.vue'
import BaseModal from './BaseModal.vue'
import FormField from './FormField.vue'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    description: string
    confirmLabel?: string
    cancelLabel?: string
    danger?: boolean
    loading?: boolean
    reasonLabel?: string
    reasonPlaceholder?: string
    reasonRequired?: boolean
  }>(),
  {
    confirmLabel: '确认',
    cancelLabel: '取消',
    danger: false,
    loading: false,
    reasonLabel: undefined,
    reasonPlaceholder: '填写本次操作原因',
    reasonRequired: false,
  },
)

const emit = defineEmits<{
  close: []
  confirm: [reason: string]
}>()

const reason = ref('')
const reasonError = ref('')

watch(
  () => props.open,
  (open) => {
    if (open) {
      reason.value = ''
      reasonError.value = ''
    }
  },
)

function confirm(): void {
  if (props.reasonRequired && !reason.value.trim()) {
    reasonError.value = '请填写操作原因'
    return
  }
  emit('confirm', reason.value.trim())
}
</script>

<template>
  <BaseModal
    :open="open"
    :title="title"
    :description="description"
    width="small"
    :close-on-backdrop="!loading"
    @close="!loading && emit('close')"
  >
    <FormField
      v-if="reasonLabel"
      :label="reasonLabel"
      for-id="confirm-reason"
      :error="reasonError"
      :required="reasonRequired"
    >
      <textarea
        id="confirm-reason"
        v-model="reason"
        class="form-control confirm-dialog__textarea"
        :placeholder="reasonPlaceholder"
        :disabled="loading"
        maxlength="300"
        @input="reasonError = ''"
      ></textarea>
    </FormField>
    <template #footer>
      <BaseButton
        variant="secondary"
        :disabled="loading"
        @click="emit('close')"
      >
        {{ cancelLabel }}
      </BaseButton>
      <BaseButton
        :variant="danger ? 'danger' : 'primary'"
        :loading="loading"
        @click="confirm"
      >
        {{ confirmLabel }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<style scoped>
.confirm-dialog__textarea {
  min-height: 92px;
  resize: vertical;
}
</style>
