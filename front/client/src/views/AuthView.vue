<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowRight,
  Eye,
  EyeOff,
  Image,
  LockKeyhole,
  Mail,
} from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import SegmentedControl from '@/components/base/SegmentedControl.vue'
import { useToast } from '@/composables/useToast'
import { AUTH_CONFIG, DEMO_ACCOUNT } from '@/config'
import { useAuthStore } from '@/stores/auth'

type AuthMode = 'login' | 'register'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const toast = useToast()
const mode = ref<AuthMode>('login')
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
const passwordValid = computed(
  () => form.password.length >= AUTH_CONFIG.minimumPasswordLength,
)
const registerValid = computed(
  () =>
    emailValid.value &&
    passwordValid.value &&
    form.password === form.confirmPassword &&
    form.acceptedTerms,
)
const canSubmit = computed(() =>
  mode.value === 'login'
    ? emailValid.value && passwordValid.value
    : registerValid.value,
)

async function submit(): Promise<void> {
  if (!canSubmit.value) return
  try {
    if (mode.value === 'login') {
      await auth.login({
        email: form.email,
        password: form.password,
        remember: form.remember,
      })
      toast.success('登录成功', '工作台已恢复')
    } else {
      await auth.register({
        email: form.email,
        password: form.password,
        remember: form.remember,
        acceptedTerms: form.acceptedTerms,
      })
      toast.success('账号已创建', '使用兑换码即可开始创作')
    }
    const redirect =
      typeof route.query.redirect === 'string'
        ? route.query.redirect
        : '/app/create'
    await router.replace(redirect)
  } catch {
    // Store exposes the normalized API message beside the form.
  }
}

function useDemo(): void {
  mode.value = 'login'
  form.email = DEMO_ACCOUNT.email
  form.password = DEMO_ACCOUNT.password
  touchedEmail.value = true
}
</script>

<template>
  <main class="auth-page">
    <div class="studio-image" aria-hidden="true">
      <img src="/demo/auth-studio.jpg" alt="" />
      <div class="image-caption">
        <span>影像校样 · 07/29</span>
        <p>从一张原图，到可以保存的成片。</p>
      </div>
    </div>

    <section class="auth-panel" aria-labelledby="auth-title">
      <div class="brand">
        <span>映</span>
        <strong>映研</strong>
      </div>

      <div class="auth-copy">
        <p>私人影像工作室</p>
        <h1 id="auth-title">
          {{ mode === 'login' ? '继续你的创作' : '创建一个工作账号' }}
        </h1>
      </div>

      <SegmentedControl
        v-model="mode"
        label="登录或注册"
        :options="[
          { value: 'login', label: '登录' },
          { value: 'register', label: '注册' },
        ]"
      />

      <form novalidate @submit.prevent="submit">
        <div class="field">
          <label for="email">邮箱</label>
          <div class="input-wrap">
            <Mail :size="18" aria-hidden="true" />
            <input
              id="email"
              v-model.trim="form.email"
              type="email"
              autocomplete="email"
              placeholder="name@example.com"
              @blur="touchedEmail = true"
            />
          </div>
          <p v-if="touchedEmail && !emailValid" class="field-error">
            请输入有效的邮箱地址
          </p>
        </div>

        <div class="field">
          <label for="password">密码</label>
          <div class="input-wrap">
            <LockKeyhole :size="18" aria-hidden="true" />
            <input
              id="password"
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
          <p v-if="form.password && !passwordValid" class="field-error">
            密码至少需要 8 位
          </p>
        </div>

        <div v-if="mode === 'register'" class="field">
          <label for="confirm-password">确认密码</label>
          <div class="input-wrap">
            <LockKeyhole :size="18" aria-hidden="true" />
            <input
              id="confirm-password"
              v-model="form.confirmPassword"
              :type="showPassword ? 'text' : 'password'"
              autocomplete="new-password"
              placeholder="再次输入密码"
            />
          </div>
          <p
            v-if="form.confirmPassword && form.password !== form.confirmPassword"
            class="field-error"
          >
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

        <BaseButton
          class="submit-button"
          type="submit"
          :loading="auth.loading"
          :disabled="!canSubmit"
        >
          {{ mode === 'login' ? '进入工作台' : '创建账号' }}
          <template #icon><ArrowRight :size="18" /></template>
        </BaseButton>
      </form>

      <div class="demo-access">
        <div>
          <Image :size="17" />
          <span>仅查看完整演示</span>
        </div>
        <button type="button" @click="useDemo">填入演示账号</button>
      </div>

      <button
        v-if="mode === 'login'"
        class="forgot"
        type="button"
        @click="toast.info('找回密码', '当前版本请联系卖家处理')"
      >
        忘记密码？
      </button>
    </section>
  </main>
