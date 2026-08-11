<script setup lang="ts">
import {
  CalendarPlus,
  Copy,
  Eye,
  EyeOff,
  MoreHorizontal,
  ShieldOff,
} from '@lucide/vue'
import { computed, ref } from 'vue'
import BasePagination from '@/components/base/BasePagination.vue'
import type { RedemptionCode } from '@/types'
import RedemptionStatusBadge from './RedemptionStatusBadge.vue'
import { formatDate, formatDateTime } from './formatters'

const props = defineProps<{
  items: RedemptionCode[]
  page: number
  pageSize: number
  total: number
  loading?: boolean
  selectedIds: string[]
  revealedCodes: Record<string, string>
}>()

const emit = defineEmits<{
  'update:page': [page: number]
  'update:selectedIds': [ids: string[]]
  reveal: [code: RedemptionCode]
  copy: [value: string]
  detail: [code: RedemptionCode]
  disable: [code: RedemptionCode]
  extend: [code: RedemptionCode]
}>()

const openMenuId = ref<string | null>(null)

const selectableItems = computed(() =>
  props.items.filter((item) => item.status === 'unused'),
)
const allSelected = computed(
  () =>
    selectableItems.value.length > 0 &&
    selectableItems.value.every((item) =>
      props.selectedIds.includes(item.id),
    ),
)

function toggleAll() {
  const pageIds = selectableItems.value.map((item) => item.id)
  if (allSelected.value) {
    emit(
      'update:selectedIds',
      props.selectedIds.filter((id) => !pageIds.includes(id)),
    )
  } else {
    emit('update:selectedIds', [
      ...new Set([...props.selectedIds, ...pageIds]),
    ])
  }
}

function toggleOne(codeId: string) {
  emit(
    'update:selectedIds',
    props.selectedIds.includes(codeId)
      ? props.selectedIds.filter((id) => id !== codeId)
      : [...props.selectedIds, codeId],
  )
}

function displayCode(code: RedemptionCode) {
  return props.revealedCodes[code.id] || code.maskedCode
}
</script>

<template>
  <section class="ledger" aria-label="兑换码台账">
    <div class="table-scroll">
      <table>
        <thead>
          <tr>
            <th class="selection">
              <input
                type="checkbox"
                :checked="allSelected"
                :disabled="!selectableItems.length"
                aria-label="选择当前页全部未使用兑换码"
                @change="toggleAll"
              />
            </th>
            <th>兑换码</th>
            <th>批次 / 商品</th>
            <th>次数</th>
            <th>状态</th>
            <th>有效期</th>
            <th>兑换用户</th>
            <th>创建时间</th>
            <th class="actions">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="props.loading && !props.items.length">
            <td colspan="9">
              <div class="table-state">正在读取兑换码台账...</div>
            </td>
          </tr>
          <tr v-else-if="!props.items.length">
            <td colspan="9">
              <div class="empty-state">
                <strong>没有符合条件的兑换码</strong>
                <span>调整筛选条件，或生成一个新批次。</span>
              </div>
            </td>
          </tr>
          <tr
            v-for="code in props.items"
            v-else
            :key="code.id"
            :class="{ selected: props.selectedIds.includes(code.id) }"
          >
            <td class="selection">
              <input
                v-if="code.status === 'unused'"
                type="checkbox"
                :checked="props.selectedIds.includes(code.id)"
                :aria-label="`选择 ${code.maskedCode}`"
                @change="toggleOne(code.id)"
              />
            </td>
            <td>
              <div class="code-cell">
                <button
                  v-if="code.status === 'unused'"
                  type="button"
                  class="reveal-button"
                  :aria-label="
                    props.revealedCodes[code.id]
                      ? `隐藏 ${code.maskedCode}`
                      : `查看 ${code.maskedCode} 完整码`
                  "
                  @click="
                    props.revealedCodes[code.id]
                      ? emit('reveal', code)
                      : emit('reveal', code)
                  "
                >
                  <EyeOff
                    v-if="props.revealedCodes[code.id]"
                    :size="15"
                    aria-hidden="true"
                  />
                  <Eye v-else :size="15" aria-hidden="true" />
                </button>
                <button
                  type="button"
                  class="code-value data-mono"
                  :title="displayCode(code)"
                  @click="emit('detail', code)"
                >
                  {{ displayCode(code) }}
                </button>
                <button
                  v-if="props.revealedCodes[code.id]"
                  type="button"
                  class="copy-button"
                  aria-label="复制完整兑换码"
                  @click="emit('copy', props.revealedCodes[code.id])"
                >
                  <Copy :size="14" aria-hidden="true" />
                </button>
              </div>
            </td>
            <td>
              <div class="stacked-cell">
                <strong>{{ code.batchName }}</strong>
                <span class="data-mono">{{ code.productCode }}</span>
              </div>
            </td>
            <td>
              <span class="credits">{{ code.credits }}</span>
            </td>
            <td>
              <RedemptionStatusBadge
                :status="code.status"
                :expiring-soon="code.expiringSoon"
              />
            </td>
            <td>
              <span :class="{ warning: code.expiringSoon }">
                {{ formatDate(code.expiresAt) }}
              </span>
            </td>
            <td>
              <div v-if="code.redeemedByEmail" class="stacked-cell">
                <strong>{{ code.redeemedByEmail }}</strong>
                <span>{{ formatDateTime(code.redeemedAt) }}</span>
              </div>
              <span v-else class="muted">尚未兑换</span>
            </td>
            <td>{{ formatDateTime(code.createdAt) }}</td>
            <td class="actions">
              <div class="action-menu">
                <button
                  type="button"
                  aria-label="更多操作"
                  @click="
                    openMenuId = openMenuId === code.id ? null : code.id
                  "
                >
                  <MoreHorizontal :size="18" aria-hidden="true" />
                </button>
                <div
                  v-if="openMenuId === code.id"
                  class="menu-popover"
                  @mouseleave="openMenuId = null"
                >
                  <button type="button" @click="emit('detail', code)">
                    <Eye :size="15" aria-hidden="true" />
                    查看详情
                  </button>
                  <button
                    v-if="code.status === 'unused' || code.status === 'expired'"
                    type="button"
                    @click="emit('extend', code)"
                  >
                    <CalendarPlus :size="15" aria-hidden="true" />
                    延长有效期
                  </button>
                  <button
                    v-if="code.status === 'unused'"
                    type="button"
                    class="danger"
                    @click="emit('disable', code)"
                  >
                    <ShieldOff :size="15" aria-hidden="true" />
                    失效兑换码
                  </button>
                </div>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="pagination-row">
      <span>共 {{ props.total }} 条记录</span>
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
  overflow: visible;
  background: #fff;
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 8px;
}

