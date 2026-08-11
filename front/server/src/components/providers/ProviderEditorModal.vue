<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { KeyRound, LockKeyhole, Server } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import FormField from '@/components/base/FormField.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseToggle from '@/components/base/BaseToggle.vue'
import type {
  AIProvider,
  CreateAIProviderPayload,
  UpdateAIProviderPayload,
} from '@/types'

const props = defineProps<{
  open: boolean
  provider?: AIProvider | null
  loading?: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [
    payload: CreateAIProviderPayload | UpdateAIProviderPayload,
  ]
}>()

const form = reactive({
  name: '',
  code: '',
  baseUrl: '',
  apiKey: '',
  note: '',
  enabled: true,
  touched: false,
})

const isEditing = computed(() => Boolean(props.provider))
const errors = computed(() => ({
  name: !form.name.trim() ? '请填写服务商名称' : '',
  code:
    !isEditing.value && !/^[a-z][a-z0-9-]{1,31}$/.test(form.code.trim())
      ? '编码需以字母开头，仅使用小写字母、数字和短横线'
      : '',
  baseUrl: (() => {
    try {
      const url = new URL(form.baseUrl)
      return url.protocol !== 'https:' ? 'Base URL 必须使用 HTTPS' : ''
    } catch {
      return '请输入有效的 HTTPS 地址'
    }
  })(),
  apiKey: !isEditing.value && !form.apiKey.trim() ? '请输入 API Key' : '',
}))

const valid = computed(() =>
  Object.values(errors.value).every((message) => !message),
)

function submit() {
  form.touched = true
  if (!valid.value) return

  if (props.provider) {
    emit('submit', {
      name: form.name.trim(),
      baseUrl: form.baseUrl.trim().replace(/\/$/, ''),
      enabled: form.enabled,
      note: form.note.trim() || undefined,
    })
    return
  }

  emit('submit', {
    name: form.name.trim(),
    code: form.code.trim().toLowerCase(),
    baseUrl: form.baseUrl.trim().replace(/\/$/, ''),
    apiKey: form.apiKey.trim(),
    enabled: form.enabled,
    note: form.note.trim() || undefined,
  })
}

watch(
  () => [props.open, props.provider] as const,
  ([open]) => {
    if (!open) return
    form.name = props.provider?.name || ''
    form.code = props.provider?.code || ''
    form.baseUrl = props.provider?.baseUrl || ''
    form.apiKey = ''
    form.note = props.provider?.note || ''
    form.enabled = props.provider?.enabled ?? true
    form.touched = false
  },
  { deep: true },
)
</script>

<template>
  <BaseModal
    :open="props.open"
    :title="isEditing ? '编辑 AI 服务商' : '新增 AI 服务商'"
    :description="
      isEditing
        ? '修改连接地址后，现有测试结果会失效，需要重新测试。'
        : 'v0.1 使用 OpenAI Compatible 协议，连接请求仅由后端发起。'
    "
    @close="emit('close')"
  >
    <form class="provider-form" @submit.prevent="submit">
      <div class="protocol-mark">
        <Server :size="19" aria-hidden="true" />
        <div>
          <strong>OpenAI Compatible</strong>
          <span>连接信息不会发送到 Client 或记录在前端日志中</span>
        </div>
      </div>

      <div class="form-grid">
        <FormField
          label="服务商名称"
          for-id="provider-name"
          required
          :error="form.touched ? errors.name : ''"
        >
          <input
            id="provider-name"
            v-model="form.name"
            maxlength="50"
            placeholder="例如：test1"
          />
        </FormField>
        <FormField
          label="服务商编码"
          for-id="provider-code"
          required
          :error="form.touched ? errors.code : ''"
          :hint="isEditing ? '创建后保持稳定，不可修改' : '用于服务端稳定识别'"
        >
          <input
            id="provider-code"
            v-model="form.code"
            class="data-mono"
            maxlength="32"
            placeholder="test1"
            :disabled="isEditing"
          />
        </FormField>
      </div>

      <FormField
        label="Base URL"
        for-id="provider-base-url"
        required
        :error="form.touched ? errors.baseUrl : ''"
        hint="需使用公网 HTTPS 地址，服务端会额外校验出站安全"
      >
        <div class="input-with-icon">
          <LockKeyhole :size="16" aria-hidden="true" />
          <input
            id="provider-base-url"
            v-model="form.baseUrl"
            class="data-mono"
            placeholder="https://api.example.com/v1"
            autocomplete="url"
          />
        </div>
      </FormField>

      <FormField
        v-if="!isEditing"
        label="API Key"
        for-id="provider-api-key"
        required
        :error="form.touched ? errors.apiKey : ''"
        hint="保存后只返回掩码，无法重新读取完整密钥"
      >
        <div class="input-with-icon">
          <KeyRound :size="16" aria-hidden="true" />
          <input
            id="provider-api-key"
            v-model="form.apiKey"
            class="data-mono"
            type="password"
            placeholder="sk-..."
            autocomplete="new-password"
          />
        </div>
      </FormField>

      <FormField label="内部备注" for-id="provider-note">
        <textarea
          id="provider-note"
          v-model="form.note"
          rows="3"
          maxlength="300"
          placeholder="记录服务用途、账号归属或迁移说明"
        />
      </FormField>

      <div class="toggle-panel">
        <BaseToggle
          v-model="form.enabled"
          label="允许平台调用"
          description="关闭后新任务不能使用该服务商；当前绑定需先解除或切换"
        />
      </div>

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
          {{ isEditing ? '保存服务商' : '创建服务商' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.provider-form {
  display: grid;
  gap: 18px;
}

.protocol-mark {
  display: flex;
  gap: 11px;
  align-items: center;
  padding: 12px 13px;
  color: var(--color-primary, #236c62);
  background: #eef5f3;
  border-left: 3px solid currentColor;
  border-radius: 4px 7px 7px 4px;
}

.protocol-mark div {
  display: grid;
  gap: 3px;
}

.protocol-mark strong {
  font-size: 12px;
}

.protocol-mark span {
  color: var(--color-text-muted, #68716f);
  font-size: 10px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 13px;
}

input,
textarea {
  width: 100%;
}

.input-with-icon {
  position: relative;
}

.input-with-icon svg {
  position: absolute;
  top: 50%;
  left: 12px;
  color: var(--color-text-muted, #68716f);
  transform: translateY(-50%);
  pointer-events: none;
}

.input-with-icon input {
  padding-left: 38px;
}

.toggle-panel {
  padding: 12px 13px;
  background: #f7f8f7;
  border: 1px solid var(--color-border-soft, #edf0ef);
  border-radius: 7px;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 9px;
}

@media (max-width: 560px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
