<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { CalendarPlus, Infinity as InfinityIcon } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import FormField from '@/components/base/FormField.vue'
import BaseModal from '@/components/base/BaseModal.vue'

const props = defineProps<{
  open: boolean
  targetLabel: string
  count: number
  loading?: boolean
}>()

const emit = defineEmits<{
  close: []
  confirm: [payload: { expiresAt: string | null; reason: string }]
}>()

const form = reactive({
  neverExpires: false,
  expiresAt: '',
  reason: '',
  touched: false,
})

const minDate = computed(() =>
  new Date(Date.now() + 86400000).toISOString().slice(0, 10),
)

const errors = computed(() => ({
  expiresAt:
    !form.neverExpires &&
    (!form.expiresAt ||
      new Date(`${form.expiresAt}T23:59:59`).getTime() <= Date.now())
      ? '请选择未来的到期日期'
      : '',
  reason: !form.reason.trim() ? '请填写延期原因' : '',
}))

const valid = computed(() =>
  Object.values(errors.value).every((message) => !message),
)

function submit() {
  form.touched = true
  if (!valid.value) return

  emit('confirm', {
    expiresAt: form.neverExpires
      ? null
      : `${form.expiresAt}T23:59:59.000Z`,
    reason: form.reason.trim(),
  })
}

watch(
  () => props.open,
  (open) => {
    if (!open) return
    const date = new Date()
    date.setDate(date.getDate() + 90)
    form.neverExpires = false
    form.expiresAt = date.toISOString().slice(0, 10)
    form.reason = ''
    form.touched = false
  },
)
</script>

<template>
  <BaseModal
    :open="props.open"
    title="延长兑换码有效期"
    :description="`${props.targetLabel} · 共 ${props.count} 个目标，服务端会跳过已兑换和已失效记录。`"
    @close="emit('close')"
  >
    <form class="extend-form" @submit.prevent="submit">
      <div class="notice">
        <CalendarPlus :size="19" aria-hidden="true" />
        <p>
          过期但未兑换的兑换码延期后会恢复为“未使用”；已兑换和显式失效码不会改变。
        </p>
      </div>

      <div class="validity-switch">
        <button
          type="button"
          :class="{ active: !form.neverExpires }"
          @click="form.neverExpires = false"
        >
          <CalendarPlus :size="17" aria-hidden="true" />
          指定到期日期
        </button>
        <button
          type="button"
          :class="{ active: form.neverExpires }"
          @click="form.neverExpires = true"
        >
          <InfinityIcon :size="17" aria-hidden="true" />
          改为永久有效
        </button>
      </div>

      <FormField
        v-if="!form.neverExpires"
        label="新的到期日期"
        for-id="extend-expires-at"
        required
        :error="form.touched ? errors.expiresAt : ''"
      >
        <input
          id="extend-expires-at"
          v-model="form.expiresAt"
          type="date"
          :min="minDate"
        />
      </FormField>

      <FormField
        label="延期原因"
        for-id="extend-reason"
        required
        :error="form.touched ? errors.reason : ''"
        hint="该说明会进入兑换码操作历史与审计日志"
      >
        <textarea
          id="extend-reason"
          v-model="form.reason"
          rows="3"
          maxlength="240"
          placeholder="例如：买家延迟领取，延长本批次发放有效期"
        />
      </FormField>

      <div class="actions">
        <BaseButton
          type="button"
          variant="ghost"
          :disabled="props.loading"
          @click="emit('close')"
        >
          取消
        </BaseButton>
        <BaseButton type="submit" :loading="props.loading">
          确认延期
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.extend-form {
  display: grid;
  gap: 18px;
}

.notice {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  padding: 12px 13px;
  color: #8a6726;
  background: #fbf6e9;
  border-left: 3px solid var(--color-warning, #c59a44);
  border-radius: 4px 7px 7px 4px;
}

.notice svg {
  flex: 0 0 auto;
  margin-top: 1px;
}

.notice p {
  margin: 0;
  font-size: 11px;
  line-height: 1.65;
}

.validity-switch {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 9px;
}

.validity-switch button {
  display: flex;
  gap: 8px;
  align-items: center;
  min-height: 48px;
  padding: 0 12px;
  color: var(--color-text-muted, #68716f);
  font-size: 12px;
  background: #fff;
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 7px;
  cursor: pointer;
}

.validity-switch button.active {
  color: var(--color-primary, #236c62);
  font-weight: 700;
  background: #f2f7f5;
  border-color: var(--color-primary, #236c62);
}

input,
textarea {
  width: 100%;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 9px;
}

@media (max-width: 520px) {
  .validity-switch {
    grid-template-columns: 1fr;
  }
}
</style>
