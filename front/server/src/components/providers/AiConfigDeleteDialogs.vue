<script setup lang="ts">
import ConfirmDialog from '@/components/base/ConfirmDialog.vue'
import type { AIModel, AIProvider } from '@/types'

defineProps<{
  provider: AIProvider | null
  model: AIModel | null
  busy?: boolean
}>()

const emit = defineEmits<{
  closeProvider: []
  confirmProvider: []
  closeModel: []
  confirmModel: []
}>()
</script>

<template>
  <ConfirmDialog
    :open="Boolean(provider)"
    title="删除服务商"
    :description="`${provider?.name || ''} 删除后无法恢复。服务商下仍有模型时会拒绝删除，请先逐一删除模型。历史任务快照不会受影响。`"
    confirm-label="确认删除"
    danger
    :loading="busy"
    @close="emit('closeProvider')"
    @confirm="emit('confirmProvider')"
  />
  <ConfirmDialog
    :open="Boolean(model)"
    title="删除模型"
    :description="`${model?.displayName || ''} 删除后无法恢复。当前平台模型必须先解除绑定，历史任务快照不会受影响。`"
    confirm-label="确认删除"
    danger
    :loading="busy"
    @close="emit('closeModel')"
    @confirm="emit('confirmModel')"
  />
</template>
