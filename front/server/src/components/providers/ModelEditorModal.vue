<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { Image, MessageSquareText, Sparkles } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import FormField from '@/components/base/FormField.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseToggle from '@/components/base/BaseToggle.vue'
import type {
  AIModel,
  AIProvider,
  CreateAIModelPayload,
  ModelType,
  UpdateAIModelPayload,
} from '@/types'

const props = defineProps<{
  open: boolean
  provider: AIProvider | null
  model?: AIModel | null
  loading?: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [payload: CreateAIModelPayload | UpdateAIModelPayload]
  unchanged: []
}>()

const form = reactive({
  displayName: '',
  modelId: '',
  type: 'chat' as ModelType,
  enabled: true,
  promptOptimization: true,
  visionInput: true,
  textToImage: true,
  imageToImage: true,
  touched: false,
})

const isEditing = computed(() => Boolean(props.model))
const errors = computed(() => ({
  displayName: !form.displayName.trim() ? '请填写模型展示名称' : '',
  modelId: !form.modelId.trim() ? '请填写实际模型 ID' : '',
}))
const valid = computed(() =>
  Object.values(errors.value).every((message) => !message),
)

function capabilitiesForForm() {
  return form.type === 'chat'
    ? {
        promptOptimization: form.promptOptimization,
        visionInput: form.visionInput,
      }
    : {
        textToImage: form.textToImage,
        imageToImage: form.imageToImage,
      }
}

function capabilitiesChanged(model: AIModel): boolean {
  if (model.type === 'chat') {
    return (
      Boolean(model.capabilities.promptOptimization) !==
        form.promptOptimization ||
      Boolean(model.capabilities.visionInput) !== form.visionInput
    )
  }
  return (
    Boolean(model.capabilities.textToImage) !== form.textToImage ||
    Boolean(model.capabilities.imageToImage) !== form.imageToImage
  )
}

function submit() {
  form.touched = true
  if (!props.provider || !valid.value) return

  const displayName = form.displayName.trim()
  const modelId = form.modelId.trim()
  const capabilities = capabilitiesForForm()

  if (props.model) {
    const payload: UpdateAIModelPayload = {}
    if (displayName !== props.model.displayName) payload.displayName = displayName
    if (modelId !== props.model.modelId) payload.modelId = modelId
    if (form.enabled !== props.model.enabled) payload.enabled = form.enabled
    if (capabilitiesChanged(props.model)) payload.capabilities = capabilities
    if (!Object.keys(payload).length) {
      emit('unchanged')
      return
    }
    emit('submit', payload)
    return
  }

  emit('submit', {
    providerId: props.provider.id,
    displayName,
    modelId,
    type: form.type,
    enabled: form.enabled,
    capabilities,
  })
}

watch(
  () => [props.open, props.model, props.provider] as const,
  ([open]) => {
    if (!open) return
    form.displayName = props.model?.displayName || ''
    form.modelId = props.model?.modelId || ''
    form.type = props.model?.type || 'chat'
    form.enabled = props.model?.enabled ?? true
    form.promptOptimization = props.model
      ? Boolean(props.model.capabilities.promptOptimization)
      : true
    form.visionInput = props.model
      ? Boolean(props.model.capabilities.visionInput)
      : true
    form.textToImage = props.model
      ? Boolean(props.model.capabilities.textToImage)
      : true
    form.imageToImage = props.model
      ? Boolean(props.model.capabilities.imageToImage)
      : true
    form.touched = false
  },
  { deep: true },
)
</script>

