<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { BadgeCheck, Ticket } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import { useToast } from '@/composables/useToast'
import { MOCK_REDEMPTION_CODES } from '@/config'
import { useEntitlementStore } from '@/stores/entitlement'

const props = defineProps<{ open: boolean }>()
defineEmits<{ close: [] }>()
const entitlement = useEntitlementStore()
const toast = useToast()

const code = ref('')
const loading = ref(false)
const error = ref('')
const success = ref<{ added: number; balance: number } | null>(null)
const normalized = computed(() => code.value.trim().toUpperCase())

watch(
  () => props.open,
  (open) => {
    if (!open) return
    code.value = ''
    error.value = ''
    success.value = null
  },
)

async function redeem(): Promise<void> {
  if (!normalized.value) {
    error.value = '请输入兑换码'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const result = await entitlement.redeem(normalized.value)
    success.value = {
      added: result.added,
      balance: result.entitlement.balance,
    }
    toast.success('兑换成功', `已增加 ${result.added} 次`)
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : '兑换失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <BaseModal
    :open="open"
    title="兑换使用次数"
    description="使用卖家发给你的兑换码，次数会绑定到当前账号。"
    size="small"
    @close="$emit('close')"
  >
    <div v-if="success" class="redeem-success">
      <div class="stamp" aria-hidden="true">
        <BadgeCheck :size="34" />
        <span>已兑换</span>
      </div>
      <h3>已增加 {{ success.added }} 次</h3>
      <p>当前剩余 {{ success.balance }} 次，可立即开始创作。</p>
      <BaseButton @click="$emit('close')">返回工作台</BaseButton>
    </div>
    <form v-else class="redeem-form" @submit.prevent="redeem">
      <label for="redeem-code">兑换码</label>
      <div class="code-field">
        <Ticket :size="19" aria-hidden="true" />
        <input
          id="redeem-code"
          v-model="code"
          autocomplete="off"
          placeholder="例如 YINGYAN-START-10"
          spellcheck="false"
          @input="error = ''"
        />
      </div>
      <p v-if="error" class="field-error" role="alert">{{ error }}</p>
      <BaseButton type="submit" :loading="loading" :disabled="!normalized">
        兑换次数
      </BaseButton>
      <p class="rules">
        每个兑换码只能使用一次。同一账号可以继续兑换新码，已领取次数不会因更换浏览器而丢失。
      </p>
      <details>
        <summary>查看演示兑换码</summary>
        <code
          v-for="item in MOCK_REDEMPTION_CODES.filter(
            (candidate) => candidate.state === 'active',
          )"
          :key="item.code"
        >
          {{ item.code }}
        </code>
      </details>
    </form>
  </BaseModal>
</template>

<style scoped>
.redeem-form {
  display: grid;
  gap: 12px;
}

label {
  font-size: 14px;
  font-weight: 680;
}

.code-field {
  display: grid;
  min-height: 50px;
  grid-template-columns: 24px 1fr;
  align-items: center;
  gap: 8px;
  padding: 0 14px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
}

.code-field:focus-within {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px rgb(20 108 99 / 10%);
}

input {
  min-width: 0;
  height: 48px;
  border: 0;
  outline: 0;
  background: transparent;
  font-size: 14px;
  text-transform: uppercase;
}

.field-error {
  color: var(--danger);
  font-size: 13px;
}

.rules {
  color: var(--ink-muted);
  font-size: 12px;
}

details {
  padding-top: 6px;
  color: var(--ink-muted);
  font-size: 12px;
}

details code {
  display: block;
  margin-top: 8px;
  color: var(--primary);
}

.redeem-success {
  display: grid;
  justify-items: center;
  padding: 8px 0 4px;
  text-align: center;
}

.stamp {
  display: grid;
  width: 112px;
  height: 112px;
  place-items: center;
  border: 3px double var(--coral);
  border-radius: 50%;
  color: var(--coral);
  transform: rotate(-7deg);
  animation: stamp-in 300ms var(--ease-out) both;
}

.stamp span {
  margin-top: -24px;
  font-family: 'Songti SC', serif;
  font-size: 15px;
  font-weight: 700;
}

.redeem-success h3 {
  margin-top: 22px;
  font-size: 22px;
}

.redeem-success p {
  margin: 6px 0 22px;
  color: var(--ink-muted);
  font-size: 14px;
}

@keyframes stamp-in {
  from {
    opacity: 0;
    transform: scale(1.35) rotate(-7deg);
  }
}
</style>
