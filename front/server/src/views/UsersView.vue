<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Eye, RefreshCw, Search, UsersRound } from '@lucide/vue'
import { useRoute } from 'vue-router'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDataTable, {
  type DataTableColumn,
} from '@/components/base/BaseDataTable.vue'
import FilterBar from '@/components/base/FilterBar.vue'
import FormField from '@/components/base/FormField.vue'
import PaginationBar from '@/components/base/PaginationBar.vue'
import UserStatusBadge from '@/components/shared/UserStatusBadge.vue'
import UserActionModal, {
  type UserAction,
  type UserActionPayload,
} from '@/components/users/UserActionModal.vue'
import UserDetailDrawer from '@/components/users/UserDetailDrawer.vue'
import { USER_STATUS_LABELS } from '@/config'
import { useToast } from '@/composables/useToast'
import { useUserStore } from '@/stores/users'
import type {
  ManagedUser,
  ManagedUserQuery,
  ResetPasswordResult,
  UserStatus,
} from '@/types/domain'
import { formatDateTime } from '@/utils/format'

const store = useUserStore()
const toast = useToast()
const route = useRoute()
const keyword = ref(
  typeof route.query.keyword === 'string' ? route.query.keyword : '',
)
const status = ref<UserStatus | ''>(
  typeof route.query.status === 'string'
    ? (route.query.status as UserStatus)
    : '',
)
const selectedId = ref('')
const action = ref<UserAction | null>(null)
const passwordResult = ref<ResetPasswordResult | null>(null)

const columns: DataTableColumn[] = [
  { key: 'user', label: '用户', width: '27%' },
  { key: 'status', label: '状态', width: '13%' },
  { key: 'credits', label: '可用次数', width: '12%', align: 'right' },
  { key: 'usage', label: '兑换 / 消耗', width: '16%', align: 'right' },
  { key: 'business', label: '任务 / 工单', width: '14%', align: 'right' },
  { key: 'lastLogin', label: '最近登录', width: '15%' },
  { key: 'actions', label: '', width: '64px', align: 'right' },
]

function currentQuery(page = 1): ManagedUserQuery {
  return {
    page,
    pageSize: store.users.pageSize,
    keyword: keyword.value.trim() || undefined,
    status: status.value || undefined,
  }
}

async function load(page = 1): Promise<void> {
  try {
    await store.loadUsers(currentQuery(page))
  } catch (error) {
    toast.error({
      title: '用户列表加载失败',
      message: error instanceof Error ? error.message : '请稍后重试',
    })
  }
}

async function openUser(user: ManagedUser): Promise<void> {
  selectedId.value = user.id
  try {
    await store.loadUser(user.id)
  } catch (error) {
    selectedId.value = ''
    toast.error({
      title: '用户详情加载失败',
      message: error instanceof Error ? error.message : '请稍后重试',
    })
  }
}

function openAction(nextAction: UserAction): void {
  passwordResult.value = null
  action.value = nextAction
}

async function submitAction(payload: UserActionPayload): Promise<void> {
  const user = store.currentUser
  if (!user || !action.value) return
  try {
    if (action.value === 'adjust') {
      await store.adjustCredits(user.id, {
        amount: payload.amount ?? 0,
        reason: payload.reason ?? '',
        referenceNo: payload.referenceNo,
      })
      toast.success('用户次数已调整')
      action.value = null
    } else if (action.value === 'disable' || action.value === 'enable') {
      await store.setStatus(
        user.id,
        action.value === 'disable' ? 'disabled' : 'active',
        payload.reason ?? '',
      )
      toast.success(action.value === 'disable' ? '用户已停用' : '用户已恢复')
      action.value = null
    } else {
      passwordResult.value = await store.resetPassword(user.id)
      toast.success('临时密码已生成')
    }
  } catch (error) {
    toast.error({
      title: '用户操作未完成',
      message: error instanceof Error ? error.message : '请稍后重试',
    })
  }
}

function closeDrawer(): void {
  if (store.isMutating) return
  selectedId.value = ''
  store.currentUser = null
  action.value = null
  passwordResult.value = null
}

onMounted(() => void load())
</script>

