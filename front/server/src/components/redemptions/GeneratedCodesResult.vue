<script setup lang="ts">
import { CheckCircle2, Copy, Download, TicketCheck } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import type { CreateRedemptionBatchResult } from '@/types'
import { formatDate } from './formatters'

const props = defineProps<{
  result: CreateRedemptionBatchResult
  exporting?: boolean
}>()

const emit = defineEmits<{
  copy: [value: string, label: string]
  export: [batchId: string]
  done: []
}>()

function copyAll() {
  emit(
    'copy',
    props.result.codes.map((item) => item.fullCode).join('\n'),
    '全部兑换码',
  )
}
</script>

<template>
  <div class="result">
    <div class="success-mark" aria-hidden="true">
      <CheckCircle2 :size="24" />
    </div>
    <div>
      <p class="eyebrow">生成完成</p>
      <h3>{{ props.result.batch.name }}</h3>
      <p class="summary">
        已生成 {{ props.result.batch.quantity }} 个兑换码，每码
        {{ props.result.batch.creditsPerCode }} 次，有效期至
        {{ formatDate(props.result.batch.expiresAt) }}。
      </p>
    </div>
  </div>

  <div class="result-meta">
    <div>
      <span>批次编号</span>
      <strong class="data-mono">{{ props.result.batch.id }}</strong>
    </div>
    <div>
      <span>商品标识</span>
      <strong class="data-mono">{{ props.result.batch.productCode }}</strong>
    </div>
  </div>

  <div class="code-ledger">
    <div class="ledger-heading">
      <span>完整兑换码</span>
      <small>关闭后仍可在批次详情重新查看</small>
    </div>
    <div class="code-list" role="list">
      <button
        v-for="item in props.result.codes"
        :key="item.id"
        class="code-row"
        type="button"
        role="listitem"
        :aria-label="`复制兑换码 ${item.fullCode}`"
        @click="emit('copy', item.fullCode, '兑换码')"
      >
        <TicketCheck :size="15" aria-hidden="true" />
        <span class="data-mono">{{ item.fullCode }}</span>
        <Copy :size="15" aria-hidden="true" />
      </button>
    </div>
  </div>

  <div class="result-actions">
    <BaseButton variant="secondary" @click="copyAll">
      <Copy :size="16" aria-hidden="true" />
      复制全部
    </BaseButton>
    <BaseButton
      variant="secondary"
      :loading="props.exporting"
      @click="emit('export', props.result.batch.id)"
    >
      <Download :size="16" aria-hidden="true" />
      导出 CSV
    </BaseButton>
    <BaseButton @click="emit('done')">完成</BaseButton>
  </div>
</template>

<style scoped>
.result {
  display: grid;
  grid-template-columns: 44px 1fr;
  gap: 14px;
  align-items: start;
}

.success-mark {
  display: grid;
  width: 44px;
  height: 44px;
  color: var(--color-primary, #236c62);
  background: #e8f1ef;
  border-radius: 50%;
  place-items: center;
}

.eyebrow {
  margin: 1px 0 5px;
  color: var(--color-primary, #236c62);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0;
  text-transform: uppercase;
}

h3 {
  margin: 0;
  color: var(--color-text, #1b1f1f);
  font-family: var(--font-display, serif);
  font-size: 21px;
  font-weight: 600;
}

.summary {
  margin: 6px 0 0;
  color: var(--color-text-muted, #68716f);
  font-size: 13px;
  line-height: 1.65;
}

.result-meta {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1px;
  margin-top: 20px;
  overflow: hidden;
  background: var(--color-border, #dce1df);
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 7px;
}

.result-meta > div {
  display: grid;
  gap: 6px;
  min-width: 0;
  padding: 13px 15px;
  background: #fff;
}

.result-meta span,
.ledger-heading small {
  color: var(--color-text-muted, #68716f);
  font-size: 11px;
}

.result-meta strong {
  overflow: hidden;
  color: var(--color-text, #1b1f1f);
  font-size: 12px;
  text-overflow: ellipsis;
}

.code-ledger {
  margin-top: 18px;
}

.ledger-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 9px;
  color: var(--color-text, #1b1f1f);
  font-size: 13px;
  font-weight: 700;
}

.code-list {
  max-height: 278px;
  overflow-y: auto;
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 7px;
}

.code-row {
  display: grid;
  grid-template-columns: 18px 1fr 18px;
  gap: 10px;
  align-items: center;
  width: 100%;
  min-height: 42px;
  padding: 0 13px;
  color: var(--color-text, #1b1f1f);
  text-align: left;
  background: #fff;
  border: 0;
  border-bottom: 1px solid var(--color-border-soft, #edf0ef);
  cursor: pointer;
}

.code-row:last-child {
  border-bottom: 0;
}

.code-row:hover {
  background: #f6f8f7;
}

.code-row svg:last-child {
  color: var(--color-text-muted, #68716f);
}

.result-actions {
  display: flex;
  justify-content: flex-end;
  gap: 9px;
  margin-top: 20px;
}

@media (max-width: 560px) {
  .result-meta {
    grid-template-columns: 1fr;
  }

  .ledger-heading {
    align-items: flex-start;
    flex-direction: column;
    gap: 2px;
  }

  .result-actions {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
