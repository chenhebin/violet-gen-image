<script setup lang="ts">
import {
  CalendarPlus,
  Copy,
  Download,
  Eye,
  Layers3,
  PackageOpen,
  ShieldOff,
} from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDrawer from '@/components/base/BaseDrawer.vue'
import type { RedemptionBatch, RedemptionCode } from '@/types'
import RedemptionStatusBadge from './RedemptionStatusBadge.vue'
import { formatDate, formatDateTime, formatPercent } from './formatters'

const props = defineProps<{
  open: boolean
  batch: RedemptionBatch | null
  codes: RedemptionCode[]
  revealedCodes: Record<string, string>
  loading?: boolean
  revealing?: boolean
  exporting?: boolean
}>()

const emit = defineEmits<{
  close: []
  reveal: [code: RedemptionCode]
  revealAll: [batch: RedemptionBatch]
  copy: [value: string, label: string]
  export: [batch: RedemptionBatch]
  exportXianyu: [batch: RedemptionBatch]
  disable: [batch: RedemptionBatch]
  extend: [batch: RedemptionBatch]
}>()

function copyRevealed() {
  if (!props.batch) return
  const visible = props.codes
    .map((code) => props.revealedCodes[code.id])
    .filter(Boolean)
  emit('copy', visible.join('\n'), `${props.batch.name}完整兑换码`)
}
</script>