</template>

<style scoped>
.auth-page {
  display: grid;
  width: 100%;
  height: 100dvh;
  min-height: 0;
  grid-template-columns: minmax(0, 1.2fr) minmax(420px, 0.8fr);
  grid-template-rows: minmax(0, 1fr);
  overflow: hidden;
  background: var(--surface);
}

.studio-image {
  position: relative;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: var(--ink);
}

.studio-image img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center 38%;
}

.studio-image::after {
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, transparent 55%, rgb(16 20 22 / 72%));
  content: '';
}

.image-caption {
  position: absolute;
  z-index: 1;
  right: 48px;
  bottom: 44px;
  left: 48px;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 32px;
  color: white;
}

.image-caption span {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
}

.image-caption p {
  max-width: 330px;
  font-family: 'Songti SC', 'STSong', serif;
  font-size: 24px;
  line-height: 1.35;
}

.auth-panel {
  display: flex;
  width: min(100%, 510px);
  height: 100%;
  min-height: 0;
  flex-direction: column;
  justify-content: center;
  padding: 44px clamp(32px, 5vw, 74px);
  margin: auto;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
}

.brand > span {
  display: grid;
  width: 32px;
  height: 36px;
  place-items: center;
  border-radius: 5px 5px 7px 7px;
  background: var(--ink);
  color: white;
  font-family: 'Songti SC', serif;
}

.brand strong {
  font-size: 18px;
}

.auth-copy {
  margin: 46px 0 24px;
}

.auth-copy p {
  color: var(--primary);
  font-size: 12px;
  font-weight: 720;
}

.auth-copy h1 {
  margin-top: 7px;
  font-family: 'Songti SC', 'STSong', serif;
  font-size: clamp(30px, 3vw, 40px);
  font-weight: 600;
  line-height: 1.2;
}

.auth-panel > .segmented {
  width: 100%;
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

.demo-access {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 0;
  margin-top: 20px;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  font-size: 12px;
}

.demo-access div {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--ink-muted);
}

.demo-access button,
.forgot {
  min-height: 36px;
  background: transparent;
  color: var(--primary);
  font-size: 12px;
  font-weight: 680;
}

.forgot {
  align-self: center;
  margin-top: 8px;
}

@media (max-width: 840px) {
  .auth-page {
    grid-template-columns: 1fr;
    place-items: center;
    background:
      linear-gradient(rgb(18 22 24 / 64%), rgb(18 22 24 / 64%)),
      url('/demo/auth-studio.jpg') center / cover;
  }

  .studio-image {
    display: none;
  }

  .auth-panel {
    width: min(calc(100% - 28px), 500px);
    height: auto;
    max-height: calc(100dvh - 48px);
    padding: 28px;
    border: 1px solid rgb(255 255 255 / 38%);
    border-radius: var(--radius-md);
    margin-block: 24px;
    background: rgb(255 255 255 / 96%);
    box-shadow: var(--shadow-md);
  }
}

</style>

<style scoped src="../styles/auth-responsive.css"></style>
