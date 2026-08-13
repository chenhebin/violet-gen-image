<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Eye, EyeOff, LockKeyhole, ShieldCheck } from '@lucide/vue'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '@/components/base/BaseButton.vue'
import FormField from '@/components/base/FormField.vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const showPassword = ref(false)
const touched = reactive({ email: false, password: false })
const form = reactive({
  email: '',
  password: '',
  remember: true,
})

const errors = computed(() => ({
  email: !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)
    ? '请输入有效的邮箱地址'
    : '',
  password: form.password.length < 8 ? '密码至少需要 8 位' : '',
}))

function safeRedirect(): string {
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : ''
  return redirect.startsWith('/manage/') ? redirect : '/manage/dashboard'
}

async function submit(): Promise<void> {
  touched.email = true
  touched.password = true
  if (errors.value.email || errors.value.password) return

  try {
    await auth.login({
      email: form.email.trim(),
      password: form.password,
      remember: form.remember,
    })
    await router.replace(safeRedirect())
  } catch {
    // The store exposes a normalized, user-facing error below the form.
  }
}
</script>

<template>
  <main class="login-view">
    <section class="login-visual" aria-label="映研影像运营工作台">
      <img src="/demo/auth-studio.jpg" alt="" />
      <div class="login-visual__scrim"></div>
      <header class="login-visual__brand">
        <span aria-hidden="true">映</span>
        <div>
          <strong>映研</strong>
          <small>平台运营管理端</small>
        </div>
      </header>
      <div class="login-visual__ledger">
        <p>影像业务运行台账</p>
        <h1>从兑换发放，到每一次成片交付。</h1>
        <ol>
          <li><span>01</span> 管理兑换库存</li>
          <li><span>02</span> 监控模型服务</li>
          <li><span>03</span> 追踪人工交付</li>
        </ol>
      </div>
    </section>

    <section class="login-panel">
      <div class="login-panel__inner">
        <div class="login-panel__heading">
          <span class="login-panel__security">
            <ShieldCheck :size="15" />
            内部运营入口
          </span>
          <h2>登录管理端</h2>
          <p>使用已授权的管理员账号继续。</p>
        </div>

        <form class="login-form" novalidate @submit.prevent="submit">
          <FormField
            label="邮箱"
            for-id="admin-email"
            required
            :error="touched.email ? errors.email : ''"
          >
            <input
              id="admin-email"
              v-model.trim="form.email"
              class="form-control"
              type="email"
              autocomplete="username"
              placeholder="name@yingyan.local"
              @blur="touched.email = true"
              @input="auth.clearError()"
            />
          </FormField>

          <FormField
            label="密码"
            for-id="admin-password"
            required
            :error="touched.password ? errors.password : ''"
          >
            <div class="password-input">
              <LockKeyhole :size="17" aria-hidden="true" />
              <input
                id="admin-password"
                v-model="form.password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                placeholder="输入管理密码"
                @blur="touched.password = true"
                @input="auth.clearError()"
              />
              <button
                type="button"
                :aria-label="showPassword ? '隐藏密码' : '显示密码'"
                @click="showPassword = !showPassword"
              >
                <EyeOff v-if="showPassword" :size="18" />
                <Eye v-else :size="18" />
              </button>
            </div>
          </FormField>

          <label class="remember-field">
            <input v-model="form.remember" type="checkbox" />
            <span>在这台设备上保持登录</span>
          </label>

          <p v-if="auth.error" class="login-form__error" role="alert">
            {{ auth.error.message }}
          </p>

          <BaseButton type="submit" :loading="auth.isLoading" block>
            登录管理端
          </BaseButton>
        </form>
      </div>
      <p class="login-panel__footnote">
        管理操作会记录账号、对象、结果与请求追踪信息。
      </p>
    </section>
  </main>
</template>

<style scoped>
.login-view {
  display: grid;
  width: 100%;
  height: 100dvh;
  overflow: hidden;
  background: var(--surface);
  grid-template-columns: minmax(420px, 46%) 1fr;
}

.login-visual {
  position: relative;
  min-height: 0;
  overflow: hidden;
  background: var(--sidebar);
  color: #fff;
}

.login-visual > img,
.login-visual__scrim {
  position: absolute;
  width: 100%;
  height: 100%;
  inset: 0;
}

.login-visual > img {
  object-fit: cover;
  object-position: center 42%;
}

