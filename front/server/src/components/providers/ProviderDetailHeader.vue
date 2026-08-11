<script setup lang="ts">
import {
  KeyRound,
  Pencil,
  Power,
  RefreshCw,
  Trash2,
  Wifi,
} from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import type { AIProvider } from '@/types'
import ProviderStatusBadge from './ProviderStatusBadge.vue'

const props = defineProps<{
  provider: AIProvider
  modelCount: number
  testing?: boolean
  busy?: boolean
}>()

const emit = defineEmits<{
  edit: [provider: AIProvider]
  rotateKey: [provider: AIProvider]
  test: [provider: AIProvider]
  toggleEnabled: [provider: AIProvider]
  delete: [provider: AIProvider]
}>()

function formatTestTime(value?: string) {
  if (!value) return '从未测试'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(new Date(value))
}
</script>

<template>
  <header class="provider-header">
    <div class="provider-title">
      <span class="provider-index data-mono">{{ props.provider.code }}</span>
      <div>
        <div class="name-line">
          <h2>{{ props.provider.name }}</h2>
          <ProviderStatusBadge :status="props.provider.connectionStatus" />
          <span v-if="!props.provider.enabled" class="disabled-label">
            已停用
          </span>
        </div>
        <p class="data-mono">{{ props.provider.baseUrl }}</p>
      </div>
    </div>

    <div class="header-actions">
      <BaseButton
        variant="secondary"
        size="sm"
        :loading="props.testing"
        :disabled="props.busy"
        @click="emit('test', props.provider)"
      >
        <Wifi :size="15" aria-hidden="true" />
        测试连接
      </BaseButton>
      <BaseButton
        variant="ghost"
        size="sm"
        :disabled="props.busy"
        @click="emit('edit', props.provider)"
      >
        <Pencil :size="15" aria-hidden="true" />
        编辑
      </BaseButton>
      <BaseButton
        variant="ghost"
        size="sm"
        :disabled="props.busy"
        @click="emit('rotateKey', props.provider)"
      >
        <KeyRound :size="15" aria-hidden="true" />
        轮换密钥
      </BaseButton>
      <BaseButton
        :variant="props.provider.enabled ? 'danger' : 'secondary'"
        size="sm"
        :disabled="props.busy"
        @click="emit('toggleEnabled', props.provider)"
      >
        <Power :size="15" aria-hidden="true" />
        {{ props.provider.enabled ? '停用' : '启用' }}
      </BaseButton>
      <BaseButton
        variant="ghost"
        size="sm"
        :disabled="props.busy"
        title="删除服务商"
        @click="emit('delete', props.provider)"
      >
        <Trash2 :size="15" aria-hidden="true" />
        删除
      </BaseButton>
    </div>

    <div class="connection-ledger">
      <div>
        <span>API Key</span>
        <strong class="data-mono">{{ props.provider.maskedApiKey }}</strong>
      </div>
      <div>
        <span>协议</span>
        <strong>OpenAI Compatible</strong>
      </div>
      <div>
        <span>已配置模型</span>
        <strong>{{ props.modelCount }} 个</strong>
      </div>
      <div>
        <span>最近测试</span>
        <strong>
          {{
            formatTestTime(props.provider.lastTest?.testedAt)
          }}
        </strong>
      </div>
      <div class="test-message">
        <RefreshCw :size="13" aria-hidden="true" />
        <span>
          {{
            props.provider.lastTest?.message ||
              '保存连接信息后执行一次测试，才能绑定平台模型。'
          }}
        </span>
      </div>
    </div>
  </header>
</template>

<style scoped>
.provider-header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 16px;
  padding: 18px;
  background: #fff;
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 8px;
}

.provider-title {
  display: flex;
  gap: 13px;
  align-items: center;
  min-width: 0;
}

.provider-index {
  display: grid;
  flex: 0 0 auto;
  width: 54px;
  height: 54px;
  color: var(--color-primary, #236c62);
  font-size: 11px;
  background: #edf4f2;
  border: 1px solid #d4e4e0;
  border-radius: 7px;
  place-items: center;
}

.provider-title > div {
  min-width: 0;
}

.name-line {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

h2 {
  margin: 0;
  font-family: var(--font-display, serif);
  font-size: 21px;
  font-weight: 600;
}

.provider-title p {
  margin: 5px 0 0;
  overflow: hidden;
  color: var(--color-text-muted, #68716f);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.disabled-label {
  color: var(--color-danger, #b8574b);
  font-size: 10px;
}

.header-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  align-items: center;
  justify-content: flex-end;
}

.connection-ledger {
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: repeat(4, minmax(120px, 1fr));
  gap: 1px;
  overflow: hidden;
  background: var(--color-border-soft, #edf0ef);
  border: 1px solid var(--color-border-soft, #edf0ef);
  border-radius: 7px;
}

.connection-ledger > div {
  display: grid;
  gap: 5px;
  min-height: 56px;
  padding: 10px 12px;
  background: #f9faf9;
}

.connection-ledger span {
  color: var(--color-text-muted, #68716f);
  font-size: 9px;
}

.connection-ledger strong {
  overflow: hidden;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.connection-ledger .test-message {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  min-height: 36px;
  color: var(--color-text-muted, #68716f);
  background: #fff;
}

@media (max-width: 900px) {
  .provider-header {
    grid-template-columns: 1fr;
  }

  .header-actions {
    justify-content: flex-start;
  }

  .connection-ledger {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 520px) {
  .provider-title {
    align-items: flex-start;
  }

  .connection-ledger {
    grid-template-columns: 1fr;
  }
}
</style>
