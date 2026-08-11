<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Plus, ServerCog, ShieldCheck } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import ConfirmDialog from '@/components/base/ConfirmDialog.vue'
import AiConfigDeleteDialogs from '@/components/providers/AiConfigDeleteDialogs.vue'
import ModelEditorModal from '@/components/providers/ModelEditorModal.vue'
import ModelTestModal from '@/components/providers/ModelTestModal.vue'
import ModelsLedger from '@/components/providers/ModelsLedger.vue'
import PlatformBindingsSpine from '@/components/providers/PlatformBindingsSpine.vue'
import ProviderDetailHeader from '@/components/providers/ProviderDetailHeader.vue'
import ProviderEditorModal from '@/components/providers/ProviderEditorModal.vue'
import ProviderRail from '@/components/providers/ProviderRail.vue'
import RotateProviderKeyModal from '@/components/providers/RotateProviderKeyModal.vue'
import { useToast } from '@/composables/useToast'
import { useProviderStore } from '@/stores/providers'
import type {
  AIModel,
  AIProvider,
  CreateAIModelPayload,
  CreateAIProviderPayload,
  ModelType,
  UpdateAIModelPayload,
  UpdateAIProviderPayload,
} from '@/types'
const route = useRoute()
const router = useRouter()
const store = useProviderStore()
const toast = useToast()
const selectedProviderId = ref(String(route.query.provider || ''))
const providerEditorOpen = ref(false)
const editingProvider = ref<AIProvider | null>(null)
const rotateKeyProvider = ref<AIProvider | null>(null)
const modelEditorOpen = ref(false)
const editingModel = ref<AIModel | null>(null)
const testingProviderId = ref<string | null>(null)
const testingModelId = ref<string | null>(null)
const pendingModelTest = ref<AIModel | null>(null)
const modelTestError = ref('')
const pendingBinding = ref<AIModel | null>(null)
const pendingUnbind = ref<ModelType | null>(null)
const pendingProviderToggle = ref<AIProvider | null>(null)
const pendingProviderDelete = ref<AIProvider | null>(null)
const pendingModelDelete = ref<AIModel | null>(null)
const selectedProvider = computed(
  () =>
    store.providers.find(
      (provider) => provider.id === selectedProviderId.value,
    ) || null,
)
const selectedModels = computed(() =>
  store.models.filter(
    (model) => model.providerId === selectedProviderId.value,
  ),
)
const pendingTestProvider = computed(() =>
  pendingModelTest.value
    ? store.providers.find(
        (provider) => provider.id === pendingModelTest.value?.providerId,
      ) || null
    : null,
)
const replacingModel = computed(() => {
  if (!pendingBinding.value) return undefined
  return pendingBinding.value.type === 'chat'
    ? store.currentChatModel
    : store.currentImageModel
})
function errorText(error: unknown) {
  return error instanceof Error ? error.message : '操作未完成，请稍后重试'
}
function selectProvider(provider: AIProvider) {
  selectedProviderId.value = provider.id
  void router.replace({ query: { provider: provider.id } })
}
function createProvider() {
  editingProvider.value = null
  providerEditorOpen.value = true
}
function editProvider(provider: AIProvider) {
  editingProvider.value = provider
  providerEditorOpen.value = true
}
async function saveProvider(
  payload: CreateAIProviderPayload | UpdateAIProviderPayload,
) {
  try {
    if (editingProvider.value) {
      const updated = await store.updateProvider(
        editingProvider.value.id,
        payload as UpdateAIProviderPayload,
      )
      toast.success(`${updated.name} 已更新`)
    } else {
      const created = await store.createProvider(
        payload as CreateAIProviderPayload,
      )
      selectProvider(created)
      toast.success(`${created.name} 已创建，请先测试连接`)
    }
    providerEditorOpen.value = false
  } catch (error) {
    toast.error({ title: '服务商保存失败', message: errorText(error) })
  }
}
async function testProvider(provider: AIProvider) {
  testingProviderId.value = provider.id
  try {
    const result = await store.testProvider(provider.id)
    if (result.connectionStatus === 'healthy') {
      toast.success({
        title: `${result.name} 连接正常`,
        message: result.lastTest?.message,
      })
    } else {
      toast.error({
        title: `${result.name} 连接失败`,
        message: result.lastTest?.message ?? '请检查 Base URL、API Key 和网络配置',
      })
    }
  } catch (error) {
    toast.error({ title: '连接测试失败', message: errorText(error) })
  } finally {
    testingProviderId.value = null
  }
}
async function rotateKey(apiKey: string) {
  if (!rotateKeyProvider.value) return
  try {
    await store.rotateKey(rotateKeyProvider.value.id, apiKey)
    toast.success('API Key 已轮换，请重新测试连接与模型')
    rotateKeyProvider.value = null
  } catch (error) {
    toast.error({ title: '密钥轮换失败', message: errorText(error) })
  }
}
async function toggleProvider() {
  if (!pendingProviderToggle.value) return
  const provider = pendingProviderToggle.value
  try {
    await store.updateProvider(provider.id, {
      enabled: !provider.enabled,
    })
    toast.success(`${provider.name} 已${provider.enabled ? '停用' : '启用'}`)
    pendingProviderToggle.value = null
  } catch (error) {
    toast.error({ title: '服务商状态更新失败', message: errorText(error) })
  }
}
function createModel() {
  editingModel.value = null
  modelEditorOpen.value = true
}
function editModel(model: AIModel) {
  editingModel.value = model
  modelEditorOpen.value = true
}

