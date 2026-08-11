<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  CircleAlert,
  FlaskConical,
  Image,
  MessageSquareText,
  Pencil,
  Plus,
  Route,
  Trash2,
} from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import StatusBadge from '@/components/base/StatusBadge.vue'
import { MODEL_TEST_STATUS_LABELS, MODEL_TYPE_LABELS } from '@/config'
import type { AIModel, AIProvider, ModelType } from '@/types'
const props = defineProps<{
  provider: AIProvider
  models: AIModel[]
  testingId?: string | null
  bindingId?: string | null
  busy?: boolean
}>()
const emit = defineEmits<{
  create: []
  edit: [model: AIModel]
  test: [model: AIModel]
  bind: [model: AIModel]
  unbind: [model: AIModel]
  delete: [model: AIModel]
}>()
const activeType = ref<'all' | ModelType>('all')
const filteredModels = computed(() =>
  activeType.value === 'all'
    ? props.models
    : props.models.filter((model) => model.type === activeType.value),
)
function capabilityLabels(model: AIModel) {
  if (model.type === 'chat') {
    return [
      model.capabilities.promptOptimization ? '提示词优化' : '',
      model.capabilities.visionInput ? '图片理解' : '',
    ].filter(Boolean)
  }
  return [
    model.capabilities.textToImage ? '文生图' : '',
    model.capabilities.imageToImage ? '图生图 / 编辑' : '',
  ].filter(Boolean)
}
function bindDisabledReason(model: AIModel) {
  if (!props.provider.enabled) return '服务商已停用'
  if (!model.enabled) return '模型已停用'
  if (model.connectionStatus !== 'healthy') return '模型尚未通过能力测试'
  if (
    model.type === 'image' &&
    (!model.capabilities.textToImage || !model.capabilities.imageToImage)
  ) {
    return '生图模型需同时支持文生图和图生图'
  }
  if (
    model.type === 'chat' &&
    (!model.capabilities.promptOptimization || !model.capabilities.visionInput)
  ) {
    return '对话模型需同时支持提示词优化和图片理解'
  }
  return ''
}

function testTone(model: AIModel): 'success' | 'danger' | 'neutral' {
  return model.connectionStatus === 'healthy'
    ? 'success'
    : model.connectionStatus === 'error' ? 'danger' : 'neutral'
}
</script>

<template>
  <section class="models-ledger">
    <header class="ledger-heading">
      <div>
        <span>模型台账</span>
        <h3>连接能力与平台用途</h3>
        <small>启用、测试通过和平台绑定是三个独立状态。</small>
      </div>
      <div class="heading-actions">
        <div class="type-tabs" aria-label="模型类型筛选">
          <button
            v-for="option in [
              { value: 'all', label: '全部' },
              { value: 'chat', label: '对话' },
              { value: 'image', label: '生图' },
            ] as const"
            :key="option.value"
            type="button"
            :class="{ active: activeType === option.value }"
            @click="activeType = option.value"
          >
            {{ option.label }}
          </button>
        </div>
        <BaseButton size="sm" @click="emit('create')">
          <Plus :size="15" aria-hidden="true" />
          新增模型
        </BaseButton>
      </div>
    </header>

    <div class="models-table">
      <div class="table-head">
        <span>模型</span>
        <span>能力</span>
        <span>测试状态</span>
        <span>平台用途</span>
        <span>操作</span>
      </div>

      <article v-for="model in filteredModels" :key="model.id">
        <div class="model-identity">
          <span
            class="model-icon"
            :class="model.type"
            aria-hidden="true"
          >
            <MessageSquareText v-if="model.type === 'chat'" :size="18" />
            <Image v-else :size="18" />
          </span>
          <span class="model-copy">
            <strong>{{ model.displayName }}</strong>
            <small class="data-mono">{{ model.modelId }}</small>
            <em :class="model.enabled ? 'enabled' : 'disabled'">
              {{ model.enabled ? '已启用' : '已停用' }}
            </em>
          </span>
        </div>

        <div class="capabilities">
          <span class="type-label">{{ MODEL_TYPE_LABELS[model.type] }}</span>
          <span
            v-for="label in capabilityLabels(model)"
            :key="label"
            class="capability"
          >
            {{ label }}
          </span>
          <span v-if="!capabilityLabels(model).length" class="empty-capability">
            未声明能力
          </span>
        </div>
        <div class="test-status">
          <StatusBadge :tone="testTone(model)">
            {{ MODEL_TEST_STATUS_LABELS[model.connectionStatus] }}
          </StatusBadge>
          <small v-if="model.lastTestAt">
            {{
              new Intl.DateTimeFormat('zh-CN', {
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                hour12: false,
              }).format(new Date(model.lastTestAt))
            }}
          </small>
        </div>
        <div class="platform-binding">
          <button
            type="button"
            class="binding-switch"
            :class="{ active: model.isPlatformModel }"
            :disabled="
              Boolean(!model.isPlatformModel && bindDisabledReason(model)) ||
                props.busy
            "
            :title="
              bindDisabledReason(model) ||
                (model.isPlatformModel ? '解除平台绑定' : '设为平台模型')
            "
            :aria-pressed="model.isPlatformModel"
            @click="
              model.isPlatformModel
                ? emit('unbind', model)
                : emit('bind', model)
            "
          >
            <span aria-hidden="true" />
          </button>
          <span>
            {{
              model.isPlatformModel
                ? model.type === 'chat'
                  ? '平台对话模型'
                  : '平台生图模型'
                : '未用于平台'
            }}
          </span>
          <CircleAlert
            v-if="bindDisabledReason(model) && !model.isPlatformModel"
            :size="14"
            :aria-label="bindDisabledReason(model)"
          />
        </div>
        <div class="model-actions">
          <BaseButton
            variant="ghost"
            size="sm"
            :loading="props.testingId === model.id"
            :disabled="props.busy && props.testingId !== model.id"
            @click="emit('test', model)"
          >
            <FlaskConical :size="15" aria-hidden="true" />
            测试
          </BaseButton>
          <BaseButton
            variant="ghost"
            size="sm"
            :disabled="props.busy"
            @click="emit('edit', model)"
          >
            <Pencil :size="15" aria-hidden="true" />
            编辑
          </BaseButton>
          <BaseButton
            variant="ghost"
            size="sm"
            :disabled="props.busy"
            title="删除模型"
            @click="emit('delete', model)"
          >
            <Trash2 :size="15" aria-hidden="true" />
            删除
          </BaseButton>
        </div>
      </article>

      <div v-if="!filteredModels.length" class="empty-models">
        <Route :size="24" aria-hidden="true" />
        <strong>当前分类还没有模型</strong>
        <span>新增模型并通过能力测试后，才能将它绑定到生图平台。</span>
        <BaseButton size="sm" @click="emit('create')">
          <Plus :size="15" aria-hidden="true" />
          新增模型
        </BaseButton>
      </div>
    </div>
  </section>