<template>
  <BaseDrawer
    :open="props.open"
    title="生成批次详情"
    description="查看库存分布、完整码与批次管理操作"
    size="large"
    @close="emit('close')"
  >
    <div v-if="props.loading && !props.batch" class="drawer-state">
      正在读取批次...
    </div>

    <div v-else-if="props.batch" class="drawer-content">
      <section class="batch-identity">
        <div class="mark" aria-hidden="true">
          <Layers3 :size="25" />
        </div>
        <div>
          <span>生成批次</span>
          <h2>{{ props.batch.name }}</h2>
          <small class="data-mono">{{ props.batch.id }}</small>
        </div>
        <div class="rate">
          <strong>{{ formatPercent(props.batch.usageRate) }}</strong>
          <span>已兑换</span>
        </div>
      </section>

      <section class="inventory-spine" aria-label="批次库存分布">
        <div class="spine-line" aria-hidden="true">
          <span
            class="unused"
            :style="{ flex: Math.max(props.batch.counts.unused, 0.001) }"
          />
          <span
            class="redeemed"
            :style="{ flex: Math.max(props.batch.counts.redeemed, 0.001) }"
          />
          <span
            class="expired"
            :style="{ flex: Math.max(props.batch.counts.expired, 0.001) }"
          />
          <span
            class="disabled"
            :style="{ flex: Math.max(props.batch.counts.disabled, 0.001) }"
          />
        </div>
        <div class="inventory-values">
          <div><i class="unused" />未使用 <strong>{{ props.batch.counts.unused }}</strong></div>
          <div><i class="redeemed" />已兑换 <strong>{{ props.batch.counts.redeemed }}</strong></div>
          <div><i class="expired" />已过期 <strong>{{ props.batch.counts.expired }}</strong></div>
          <div><i class="disabled" />已失效 <strong>{{ props.batch.counts.disabled }}</strong></div>
        </div>
      </section>

      <dl class="batch-config">
        <div>
          <dt>生成数量</dt>
          <dd>{{ props.batch.quantity }} 个</dd>
        </div>
        <div>
          <dt>每码次数</dt>
          <dd>{{ props.batch.creditsPerCode }} 次</dd>
        </div>
        <div>
          <dt>商品标识</dt>
          <dd class="data-mono">{{ props.batch.productCode }}</dd>
        </div>
        <div>
          <dt>有效期</dt>
          <dd>{{ formatDate(props.batch.expiresAt) }}</dd>
        </div>
        <div>
          <dt>创建人</dt>
          <dd>{{ props.batch.createdBy }}</dd>
        </div>
        <div>
          <dt>创建时间</dt>
          <dd>{{ formatDateTime(props.batch.createdAt) }}</dd>
        </div>
        <div class="wide">
          <dt>内部备注</dt>
          <dd>{{ props.batch.note || '未填写' }}</dd>
        </div>
      </dl>

      <section class="codes-section">
        <header>
          <div>
            <h3>批次兑换码</h3>
            <p>当前展示前 {{ props.codes.length }} 条，完整码只在本次授权操作中显示。</p>
          </div>
          <div class="header-actions">
            <BaseButton
              v-if="props.batch.counts.unused"
              variant="secondary"
              size="sm"
              :loading="props.revealing"
              @click="emit('revealAll', props.batch)"
            >
              <Eye :size="15" aria-hidden="true" />
              查看未使用完整码
            </BaseButton>
            <BaseButton
              v-if="Object.keys(props.revealedCodes).length"
              variant="ghost"
              size="sm"
              @click="copyRevealed"
            >
              <Copy :size="15" aria-hidden="true" />
              复制已显示
            </BaseButton>
          </div>
        </header>

        <div class="code-list">
          <div
            v-for="code in props.codes"
            :key="code.id"
            class="code-row"
          >
            <button
              v-if="code.status === 'unused'"
              type="button"
              class="reveal-one"
              aria-label="查看完整兑换码"
              @click="emit('reveal', code)"
            >
              <Eye :size="14" aria-hidden="true" />
            </button>
            <span v-else class="reveal-placeholder" />
            <strong class="data-mono">
              {{ props.revealedCodes[code.id] || code.maskedCode }}
            </strong>
            <RedemptionStatusBadge
              :status="code.status"
              :expiring-soon="code.expiringSoon"
            />
            <span>{{ formatDate(code.expiresAt) }}</span>
          </div>
          <div v-if="!props.codes.length" class="codes-empty">
            正在加载本批次兑换码...
          </div>
        </div>
      </section>

      <div class="drawer-actions">
        <BaseButton
          variant="secondary"
          :loading="props.exporting"
          @click="emit('export', props.batch)"
        >
          <Download :size="16" aria-hidden="true" />
          导出 CSV
        </BaseButton>
        <BaseButton
          variant="secondary"
          :loading="props.exporting"
          :disabled="!props.batch.counts.unused"
          @click="emit('exportXianyu', props.batch)"
        >
          <PackageOpen :size="16" aria-hidden="true" />
          导出闲鱼库存
        </BaseButton>
        <BaseButton
          variant="secondary"
          @click="emit('extend', props.batch)"
        >
          <CalendarPlus :size="16" aria-hidden="true" />
          批量延期
        </BaseButton>
        <BaseButton
          v-if="props.batch.counts.unused"
          variant="danger"
          @click="emit('disable', props.batch)"
        >
          <ShieldOff :size="16" aria-hidden="true" />
          失效未使用码
        </BaseButton>
      </div>
    </div>
  </BaseDrawer>
</template>

