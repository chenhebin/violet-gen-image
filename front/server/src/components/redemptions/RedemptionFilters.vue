<script setup lang="ts">
import { Search, SlidersHorizontal, X } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import type { RedemptionBatch, RedemptionCodeStatus } from '@/types'

export interface RedemptionFilterValue {
  keyword: string
  status: '' | RedemptionCodeStatus
  batchId: string
  redeemedBy: string
  expiringSoon: boolean
}

const props = defineProps<{
  modelValue: RedemptionFilterValue
  batches: RedemptionBatch[]
  loading?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: RedemptionFilterValue]
  search: []
  reset: []
}>()

function patch(patchValue: Partial<RedemptionFilterValue>) {
  emit('update:modelValue', { ...props.modelValue, ...patchValue })
}
</script>

<template>
  <section class="filter-panel" aria-label="兑换码筛选">
    <div class="filter-heading">
      <SlidersHorizontal :size="17" aria-hidden="true" />
      <span>筛选兑换码</span>
    </div>

    <div class="filter-grid">
      <label class="field field--wide">
        <span>兑换码或关联信息</span>
        <input
          :value="props.modelValue.keyword"
          placeholder="掩码、邮箱或批次名称"
          @input="
            patch({
              keyword: ($event.target as HTMLInputElement).value,
            })
          "
          @keydown.enter="emit('search')"
        />
      </label>

      <label class="field">
        <span>状态</span>
        <select
          :value="props.modelValue.status"
          @change="
            patch({
              status: ($event.target as HTMLSelectElement)
                .value as RedemptionFilterValue['status'],
            })
          "
        >
          <option value="">全部状态</option>
          <option value="unused">未使用</option>
          <option value="redeemed">已兑换</option>
          <option value="expired">已过期</option>
          <option value="disabled">已失效</option>
        </select>
      </label>

      <label class="field">
        <span>生成批次</span>
        <select
          :value="props.modelValue.batchId"
          @change="
            patch({
              batchId: ($event.target as HTMLSelectElement).value,
            })
          "
        >
          <option value="">全部批次</option>
          <option
            v-for="batch in props.batches"
            :key="batch.id"
            :value="batch.id"
          >
            {{ batch.name }}
          </option>
        </select>
      </label>

      <label class="field">
        <span>兑换用户</span>
        <input
          :value="props.modelValue.redeemedBy"
          placeholder="用户邮箱"
          @input="
            patch({
              redeemedBy: ($event.target as HTMLInputElement).value,
            })
          "
          @keydown.enter="emit('search')"
        />
      </label>

      <label class="check-field">
        <input
          type="checkbox"
          :checked="props.modelValue.expiringSoon"
          @change="
            patch({
              expiringSoon: ($event.target as HTMLInputElement).checked,
            })
          "
        />
        <span>仅看 7 天内过期</span>
      </label>
    </div>

    <div class="filter-actions">
      <BaseButton
        variant="ghost"
        size="sm"
        :disabled="props.loading"
        @click="emit('reset')"
      >
        <X :size="16" aria-hidden="true" />
        清空
      </BaseButton>
      <BaseButton
        size="sm"
        :loading="props.loading"
        @click="emit('search')"
      >
        <Search :size="16" aria-hidden="true" />
        查询
      </BaseButton>
    </div>
  </section>
</template>

<style scoped>
.filter-panel {
  display: grid;
  grid-template-columns: 138px minmax(0, 1fr) auto;
  gap: 20px;
  align-items: end;
  padding: 18px 20px;
  background: var(--color-surface, #fff);
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 8px;
}

.filter-heading {
  align-self: center;
  display: flex;
  align-items: center;
  gap: 9px;
  color: var(--color-text, #1b1f1f);
  font-size: 13px;
  font-weight: 700;
}

.filter-grid {
  display: grid;
  grid-template-columns: minmax(190px, 1.35fr) repeat(3, minmax(138px, 1fr)) auto;
  gap: 12px;
  align-items: end;
}

.field {
  display: grid;
  gap: 7px;
  min-width: 0;
}

.field > span {
  color: var(--color-text-muted, #6c7472);
  font-size: 12px;
}

.field input,
.field select {
  width: 100%;
  height: 38px;
  padding: 0 11px;
  color: var(--color-text, #1b1f1f);
  background: var(--color-canvas, #f3f5f4);
  border: 1px solid transparent;
  border-radius: 6px;
  outline: none;
  transition: border-color 160ms ease, background-color 160ms ease;
}

.field input:focus,
.field select:focus {
  background: #fff;
  border-color: var(--color-primary, #236c62);
}

.check-field {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  color: var(--color-text-muted, #6c7472);
  font-size: 12px;
  white-space: nowrap;
}

.check-field input {
  accent-color: var(--color-primary, #236c62);
}

.filter-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

@media (max-width: 1280px) {
  .filter-panel {
    grid-template-columns: 1fr auto;
  }

  .filter-heading {
    grid-column: 1 / -1;
  }

  .filter-grid {
    grid-template-columns: repeat(3, minmax(148px, 1fr));
  }
}

@media (max-width: 768px) {
  .filter-panel {
    grid-template-columns: 1fr;
    padding: 16px;
  }

  .filter-grid {
    grid-template-columns: 1fr;
  }

  .filter-actions {
    justify-content: flex-end;
  }
}
</style>
