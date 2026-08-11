<script setup lang="ts">
import { computed } from 'vue'
import { AlertCircle, ArrowRight, Coins } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import type { UsageQuote } from '@/types/domain'

const props = defineProps<{
  quote: UsageQuote | null
  quoting: boolean
  submitting: boolean
  ready: boolean
  outputCount: number
}>()

defineEmits<{ submit: [] }>()

const blockedReason = computed(() => {
  if (!props.quote) return '正在获取本次报价'
  if (!props.quote.canSubmit) return '剩余次数不足，请先兑换新码'
  if (!props.ready) return '填写完整的画面需求后可生成'
  return ''
})

const disabled = computed(
  () =>
    props.quoting ||
    props.submitting ||
    !props.quote?.canSubmit ||
    !props.ready,
)
</script>

<template>
  <footer class="quote-bar">
    <div class="quote-summary">
      <span>本次生成</span>
      <strong>
        {{ outputCount }} 张
        <i>·</i>
        消耗 {{ quote?.cost ?? outputCount }} 次
      </strong>
      <p>
        <Coins :size="14" />
        当前余额 {{ quote?.balance ?? 0 }} 次
      </p>
    </div>
    <BaseButton
      :loading="submitting || quoting"
      :disabled="disabled"
      @click="$emit('submit')"
    >
      {{
        submitting
          ? '正在提交'
          : quote?.canSubmit === false
            ? '次数不足'
            : '生成图片'
      }}
      <template #icon>
        <ArrowRight v-if="!quoting" :size="17" />
      </template>
    </BaseButton>
    <p v-if="blockedReason" class="blocked-reason">
      <AlertCircle :size="13" />
      {{ blockedReason }}
    </p>
  </footer>
</template>

<style scoped>
.quote-bar {
  position: sticky;
  bottom: 0;
  z-index: 4;
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 12px 14px;
  align-items: center;
  padding: 14px 18px 16px;
  border-top: 1px solid var(--border);
  background: rgb(255 255 255 / 96%);
  backdrop-filter: blur(12px);
}

.quote-summary > span {
  color: var(--ink-faint);
  font-size: 9px;
}

.quote-summary strong {
  display: block;
  margin-top: 1px;
  font-size: 13px;
}

.quote-summary i {
  margin-inline: 3px;
  color: var(--ink-faint);
  font-style: normal;
}

.quote-summary p {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 2px;
  color: var(--ink-muted);
  font-size: 9px;
}

.blocked-reason {
  display: flex;
  grid-column: 1 / -1;
  align-items: center;
  gap: 5px;
  color: var(--coral);
  font-size: 9px;
}

@media (max-width: 1100px) and (min-width: 901px) {
  .quote-bar {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .quote-bar {
    position: static;
    bottom: auto;
    border-radius: 0 0 var(--radius-md) var(--radius-md);
    backdrop-filter: none;
  }
}

@media (max-width: 560px) {
  .quote-bar {
    grid-template-columns: 1fr;
  }
}
</style>
