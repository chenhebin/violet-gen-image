<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowRight,
  CheckCircle2,
  KeyRound,
  LoaderCircle,
  LogOut,
  RefreshCw,
  ShieldCheck,
} from '@lucide/vue'
import AuthForm, { type AuthMode } from '@/components/auth/AuthForm.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { entitlementApi } from '@/services/api'
import {
  clearPendingClaim,
  isClaimCodeFormatValid,
  maskClaimCode,
  readClaimIdempotencyKey,
  readPendingClaimCode,
  saveClaimIdempotencyKey,
  savePendingClaimCode,
} from '@/services/claim-session'
import { createIdempotencyKey } from '@/services/http'
import { useAuthStore } from '@/stores/auth'
import { useEntitlementStore } from '@/stores/entitlement'
import { AppError, ErrorCode } from '@/types/api'
import type { RedemptionPreview, RedemptionResult } from '@/types/domain'

type ClaimState =
  | 'initializing'
  | 'missing_code'
  | 'invalid_format'
  | 'unauthenticated'
  | 'authenticated'
  | 'claiming'
  | 'success'
  | 'invalid'
  | 'used'
  | 'expired'
  | 'network_error'
  | 'service_error'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const entitlement = useEntitlementStore()

const state = ref<ClaimState>('initializing')
const pendingCode = ref('')
const manualCode = ref('')
const preview = ref<RedemptionPreview | null>(null)
const result = ref<RedemptionResult | null>(null)
const authMode = ref<AuthMode>('register')
const accountCreated = ref(false)
const usedByCurrentUser = ref(false)
const lastAction = ref<'preview' | 'claim'>('preview')

const maskedCode = computed(() => preview.value?.maskedCode || maskClaimCode(pendingCode.value))
const credits = computed(() => preview.value?.credits ?? null)
const creditText = computed(() => credits.value ? `${credits.value} 次` : '本兑换码额度')
const errorCopy = computed(() => {
  if (state.value === 'invalid') return ['兑换码无效', '请核对闲鱼消息中的备用兑换码，或联系卖家处理。']
  if (state.value === 'used' && usedByCurrentUser.value) {
    return ['你已领取过这个兑换码', '次数已经在当前账号中，无需重复领取。']
  }
  if (state.value === 'used') return ['兑换码已使用', '该兑换码不能重复领取。如有疑问，请联系卖家核对订单。']
  if (state.value === 'expired') return ['兑换码已过期', '请联系卖家确认订单并更换可用兑换码。']
  if (state.value === 'network_error') return ['网络连接中断', '领取信息仍保留在当前页面，可以直接重试。']
  return ['服务暂不可用', '领取信息仍已保留，请稍后重新尝试。']
})

function isNetworkError(error: unknown): boolean {
  return error instanceof AppError && typeof error.details === 'string' && /network|timeout|fetch/i.test(error.details)
}

function clearAndSet(nextState: ClaimState): void {
  clearPendingClaim()
  pendingCode.value = ''
  state.value = nextState
}

function wasClaimedByCurrentUser(error: AppError): boolean {
  return (
    error.code === ErrorCode.CodeUsed &&
    typeof error.details === 'object' &&
    error.details !== null &&
    'claimedByCurrentUser' in error.details &&
    error.details.claimedByCurrentUser === true
  )
}

async function handleClaimError(error: unknown): Promise<void> {
  if (error instanceof AppError) {
    if (error.code === ErrorCode.AuthRequired) {
      auth.invalidateSession()
      entitlement.reset()
      authMode.value = 'login'
      state.value = 'unauthenticated'
      return
    }
    if (wasClaimedByCurrentUser(error)) {
      clearPendingClaim()
      pendingCode.value = ''
      usedByCurrentUser.value = true
      try {
        await entitlement.load()
      } catch {
        // The already-claimed state remains useful even when balance refresh fails.
      }
      state.value = 'used'
      return
    }
    if (error.code === ErrorCode.CodeInvalid || error.code === ErrorCode.ProductMismatch) {
      clearAndSet('invalid')
      return
    }
    if (error.code === ErrorCode.CodeUsed) {
      clearAndSet('used')
      return
    }
    if (error.code === ErrorCode.CodeExpired) {
      clearAndSet('expired')
      return
    }
  }
  state.value = isNetworkError(error) ? 'network_error' : 'service_error'
}

async function initialize(): Promise<void> {
  state.value = 'initializing'
  const queryCode = typeof route.query.code === 'string' ? route.query.code : ''
  if (queryCode) {
    pendingCode.value = savePendingClaimCode(queryCode)
    await router.replace({ name: 'claim' })
  } else {
    pendingCode.value = readPendingClaimCode()
  }
  if (!pendingCode.value) {
    state.value = 'missing_code'
    return
  }
  if (!isClaimCodeFormatValid(pendingCode.value)) {
    clearAndSet('invalid_format')
    return
  }
  await previewCode()
}