<style scoped>
.drawer-state {
  min-height: 360px;
  color: var(--color-text-muted, #68716f);
  place-content: center;
}
.drawer-content {
  display: grid;
  gap: 20px;
}
.batch-identity {
  display: grid;
  grid-template-columns: 50px minmax(0, 1fr) auto;
  gap: 14px;
  align-items: center;
  padding: 18px;
  background: #f2f6f5;
  border-left: 3px solid var(--color-primary, #236c62);
  border-radius: 5px 8px 8px 5px;
}
.mark {
  display: grid;
  width: 50px;
  height: 50px;
  color: var(--color-primary, #236c62);
  background: #fff;
  border: 1px solid #d8e6e2;
  border-radius: 7px;
  place-items: center;
}
.batch-identity > div:nth-child(2) {
  display: grid;
  gap: 3px;
  min-width: 0;
}
.batch-identity span,
.batch-identity small {
  color: var(--color-text-muted, #68716f);
  font-size: 11px;
}

.batch-identity h2 {
  margin: 0;
  font-family: var(--font-display, serif);
  font-size: 21px;
  font-weight: 600;
}

.rate {
  display: grid;
  text-align: right;
}

.rate strong {
  color: var(--color-primary, #236c62);
  font-family: var(--font-mono, monospace);
  font-size: 22px;
}

.inventory-spine {
  padding: 14px 16px;
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 7px;
}

.spine-line {
  display: flex;
  height: 7px;
  overflow: hidden;
  border-radius: 2px;
}

.spine-line .unused,
.inventory-values i.unused {
  background: #4d8b81;
}

.spine-line .redeemed,
.inventory-values i.redeemed {
  background: #b8c0be;
}

.spine-line .expired,
.inventory-values i.expired {
  background: #c59a44;
}

.spine-line .disabled,
.inventory-values i.disabled {
  background: #b8574b;
}

.inventory-values {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 22px;
  margin-top: 11px;
  color: var(--color-text-muted, #68716f);
  font-size: 11px;
}

.inventory-values div {
  display: flex;
  gap: 6px;
  align-items: center;
}

.inventory-values i {
  width: 7px;
  height: 7px;
  border-radius: 2px;
}

.inventory-values strong {
  color: var(--color-text, #1b1f1f);
}

.batch-config {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  margin: 0;
  background: var(--color-border-soft, #edf0ef);
  border: 1px solid var(--color-border-soft, #edf0ef);
  border-radius: 7px;
}

.batch-config > div {
  display: grid;
  gap: 6px;
  min-height: 62px;
  padding: 12px 14px;
  background: #fff;
}

.batch-config .wide {
  grid-column: 1 / -1;
}

dt {
  color: var(--color-text-muted, #68716f);
  font-size: 10px;
}

dd {
  margin: 0;
  font-size: 12px;
  overflow-wrap: anywhere;
}

.codes-section {
  min-width: 0;
}

.codes-section header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 10px;
}

.codes-section h3 {
  margin: 0;
  font-size: 13px;
}

.codes-section p {
  margin: 4px 0 0;
  color: var(--color-text-muted, #68716f);
  font-size: 10px;
}

.header-actions {
  display: flex;
  gap: 7px;
}

.code-list {
  max-height: 276px;
  overflow: auto;
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 7px;
}

.code-row {
  display: grid;
  grid-template-columns: 28px minmax(170px, 1fr) 120px 100px;
  gap: 10px;
  align-items: center;
  min-width: 570px;
  min-height: 43px;
  padding: 0 12px;
  font-size: 11px;
  border-bottom: 1px solid var(--color-border-soft, #edf0ef);
}

.code-row:last-child {
  border-bottom: 0;
}

.reveal-one {
  display: grid;
  width: 28px;
  height: 28px;
  padding: 0;
  color: var(--color-text-muted, #68716f);
  background: transparent;
  border: 0;
  border-radius: 5px;
  cursor: pointer;
  place-items: center;
}

.reveal-one:hover {
  color: var(--color-primary, #236c62);
  background: #edf4f2;
}

.reveal-placeholder {
  width: 28px;
}

.codes-empty {
  display: grid;
  min-height: 120px;
  color: var(--color-text-muted, #68716f);
  font-size: 11px;
  place-content: center;
}

.drawer-actions { position: sticky; bottom: -20px; display: flex; justify-content: flex-end; gap: 9px; padding: 14px 0 2px; background: #fff; border-top: 1px solid var(--color-border-soft, #edf0ef); }

@media (max-width: 640px) {
  .batch-identity {
    grid-template-columns: 42px 1fr;
  }

  .mark {
    width: 42px;
    height: 42px;
  }

  .rate {
    grid-column: 1 / -1;
    text-align: left;
  }

  .batch-config {
    grid-template-columns: 1fr;
  }

  .batch-config .wide {
    grid-column: auto;
  }

  .codes-section header,
  .header-actions,
  .drawer-actions {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
