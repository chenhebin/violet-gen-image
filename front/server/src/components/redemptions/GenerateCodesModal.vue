<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { CalendarDays, Infinity as InfinityIcon, Layers3 } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import FormField from '@/components/base/FormField.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import { APP_CONFIG, REDEMPTION_CONFIG } from '@/config'
import type {
  CreateRedemptionBatchPayload,
  CreateRedemptionBatchResult,
} from '@/types'
import GeneratedCodesResult from './GeneratedCodesResult.vue'

type ValidityMode = 'default' | 'custom' | 'never'

const props = defineProps<{
  open: boolean
  loading?: boolean
  exporting?: boolean
  result?: CreateRedemptionBatchResult | null
}>()

const emit = defineEmits<{
  close: []
  submit: [payload: CreateRedemptionBatchPayload]
  copy: [value: string, label: string]
  export: [batchId: string]
}>()

const form = reactive({
  name: '',
  quantity: 20,
  creditsPerCode: 10,
  productCode: APP_CONFIG.productCode,
  validityMode: 'default' as ValidityMode,
  expiresAt: '',
  note: '',
})

const touched = reactive({
  name: false,
  quantity: false,
  creditsPerCode: false,
  expiresAt: false,
})

const defaultExpiry = computed(() => {
  const date = new Date()
  date.setDate(date.getDate() + REDEMPTION_CONFIG.defaultValidityDays)
  return date.toISOString().slice(0, 10)
})

const errors = computed(() => ({
  name: !form.name.trim()
    ? '请填写批次名称'
    : Array.from(form.name.trim()).length > REDEMPTION_CONFIG.batchNameMaxLength
      ? `批次名称不能超过 ${REDEMPTION_CONFIG.batchNameMaxLength} 个字符`
      : '',
  quantity:
    !Number.isInteger(form.quantity) ||
    form.quantity < REDEMPTION_CONFIG.minQuantity ||
    form.quantity > REDEMPTION_CONFIG.maxQuantity
      ? `生成数量需为 ${REDEMPTION_CONFIG.minQuantity}-${REDEMPTION_CONFIG.maxQuantity} 的整数`
      : '',
  creditsPerCode:
    !Number.isInteger(form.creditsPerCode) ||
    form.creditsPerCode < REDEMPTION_CONFIG.minCredits
      ? '每码次数需为大于 0 的整数'
      : '',
  expiresAt:
    form.validityMode === 'custom' &&
    (!form.expiresAt ||
      new Date(`${form.expiresAt}T23:59:59`).getTime() <= Date.now())
      ? '请选择未来的到期日期'
      : '',
}))

const isValid = computed(() =>
  Object.values(errors.value).every((message) => !message),
)

function reset() {
  form.name = ''
  form.quantity = 20
  form.creditsPerCode = 10
  form.productCode = APP_CONFIG.productCode
  form.validityMode = 'default'
  form.expiresAt = ''
  form.note = ''
  Object.keys(touched).forEach((key) => {
    touched[key as keyof typeof touched] = false
  })
}

function submit() {
  Object.keys(touched).forEach((key) => {
    touched[key as keyof typeof touched] = true
  })
  if (!isValid.value) return

  const expiresAt =
    form.validityMode === 'never'
      ? null
      : `${form.validityMode === 'default' ? defaultExpiry.value : form.expiresAt}T23:59:59.000Z`

  emit('submit', {
    name: form.name.trim(),
    quantity: Number(form.quantity),
    creditsPerCode: Number(form.creditsPerCode),
    productCode: form.productCode,
    expiresAt,
    neverExpires: form.validityMode === 'never',
    note: form.note.trim() || undefined,
  })
}

watch(
  () => props.open,
  (open) => {
    if (open && !props.result) reset()
  },
)
</script>

