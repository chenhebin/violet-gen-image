<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ImageUp, Play, Send, ShieldAlert } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import FormField from '@/components/base/FormField.vue'
import { RETOUCH_UPLOAD_CONFIG } from '@/config'
import type { ManageRetouchTicket } from '@/types/domain'

export type RetouchAction = 'quote' | 'start' | 'deliver' | 'reject' | 'fail'

export interface RetouchActionPayload {
  credits?: number
  note?: string
  files?: File[]
  reason?: string
}

const props = defineProps<{
  action: RetouchAction | null
  ticket: ManageRetouchTicket | null
  loading: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [payload: RetouchActionPayload]
}>()

const credits = ref(1)
const note = ref('')
const reason = ref('')
const files = ref<File[]>([])
const validationError = ref('')

const actionMeta = computed(() => {
  switch (props.action) {
    case 'quote':
      return {
        title: props.ticket?.quote ? '调整人工报价' : '给出人工报价',
        description: '报价以平台次数计价。用户接受前可以重新报价。',
        submit: '发送报价',
      }
    case 'start':
      return {
        title: '确认开始处理',
        description: '开工后将结算用户已预占的次数，状态进入处理中。',
        submit: '确认开工',
      }
    case 'deliver':
      return {
        title: props.ticket?.revision ? '交付返修成片' : '交付人工成片',
        description: '上传 1–4 张 JPG、PNG 或 WebP，交付后等待用户确认。',
        submit: '确认交付',
      }
    case 'reject':
      return {
        title: '拒绝人工修图需求',
        description: '仅限尚未被用户接受的需求。拒单不会扣减或退回次数。',
        submit: '确认拒单',
      }
    case 'fail':
      return {
        title: '标记履约失败',
        description: '工单会立即结束，并将已预占或已结算次数全额退回用户。',
        submit: '失败并退款',
      }
    default:
      return { title: '', description: '', submit: '' }
  }
})

watch(
  () => props.action,
  () => {
    credits.value = props.ticket?.quote?.credits ?? 1
    note.value = ''
    reason.value = ''
    files.value = []
    validationError.value = ''
  },
)

function onFiles(event: Event): void {
  const input = event.target as HTMLInputElement
  files.value = Array.from(input.files ?? [])
  validationError.value = ''
}

function submit(): void {
  validationError.value = ''
  if (props.action === 'quote') {
    if (!Number.isInteger(credits.value) || credits.value < 1 || credits.value > 999) {
      validationError.value = '报价必须是 1–999 的整数'
      return
    }
    emit('submit', { credits: credits.value, note: note.value.trim() || undefined })
    return
  }
  if (props.action === 'deliver') {
    if (!files.value.length || files.value.length > RETOUCH_UPLOAD_CONFIG.maxFiles) {
      validationError.value = `请选择 1–${RETOUCH_UPLOAD_CONFIG.maxFiles} 张成片`
      return
    }
    const invalid = files.value.find(
      (file) =>
        !RETOUCH_UPLOAD_CONFIG.allowedTypes.includes(
          file.type as (typeof RETOUCH_UPLOAD_CONFIG.allowedTypes)[number],
        ) || file.size > RETOUCH_UPLOAD_CONFIG.maxFileSize,
    )
    if (invalid) {
      validationError.value = `${invalid.name} 的格式或大小不符合要求`
      return
    }
    emit('submit', { files: files.value, note: note.value.trim() || undefined })
    return
  }
  if (props.action === 'reject' || props.action === 'fail') {
    if (!reason.value.trim()) {
      validationError.value = '请填写操作原因'
      return
    }
    emit('submit', { reason: reason.value.trim() })
    return
  }
  emit('submit', {})
}
</script>

<template>
  <BaseModal
    :open="Boolean(action)"
    :title="actionMeta.title"
    :description="actionMeta.description"
    width="small"
    :close-on-backdrop="!loading"
    @close="$emit('close')"
  >
    <div v-if="action === 'quote'" class="action-form">
      <FormField label="报价次数" for-id="retouch-credits" required>
        <input
          id="retouch-credits"
          v-model.number="credits"
          class="form-control"
          type="number"
          min="1"
          max="999"
        />
      </FormField>
      <FormField label="报价说明" for-id="retouch-quote-note" hint="用户会看到这段说明">
        <textarea
          id="retouch-quote-note"
          v-model="note"
          class="form-control"
          rows="4"
          maxlength="500"
          placeholder="例如：包含皮肤、碎发和背景细节处理"
        />
      </FormField>
    </div>

    <div v-else-if="action === 'deliver'" class="action-form">
      <FormField label="人工成片" for-id="retouch-files" required>
        <label class="file-picker" for="retouch-files">
          <ImageUp :size="22" />
          <span>{{ files.length ? `已选择 ${files.length} 张` : '选择成片文件' }}</span>
          <small>每张不超过 30MB</small>
        </label>
        <input
          id="retouch-files"
          class="sr-only"
          type="file"
          accept="image/jpeg,image/png,image/webp"
          multiple
          @change="onFiles"
        />
      </FormField>
      <FormField label="交付说明" for-id="retouch-delivery-note">
        <textarea
          id="retouch-delivery-note"
          v-model="note"
          class="form-control"
          rows="4"
          maxlength="500"
          placeholder="说明本次处理重点"
        />
      </FormField>
    </div>

    <FormField
      v-else-if="action === 'reject' || action === 'fail'"
      label="操作原因"
      for-id="retouch-reason"
      required
    >
      <textarea
        id="retouch-reason"
        v-model="reason"
        class="form-control"
        rows="5"
        maxlength="500"
        :placeholder="
          action === 'reject'
            ? '说明当前需求无法承接的原因'
            : '说明无法继续履约的原因'
        "
      />
    </FormField>

    <div v-else class="confirm-copy">
      <Play :size="22" />
      <p>
        用户已接受 {{ ticket?.quote?.credits ?? 0 }} 次报价。开工后不得再修改报价。
      </p>
    </div>

    <p v-if="validationError" class="validation-error" role="alert">
      {{ validationError }}
    </p>

    <template #footer>
      <BaseButton variant="secondary" :disabled="loading" @click="$emit('close')">
        取消
      </BaseButton>
      <BaseButton
        :variant="action === 'reject' || action === 'fail' ? 'danger' : 'primary'"
        :loading="loading"
        @click="submit"
      >
        <template #icon>
          <ShieldAlert v-if="action === 'reject' || action === 'fail'" :size="16" />
          <Send v-else :size="16" />
        </template>
        {{ actionMeta.submit }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<style scoped>
.action-form {
  display: grid;
  gap: 18px;
}

textarea.form-control {
  min-height: 104px;
  resize: vertical;
}

.file-picker {
  display: grid;
  min-height: 132px;
  place-items: center;
  align-content: center;
  gap: 5px;
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--surface-soft);
  color: var(--primary);
  text-align: center;
}

.file-picker:hover {
  border-color: var(--primary);
}

.file-picker span {
  color: var(--ink);
  font-size: 13px;
  font-weight: 700;
}

.file-picker small {
  color: var(--ink-muted);
  font-size: 11px;
}

.confirm-copy {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-soft);
  color: var(--ink-muted);
}

.confirm-copy svg {
  flex: 0 0 auto;
  color: var(--primary);
}

.validation-error {
  margin-top: 12px;
  color: var(--danger);
  font-size: 12px;
}
</style>