async function previewCode(): Promise<void> {
  lastAction.value = 'preview'
  state.value = 'initializing'
  try {
    preview.value = await entitlementApi.previewRedemption(pendingCode.value)
    state.value = auth.isAuthenticated ? 'authenticated' : 'unauthenticated'
  } catch (error) {
    await handleClaimError(error)
  }
}

async function continueWithManualCode(): Promise<void> {
  if (!isClaimCodeFormatValid(manualCode.value)) {
    state.value = 'invalid_format'
    return
  }
  pendingCode.value = savePendingClaimCode(manualCode.value)
  manualCode.value = ''
  result.value = null
  preview.value = null
  await previewCode()
}

async function claim(): Promise<void> {
  if (!pendingCode.value || state.value === 'claiming') return
  lastAction.value = 'claim'
  state.value = 'claiming'
  const key = readClaimIdempotencyKey() || createIdempotencyKey('claim_flow')
  saveClaimIdempotencyKey(key)
  try {
    result.value = await entitlement.redeem(pendingCode.value, key)
    clearPendingClaim()
    pendingCode.value = ''
    state.value = 'success'
  } catch (error) {
    await handleClaimError(error)
  }
}

async function authenticated(mode: AuthMode): Promise<void> {
  accountCreated.value = mode === 'register'
  if (!auth.user) {
    state.value = 'service_error'
    return
  }
  await claim()
}

async function switchAccount(): Promise<void> {
  try {
    await auth.logout()
  } finally {
    entitlement.reset()
    authMode.value = 'login'
    state.value = 'unauthenticated'
  }
}

async function retry(): Promise<void> {
  if (lastAction.value === 'claim' && auth.isAuthenticated) await claim()
  else await previewCode()
}

function enterAnotherCode(): void {
  clearPendingClaim()
  pendingCode.value = ''
  preview.value = null
  result.value = null
  manualCode.value = ''
  state.value = 'missing_code'
}

onMounted(() => void initialize())
</script>

