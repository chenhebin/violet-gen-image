<script setup lang="ts">
import { computed, ref } from 'vue'
import { Plus, Search, ServerCog } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import type { AIProvider } from '@/types'
import ProviderStatusBadge from './ProviderStatusBadge.vue'

const props = defineProps<{
  providers: AIProvider[]
  selectedId?: string | null
  loading?: boolean
}>()

const emit = defineEmits<{
  select: [provider: AIProvider]
  create: []
}>()

const keyword = ref('')
const filtered = computed(() => {
  const query = keyword.value.trim().toLowerCase()
  if (!query) return props.providers
  return props.providers.filter(
    (provider) =>
      provider.name.toLowerCase().includes(query) ||
      provider.code.toLowerCase().includes(query) ||
      provider.baseUrl.toLowerCase().includes(query),
  )
})
</script>

<template>
  <aside class="provider-rail">
    <div class="rail-heading">
      <div>
        <span>连接目录</span>
        <strong>{{ props.providers.length }} 个服务商</strong>
      </div>
      <BaseButton size="sm" @click="emit('create')">
        <Plus :size="15" aria-hidden="true" />
        新增
      </BaseButton>
    </div>

    <label class="rail-search">
      <Search :size="15" aria-hidden="true" />
      <input v-model="keyword" placeholder="搜索服务商" />
    </label>

    <div class="provider-list">
      <button
        v-for="provider in filtered"
        :key="provider.id"
        type="button"
        class="provider-item"
        :class="{ active: provider.id === props.selectedId }"
        @click="emit('select', provider)"
      >
        <span class="provider-icon" aria-hidden="true">
          <ServerCog :size="18" />
        </span>
        <span class="provider-copy">
          <strong>{{ provider.name }}</strong>
          <small class="data-mono">{{ provider.code }}</small>
        </span>
        <span
          class="health-dot"
          :class="provider.connectionStatus"
          :title="provider.connectionStatus"
        />
        <ProviderStatusBadge
          class="provider-badge"
          :status="provider.connectionStatus"
        />
        <span v-if="!provider.enabled" class="disabled-note">已停用</span>
      </button>

      <div v-if="props.loading && !props.providers.length" class="rail-empty">
        正在读取服务商...
      </div>
      <div v-else-if="!filtered.length" class="rail-empty">
        没有匹配的服务商
      </div>
    </div>

    <div class="rail-note">
      <span>连接配置只保存在服务端</span>
      <small>Client 始终通过映研 `/api` 访问</small>
    </div>
  </aside>
</template>

<style scoped>
.provider-rail {
  display: grid;
  grid-template-rows: auto auto minmax(240px, 1fr) auto;
  min-width: 0;
  overflow: hidden;
  background: #fff;
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 8px;
}

.rail-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 68px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--color-border-soft, #edf0ef);
}

.rail-heading > div {
  display: grid;
  gap: 3px;
}

.rail-heading span {
  color: var(--color-text-muted, #68716f);
  font-size: 10px;
}

.rail-heading strong {
  font-size: 13px;
}

.rail-search {
  display: flex;
  gap: 8px;
  align-items: center;
  height: 38px;
  margin: 11px 12px 7px;
  padding: 0 10px;
  color: var(--color-text-muted, #68716f);
  background: var(--color-canvas, #f3f5f4);
  border: 1px solid transparent;
  border-radius: 6px;
}

.rail-search:focus-within {
  background: #fff;
  border-color: var(--color-primary, #236c62);
}

.rail-search input {
  min-width: 0;
  width: 100%;
  padding: 0;
  background: transparent;
  border: 0;
  outline: none;
}

.provider-list {
  display: grid;
  align-content: start;
  gap: 4px;
  padding: 5px 7px 10px;
  overflow-y: auto;
}

.provider-item {
  position: relative;
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr) auto;
  grid-template-rows: auto auto;
  gap: 2px 9px;
  align-items: center;
  min-height: 66px;
  padding: 8px 10px;
  color: var(--color-text, #1b1f1f);
  text-align: left;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 7px;
  cursor: pointer;
}

.provider-item:hover {
  background: #f6f8f7;
}

.provider-item.active {
  background: #edf4f2;
  border-color: #d3e4df;
}

.provider-icon {
  grid-row: 1 / 3;
  display: grid;
  width: 36px;
  height: 36px;
  color: var(--color-primary, #236c62);
  background: #fff;
  border: 1px solid var(--color-border-soft, #e8eceb);
  border-radius: 6px;
  place-items: center;
}

.provider-copy {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.provider-copy strong,
.provider-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.provider-copy strong {
  font-size: 12px;
}

.provider-copy small {
  color: var(--color-text-muted, #68716f);
  font-size: 9px;
}

.health-dot {
  width: 7px;
  height: 7px;
  background: #9ba3a1;
  border-radius: 50%;
}

.health-dot.healthy {
  background: #4d8b81;
}

.health-dot.error {
  background: #b8574b;
}

.provider-badge {
  display: none;
}

.disabled-note {
  grid-column: 2 / -1;
  color: var(--color-danger, #b8574b);
  font-size: 9px;
}

.rail-empty {
  display: grid;
  min-height: 150px;
  color: var(--color-text-muted, #68716f);
  font-size: 11px;
  place-content: center;
}

.rail-note {
  display: grid;
  gap: 3px;
  padding: 12px 14px;
  color: var(--color-text-muted, #68716f);
  background: #f8f9f8;
  border-top: 1px solid var(--color-border-soft, #edf0ef);
}

.rail-note span {
  font-size: 10px;
  font-weight: 700;
}

.rail-note small {
  font-size: 9px;
}

@media (max-width: 840px) {
  .provider-rail {
    grid-template-rows: auto auto auto auto;
  }

  .provider-list {
    grid-auto-columns: minmax(220px, 1fr);
    grid-auto-flow: column;
    overflow-x: auto;
  }
}
</style>
