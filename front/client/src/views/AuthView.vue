<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AuthForm, { type AuthMode } from '@/components/auth/AuthForm.vue'
import { useToast } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const mode = ref<AuthMode>('login')

async function authenticated(mode: AuthMode): Promise<void> {
  if (mode === 'login') toast.success('登录成功', '工作台已恢复')
  else toast.success('账号已创建', '使用兑换码即可开始创作')
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/app/create'
  await router.replace(redirect)
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

    <section
      class="auth-panel"
      :class="{ 'is-register': mode === 'register' }"
      aria-labelledby="auth-title"
    >
      <div class="brand"><span>映</span><strong>映研</strong></div>
      <div class="auth-copy">
        <p>私人影像工作室</p>
        <h1 id="auth-title">{{ mode === 'login' ? '继续你的创作' : '创建一个工作账号' }}</h1>
      </div>
      <AuthForm @mode-changed="mode = $event" @authenticated="authenticated" />
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
  overflow: hidden;
  background: var(--surface);
}

.studio-image {
  position: relative;
  height: 100%;
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

.brand strong { font-size: 18px; }

.auth-copy { margin: 46px 0 24px; }
.auth-copy p { color: var(--primary); font-size: 12px; font-weight: 720; }
.auth-copy h1 {
  margin-top: 7px;
  font-family: 'Songti SC', 'STSong', serif;
  font-size: clamp(30px, 3vw, 40px);
  font-weight: 600;
  line-height: 1.2;
}

@media (max-height: 760px) and (min-width: 841px) {
  .auth-panel { justify-content: flex-start; padding-block: 22px; }
  .auth-copy { margin: 24px 0 16px; }
}

@media (max-width: 840px) {
  .auth-page {
    position: relative;
    isolation: isolate;
    display: block;
    background: #161a1c;
  }

  .studio-image {
    position: absolute;
    z-index: 0;
    inset: 0;
    display: block;
    height: 100%;
  }

  .studio-image img {
    object-position: center 34%;
    animation: auth-image-settle 900ms var(--ease-out) both;
  }

  .studio-image::after {
    background:
      linear-gradient(180deg, rgb(10 13 15 / 68%) 0%, rgb(10 13 15 / 16%) 34%, rgb(10 13 15 / 28%) 52%, rgb(10 13 15 / 88%) 100%);
  }

  .image-caption { display: none; }

  .auth-panel {
    position: relative;
    z-index: 1;
    width: min(100%, 520px);
    height: 100dvh;
    max-height: none;
    justify-content: flex-start;
    padding: calc(20px + env(safe-area-inset-top)) 16px calc(14px + env(safe-area-inset-bottom));
    margin: 0 auto;
    overflow-y: auto;
    color: #fff;
    animation: auth-content-in 620ms var(--ease-out) both;
  }

  .brand > span {
    background: rgb(255 255 255 / 94%);
    color: #171b1d;
    box-shadow: 0 10px 28px rgb(4 7 8 / 18%);
  }

  .brand strong {
    color: #fff;
    text-shadow: 0 1px 16px rgb(0 0 0 / 24%);
  }

  .auth-copy {
    margin: 24px 2px 18px;
    text-shadow: 0 2px 18px rgb(0 0 0 / 30%);
  }

  .auth-copy p { color: rgb(222 241 237 / 88%); }

  .auth-copy h1 {
    max-width: 340px;
    color: #fff;
    font-size: 34px;
    line-height: 1.12;
    text-wrap: balance;
  }

  .auth-panel.is-register .auth-copy {
    margin-block: 18px 14px;
  }

  .auth-panel :deep(.auth-form) {
    width: 100%;
    flex: 0 0 auto;
    padding: 18px;
    margin-top: auto;
    border: 1px solid rgb(255 255 255 / 46%);
    border-radius: var(--radius-md);
    background: rgb(247 249 248 / 91%);
    box-shadow:
      0 24px 64px rgb(5 10 11 / 30%),
      inset 0 1px 0 rgb(255 255 255 / 70%);
    color: var(--ink);
    backdrop-filter: blur(20px) saturate(116%);
    animation: auth-sheet-in 720ms 80ms var(--ease-out) both;
  }

  .auth-panel :deep(.auth-form .segmented) {
    width: 100%;
    border-color: rgb(39 53 54 / 12%);
    background: rgb(25 40 41 / 7%);
  }

  .auth-panel :deep(.auth-form .segmented button.active) {
    background: rgb(255 255 255 / 88%);
    box-shadow: 0 1px 8px rgb(21 31 32 / 9%);
  }

  .auth-panel :deep(.auth-form form) {
    gap: 14px;
    margin-top: 16px;
  }

  .auth-panel :deep(.auth-form .input-wrap) {
    min-height: 48px;
    border-color: rgb(43 61 62 / 18%);
    background: rgb(255 255 255 / 58%);
  }

  .auth-panel :deep(.auth-form .input-wrap:focus-within) {
    border-color: var(--primary);
    background: rgb(255 255 255 / 80%);
    box-shadow: 0 0 0 3px rgb(20 108 99 / 12%);
  }

  .auth-panel :deep(.auth-form .input-wrap input) { height: 46px; }

  .auth-panel :deep(.auth-form .submit-button) {
    min-height: 48px;
    background: #185f57;
    box-shadow: 0 10px 24px rgb(13 82 75 / 18%);
  }

  .auth-panel :deep(.auth-form .forgot) {
    margin-top: 10px;
    color: #145f58;
  }
}

@media (max-width: 480px) {
  .auth-panel {
    width: 100%;
    padding-inline: 14px;
    border: 0;
    border-radius: 0;
  }

  .auth-copy { margin-inline: 2px; }

  .auth-panel :deep(.auth-form) { padding: 17px; }
}

@media (max-width: 480px) and (max-height: 660px) {
  .auth-panel {
    padding-top: calc(12px + env(safe-area-inset-top));
    padding-bottom: calc(10px + env(safe-area-inset-bottom));
  }

  .brand > span {
    width: 30px;
    height: 32px;
  }

  .auth-copy,
  .auth-panel.is-register .auth-copy {
    margin-block: 12px 10px;
  }

  .auth-copy h1 { font-size: 28px; }

  .auth-panel :deep(.auth-form) { padding: 13px 15px; }

  .auth-panel :deep(.auth-form form) {
    gap: 10px;
    margin-top: 12px;
  }

  .auth-panel :deep(.auth-form .input-wrap) { min-height: 44px; }
  .auth-panel :deep(.auth-form .input-wrap input) { height: 42px; }
  .auth-panel :deep(.auth-form .check-row) { min-height: 28px; }
  .auth-panel :deep(.auth-form .submit-button) { min-height: 44px; }
  .auth-panel :deep(.auth-form .forgot) { min-height: 30px; margin-top: 6px; }
}

@keyframes auth-content-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes auth-image-settle {
  from { transform: scale(1.035); }
  to { transform: scale(1); }
}

@keyframes auth-sheet-in {
  from { opacity: 0; transform: translateY(18px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