.login-visual__scrim {
  background:
    linear-gradient(180deg, rgb(18 25 23 / 28%), rgb(18 25 23 / 82%)),
    linear-gradient(90deg, rgb(18 25 23 / 20%), transparent);
}

.login-visual__brand {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 28px 32px;
}

.login-visual__brand > span {
  display: grid;
  width: 38px;
  height: 38px;
  border: 1px solid rgb(255 255 255 / 42%);
  border-radius: var(--radius-sm);
  font-family: var(--font-display);
  font-size: 20px;
  place-items: center;
}

.login-visual__brand strong,
.login-visual__brand small {
  display: block;
}

.login-visual__brand strong {
  font-family: var(--font-display);
  font-size: 20px;
}

.login-visual__brand small {
  margin-top: 2px;
  color: rgb(255 255 255 / 62%);
  font-size: 10px;
}

.login-visual__ledger {
  position: absolute;
  z-index: 1;
  right: 32px;
  bottom: 36px;
  left: 32px;
  padding-left: 20px;
  border-left: 1px solid rgb(255 255 255 / 42%);
}

.login-visual__ledger > p {
  color: rgb(255 255 255 / 66%);
  font-size: 11px;
  font-weight: 700;
}

.login-visual__ledger h1 {
  max-width: 520px;
  margin-top: 9px;
  font-family: var(--font-display);
  font-size: 42px;
  line-height: 1.22;
}

.login-visual__ledger ol {
  display: flex;
  padding: 0;
  margin: 26px 0 0;
  list-style: none;
  gap: 20px;
}

.login-visual__ledger li {
  color: rgb(255 255 255 / 68%);
  font-size: 11px;
}

.login-visual__ledger li span {
  display: block;
  margin-bottom: 4px;
  color: rgb(255 255 255 / 38%);
  font-family: var(--font-mono);
  font-size: 9px;
}

.login-panel {
  display: grid;
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 36px clamp(28px, 7vw, 110px) 22px;
  grid-template-rows: 1fr auto;
}

.login-panel__inner {
  width: min(100%, 420px);
  align-self: center;
  justify-self: center;
}

.login-panel__security {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--primary);
  font-size: 11px;
  font-weight: 750;
}

.login-panel__heading h2 {
  margin-top: 14px;
  font-family: var(--font-display);
  font-size: 30px;
}

.login-panel__heading p {
  margin-top: 6px;
  color: var(--ink-muted);
  font-size: 13px;
}

.login-form {
  display: grid;
  gap: 18px;
  margin-top: 30px;
}

.password-input {
  display: grid;
  height: 44px;
  align-items: center;
  padding-left: 12px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  color: var(--ink-muted);
  grid-template-columns: auto 1fr auto;
}

.password-input:focus-within {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px rgb(35 108 98 / 12%);
}

.password-input input {
  min-width: 0;
  height: 100%;
  padding: 0 10px;
  border: 0;
  outline: 0;
}

.password-input button {
  display: grid;
  width: 43px;
  height: 42px;
  background: transparent;
  color: var(--ink-muted);
  place-items: center;
}

.remember-field {
  display: inline-flex;
  width: fit-content;
  align-items: center;
  gap: 8px;
  color: var(--ink-muted);
  font-size: 12px;
}

.remember-field input {
  width: 16px;
  height: 16px;
  accent-color: var(--primary);
}

.login-form__error {
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  background: var(--danger-soft);
  color: var(--danger);
  font-size: 12px;
}

.login-panel__footnote {
  margin-top: 24px;
  color: var(--ink-faint);
  font-size: 10px;
  text-align: center;
}

@media (max-width: 840px) {
  .login-view {
    overflow: auto;
    grid-template-columns: 1fr;
    grid-template-rows: 180px minmax(0, 1fr);
  }

  .login-visual__brand {
    padding: 20px;
  }

  .login-visual__ledger {
    right: 20px;
    bottom: 16px;
    left: 20px;
  }

  .login-visual__ledger > p,
  .login-visual__ledger ol {
    display: none;
  }

  .login-visual__ledger h1 {
    max-width: 440px;
    margin: 0;
    font-size: 23px;
  }

  .login-panel {
    overflow: visible;
    padding: 28px 20px 20px;
  }
}

@media (max-height: 720px) and (min-width: 841px) {
  .login-panel {
    padding-top: 20px;
  }

  .login-form {
    gap: 12px;
    margin-top: 20px;
  }
}
</style>
