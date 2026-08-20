<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Check, Clipboard, FlaskConical, ShieldCheck } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import IconButton from '@/components/base/IconButton.vue'
import type { AIModel, AIProvider } from '@/types'
import {
  buildImageModelTestRequests,
  type ModelTestRequestKind,
} from '@/utils/providerTest'

const props = defineProps<{
  open: boolean
  model: AIModel | null
  provider: AIProvider | null
  loading?: boolean
  error?: string
}>()

const emit = defineEmits<{
  close: []
  confirm: []
}>()

const copied = ref<ModelTestRequestKind | null>(null)
const requests = computed(() =>
  props.model && props.provider
    ? buildImageModelTestRequests(props.provider, props.model)
    : [],
)
const result = computed(() => props.model?.lastTest)

watch(
  () => props.open,
  (open) => {
    if (open) copied.value = null
  },
)

async function copyRequest(
  kind: ModelTestRequestKind,
  curl: string,
): Promise<void> {
  await navigator.clipboard.writeText(curl)
  copied.value = kind
}
</script>

<template>
  <BaseModal
    :open="open"
    title="执行生图模型能力测试"
    :description="`${model?.displayName || ''} 将按下列顺序请求服务商，可能产生费用。`"
    width="large"
    :close-on-backdrop="!loading"
    @close="!loading && emit('close')"
  >
    <div class="test-intro">
      <FlaskConical :size="18" aria-hidden="true" />
      <p>
        后端会使用已保存的真实 API Key。这里展示的是等价脱敏请求，可复制后将
        <code>&lt;API_KEY&gt;</code> 替换为临时 Key 进行独立验证。
      </p>
    </div>

    <div
      v-if="error || result"
      class="test-result"
      :class="{
        success: !error && result?.success,
        failed: Boolean(error) || result?.success === false,
      }"
      role="status"
    >
      <strong>
        {{ error ? '请求未完成' : result?.success ? '能力测试通过' : '能力测试失败' }}
      </strong>
      <span>{{ error || result?.message }}</span>
    </div>

    <div v-if="result?.requestSummary" class="request-summary" aria-live="polite">
      <span>实际请求摘要</span>
      <code>
        {{ result.requestSummary.method }} {{ result.requestSummary.path }}
        · HTTP {{ result.requestSummary.status || '—' }}
        · {{ result.requestSummary.latencyMs }} ms
        <template v-if="result.requestSummary.requestId">
          · {{ result.requestSummary.requestId }}
        </template>
      </code>
    </div>

    <div class="request-list">
      <section
        v-for="request in requests"
        :key="request.kind"
        class="request-preview"
      >
        <header>
          <div>
            <span>{{ request.method }}</span>
            <strong>{{ request.label }}</strong>
          </div>
          <IconButton
            :label="copied === request.kind ? '已复制请求' : `复制${request.label}请求`"
            @click="copyRequest(request.kind, request.curl)"
          >
            <Check v-if="copied === request.kind" :size="16" />
            <Clipboard v-else :size="16" />
          </IconButton>
        </header>
        <code class="request-url">{{ request.url }}</code>
        <pre><code>{{ request.curl }}</code></pre>
      </section>
    </div>

    <div class="security-note">
      <ShieldCheck :size="16" aria-hidden="true" />
      页面、响应和日志都不会输出真实 API Key。生图响应较慢时，测试最多等待约 3 分钟。
    </div>

    <template #footer>
      <BaseButton
        variant="secondary"
        :disabled="loading"
        @click="emit('close')"
      >
        {{ result || error ? '关闭' : '取消' }}
      </BaseButton>
      <BaseButton
        :loading="loading"
        :disabled="!requests.length"
        @click="emit('confirm')"
      >
        {{ result || error ? '重新测试' : '确认并测试' }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<style scoped>
.test-intro,
.security-note,
.test-result {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  border-radius: 8px;
  font-size: 13px;
  line-height: 1.65;
}

.test-intro {
  padding: 12px 14px;
  background: var(--primary-soft);
  color: var(--primary);
}

.test-intro svg,
.security-note svg {
  flex: 0 0 auto;
  margin-top: 2px;
}

.test-intro code {
  padding: 1px 4px;
  border-radius: 4px;
  background: rgb(255 255 255 / 66%);
  font-family: var(--font-mono);
}

.test-result {
  display: grid;
  gap: 2px;
  margin-top: 12px;
  padding: 11px 14px;
  border: 1px solid var(--border);
}

.test-result.success {
  border-color: rgb(35 108 98 / 22%);
  background: var(--primary-soft);
  color: var(--primary);
}

.test-result.failed {
  border-color: rgb(184 87 75 / 24%);
  background: #fbefed;
  color: var(--danger);
}

.request-summary {
  display: grid;
  gap: 5px;
  margin-top: 10px;
  padding: 10px 14px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-soft);
  color: var(--ink-muted);
  font-size: 11px;
}

.request-summary code {
  color: var(--ink);
  font-family: var(--font-mono);
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.request-list {
  display: grid;
  gap: 14px;
  margin-top: 16px;
}

.request-preview {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: #fff;
}

.request-preview > header {
  display: flex;
  min-height: 48px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 7px 10px 7px 14px;
  border-bottom: 1px solid var(--border);
}

.request-preview > header > div {
  display: flex;
  align-items: center;
  gap: 9px;
}

.request-preview > header span {
  padding: 3px 6px;
  border-radius: 4px;
  background: var(--ink);
  color: #fff;
  font-family: var(--font-mono);
  font-size: 10px;
}

.request-preview > header strong {
  font-size: 13px;
}

.request-url {
  display: block;
  overflow: hidden;
  padding: 9px 14px;
  color: var(--ink-muted);
  font-family: var(--font-mono);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.request-preview pre {
  max-height: 220px;
  overflow: auto;
  margin: 0;
  padding: 14px;
  background: #202625;
  color: #eef4f1;
  font-family: var(--font-mono);
  font-size: 11px;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

.security-note {
  margin-top: 14px;
  color: var(--ink-muted);
}

@media (max-width: 640px) {
  .request-preview pre {
    max-height: 190px;
    font-size: 10px;
  }

  .request-preview > header {
    padding-left: 11px;
  }
}
</style>