async function saveModel(
  payload: CreateAIModelPayload | UpdateAIModelPayload,
) {
  try {
    if (editingModel.value) {
      const requiresRetest =
        'modelId' in payload || 'capabilities' in payload
      const updated = await store.updateModel(
        editingModel.value.id,
        payload as UpdateAIModelPayload,
      )
      toast.success(
        requiresRetest
          ? `${updated.displayName} 已更新，请重新测试能力`
          : `${updated.displayName} 已更新，原测试状态保持不变`,
      )
    } else {
      const created = await store.createModel(
        payload as CreateAIModelPayload,
      )
      toast.success(`${created.displayName} 已创建`)
    }
    modelEditorOpen.value = false
  } catch (error) {
    toast.error({ title: '模型保存失败', message: errorText(error) })
  }
}

function closeUnchangedModelEditor() {
  modelEditorOpen.value = false
  editingModel.value = null
  toast.info('模型配置没有变化，未发送更新请求')
}

async function confirmProviderDelete() {
  if (!pendingProviderDelete.value) return
  const provider = pendingProviderDelete.value
  const index = store.providers.findIndex((item) => item.id === provider.id)
  try {
    await store.deleteProvider(provider.id)
    const next = store.providers[Math.min(index, store.providers.length - 1)]
    pendingProviderDelete.value = null
    selectedProviderId.value = next?.id || ''
    await router.replace({ query: next ? { provider: next.id } : {} })
    toast.success(`${provider.name} 已删除`)
  } catch (error) {
    toast.error({ title: '服务商删除失败', message: errorText(error) })
  }
}

async function confirmModelDelete() {
  if (!pendingModelDelete.value) return
  const model = pendingModelDelete.value
  try {
    await store.deleteModel(model.id)
    pendingModelDelete.value = null
    toast.success(`${model.displayName} 已删除`)
  } catch (error) {
    toast.error({ title: '模型删除失败', message: errorText(error) })
  }
}

function requestModelTest(model: AIModel) {
  if (model.type === 'image') {
    modelTestError.value = ''
    pendingModelTest.value = model
  } else {
    void testModel(model)
  }
}

