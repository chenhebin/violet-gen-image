<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { PencilLine } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import FormField from '@/components/base/FormField.vue'
import { REDEMPTION_CONFIG } from '@/config'
import type { RedemptionBatch, UpdateRedemptionBatchPayload } from '@/types'

const props = defineProps<{
  open: boolean
  batch: RedemptionBatch | null
  loading?: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [payload: UpdateRedemptionBatchPayload]
}>()

const name = ref('')
const touched = ref(false)
const input = ref<HTMLInputElement | null>(null)

const normalizedName = computed(() => name.value.trim())
const error = computed(() => {
  if (!normalizedName.value) return '请填写批次名称'
  if (
    Array.from(normalizedName.value).length >
    REDEMPTION_CONFIG.batchNameMaxLength
  ) {
    return `批次名称不能超过 ${REDEMPTION_CONFIG.batchNameMaxLength} 个字符`
  }
  return ''
})
const unchanged = computed(
  () => normalizedName.value === props.batch?.name.trim(),
)
const canSubmit = computed(
  () => Boolean(props.batch) && !error.value && !unchanged.value && !props.loading,
)

function submit() {
  touched.value = true
  if (!canSubmit.value) return
  emit('submit', { name: normalizedName.value })
}

watch(
  () => [props.open, props.batch?.id] as const,
  async ([open]) => {
    if (!open) return
    name.value = props.batch?.name ?? ''
    touched.value = false
    await nextTick()
    input.value?.focus()
    input.value?.select()
  },
  { immediate: true },
)
</script>

<template>
  <BaseModal
    :open="props.open"
    title="修改批次名称"
    description="只更新运营展示名称，不会改变批次内兑换码、次数或有效期。"
    width="small"
    :close-on-backdrop="!props.loading"
    @close="!props.loading && emit('close')"
  >
    <form class="rename-form" @submit.prevent="submit">
      <div class="batch-context">
        <PencilLine :size="18" aria-hidden="true" />
        <div>
          <span>当前批次</span>
          <strong class="data-mono">{{ props.batch?.id }}</strong>
        </div>
      </div>

      <FormField
        label="批次名称"
        for-id="rename-batch-name"
        required
        :error="touched ? error : ''"
        :hint="`最多 ${REDEMPTION_CONFIG.batchNameMaxLength} 个字符`"
      >
        <input
          id="rename-batch-name"
          ref="input"
          v-model="name"
          :maxlength="REDEMPTION_CONFIG.batchNameMaxLength"
          autocomplete="off"
          @blur="touched = true"
        />
      </FormField>

      <p v-if="unchanged && !error" class="unchanged-hint" role="status">
        名称尚未修改
      </p>

      <div class="form-actions">
        <BaseButton
          type="button"
          variant="ghost"
          :disabled="props.loading"
          @click="emit('close')"
        >
          取消
        </BaseButton>
        <BaseButton type="submit" :loading="props.loading" :disabled="!canSubmit">
          保存名称
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.rename-form {
  display: grid;
  gap: 18px;
}

.batch-context {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 12px 13px;
  color: var(--primary);
  background: var(--surface-soft);
  border-left: 3px solid currentColor;
  border-radius: 4px var(--radius-sm) var(--radius-sm) 4px;
}

.batch-context > div {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.batch-context span {
  color: var(--ink-muted);
  font-size: 11px;
}

.batch-context strong {
  overflow: hidden;
  color: var(--ink);
  font-size: 11px;
  text-overflow: ellipsis;
}

input {
  width: 100%;
}

.unchanged-hint {
  margin-top: -9px;
  color: var(--ink-muted);
  font-size: 12px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 9px;
}

@media (max-width: 640px) {
  .form-actions > :deep(.base-button) {
    flex: 1;
  }
}
</style>