<template>
  <BaseModal
    :open="props.open"
    :title="props.result ? '兑换码已生成' : '生成兑换码'"
    :description="
      props.result
        ? '完整码仅在授权操作中展示，请通过安全渠道发放。'
        : '单个兑换码也会归入独立批次，便于后续追踪和审计。'
    "
    @close="emit('close')"
  >
    <GeneratedCodesResult
      v-if="props.result"
      :result="props.result"
      :exporting="props.exporting"
      @copy="(value, label) => emit('copy', value, label)"
      @export="(batchId) => emit('export', batchId)"
      @done="emit('close')"
    />

    <form v-else class="generate-form" @submit.prevent="submit">
      <div class="batch-mark">
        <Layers3 :size="21" aria-hidden="true" />
        <div>
          <strong>新生成批次</strong>
          <span>服务端原子生成，重复请求不会创建多个批次</span>
        </div>
      </div>

      <FormField
        label="批次名称"
        for-id="batch-name"
        required
        :error="touched.name ? errors.name : ''"
        hint="建议填写咸鱼商品或活动名称"
      >
        <input
          id="batch-name"
          v-model="form.name"
          :maxlength="REDEMPTION_CONFIG.batchNameMaxLength"
          placeholder="例如：暑期人像修图 · 第 02 批"
          @blur="touched.name = true"
        />
      </FormField>

      <div class="form-grid">
        <FormField
          label="生成数量"
          for-id="batch-quantity"
          required
          :error="touched.quantity ? errors.quantity : ''"
          hint="一次最多生成 500 个"
        >
          <input
            id="batch-quantity"
            v-model.number="form.quantity"
            type="number"
            min="1"
            max="500"
            step="1"
            @blur="touched.quantity = true"
          />
        </FormField>

        <FormField
          label="每码次数"
          for-id="batch-credits"
          required
          :error="touched.creditsPerCode ? errors.creditsPerCode : ''"
          hint="兑换成功后一次性发放"
        >
          <input
            id="batch-credits"
            v-model.number="form.creditsPerCode"
            type="number"
            min="1"
            step="1"
            @blur="touched.creditsPerCode = true"
          />
        </FormField>
      </div>

      <FormField label="商品标识" for-id="product-code">
        <input
          id="product-code"
          v-model="form.productCode"
          class="data-mono"
          disabled
        />
      </FormField>

      <fieldset class="validity">
        <legend>有效期</legend>
        <button
          type="button"
          :class="{ active: form.validityMode === 'default' }"
          @click="form.validityMode = 'default'"
        >
          <CalendarDays :size="17" aria-hidden="true" />
          <span>
            <strong>默认 90 天</strong>
            <small>至 {{ defaultExpiry }}</small>
          </span>
        </button>
        <button
          type="button"
          :class="{ active: form.validityMode === 'custom' }"
          @click="form.validityMode = 'custom'"
        >
          <CalendarDays :size="17" aria-hidden="true" />
          <span>
            <strong>指定日期</strong>
            <small>自定义未来到期时间</small>
          </span>
        </button>
        <button
          type="button"
          :class="{ active: form.validityMode === 'never' }"
          @click="form.validityMode = 'never'"
        >
          <InfinityIcon :size="17" aria-hidden="true" />
          <span>
            <strong>永久有效</strong>
            <small>不会自动过期</small>
          </span>
        </button>
      </fieldset>

      <FormField
        v-if="form.validityMode === 'custom'"
        label="到期日期"
        for-id="expires-at"
        required
        :error="touched.expiresAt ? errors.expiresAt : ''"
      >
        <input
          id="expires-at"
          v-model="form.expiresAt"
          type="date"
          :min="new Date(Date.now() + 86400000).toISOString().slice(0, 10)"
          @blur="touched.expiresAt = true"
        />
      </FormField>

      <FormField
        label="内部备注"
        for-id="batch-note"
        hint="不会向用户端展示"
      >
        <textarea
          id="batch-note"
          v-model="form.note"
          rows="3"
          maxlength="300"
          placeholder="记录发放渠道、对应商品或运营说明"
        />
      </FormField>

      <div class="form-actions">
        <BaseButton
          type="button"
          variant="ghost"
          :disabled="props.loading"
          @click="emit('close')"
        >
          取消
        </BaseButton>
        <BaseButton type="submit" :loading="props.loading">
          生成 {{ form.quantity }} 个兑换码
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.generate-form {
  display: grid;
  gap: 18px;
}

.batch-mark {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 13px 14px;
  color: var(--color-primary, #236c62);
  background: #eef5f3;
  border-left: 3px solid currentColor;
  border-radius: 4px 7px 7px 4px;
}

.batch-mark > div {
  display: grid;
  gap: 3px;
}

.batch-mark strong {
  font-size: 13px;
}

.batch-mark span {
  color: var(--color-text-muted, #68716f);
  font-size: 11px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

input,
textarea {
  width: 100%;
}

.validity {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 9px;
  padding: 0;
  border: 0;
}

.validity legend {
  margin-bottom: 8px;
  color: var(--color-text, #1b1f1f);
  font-size: 12px;
  font-weight: 700;
}

.validity button {
  display: flex;
  gap: 9px;
  align-items: flex-start;
  min-height: 66px;
  padding: 11px;
  color: var(--color-text-muted, #68716f);
  text-align: left;
  background: #fff;
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 7px;
  cursor: pointer;
}

.validity button.active {
  color: var(--color-primary, #236c62);
  background: #f4f8f7;
  border-color: var(--color-primary, #236c62);
}

.validity button span {
  display: grid;
  gap: 3px;
}

.validity strong {
  color: var(--color-text, #1b1f1f);
  font-size: 12px;
}

.validity small {
  font-size: 10px;
  line-height: 1.4;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 9px;
  padding-top: 3px;
}

@media (max-width: 600px) {
  .form-grid,
  .validity {
    grid-template-columns: 1fr;
  }
}
</style>
