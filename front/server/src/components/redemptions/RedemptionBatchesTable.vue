<script setup lang="ts">
import {
  CalendarPlus,
  Download,
  PackageOpen,
  Eye,
  MoreHorizontal,
  PencilLine,
  ShieldOff,
} from '@lucide/vue'
import BasePagination from '@/components/base/BasePagination.vue'
import { useFloatingActionMenu } from '@/composables/useFloatingActionMenu'
import type { RedemptionBatch } from '@/types'
import { formatDate, formatDateTime, formatPercent } from './formatters'

const props = defineProps<{
  items: RedemptionBatch[]
  page: number
  pageSize: number
  total: number
  loading?: boolean
  exportingId?: string | null
}>()

const emit = defineEmits<{
  'update:page': [page: number]
  detail: [batch: RedemptionBatch]
  export: [batch: RedemptionBatch]
  exportXianyu: [batch: RedemptionBatch]
  disable: [batch: RedemptionBatch]
  extend: [batch: RedemptionBatch]
  rename: [batch: RedemptionBatch]
}>()

const { openMenuId, menuPosition, closeMenu, toggleMenu } =
  useFloatingActionMenu(190)
</script>

<template>
  <section class="ledger" aria-label="兑换码生成批次">
    <div class="table-scroll">
      <table>
        <thead>
          <tr>
            <th>批次</th>
            <th>库存分布</th>
            <th>使用率</th>
            <th>每码次数</th>
            <th>商品标识</th>
            <th>有效期</th>
            <th>创建人</th>
            <th>创建时间</th>
            <th class="actions">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="props.loading && !props.items.length">
            <td colspan="9">
              <div class="table-state">正在读取生成批次...</div>
            </td>
          </tr>
          <tr v-else-if="!props.items.length">
            <td colspan="9">
              <div class="empty-state">
                <strong>还没有生成批次</strong>
                <span>生成一批兑换码后，库存和核销情况会在这里汇总。</span>
              </div>
            </td>
          </tr>
          <tr
            v-for="batch in props.items"
            v-else
            :key="batch.id"
            @dblclick="emit('detail', batch)"
          >
            <td>
              <button
                type="button"
                class="batch-name"
                @click="emit('detail', batch)"
              >
                <strong>{{ batch.name }}</strong>
                <span class="data-mono">{{ batch.id }}</span>
              </button>
            </td>
            <td>
              <div class="distribution">
                <span class="unused" :title="`未使用 ${batch.counts.unused}`">
                  {{ batch.counts.unused }}
                </span>
                <span
                  class="redeemed"
                  :title="`已兑换 ${batch.counts.redeemed}`"
                >
                  {{ batch.counts.redeemed }}
                </span>
                <span
                  class="expired"
                  :title="`已过期 ${batch.counts.expired}`"
                >
                  {{ batch.counts.expired }}
                </span>
                <span
                  class="disabled"
                  :title="`已失效 ${batch.counts.disabled}`"
                >
                  {{ batch.counts.disabled }}
                </span>
              </div>
              <small>共 {{ batch.quantity }} 个</small>
            </td>
            <td>
              <div class="usage">
                <div>
                  <span :style="{ width: formatPercent(batch.usageRate) }" />
                </div>
                <strong>{{ formatPercent(batch.usageRate) }}</strong>
              </div>
            </td>
            <td>
              <span class="credit-value">{{ batch.creditsPerCode }}</span>
            </td>
            <td class="data-mono">{{ batch.productCode }}</td>
            <td>{{ formatDate(batch.expiresAt) }}</td>
            <td>{{ batch.createdBy }}</td>
            <td>{{ formatDateTime(batch.createdAt) }}</td>
            <td class="actions">
              <div class="action-menu">
                <button
                  type="button"
                  aria-label="更多操作"
                  @click="toggleMenu($event, batch.id)"
                >
                  <MoreHorizontal :size="18" aria-hidden="true" />
                </button>
                <Teleport to="body">
                  <div
                    v-if="openMenuId === batch.id"
                    class="menu-popover"
                    :style="{
                      top: `${menuPosition.top}px`,
                      right: `${menuPosition.right}px`,
                    }"
                    @mouseleave="closeMenu"
                  >
                    <button
                      type="button"
                      @click="closeMenu(); emit('detail', batch)"
                    >
                      <Eye :size="15" aria-hidden="true" />
                      查看批次
                    </button>
                    <button
                      type="button"
                      @click="closeMenu(); emit('rename', batch)"
                    >
                      <PencilLine :size="15" aria-hidden="true" />
                      修改批次名称
                    </button>
                    <button
                      type="button"
                      :disabled="props.exportingId === batch.id"
                      @click="closeMenu(); emit('export', batch)"
                    >
                      <Download :size="15" aria-hidden="true" />
                      导出未使用码
                    </button>
                    <button
                      type="button"
                      :disabled="props.exportingId === batch.id || batch.counts.unused === 0"
                      @click="closeMenu(); emit('exportXianyu', batch)"
                    >
                      <PackageOpen :size="15" aria-hidden="true" />
                      导出闲鱼库存
                    </button>
                    <button
                      type="button"
                      @click="closeMenu(); emit('extend', batch)"
                    >
                      <CalendarPlus :size="15" aria-hidden="true" />
                      批量延期
                    </button>
                    <button
                      type="button"
                      class="danger"
                      :disabled="batch.counts.unused === 0"
                      @click="closeMenu(); emit('disable', batch)"
                    >
                      <ShieldOff :size="15" aria-hidden="true" />
                      失效未使用码
                    </button>
                  </div>
                </Teleport>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="pagination-row">
      <span>共 {{ props.total }} 个批次</span>
      <BasePagination
        :page="props.page"
        :page-size="props.pageSize"
        :total="props.total"
        @update:page="emit('update:page', $event)"
      />
    </div>
  </section>