<template>
  <main class="page">
    <header class="page__header">
      <div>
        <p class="page__eyebrow">Customer accounts</p>
        <h1 class="page__title">用户与次数</h1>
        <p class="page__description">
          查询账号、次数余额和完整业务链路。人工调整会生成不可修改的次数流水。
        </p>
      </div>
      <div class="page__actions">
        <BaseButton variant="secondary" :loading="store.isLoading" @click="load(store.users.page)">
          <template #icon><RefreshCw :size="16" /></template>
          刷新
        </BaseButton>
      </div>
    </header>

    <FilterBar>
      <FormField label="检索用户" for-id="user-search">
        <div class="search-control">
          <Search :size="16" />
          <input
            id="user-search"
            v-model="keyword"
            class="form-control"
            type="search"
            placeholder="用户邮箱或编号"
            @keyup.enter="load()"
          />
        </div>
      </FormField>
      <FormField label="账号状态" for-id="user-status">
        <select id="user-status" v-model="status" class="form-control" @change="load()">
          <option value="">全部状态</option>
          <option v-for="(label, value) in USER_STATUS_LABELS" :key="value" :value="value">
            {{ label }}
          </option>
        </select>
      </FormField>
      <template #actions>
        <BaseButton @click="load()">
          <template #icon><Search :size="16" /></template>
          查询
        </BaseButton>
      </template>
    </FilterBar>

    <section class="table-section" aria-label="用户列表">
      <div class="table-summary">
        <span><UsersRound :size="15" />平台注册用户</span>
        <strong>{{ store.users.total }} 人</strong>
      </div>
      <BaseDataTable
        :columns="columns"
        :loading="store.isLoading"
        :has-rows="store.users.items.length > 0"
        empty-title="没有匹配的用户"
        empty-description="调整邮箱关键词或账号状态后再试。"
        :min-width="980"
      >
        <template #body>
          <tr
            v-for="user in store.users.items"
            :key="user.id"
            tabindex="0"
            @click="openUser(user)"
            @keyup.enter="openUser(user)"
          >
            <td>
              <div class="primary-cell">
                <strong>{{ user.email }}</strong>
                <span class="mono">{{ user.id }}</span>
              </div>
            </td>
            <td><UserStatusBadge :status="user.status" /></td>
            <td class="number-cell balance">{{ user.balance }}</td>
            <td class="number-cell">
              <span>{{ user.totalRedeemed }}</span>
              <i>/</i>
              <span>{{ user.totalConsumed }}</span>
            </td>
            <td class="number-cell">
              <span>{{ user.taskCount }}</span>
              <i>/</i>
              <span>{{ user.ticketCount }}</span>
            </td>
            <td class="muted-cell">{{ formatDateTime(user.lastLoginAt) }}</td>
            <td class="row-action">
              <button aria-label="查看用户详情" @click.stop="openUser(user)">
                <Eye :size="17" />
              </button>
            </td>
          </tr>
        </template>
      </BaseDataTable>
      <PaginationBar
        :page="store.users.page"
        :page-size="store.users.pageSize"
        :total="store.users.total"
        :has-more="store.users.hasMore"
        :loading="store.isLoading"
        @change="load"
      />
    </section>
  </main>

  <UserDetailDrawer
    :open="Boolean(selectedId)"
    :user="store.currentUser"
    :loading="store.isLoading"
    @close="closeDrawer"
    @action="openAction"
  />
  <UserActionModal
    :action="action"
    :user="store.currentUser"
    :loading="store.isMutating"
    :password-result="passwordResult"
    @close="action = null"
    @submit="submitAction"
  />
</template>

<style scoped>
.search-control {
  position: relative;
  width: min(360px, 100%);
}

.search-control svg {
  position: absolute;
  z-index: 1;
  top: 14px;
  left: 12px;
  color: var(--ink-faint);
}

.search-control input {
  padding-left: 37px;
}

select.form-control {
  min-width: 180px;
}

.table-section {
  margin-top: 18px;
}

.table-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 2px 10px;
  color: var(--ink-muted);
  font-size: 12px;
}

.table-summary span {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

.table-summary strong {
  color: var(--ink);
  font-family: var(--font-mono);
}

tbody tr {
  cursor: pointer;
}

.primary-cell {
  display: grid;
  gap: 3px;
}

.primary-cell strong {
  font-size: 12px;
}

.primary-cell span,
.muted-cell {
  color: var(--ink-muted);
  font-size: 10px;
}

.number-cell {
  color: var(--ink-muted);
  font-family: var(--font-mono);
  text-align: right;
}

.number-cell i {
  padding: 0 5px;
  color: var(--ink-faint);
  font-style: normal;
}

.number-cell.balance {
  color: var(--ink);
  font-size: 15px;
  font-weight: 750;
}

.row-action {
  text-align: right;
}

.row-action button {
  display: inline-grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--ink-muted);
}

.row-action button:hover {
  background: var(--primary-soft);
  color: var(--primary);
}
</style>