<template>
  <main class="claim-page">
    <header class="claim-header">
      <RouterLink class="brand" to="/app/create" aria-label="映研首页">
        <span>映</span><strong>映研</strong>
      </RouterLink>
      <span class="secure-note"><ShieldCheck :size="15" /> 安全领取</span>
    </header>

    <section class="claim-shell" aria-live="polite">
      <div class="claim-intro">
        <p class="eyebrow">Xianyu order claim</p>
        <h1>把购买次数<br />领取到你的账号</h1>
        <p>兑换资格只会绑定到你确认的映研账号。关闭当前标签页后，待领取信息会自动清除。</p>
        <div v-if="maskedCode" class="claim-ticket">
          <span>待领取兑换码</span>
          <strong>{{ maskedCode }}</strong>
          <small v-if="credits">{{ preview?.productName }} · {{ credits }} 次</small>
          <small v-else>实际额度以服务端核销结果为准</small>
        </div>
      </div>

      <div class="claim-panel">
        <div v-if="state === 'initializing' || state === 'claiming'" class="state-panel loading-state">
          <LoaderCircle class="spin" :size="30" aria-hidden="true" />
          <h2>{{ state === 'claiming' ? '正在领取次数' : '正在检查领取信息' }}</h2>
          <p>{{ state === 'claiming' ? '正在核验兑换码并写入账号，请不要重复提交。' : '正在确认兑换码和当前登录状态。' }}</p>
        </div>

        <div v-else-if="state === 'missing_code' || state === 'invalid_format'" class="state-panel manual-state">
          <div class="state-icon"><KeyRound :size="22" /></div>
          <p class="panel-kicker">备用领取方式</p>
          <h2>{{ state === 'invalid_format' ? '兑换码格式不正确' : '输入备用兑换码' }}</h2>
          <p>从闲鱼自动发货消息中复制“备用兑换码”，完整粘贴到下方。</p>
          <form @submit.prevent="continueWithManualCode">
            <label for="claim-code">兑换码</label>
            <input id="claim-code" v-model.trim="manualCode" autocomplete="off" spellcheck="false" placeholder="YY-XXXX-XXXX-XXXX" />
            <BaseButton type="submit" :disabled="!manualCode.trim()">
              继续领取 <ArrowRight :size="17" />
            </BaseButton>
          </form>
        </div>

        <div v-else-if="state === 'unauthenticated'" class="auth-state">
          <div class="panel-heading">
            <p class="panel-kicker">登录状态未恢复</p>
            <h2>{{ authMode === 'register' ? '注册后自动领取' : '登录后自动领取' }} {{ creditText }}</h2>
            <p>认证完成后会继续当前领取流程，不需要再次粘贴兑换码。</p>
          </div>
          <AuthForm
            context="claim"
            :credits="credits"
            :initial-mode="authMode"
            @mode-changed="authMode = $event"
            @authenticated="authenticated"
          />
        </div>

        <div v-else-if="state === 'authenticated'" class="state-panel account-state">
          <div class="state-icon"><ShieldCheck :size="22" /></div>
          <p class="panel-kicker">确认领取账号</p>
          <h2>领取 {{ creditText }} 到此账号</h2>
          <p>请先确认邮箱。兑换成功后，次数不能转移到其他账号。</p>
          <div class="account-row">
            <span>额度将领取到</span>
            <strong>{{ auth.user?.email }}</strong>
          </div>
          <BaseButton class="wide-button" @click="claim">领取到此账号 <ArrowRight :size="17" /></BaseButton>
          <button class="text-action" type="button" @click="switchAccount"><LogOut :size="15" /> 不是我的账号，切换账号</button>
        </div>

        <div v-else-if="state === 'success'" class="state-panel success-state">
          <div class="success-mark"><CheckCircle2 :size="28" /></div>
          <p class="panel-kicker">领取完成</p>
          <h2>已增加 {{ result?.added }} 次</h2>
          <p>次数已经写入当前账号，可以直接开始创作。</p>
          <div class="balance-row"><span>当前剩余次数</span><strong>{{ result?.entitlement.balance }}</strong></div>
          <BaseButton class="wide-button" @click="router.push('/app/create')">开始创作 <ArrowRight :size="17" /></BaseButton>
        </div>

        <div v-else class="state-panel error-state">
          <div class="error-mark">{{ state === 'network_error' || state === 'service_error' ? '暂' : '!' }}</div>
          <p v-if="accountCreated" class="account-created">账号已创建，领取暂未完成</p>
          <h2>{{ errorCopy[0] }}</h2>
          <p>{{ errorCopy[1] }}</p>
          <div v-if="state === 'used' && usedByCurrentUser && entitlement.entitlement" class="balance-row">
            <span>当前剩余次数</span><strong>{{ entitlement.balance }}</strong>
          </div>
          <BaseButton v-if="state === 'used' && usedByCurrentUser" class="wide-button" @click="router.push('/app/create')">
            开始创作 <ArrowRight :size="17" />
          </BaseButton>
          <BaseButton v-if="state === 'network_error' || state === 'service_error'" class="wide-button" @click="retry">
            <RefreshCw :size="16" /> 重新领取
          </BaseButton>
          <BaseButton v-else-if="!(state === 'used' && usedByCurrentUser)" class="wide-button" variant="secondary" @click="enterAnotherCode">输入备用兑换码</BaseButton>
          <a v-if="!(state === 'used' && usedByCurrentUser)" class="text-action" href="https://www.goofish.com" rel="noreferrer">联系卖家处理</a>
        </div>
      </div>
    </section>

    <footer>映研私人影像工作室 · 兑换结果以服务端记录为准</footer>
  </main>
</template>

<style scoped>
.claim-page {
  min-height: 100dvh;
  padding: 0 clamp(18px, 4vw, 72px);
  overflow-x: hidden;
  background:
    linear-gradient(90deg, rgb(20 108 99 / 4%) 1px, transparent 1px) 0 0 / 72px 72px,
    var(--canvas);
}

.claim-header {
  display: flex;
  max-width: 1180px;
  min-height: 72px;
  align-items: center;
  justify-content: space-between;
  margin: 0 auto;
}

.brand { display: flex; align-items: center; gap: 10px; }
.brand > span {
  display: grid;
  width: 31px;
  height: 35px;
  place-items: center;
  border-radius: 5px 5px 7px 7px;
  background: var(--ink);
  color: #fff;
  font-family: 'Songti SC', serif;
}
.brand strong { font-size: 17px; }
.secure-note { display: flex; align-items: center; gap: 6px; color: var(--primary); font-size: 12px; font-weight: 700; }

.claim-shell {
  display: grid;
  width: min(100%, 1080px);
  min-height: calc(100dvh - 132px);
  grid-template-columns: minmax(0, 0.86fr) minmax(420px, 0.74fr);
  align-items: center;
  gap: clamp(56px, 8vw, 120px);
  padding: 40px 0 76px;
  margin: 0 auto;
}

.claim-intro { align-self: center; }
.eyebrow, .panel-kicker { color: var(--primary); font-size: 11px; font-weight: 760; text-transform: uppercase; }
.claim-intro h1 {
  margin: 14px 0 24px;
  font-family: 'Songti SC', 'STSong', serif;
  font-size: clamp(38px, 4.4vw, 62px);
  font-weight: 600;
  line-height: 1.12;
}
.claim-intro > p:last-of-type { max-width: 470px; color: var(--ink-muted); font-size: 14px; line-height: 1.8; }

