<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Check, RotateCcw, ShieldCheck, Trash2 } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import { RETOUCH_TICKET_CONFIG } from '@/config'
import type { RetouchTicket } from '@/types/domain'

export type RetouchAction = 'accept' | 'cancel' | 'revision' | 'confirm'

const props = defineProps<{
  action: RetouchAction | null
  ticket: RetouchTicket
  busy: boolean
}>()

const emit = defineEmits<{
  close: []
  confirm: [message?: string]
}>()

const revisionMessage = ref('')
const modalCopy = computed(() => {
  if (props.action === 'accept') {
    return {
      title: '接受人工精修报价',
      description: `确认后将预占 ${props.ticket.quote?.credits ?? 0} 次额度，并进入人工处理。`,
      confirmText: '接受并开始处理',
    }
  }
  if (props.action === 'cancel') {
    return {
      title: '取消精修工单',
      description:
        props.ticket.status === 'accepted'
          ? `取消后将退回已预占的 ${props.ticket.reservedCredits} 次额度。`
          : '工单取消后将停止报价或后续处理。',
      confirmText: '确认取消',
    }
  }
  if (props.action === 'revision') {
    return {
      title: '申请一次返修',
      description: '请准确说明需要调整的位置和效果。本工单仅有一次返修机会。',
      confirmText: '提交返修要求',
    }
  }
  return {
    title: '确认精修交付',
    description: '确认后工单将完成结算，交付文件仍可在记录中下载。',
    confirmText: '确认交付完成',
  }
})

const trimmedMessage = computed(() => revisionMessage.value.trim())
const submitDisabled = computed(
  () => props.action === 'revision' && !trimmedMessage.value,
)

watch(
  () => props.action,
  () => {
    revisionMessage.value = ''
  },
)

function submit(): void {
  if (submitDisabled.value) return
  emit(
    'confirm',
    props.action === 'revision' ? trimmedMessage.value : undefined,
  )
}
</script>

<template>
  <BaseModal
    :open="Boolean(action)"
    :title="modalCopy.title"
    :description="modalCopy.description"
    size="small"
    @close="!busy && $emit('close')"
  >
    <div v-if="action === 'accept'" class="quote-summary">
      <span><ShieldCheck :size="19" /></span>
      <div>
        <small>本次精修报价</small>
        <strong>{{ ticket.quote?.credits ?? 0 }} 次</strong>
      </div>
    </div>

    <label v-else-if="action === 'revision'" class="revision-field">
      <span>返修要求</span>
      <textarea
        v-model="revisionMessage"
        :maxlength="RETOUCH_TICKET_CONFIG.revisionMaxLength"
        rows="6"
        placeholder="例如：请减弱面部磨皮，并保留左侧碎发的自然层次。"
        autofocus
      />
      <small>
        {{ revisionMessage.length }} /
        {{ RETOUCH_TICKET_CONFIG.revisionMaxLength }}
      </small>
    </label>

    <div class="modal-actions">
      <BaseButton variant="secondary" :disabled="busy" @click="$emit('close')">
        返回
      </BaseButton>
      <BaseButton
        :variant="action === 'cancel' ? 'danger' : 'primary'"
        :loading="busy"
        :disabled="submitDisabled"
        @click="submit"
      >
        <template #icon>
          <Trash2 v-if="action === 'cancel'" :size="17" />
          <RotateCcw v-else-if="action === 'revision'" :size="17" />
          <Check v-else :size="17" />
        </template>
        {{ modalCopy.confirmText }}
      </BaseButton>
    </div>
  </BaseModal>
</template>

<style scoped>
.quote-summary {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-soft);
}

.quote-summary > span {
  display: grid;
  width: 40px;
  height: 40px;
  flex: 0 0 40px;
  place-items: center;
  border-radius: 50%;
  background: var(--primary-soft);
  color: var(--primary);
}

.quote-summary small,
.quote-summary strong {
  display: block;
}

.quote-summary small {
  color: var(--ink-muted);
  font-size: 11px;
}

.quote-summary strong {
  margin-top: 2px;
  font-size: 20px;
}

.revision-field {
  display: grid;
  gap: 8px;
}

.revision-field > span {
  font-size: 12px;
  font-weight: 700;
}

.revision-field textarea {
  width: 100%;
  resize: vertical;
  min-height: 140px;
  padding: 12px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--surface);
  font-size: 13px;
  line-height: 1.65;
}

.revision-field textarea::placeholder {
  color: var(--ink-faint);
}

.revision-field > small {
  justify-self: end;
  color: var(--ink-faint);
  font-size: 10px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 22px;
}

@media (max-width: 480px) {
  .modal-actions {
    align-items: stretch;
    flex-direction: column-reverse;
  }
}
</style>