</template>

<style scoped>
.ledger {
  min-width: 0;
  background: #fff;
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 8px;
}

.table-scroll {
  min-height: clamp(300px, calc(100dvh - 470px), 500px);
  overflow-x: auto;
}

table {
  width: 100%;
  min-width: 1080px;
  border-collapse: collapse;
}

th,
td {
  padding: 13px 14px;
  color: var(--color-text, #1b1f1f);
  font-size: 12px;
  text-align: left;
  border-bottom: 1px solid var(--color-border-soft, #edf0ef);
}

th {
  height: 43px;
  color: var(--color-text-muted, #68716f);
  font-size: 11px;
  background: #f7f8f7;
}

tbody tr {
  transition: background-color 140ms ease;
}

tbody tr:hover {
  background: #f7faf9;
}

tbody tr:last-child td {
  border-bottom: 0;
}

.batch-name {
  display: grid;
  gap: 5px;
  min-width: 170px;
  padding: 0;
  text-align: left;
  background: transparent;
  border: 0;
  cursor: pointer;
}

.batch-name strong {
  color: var(--color-text, #1b1f1f);
  font-size: 12px;
}

.batch-name span,
td small {
  color: var(--color-text-muted, #68716f);
  font-size: 10px;
}

.distribution {
  display: flex;
  width: 150px;
  height: 7px;
  margin-bottom: 7px;
  overflow: hidden;
  background: #edf0ef;
  border-radius: 2px;
}

.distribution span {
  min-width: 4px;
  text-indent: -9999px;
}

.distribution .unused {
  flex: var(--unused, 1);
  background: #4d8b81;
}

.distribution .redeemed {
  background: #bac2c0;
}

.distribution .expired {
  background: #c59a44;
}

.distribution .disabled {
  background: #b8574b;
}

.usage {
  display: flex;
  gap: 8px;
  align-items: center;
}

.usage > div {
  width: 64px;
  height: 5px;
  overflow: hidden;
  background: #e8eceb;
  border-radius: 2px;
}

.usage > div span {
  display: block;
  height: 100%;
  background: var(--color-primary, #236c62);
}

.usage strong {
  font-size: 11px;
}

.credit-value {
  display: inline-grid;
  min-width: 30px;
  height: 28px;
  padding: 0 8px;
  color: var(--color-primary, #236c62);
  font-weight: 800;
  background: #edf4f2;
  border-radius: 5px;
  place-items: center;
}

.actions {
  width: 64px;
  text-align: right;
}

.action-menu {
  position: relative;
  display: inline-block;
}

.action-menu > button {
  display: grid;
  width: 30px;
  height: 30px;
  padding: 0;
  color: var(--color-text-muted, #68716f);
  background: transparent;
  border: 0;
  border-radius: 5px;
  cursor: pointer;
  place-items: center;
}

.action-menu > button:hover {
  color: var(--color-primary, #236c62);
  background: #edf4f2;
}

.menu-popover {
  position: fixed;
  z-index: 60;
  display: grid;
  width: 190px;
  padding: 5px;
  background: #fff;
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 7px;
  box-shadow: 0 14px 34px rgb(22 33 31 / 14%);
}

.menu-popover button {
  display: flex;
  gap: 8px;
  align-items: center;
  min-height: 34px;
  padding: 0 9px;
  color: var(--color-text, #1b1f1f);
  font-size: 12px;
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: 5px;
  cursor: pointer;
}

.menu-popover button:hover {
  background: #f3f5f4;
}

.menu-popover button.danger {
  color: var(--color-danger, #b8574b);
}

.menu-popover button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.table-state,
.empty-state {
  display: grid;
  gap: 5px;
  min-height: 220px;
  color: var(--color-text-muted, #68716f);
  text-align: center;
  place-content: center;
}

.empty-state strong {
  color: var(--color-text, #1b1f1f);
  font-family: var(--font-display, serif);
  font-size: 17px;
}

.pagination-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 58px;
  padding: 0 15px;
  color: var(--color-text-muted, #68716f);
  font-size: 11px;
  border-top: 1px solid var(--color-border-soft, #edf0ef);
}

@media (max-width: 768px) {
  .table-scroll {
    min-height: 300px;
  }

  .pagination-row {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
    padding: 12px;
  }
}
</style>
