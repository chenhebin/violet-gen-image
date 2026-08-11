<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Copy, KeyRound, ShieldCheck, UserMinus } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import FormField from '@/components/base/FormField.vue'
import type { ManagedUser, ResetPasswordResult } from '@/types/domain'
import { formatDateTime } from '@/utils/format'

export type UserAction = 'adjust' | 'disable' | 'enable' | 'reset'

export interface UserActionPayload {
  amount?: number
  reason?: string
  referenceNo?: string
}

const props = defineProps<{
  action: UserAction | null
  user: ManagedUser | null
  loading: boolean
  passwordResult?: ResetPasswordResult | null
}>()

const emit = defineEmits<{
  close: []
  submit: [payload: UserActionPayload]
}>()

const amount = ref(1)
const reason = ref('')
const referenceNo = ref('')
const error = ref('')

const meta = computed(() => {
  if (props.action === 'adjust') {
    return {
      title: '人工调整次数',
      description: `当前余额 ${props.user?.balance ?? 0} 次。增加填正数，减少填负数。`,
      submit: '确认调整',
    }
  }
  if (props.action === 'disable') {
    return {
      title: '停用用户账号',
      description: '停用后会阻止登录、兑换和收费操作，并撤销已有会话。',
      submit: '确认停用',
    }
  }
  if (props.action === 'enable') {
    return {
      title: '恢复用户账号',
      description: '恢复后用户可以重新登录和使用平台次数。',
      submit: '确认恢复',
    }
  }
  return {
    title: '重置用户密码',
    description: '系统将生成一次性临时密码，不会显示或读取用户原密码。',
    submit: '生成临时密码',
  }
})

watch(
  () => props.action,
  () => {
    amount.value = 1
    reason.value = ''
    referenceNo.value = ''
    error.value = ''
  },
)

function submit(): void {
  error.value = ''
  if (props.action === 'adjust') {
    if (!Number.isInteger(amount.value) || amount.value === 0) {
      error.value = '调整值必须是非零整数'
      return
    }
    if ((props.user?.balance ?? 0) + amount.value < 0) {
      error.value = '调整后的可用余额不能小于 0'
      return
    }
  }
  if (props.action !== 'reset' && !reason.value.trim()) {
    error.value = '请填写操作原因'
    return
  }
  emit('submit', {
    amount: props.action === 'adjust' ? amount.value : undefined,
    reason: reason.value.trim() || undefined,
    referenceNo: referenceNo.value.trim() || undefined,
  })
}

async function copyPassword(): Promise<void> {
  if (!props.passwordResult) return
  await navigator.clipboard.writeText(props.passwordResult.temporaryPassword)
}
</script>

<template>
  <BaseModal
    :open="Boolean(action)"
    :title="meta.title"
    :description="meta.description"
    width="small"
    :close-on-backdrop="!loading"
    @close="$emit('close')"
  >
    <div v-if="action === 'adjust'" class="action-form">
      <FormField label="调整次数" for-id="adjust-amount" required>
        <input
          id="adjust-amount"
          v-model.number="amount"
          class="form-control mono"
          type="number"
          step="1"
        />
      </FormField>
      <FormField label="操作原因" for-id="adjust-reason" required>
        <textarea
          id="adjust-reason"
          v-model="reason"
          class="form-control"
          rows="4"
          maxlength="500"
          placeholder="说明调整依据，记录将不可修改"
        />
      </FormField>
      <FormField label="外部参考号" for-id="adjust-reference" hint="可填写咸鱼订单号或内部凭证">
        <input
          id="adjust-reference"
          v-model="referenceNo"
          class="form-control mono"
          maxlength="100"
          placeholder="可选"
        />
      </FormField>
    </div>

    <FormField
      v-else-if="action === 'disable' || action === 'enable'"
      label="操作原因"
      for-id="user-status-reason"
      required
    >
      <textarea
        id="user-status-reason"
        v-model="reason"
        class="form-control"
        rows="5"
        maxlength="500"
        :placeholder="action === 'disable' ? '说明停用账号的原因' : '说明恢复账号的依据'"
      />
    </FormField>

    <div v-else-if="passwordResult" class="password-result">
      <span>一次性临时密码</span>
      <strong class="mono">{{ passwordResult.temporaryPassword }}</strong>
      <p>有效期至 {{ formatDateTime(passwordResult.expiresAt) }}</p>
      <BaseButton variant="secondary" @click="copyPassword">
        <template #icon><Copy :size="16" /></template>
        复制临时密码
      </BaseButton>
    </div>

    <div v-else class="reset-confirm">
      <KeyRound :size="23" />
      <p>
        临时密码仅在本次操作结果中显示。请通过可信渠道单独发送给用户。
      </p>
    </div>

    <p v-if="error" class="validation-error" role="alert">{{ error }}</p>

    <template #footer>
      <BaseButton variant="secondary" :disabled="loading" @click="$emit('close')">
        {{ passwordResult ? '关闭' : '取消' }}
      </BaseButton>
      <BaseButton
        v-if="!passwordResult"
        :variant="action === 'disable' ? 'danger' : 'primary'"
        :loading="loading"
        @click="submit"
      >
        <template #icon>
          <UserMinus v-if="action === 'disable'" :size="16" />
          <ShieldCheck v-else :size="16" />
        </template>
        {{ meta.submit }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<style scoped>
.action-form {
  display: grid;
  gap: 18px;
}

textarea.form-control {
  min-height: 104px;
  resize: vertical;
}

.reset-confirm {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-soft);
  color: var(--ink-muted);
}

.reset-confirm svg {
  flex: 0 0 auto;
  color: var(--primary);
}

.password-result {
  display: grid;
  justify-items: start;
  gap: 8px;
  padding: 18px;
  border: 1px solid var(--primary);
  border-radius: var(--radius-md);
  background: var(--primary-soft);
}

.password-result span,
.password-result p {
  color: var(--ink-muted);
  font-size: 11px;
}

.password-result strong {
  font-size: 20px;
  letter-spacing: 0.04em;
}

.validation-error {
  margin-top: 12px;
  color: var(--danger);
  font-size: 12px;
}
</style>