.table-scroll {
  overflow-x: auto;
}

table {
  width: 100%;
  min-width: 1180px;
  border-collapse: collapse;
}

th,
td {
  padding: 12px 13px;
  color: var(--color-text, #1b1f1f);
  font-size: 12px;
  text-align: left;
  vertical-align: middle;
  border-bottom: 1px solid var(--color-border-soft, #edf0ef);
}

th {
  height: 43px;
  color: var(--color-text-muted, #68716f);
  font-size: 11px;
  font-weight: 700;
  background: #f7f8f7;
}

tbody tr {
  transition: background-color 140ms ease;
}

tbody tr:hover,
tbody tr.selected {
  background: #f7faf9;
}

tbody tr:last-child td {
  border-bottom: 0;
}

.selection {
  width: 42px;
  padding-right: 4px;
  text-align: center;
}

input[type='checkbox'] {
  accent-color: var(--color-primary, #236c62);
}

.code-cell {
  display: flex;
  align-items: center;
  min-width: 190px;
}

.reveal-button,
.copy-button,
.action-menu > button {
  display: grid;
  flex: 0 0 auto;
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

.reveal-button:hover,
.copy-button:hover,
.action-menu > button:hover {
  color: var(--color-primary, #236c62);
  background: #edf4f2;
}

.code-value {
  max-width: 172px;
  overflow: hidden;
  color: var(--color-text, #1b1f1f);
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
  background: transparent;
  border: 0;
  cursor: pointer;
}

.stacked-cell {
  display: grid;
  gap: 4px;
  min-width: 118px;
}

.stacked-cell strong {
  overflow: hidden;
  font-size: 12px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stacked-cell span {
  color: var(--color-text-muted, #68716f);
  font-size: 10px;
}

.credits {
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

.warning {
  color: var(--color-warning, #9a7023);
  font-weight: 700;
}

.actions {
  width: 64px;
  text-align: right;
}

.action-menu {
  position: relative;
  display: inline-block;
}

.menu-popover {
  position: absolute;
  z-index: 20;
  top: 32px;
  right: 0;
  display: grid;
  width: 148px;
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
  .pagination-row {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
    padding: 12px;
  }
}
</style>