.claim-ticket {
  display: grid;
  gap: 5px;
  max-width: 420px;
  padding: 20px 22px;
  border: 1px solid var(--border);
  border-left: 3px solid var(--primary);
  border-radius: 6px 8px 8px 6px;
  margin-top: 34px;
  background: rgb(255 255 255 / 72%);
  box-shadow: 0 14px 36px rgb(26 48 44 / 6%);
}
.claim-ticket span, .claim-ticket small { color: var(--ink-faint); font-size: 11px; }
.claim-ticket strong { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 17px; letter-spacing: 0; }

.claim-panel {
  min-height: 520px;
  padding: clamp(24px, 4vw, 42px);
  border: 1px solid rgb(255 255 255 / 76%);
  border-radius: 8px;
  background: rgb(255 255 255 / 94%);
  box-shadow: 0 26px 70px rgb(24 48 43 / 12%);
}

.state-panel, .auth-state { display: flex; min-height: 434px; flex-direction: column; justify-content: center; }
.state-panel h2, .panel-heading h2 { margin: 9px 0 10px; font-family: 'Songti SC', 'STSong', serif; font-size: 28px; font-weight: 600; line-height: 1.3; }
.state-panel > p, .panel-heading > p:last-child { color: var(--ink-muted); font-size: 13px; line-height: 1.75; }
.state-icon { display: grid; width: 46px; height: 46px; place-items: center; border-radius: 8px; margin-bottom: 24px; background: var(--teal-soft); color: var(--primary); }
.loading-state { align-items: center; text-align: center; }
.loading-state h2 { margin-top: 22px; }
.spin { color: var(--primary); animation: spin 900ms linear infinite; }

.manual-state form { display: grid; gap: 10px; margin-top: 26px; }
.manual-state label { font-size: 12px; font-weight: 700; }
.manual-state input {
  width: 100%;
  min-height: 50px;
  padding: 0 14px;
  border: 1px solid var(--border-strong);
  border-radius: 7px;
  outline: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.manual-state input:focus { border-color: var(--primary); box-shadow: 0 0 0 3px rgb(20 108 99 / 10%); }
.panel-heading { margin-bottom: 23px; }
.account-row, .balance-row {
  display: grid;
  gap: 5px;
  padding: 17px 18px;
  border-radius: 7px;
  margin: 25px 0 18px;
  background: var(--canvas);
}
.account-row span, .balance-row span { color: var(--ink-faint); font-size: 11px; }
.account-row strong { font-size: 15px; overflow-wrap: anywhere; }
.balance-row { grid-template-columns: 1fr auto; align-items: center; }
.balance-row strong { color: var(--primary); font-size: 32px; font-variant-numeric: tabular-nums; }
.wide-button { width: 100%; margin-top: 18px; }
.text-action { display: flex; min-height: 40px; align-items: center; justify-content: center; gap: 7px; margin-top: 10px; background: transparent; color: var(--primary); font-size: 12px; font-weight: 680; }
.success-mark { display: grid; width: 56px; height: 56px; place-items: center; border-radius: 50%; margin-bottom: 25px; background: var(--teal-soft); color: var(--primary); animation: stamp 320ms cubic-bezier(.2,.8,.2,1); }
.error-mark { display: grid; width: 50px; height: 50px; place-items: center; border-radius: 8px; margin-bottom: 24px; background: var(--coral-soft); color: var(--danger); font-family: 'Songti SC', serif; font-size: 22px; font-weight: 700; }
.account-created { align-self: flex-start; padding: 7px 10px; border-radius: 5px; margin-bottom: 8px; background: var(--teal-soft); color: var(--primary) !important; font-weight: 700; }

footer { min-height: 60px; color: var(--ink-faint); font-size: 10px; text-align: center; }

@keyframes spin { to { transform: rotate(360deg); } }
@keyframes stamp { from { opacity: 0; transform: scale(.72) rotate(-8deg); } to { opacity: 1; transform: scale(1) rotate(0); } }

@media (max-width: 840px) {
  .claim-page { padding-inline: 18px; }
  .claim-header { min-height: 62px; }
  .claim-shell { min-height: 0; grid-template-columns: 1fr; gap: 26px; padding: 30px 0 44px; }
  .claim-intro h1 { margin: 10px 0 14px; font-size: clamp(35px, 10vw, 48px); }
  .claim-intro > p:last-of-type { font-size: 13px; }
  .claim-ticket { margin-top: 22px; }
  .claim-panel { min-height: 0; padding: 24px 20px; }
  .state-panel, .auth-state { min-height: 0; padding-block: 12px; }
  .state-panel h2, .panel-heading h2 { font-size: 25px; }
}

@media (max-width: 420px) {
  .claim-intro h1 br { display: none; }
  .secure-note { font-size: 11px; }
  .claim-ticket { padding: 17px; }
  .claim-panel { margin-inline: -4px; }
}
</style>
