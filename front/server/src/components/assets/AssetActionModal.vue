<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Archive, ArchiveRestore, Trash2 } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import FormField from '@/components/base/FormField.vue'
import type { ManagedAsset } from '@/types/domain'

export type AssetAction = 'retain' | 'release' | 'cleanup'

const props = defineProps<{
  action: AssetAction | null
  asset: ManagedAsset | null
  loading: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [reason: string]
}>()

const reason = ref('')
const error = ref('')

const meta = computed(() => {
  if (props.action === 'retain') {
    return {
      title: '设为长期保留',
      description: '该图片将跳过自动清理，直到管理员主动解除保留。',
      submit: '确认长期保留',
    }
  }
  if (props.action === 'release') {
    return {
      title: '解除长期保留',
      description: '解除后重新按照关联任务或工单的留存时间执行自动清理。',
      submit: '确认解除保留',
    }
  }
  return {
    title: '提前清理图片',
    description: '对象文件将不可恢复。系统会保留必要元数据和完整审计记录。',
    submit: '确认提前清理',
  }
})

watch(
  () => props.action,
  () => {
    reason.value = ''
    error.value = ''
  },
)

function submit(): void {
  if (!reason.value.trim()) {
    error.value = '请填写操作原因'
    return
  }
  emit('submit', reason.value.trim())
}
</script>

<template>
  <BaseModal
    :open="Boolean(action)"
    :title="meta.title"
    :description="meta.description"
    width="small"
    :close-on-backdrop="!loading"
    @close="$emit('close')"
  >
    <div class="asset-summary">
      <img v-if="asset?.previewUrl" :src="asset.previewUrl" :alt="asset.name" />
      <div>
        <strong>{{ asset?.name }}</strong>
        <span class="mono">{{ asset?.id }}</span>
      </div>
    </div>
    <FormField label="操作原因" for-id="asset-reason" required>
      <textarea
        id="asset-reason"
        v-model="reason"
        class="form-control"
        rows="4"
        maxlength="500"
        placeholder="说明保留或清理依据，记录将进入审计日志"
      />
    </FormField>
    <p v-if="error" class="validation-error" role="alert">{{ error }}</p>

    <template #footer>
      <BaseButton variant="secondary" :disabled="loading" @click="$emit('close')">
        取消
      </BaseButton>
      <BaseButton
        :variant="action === 'cleanup' ? 'danger' : 'primary'"
        :loading="loading"
        @click="submit"
      >
        <template #icon>
          <Trash2 v-if="action === 'cleanup'" :size="16" />
          <ArchiveRestore v-else-if="action === 'release'" :size="16" />
          <Archive v-else :size="16" />
        </template>
        {{ meta.submit }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<style scoped>
.asset-summary {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  padding: 10px;
  margin-bottom: 18px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-soft);
}

.asset-summary img {
  width: 64px;
  height: 50px;
  border-radius: var(--radius-sm);
  object-fit: cover;
}

.asset-summary div {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.asset-summary strong,
.asset-summary span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-summary strong {
  font-size: 12px;
}

.asset-summary span {
  color: var(--ink-muted);
  font-size: 10px;
}

textarea.form-control {
  min-height: 104px;
  resize: vertical;
}

.validation-error {
  margin-top: 12px;
  color: var(--danger);
  font-size: 12px;
}
</style>