<template>
  <BaseModal
    :open="props.open"
    :title="isEditing ? '编辑模型' : '新增模型'"
    :description="
      isEditing
        ? '修改模型 ID 或能力后需要重新测试，模型类型不可变更。'
        : `为 ${props.provider?.name || '当前服务商'} 配置实际模型与平台能力。`
    "
    @close="emit('close')"
  >
    <form class="model-form" @submit.prevent="submit">
      <fieldset class="model-types">
        <legend>模型类型</legend>
        <button
          type="button"
          :class="{ active: form.type === 'chat' }"
          :disabled="isEditing"
          @click="form.type = 'chat'"
        >
          <MessageSquareText :size="19" aria-hidden="true" />
          <span>
            <strong>对话模型</strong>
            <small>用于提示词优化</small>
          </span>
        </button>
        <button
          type="button"
          :class="{ active: form.type === 'image' }"
          :disabled="isEditing"
          @click="form.type = 'image'"
        >
          <Image :size="19" aria-hidden="true" />
          <span>
            <strong>生图模型</strong>
            <small>用于文生图与图生图</small>
          </span>
        </button>
      </fieldset>

      <div class="form-grid">
        <FormField
          label="展示名称"
          for-id="model-display-name"
          required
          :error="form.touched ? errors.displayName : ''"
        >
          <input
            id="model-display-name"
            v-model="form.displayName"
            maxlength="60"
            placeholder="例如：主力提示词模型"
          />
        </FormField>
        <FormField
          label="模型 ID"
          for-id="model-id"
          required
          :error="form.touched ? errors.modelId : ''"
          hint="填写服务商文档中的实际标识"
        >
          <input
            id="model-id"
            v-model="form.modelId"
            class="data-mono"
            maxlength="100"
            placeholder="gpt-4.1-mini"
          />
        </FormField>
      </div>

      <section class="capability-panel">
        <header>
          <Sparkles :size="17" aria-hidden="true" />
          <div>
            <strong>平台能力</strong>
            <span>修改能力会使旧测试结果失效</span>
          </div>
        </header>
        <div v-if="form.type === 'chat'" class="toggles">
          <BaseToggle
            v-model="form.promptOptimization"
            label="提示词优化"
            description="允许作为生图平台的对话能力"
          />
          <BaseToggle
            v-model="form.visionInput"
            label="图片理解"
            description="允许读取用户原图与参考图上下文"
          />
        </div>
        <div v-else class="toggles">
          <BaseToggle
            v-model="form.textToImage"
            label="文生图"
            description="根据文本需求生成图片"
          />
          <BaseToggle
            v-model="form.imageToImage"
            label="图生图 / 图片编辑"
            description="接收原图与参考图进行编辑"
          />
          <p
            v-if="!form.textToImage || !form.imageToImage"
            class="capability-warning"
          >
            平台生图模型必须同时通过文生图和图生图测试，此配置暂不能绑定。
          </p>
        </div>
      </section>

      <div class="enabled-toggle">
        <BaseToggle
          v-model="form.enabled"
          label="启用模型"
          description="停用模型前需先解除当前平台绑定"
        />
      </div>

      <div class="actions">
        <BaseButton
          type="button"
          variant="ghost"
          :disabled="props.loading"
          @click="emit('close')"
        >
          取消
        </BaseButton>
        <BaseButton type="submit" :loading="props.loading">
          {{ isEditing ? '保存模型' : '创建模型' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.model-form {
  display: grid;
  gap: 18px;
}

.model-types {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 9px;
  padding: 0;
  border: 0;
}

.model-types legend {
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 700;
}

.model-types button {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  min-height: 65px;
  padding: 11px 12px;
  color: var(--color-text-muted, #68716f);
  text-align: left;
  background: #fff;
  border: 1px solid var(--color-border, #dce1df);
  border-radius: 7px;
  cursor: pointer;
}

.model-types button.active {
  color: var(--color-primary, #236c62);
  background: #f2f7f5;
  border-color: var(--color-primary, #236c62);
}

.model-types button:disabled {
  cursor: not-allowed;
}

.model-types button span {
  display: grid;
  gap: 3px;
}

.model-types strong {
  color: var(--color-text, #1b1f1f);
  font-size: 12px;
}

.model-types small {
  font-size: 10px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 13px;
}

input {
  width: 100%;
}

.capability-panel {
  padding: 13px;
  background: #f7f8f7;
  border: 1px solid var(--color-border-soft, #edf0ef);
  border-radius: 7px;
}

.capability-panel header {
  display: flex;
  gap: 9px;
  align-items: center;
  margin-bottom: 12px;
}

.capability-panel header svg {
  color: var(--color-primary, #236c62);
}

.capability-panel header div {
  display: grid;
  gap: 2px;
}

.capability-panel header strong {
  font-size: 12px;
}

.capability-panel header span {
  color: var(--color-text-muted, #68716f);
  font-size: 9px;
}

.toggles {
  display: grid;
  gap: 10px;
}

.capability-warning {
  margin: 2px 0 0;
  padding: 8px 10px;
  color: #886321;
  font-size: 10px;
  line-height: 1.55;
  background: #fbf5e8;
  border-radius: 5px;
}

.enabled-toggle {
  padding: 11px 13px;
  border: 1px solid var(--color-border-soft, #edf0ef);
  border-radius: 7px;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 9px;
}

@media (max-width: 560px) {
  .model-types,
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
