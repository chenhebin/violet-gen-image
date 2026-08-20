<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ArrowRight, Eye, EyeOff, LockKeyhole, Mail } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import SegmentedControl from '@/components/base/SegmentedControl.vue'
import { useToast } from '@/composables/useToast'
import { AUTH_CONFIG } from '@/config'
import { useAuthStore } from '@/stores/auth'

export type AuthMode = 'login' | 'register'

const props = withDefaults(defineProps<{
  context?: 'standard' | 'claim'
  credits?: number | null
  initialMode?: AuthMode
}>(), {
  context: 'standard',
  credits: null,
  initialMode: 'login',
})

const emit = defineEmits<{
  authenticated: [mode: AuthMode]
  modeChanged: [mode: AuthMode]
}>()

const auth = useAuthStore()
const toast = useToast()
const mode = ref<AuthMode>(props.initialMode)
const showPassword = ref(false)
const touchedEmail = ref(false)
const form = reactive({
  email: '',
  password: '',
  confirmPassword: '',
  remember: true,
  acceptedTerms: false,
})

const emailValid = computed(() => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email))
const passwordValid = computed(() => form.password.length >= AUTH_CONFIG.minimumPasswordLength)
const canSubmit = computed(() => {
  if (mode.value === 'login') return emailValid.value && passwordValid.value
  return (
    emailValid.value &&
    passwordValid.value &&
    form.password === form.confirmPassword &&
    form.acceptedTerms
  )
})
const creditLabel = computed(() => props.credits ? `${props.credits} 次` : '本兑换码额度')
const submitLabel = computed(() => {
  if (props.context === 'claim') {
    return mode.value === 'login'
      ? `登录并领取${creditLabel.value}`
      : `注册并领取${creditLabel.value}`
  }
  return mode.value === 'login' ? '进入工作台' : '创建账号'
})

watch(mode, (value) => emit('modeChanged', value))

async function submit(): Promise<void> {
  if (!canSubmit.value) return
  try {
    if (mode.value === 'login') {
      await auth.login({
        email: form.email,
        password: form.password,
        remember: form.remember,
      })
    } else {
      await auth.register({
        email: form.email,
        password: form.password,
        remember: form.remember,
        acceptedTerms: form.acceptedTerms,
      })
    }
    emit('authenticated', mode.value)
  } catch {
    // The auth store exposes the normalized API message beside the form.
  }
}
</script>

<template>
  <div class="auth-form">
    <SegmentedControl
      v-model="mode"
      label="登录或注册"
      :options="[
        { value: 'login', label: props.context === 'claim' ? '登录并领取' : '登录' },
        { value: 'register', label: props.context === 'claim' ? '注册并领取' : '注册' },
      ]"
      @update:model-value="auth.clearError()"
    />

    <form novalidate @submit.prevent="submit">
      <div class="field">
        <label for="auth-email">邮箱</label>
        <div class="input-wrap">
          <Mail :size="18" aria-hidden="true" />
          <input
            id="auth-email"
            v-model.trim="form.email"
            type="email"
            autocomplete="email"
            placeholder="name@example.com"
            @blur="touchedEmail = true"
          />
        </div>
        <p v-if="touchedEmail && !emailValid" class="field-error">请输入有效的邮箱地址</p>
      </div>

      <div class="field">
        <label for="auth-password">密码</label>
        <div class="input-wrap">
          <LockKeyhole :size="18" aria-hidden="true" />
          <input
            id="auth-password"
            v-model="form.password"
            :type="showPassword ? 'text' : 'password'"
            :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
            placeholder="至少 8 位"
          />
          <button
            type="button"
            class="password-toggle"
            :aria-label="showPassword ? '隐藏密码' : '显示密码'"
            @click="showPassword = !showPassword"
          >
            <EyeOff v-if="showPassword" :size="18" />
            <Eye v-else :size="18" />
          </button>
        </div>
        <p v-if="form.password && !passwordValid" class="field-error">密码至少需要 8 位</p>
      </div>

      <div v-if="mode === 'register'" class="field">
        <label for="auth-confirm-password">确认密码</label>
        <div class="input-wrap">
          <LockKeyhole :size="18" aria-hidden="true" />
          <input
            id="auth-confirm-password"
            v-model="form.confirmPassword"
            :type="showPassword ? 'text' : 'password'"
            autocomplete="new-password"
            placeholder="再次输入密码"
          />
        </div>
        <p v-if="form.confirmPassword && form.password !== form.confirmPassword" class="field-error">
          两次输入的密码不一致
        </p>
      </div>

      <label v-if="mode === 'register'" class="check-row">
        <input v-model="form.acceptedTerms" type="checkbox" />
        <span>我已阅读并同意服务协议与隐私说明</span>
      </label>
      <label v-else class="check-row">
        <input v-model="form.remember" type="checkbox" />
        <span>在此设备保持登录</span>
      </label>

      <p v-if="auth.error" class="form-error" role="alert">{{ auth.error }}</p>

      <BaseButton class="submit-button" type="submit" :loading="auth.loading" :disabled="!canSubmit">
        {{ submitLabel }}
        <template #icon><ArrowRight :size="18" /></template>
      </BaseButton>
    </form>

    <button
      v-if="mode === 'login' && props.context === 'standard'"
      class="forgot"
      type="button"
      @click="toast.info('找回密码', '当前版本请联系卖家处理')"
    >
      忘记密码？
    </button>
  </div>
</template>

<style scoped>
.auth-form {
  display: grid;
  gap: 0;
}

form {
  display: grid;
  gap: 18px;
  margin-top: 24px;
}

.field {
  display: grid;
  gap: 7px;
}

.field label {
  font-size: 13px;
  font-weight: 680;
}

.input-wrap {
  display: grid;
  min-height: 50px;
  grid-template-columns: 22px 1fr auto;
  align-items: center;
  gap: 8px;
  padding: 0 14px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  color: var(--ink-faint);
  transition: border-color 180ms ease, box-shadow 180ms ease;
}

.input-wrap:focus-within {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px rgb(20 108 99 / 10%);
}

.input-wrap input {
  min-width: 0;
  height: 48px;
  border: 0;
  outline: 0;
  background: transparent;
}

.password-toggle {
  display: grid;
  width: 40px;
  height: 40px;
  place-items: center;
  border-radius: 6px;
  background: transparent;
  color: var(--ink-muted);
}

.field-error,
.form-error {
  color: var(--danger);
  font-size: 12px;
}

.form-error {
  padding: 10px 12px;
  border-left: 3px solid var(--danger);
  background: var(--coral-soft);
}

.check-row {
  display: flex;
  min-height: 32px;
  align-items: center;
  gap: 9px;
  color: var(--ink-muted);
  font-size: 12px;
}

.check-row input {
  width: 17px;
  height: 17px;
  accent-color: var(--primary);
}

.submit-button {
  width: 100%;
}

.forgot {
  min-height: 36px;
  margin-top: 16px;
  background: transparent;
  color: var(--primary);
  font-size: 12px;
  font-weight: 680;
}

@media (max-height: 760px) and (min-width: 841px) {
  form {
    gap: 13px;
    margin-top: 16px;
  }
}

@media (max-width: 480px) and (max-height: 720px) {
  form {
    gap: 14px;
    margin-top: 18px;
  }
}
</style>
