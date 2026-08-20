<script setup lang="ts">
import BaseButton from '@/components/base/BaseButton.vue'
import BaseModal from '@/components/base/BaseModal.vue'
import { ShieldCheck } from '@lucide/vue'
import type { AIProcessingNotice } from '@/services/api'

defineProps<{
  open: boolean
  notice: AIProcessingNotice | null
  loading?: boolean
}>()

const emit = defineEmits<{ acknowledge: [] }>()
</script>

<template>
  <BaseModal
    :open="open"
    title="使用前请确认"
    description="先了解素材如何使用，以及映研如何保护您的隐私"
    size="wide"
    @close="undefined"
  >
    <div v-if="notice" class="notice-content">
      <p class="notice-lead">{{ notice.providerDisclosure }}</p>
      <div class="privacy-note">
        <ShieldCheck :size="20" :stroke-width="1.8" aria-hidden="true" />
        <div>
          <strong>您的内容会被妥善保护</strong>
          <p>{{ notice.securitySummary }}</p>
        </div>
      </div>
      <div class="notice-block">
        <strong>处理用途</strong>
        <p>{{ notice.purpose }}</p>
      </div>
      <div class="notice-block">
        <strong>处理范围</strong>
        <ul>
          <li v-for="item in notice.processingScope" :key="item">{{ item }}</li>
        </ul>
      </div>
      <div class="notice-block">
        <strong>留存与停止使用</strong>
        <p>关联素材默认在任务终态后保留 {{ notice.retentionDays }} 天。</p>
        <p>{{ notice.stopUseDescription }}</p>
      </div>
      <BaseButton class="notice-action" :loading="loading" @click="emit('acknowledge')">
        我已了解，继续创作
      </BaseButton>
    </div>
  </BaseModal>
</template>

<style scoped>
.notice-content { display: grid; gap: 16px; color: var(--ink-muted); line-height: 1.7; }
.notice-lead { margin: 0 2px; color: var(--ink); font-size: 15px; }
.privacy-note { display: flex; align-items: flex-start; gap: 12px; padding: 14px 16px; border: 1px solid color-mix(in srgb, var(--primary) 24%, var(--border)); border-radius: var(--radius-md); background: var(--primary-soft); color: var(--ink-muted); }
.privacy-note > svg { flex: 0 0 auto; margin-top: 2px; color: var(--primary); }
.privacy-note strong { display: block; margin-bottom: 3px; color: var(--ink); font-size: 13px; }
.privacy-note p { margin: 0; font-size: 13px; }
.notice-block { padding: 15px 16px; border: 1px solid var(--border); border-radius: var(--radius-md); background: var(--surface-soft); }
.notice-block strong { display: block; margin-bottom: 4px; color: var(--ink); font-size: 13px; }
.notice-block p { margin: 0; font-size: 13px; }
.notice-block ul { display: grid; gap: 6px; margin: 5px 0 0; padding-left: 18px; font-size: 13px; }
.notice-action { width: 100%; margin-top: 4px; }

@media (max-width: 600px) {
  .notice-content { gap: 12px; }
  .notice-lead { font-size: 14px; }
  .notice-block,
  .privacy-note { padding: 13px 14px; }
}
</style>