async function testModel(model = pendingModelTest.value) {
  if (!model) return
  const keepDialogOpen = model.type === 'image'
  if (keepDialogOpen) modelTestError.value = ''
  testingModelId.value = model.id
  try {
    const result = await store.testModel(model.id)
    if (keepDialogOpen) pendingModelTest.value = result
    if (result.connectionStatus === 'healthy') {
      toast.success({
        title: `${result.displayName} 能力测试通过`,
        message: result.lastTest?.message ?? '模型能力已验证',
      })
    } else {
      toast.error({
        title: `${result.displayName} 能力测试失败`,
        message:
          result.lastTest?.message ??
          '请先检查服务商连接、模型 ID 与模型能力配置',
      })
    }
  } catch (error) {
    const message = errorText(error)
    if (keepDialogOpen) modelTestError.value = message
    toast.error({ title: '模型测试失败', message })
  } finally {
    testingModelId.value = null
  }
}

function closeModelTest(): void {
  if (testingModelId.value) return
  pendingModelTest.value = null
  modelTestError.value = ''
}

async function confirmBinding() {
  if (!pendingBinding.value) return
  const model = pendingBinding.value
  try {
    await store.bindModel(model.type, model.id)
    toast.success(
      `${model.displayName} 已设为平台${model.type === 'chat' ? '对话' : '生图'}模型`,
    )
    pendingBinding.value = null
  } catch (error) {
    toast.error({ title: '平台模型切换失败', message: errorText(error) })
  }
}

async function confirmUnbind() {
  if (!pendingUnbind.value) return
  const type = pendingUnbind.value
  try {
    await store.bindModel(type, null)
    toast.warning(
      `平台${type === 'chat' ? '提示词优化' : '图片生成'}能力已解除配置`,
    )
    pendingUnbind.value = null
  } catch (error) {
    toast.error({ title: '解除绑定失败', message: errorText(error) })
  }
}

function locateModel(model: AIModel) {
  const provider = store.providers.find(
    (item) => item.id === model.providerId,
  )
  if (provider) selectProvider(provider)
}

onMounted(async () => {
  try {
    await store.loadAll()
    if (!selectedProvider.value && store.providers[0]) {
      selectProvider(store.providers[0])
    }
  } catch (error) {
    toast.error({ title: 'AI 服务配置加载失败', message: errorText(error) })
  }
})

watch(
  () => store.providers,
  (providers) => {
    if (!selectedProvider.value && providers[0]) {
      selectedProviderId.value = providers[0].id
    }
  },
  { deep: true },
)
</script>