</template>

<style scoped>
.models-ledger {
  min-width: 0;
  overflow: hidden;
  background: #fff;
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 8px;
}

.ledger-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  min-height: 70px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border-soft, #edf0ef);
}

.ledger-heading > div:first-child {
  display: grid;
  gap: 3px;
}

.ledger-heading span {
  color: var(--color-text-muted, #68716f);
  font-size: 9px;
}

.ledger-heading h3 {
  margin: 0;
  font-family: var(--font-display, serif);
  font-size: 16px;
  font-weight: 600;
}

.heading-actions,
.type-tabs {
  display: flex;
  gap: 8px;
  align-items: center;
}

.type-tabs {
  padding: 3px;
  background: #f0f2f1;
  border-radius: 6px;
}

.type-tabs button {
  min-height: 28px;
  padding: 0 10px;
  color: var(--color-text-muted, #68716f);
  font-size: 10px;
  background: transparent;
  border: 0;
  border-radius: 4px;
  cursor: pointer;
}

.type-tabs button.active {
  color: var(--color-text, #1b1f1f);
  font-weight: 700;
  background: #fff;
  box-shadow: 0 1px 4px rgb(31 40 38 / 8%);
}

.models-table {
  min-width: 0;
  overflow-x: auto;
}

.table-head,
article {
  display: grid;
  grid-template-columns:
    minmax(210px, 1.35fr)
    minmax(190px, 1.15fr)
    minmax(118px, 0.7fr)
    minmax(180px, 1fr)
    206px;
  gap: 12px;
  align-items: center;
  min-width: 1000px;
  padding: 0 15px;
}

.table-head {
  min-height: 38px;
  color: var(--color-text-muted, #68716f);
  font-size: 9px;
  font-weight: 700;
  background: #f8f9f8;
  border-bottom: 1px solid var(--color-border-soft, #edf0ef);
}

article {
  min-height: 84px;
  border-bottom: 1px solid var(--color-border-soft, #edf0ef);
  transition: background-color 140ms ease;
}

article:hover {
  background: #f9fbfa;
}

article:last-of-type {
  border-bottom: 0;
}

.model-identity {
  display: flex;
  gap: 10px;
  align-items: center;
  min-width: 0;
}

.model-icon {
  display: grid;
  flex: 0 0 auto;
  width: 38px;
  height: 38px;
  color: #376f67;
  background: #e9f2f0;
  border-radius: 6px;
  place-items: center;
}

.model-icon.image {
  color: #856820;
  background: #f7f0df;
}

.model-copy {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.model-copy strong,
.model-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-copy strong {
  font-size: 12px;
}

.model-copy small {
  color: var(--color-text-muted, #68716f);
  font-size: 9px;
}

.model-copy em {
  color: var(--color-danger, #b8574b);
  font-size: 9px;
  font-style: normal;
}

.capabilities {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.capabilities span {
  min-height: 22px;
  padding: 3px 7px;
  font-size: 9px;
  line-height: 16px;
  border-radius: 4px;
}

.type-label {
  color: #4f5856;
  background: #edf0ef;
}

.capability {
  color: #2d665d;
  background: #eaf3f0;
}

.empty-capability {
  color: var(--color-danger, #b8574b);
  background: #f9ecea;
}

.test-status {
  display: grid;
  gap: 5px;
  justify-items: start;
}

.test-status small {
  color: var(--color-text-muted, #68716f);
  font-size: 9px;
}

</style>
<style scoped src="./ModelsLedgerSupplement.css"></style>
