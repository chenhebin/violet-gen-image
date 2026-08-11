<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { KeyRound, ShieldCheck } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import FormField from '@/components/base/FormField.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import type { AIProvider } from '@/types'

const props = defineProps<{
  open: boolean
  provider: AIProvider | null
  loading?: boolean
}>()

const emit = defineEmits<{
  close: []
  confirm: [apiKey: string]
}>()

const apiKey = ref('')
const touched = ref(false)
const error = computed(() =>
  apiKey.value.trim().length < 8 ? '请输入有效的新 API Key' : '',
)

function submit() {
  touched.value = true
  if (error.value) return
  emit('confirm', apiKey.value.trim())
}

watch(
  () => props.open,
  (open) => {
    if (!open) return
    apiKey.value = ''
    touched.value = false
  },
)
</script>

<template>
  <BaseModal
    :open="props.open"
    title="轮换 API Key"
    :description="`${props.provider?.name || '当前服务商'} · 新密钥保存后无法重新读取。`"
    @close="emit('close')"
  >
    <form class="key-form" @submit.prevent="submit">
      <div class="security-note">
        <ShieldCheck :size="20" aria-hidden="true" />
        <div>
          <strong>覆盖现有密钥</strong>
          <span>
            更新后旧连接测试立即失效；请重新测试服务商和模型，再恢复平台绑定。
          </span>
        </div>
      </div>

      <FormField
        label="现有密钥"
        for-id="masked-api-key"
        hint="管理端只持有服务端返回的掩码"
      >
        <input
          id="masked-api-key"
          class="data-mono"
          :value="props.provider?.maskedApiKey"
          disabled
        />
      </FormField>

      <FormField
        label="新 API Key"
        for-id="new-api-key"
        required
        :error="touched ? error : ''"
      >
        <div class="key-input">
          <KeyRound :size="16" aria-hidden="true" />
          <input
            id="new-api-key"
            v-model="apiKey"
            class="data-mono"
            type="password"
            placeholder="输入新的密钥"
            autocomplete="new-password"
          />
        </div>
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
          确认轮换
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.key-form {
  display: grid;
  gap: 18px;
}

.security-note {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  padding: 12px 13px;
  color: #8a6726;
  background: #fbf6e9;
  border-left: 3px solid var(--color-warning, #c59a44);
  border-radius: 4px 7px 7px 4px;
}

.security-note svg {
  flex: 0 0 auto;
}

.security-note div {
  display: grid;
  gap: 4px;
}

.security-note strong {
  font-size: 12px;
}

.security-note span {
  font-size: 10px;
  line-height: 1.55;
}

input {
  width: 100%;
}

.key-input {
  position: relative;
}

.key-input svg {
  position: absolute;
  top: 50%;
  left: 12px;
  color: var(--color-text-muted, #68716f);
  transform: translateY(-50%);
}

.key-input input {
  padding-left: 38px;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 9px;
}
</style>