<template>
  <main class="page providers-page">
    <header class="page__header">
      <div>
        <p class="page__eyebrow">Model routing desk</p>
        <h1 class="page__title">AI 服务</h1>
        <p class="page__description">
          配置后端实际调用的服务商与模型。Client 只感知映研任务状态，不接触连接地址、密钥或路由细节。
        </p>
      </div>
      <div class="page__actions">
        <div class="security-chip">
          <ShieldCheck :size="16" aria-hidden="true" />
          密钥仅服务端可用
        </div>
        <BaseButton @click="createProvider">
          <Plus :size="16" aria-hidden="true" />
          新增服务商
        </BaseButton>
      </div>
    </header>

    <PlatformBindingsSpine
      :chat-model="store.currentChatModel"
      :image-model="store.currentImageModel"
      :loading="store.isMutating"
      @locate="locateModel"
      @unbind="pendingUnbind = $event"
    />

    <div class="provider-workbench">
      <ProviderRail
        :providers="store.providers"
        :selected-id="selectedProviderId"
        :loading="store.isLoading"
        @select="selectProvider"
        @create="createProvider"
      />

      <section v-if="selectedProvider" class="provider-detail">
        <ProviderDetailHeader
          :provider="selectedProvider"
          :model-count="selectedModels.length"
          :testing="testingProviderId === selectedProvider.id"
          :busy="store.isMutating"
          @edit="editProvider"
          @rotate-key="rotateKeyProvider = $event"
          @test="testProvider"
          @toggle-enabled="pendingProviderToggle = $event"
          @delete="pendingProviderDelete = $event"
        />
        <ModelsLedger
          :provider="selectedProvider"
          :models="selectedModels"
          :testing-id="testingModelId"
          :binding-id="pendingBinding?.id"
          :busy="store.isMutating"
          @create="createModel"
          @edit="editModel"
          @test="requestModelTest"
          @bind="pendingBinding = $event"
          @unbind="pendingUnbind = $event.type"
          @delete="pendingModelDelete = $event"
        />
      </section>

      <section v-else class="provider-empty">
        <ServerCog :size="28" aria-hidden="true" />
        <h2>还没有 AI 服务商</h2>
        <p>先新增服务商，完成连接测试后再配置模型与平台用途。</p>
        <BaseButton @click="createProvider">
          <Plus :size="16" aria-hidden="true" />
          新增服务商
        </BaseButton>
      </section>
    </div>
  </main>

  <ProviderEditorModal
    :open="providerEditorOpen"
    :provider="editingProvider"
    :loading="store.isMutating"
    @close="providerEditorOpen = false"
    @submit="saveProvider"
  />

  <RotateProviderKeyModal
    :open="Boolean(rotateKeyProvider)"
    :provider="rotateKeyProvider"
    :loading="store.isMutating"
    @close="rotateKeyProvider = null"
    @confirm="rotateKey"
  />

  <ModelEditorModal
    :open="modelEditorOpen"
    :provider="selectedProvider"
    :model="editingModel"
    :loading="store.isMutating"
    @close="modelEditorOpen = false"
    @submit="saveModel"
    @unchanged="closeUnchangedModelEditor"
  />

  <ModelTestModal
    :open="Boolean(pendingModelTest)"
    :model="pendingModelTest"
    :provider="pendingTestProvider"
    :loading="testingModelId === pendingModelTest?.id"
    :error="modelTestError"
    @close="closeModelTest"
    @confirm="testModel()"
  />

  <ConfirmDialog
    :open="Boolean(pendingBinding)"
    title="切换平台模型"
    :description="
      replacingModel && replacingModel.id !== pendingBinding?.id
        ? `${pendingBinding?.displayName || ''} 将替换当前的 ${replacingModel.displayName}。切换只影响后续任务，在途任务继续使用创建时快照。`
        : `${pendingBinding?.displayName || ''} 将成为平台当前模型，只影响后续任务。`
    "
    confirm-label="确认切换"
    :loading="store.isMutating"
    @close="pendingBinding = null"
    @confirm="confirmBinding"
  />

  <ConfirmDialog
    :open="Boolean(pendingUnbind)"
    title="解除平台模型"
    :description="`解除后，Client 的${pendingUnbind === 'chat' ? '提示词优化' : '图片生成与编辑'}会暂时不可用，直到绑定新的健康模型。`"
    confirm-label="确认解除"
    danger
    :loading="store.isMutating"
    @close="pendingUnbind = null"
    @confirm="confirmUnbind"
  />

  <ConfirmDialog
    :open="Boolean(pendingProviderToggle)"
    :title="pendingProviderToggle?.enabled ? '停用服务商' : '启用服务商'"
    :description="
      pendingProviderToggle?.enabled
        ? `${pendingProviderToggle?.name || ''} 停用后不能承接新任务。存在平台绑定时，服务端会阻止不安全的停用操作。`
        : `${pendingProviderToggle?.name || ''} 启用后仍需通过连接和模型测试才能用于平台。`
    "
    :confirm-label="pendingProviderToggle?.enabled ? '确认停用' : '确认启用'"
    :danger="Boolean(pendingProviderToggle?.enabled)"
    :loading="store.isMutating"
    @close="pendingProviderToggle = null"
    @confirm="toggleProvider"
  />

  <AiConfigDeleteDialogs
    :provider="pendingProviderDelete"
    :model="pendingModelDelete"
    :busy="store.isMutating"
    @close-provider="pendingProviderDelete = null"
    @confirm-provider="confirmProviderDelete"
    @close-model="pendingModelDelete = null"
    @confirm-model="confirmModelDelete"
  />
</template>

<style scoped src="./AiProvidersView.css"></style>
