<script setup lang="ts">
import { ArrowUpRight, ShieldCheck, WandSparkles } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDrawer from '@/components/base/BaseDrawer.vue'
import RetouchStatusBadge from '@/components/shared/RetouchStatusBadge.vue'
import TaskStatusBadge from '@/components/shared/TaskStatusBadge.vue'
import { PROMPT_SECTION_LABELS, WORKSPACE_MODE_LABELS } from '@/config'
import type { ManagedGenerationTask } from '@/types/domain'
import { formatDateTime, formatFileSize } from '@/utils/format'

defineProps<{ open: boolean; task: ManagedGenerationTask | null; loading: boolean }>()

defineEmits<{ close: []; openRetouch: [ticketId: string] }>()
</script>

<template>
  <BaseDrawer
    :open="open"
    :title="task?.title ?? '生成任务详情'"
    :description="task ? `${task.ownerEmail} · ${task.id}` : '正在读取任务详情'"
    size="large"
    @close="$emit('close')"
  >
    <div v-if="loading && !task" class="drawer-loading">正在读取任务详情…</div>
    <div v-else-if="task" class="task-detail">
      <section class="task-spine">
        <div class="task-spine__status">
          <span>任务状态</span>
          <TaskStatusBadge :status="task.status" />
          <small>{{ task.progress }}% · {{ WORKSPACE_MODE_LABELS[task.mode] }}</small>
        </div>
        <dl>
          <div>
            <dt>请求图片</dt>
            <dd>{{ task.requestedCount }}</dd>
          </div>
          <div>
            <dt>成功图片</dt>
            <dd>{{ task.successfulCount }}</dd>
          </div>
          <div>
            <dt>已消耗</dt>
            <dd>{{ task.spentCredits }}</dd>
          </div>
          <div>
            <dt>已退款</dt>
            <dd>{{ task.refundedCredits }}</dd>
          </div>
        </dl>
      </section>

      <section v-if="task.errorMessage" class="error-banner">
        <strong>任务错误</strong>
        <p>{{ task.errorMessage }}</p>
      </section>

      <section class="detail-section">
        <header>
          <div>
            <span>Generation output</span>
            <h3>生成结果</h3>
          </div>
          <b>{{ task.results.length }} 张</b>
        </header>
        <div v-if="task.results.length" class="media-grid">
          <figure v-for="result in task.results" :key="result.id">
            <img :src="result.previewUrl" :alt="result.name" />
            <figcaption>
              <strong>{{ result.name }}</strong>
              <span>{{ result.width }} × {{ result.height }}</span>
            </figcaption>
          </figure>
        </div>
        <p v-else class="empty-copy">任务尚未生成可查看的图片结果</p>
      </section>

      <section class="detail-section">
        <header>
          <div>
            <span>Source assets</span>
            <h3>输入素材</h3>
          </div>
          <b>{{ task.assets.length }} 个文件</b>
        </header>
        <div v-if="task.assets.length" class="media-grid media-grid--small">
          <figure v-for="asset in task.assets" :key="asset.id">
            <img :src="asset.previewUrl" :alt="asset.name" />
            <figcaption>
              <strong>{{ asset.name }}</strong>
              <span>{{ formatFileSize(asset.size) }}</span>
            </figcaption>
          </figure>
        </div>
        <p v-else class="empty-copy">文生图任务没有上传输入素材</p>
      </section>

      <section class="detail-section requirements">
        <header>
          <div>
            <span>Prompt lineage</span>
            <h3>需求与确认提示词</h3>
          </div>
          <WandSparkles :size="18" />
        </header>
        <div class="source-requirement">
          <strong>用户原始需求</strong>
          <p>{{ task.sourceRequirement }}</p>
        </div>
        <div class="prompt-grid">
          <article
            v-for="(content, key) in task.confirmedPrompt"
            :key="key"
            :class="{ 'prompt-card--empty': !content }"
          >
            <span>{{ PROMPT_SECTION_LABELS[key] }}</span>
            <p>{{ content || '未填写' }}</p>
          </article>
        </div>
      </section>

      <section class="detail-section parameters">
        <header>
          <div>
            <span>Execution snapshot</span>
            <h3>参数与模型快照</h3>
          </div>
          <ShieldCheck :size="18" />
        </header>
        <dl>
          <div>
            <dt>服务商</dt>
            <dd>{{ task.executionSnapshot.providerName }}</dd>
          </div>
          <div>
            <dt>模型</dt>
            <dd>{{ task.executionSnapshot.modelName }}</dd>
          </div>
          <div>
            <dt>配置版本</dt>
            <dd class="mono">v{{ task.executionSnapshot.configVersion }}</dd>
          </div>
          <div>
            <dt>画面比例</dt>
            <dd>{{ task.settings.aspectRatio }}</dd>
          </div>
          <div>
            <dt>输出数量</dt>
            <dd>{{ task.settings.outputCount }}</dd>
          </div>
          <div>
            <dt>参考强度</dt>
            <dd>{{ task.settings.referenceStrength }}%</dd>
          </div>
          <div>
            <dt>创建时间</dt>
            <dd>{{ formatDateTime(task.createdAt) }}</dd>
          </div>
          <div>
            <dt>更新时间</dt>
            <dd>{{ formatDateTime(task.updatedAt) }}</dd>
          </div>
        </dl>
      </section>

      <section v-if="task.retouchTicket" class="retouch-link">
        <div>
          <span>关联人工修图</span>
          <strong class="mono">{{ task.retouchTicket.ticketNo }}</strong>
        </div>
        <RetouchStatusBadge :status="task.retouchTicket.status" />
        <BaseButton
          variant="secondary"
          size="small"
          @click="$emit('openRetouch', task.retouchTicket.id)"
        >
          <template #icon><ArrowUpRight :size="15" /></template>
          查看工单
        </BaseButton>
      </section>
    </div>
  </BaseDrawer>
</template>

<style scoped>
.drawer-loading {
  display: grid;
  min-height: 320px;
  place-items: center;
  color: var(--ink-muted);
}

.task-detail {
  display: grid;
  gap: 18px;
}

.task-spine,
.detail-section,
.retouch-link {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
}

.task-spine {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
}

.task-spine__status {
  display: grid;
  align-content: center;
  justify-items: start;
  gap: 8px;
  padding: 20px;
  border-right: 1px solid var(--border);
  background: var(--surface-soft);
}

.task-spine__status > span,
.detail-section header span,
.retouch-link span {
  color: var(--ink-muted);
  font-size: 10px;
  font-weight: 750;
}

.task-spine__status small {
  color: var(--ink-muted);
  font-family: var(--font-mono);
  font-size: 10px;
}

.task-spine dl {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
}

.task-spine dl > div {
  display: grid;
  align-content: center;
  gap: 4px;
  padding: 18px;
  border-right: 1px solid var(--border);
}

.task-spine dl > div:last-child {
  border-right: 0;
}

dt {
  color: var(--ink-muted);
  font-size: 11px;
}

dd {
  margin: 0;
  font-size: 12px;
  font-weight: 650;
}

.task-spine dd {
  font-family: var(--font-mono);
  font-size: 17px;
}

.error-banner {
  padding: 14px 16px;
  border: 1px solid #edc7c1;
  border-radius: var(--radius-md);
  background: var(--danger-soft);
  color: var(--danger);
}

.error-banner strong {
  font-size: 12px;
}

.error-banner p {
  margin-top: 3px;
  font-size: 11px;
}

.detail-section {
  padding: 18px;
}

.detail-section > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.detail-section h3 {
  margin-top: 2px;
  font-size: 15px;
}

.detail-section header b,
.detail-section header svg {
  color: var(--ink-muted);
  font-size: 11px;
}

.media-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 10px;
}

.media-grid figure {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface-soft);
}

.media-grid img {
  width: 100%;
  aspect-ratio: 4 / 3;
  object-fit: cover;
}

.media-grid figcaption {
  display: grid;
  gap: 2px;
  padding: 9px 10px;
}

.media-grid strong {
  overflow: hidden;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.media-grid span {
  color: var(--ink-muted);
  font-size: 10px;
}

.media-grid--small {
  grid-template-columns: repeat(auto-fill, minmax(145px, 1fr));
}

.source-requirement {
  padding: 14px;
  margin-bottom: 12px;
  border-left: 3px solid var(--primary);
  border-radius: var(--radius-sm);
  background: var(--primary-soft);
}

.source-requirement strong,
.prompt-card span {
  font-size: 11px;
}

.source-requirement p {
  margin-top: 5px;
  font-size: 13px;
  line-height: 1.7;
}

.prompt-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.prompt-grid article {
  min-height: 90px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.prompt-grid article:last-child {
  grid-column: 1 / -1;
}

.prompt-card--empty {
  background: var(--surface-soft);
}

.prompt-grid span {
  color: var(--primary);
  font-weight: 750;
}

.prompt-grid p {
  margin-top: 5px;
  color: var(--ink-muted);
  font-size: 12px;
  line-height: 1.65;
  white-space: pre-wrap;
}

.parameters dl {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.parameters dl > div {
  display: grid;
  grid-template-columns: 110px minmax(0, 1fr);
  gap: 10px;
  padding: 11px 12px;
  border-bottom: 1px solid var(--border);
}

.parameters dl > div:nth-child(odd) {
  border-right: 1px solid var(--border);
}

.parameters dl > div:nth-last-child(-n + 2) {
  border-bottom: 0;
}

.retouch-link {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 14px;
  padding: 15px 16px;
}

.retouch-link > div {
  display: grid;
  gap: 3px;
}

.retouch-link strong {
  font-size: 12px;
}

.empty-copy {
  padding: 24px;
  color: var(--ink-muted);
  font-size: 12px;
  text-align: center;
}

@media (max-width: 780px) {
  .task-spine {
    grid-template-columns: 1fr;
  }

  .task-spine__status {
    border-right: 0;
    border-bottom: 1px solid var(--border);
  }

  .task-spine dl {
    grid-template-columns: repeat(2, 1fr);
  }

  .task-spine dl > div:nth-child(2) {
    border-right: 0;
  }

  .task-spine dl > div:nth-child(-n + 2) {
    border-bottom: 1px solid var(--border);
  }

  .prompt-grid,
  .parameters dl {
    grid-template-columns: 1fr;
  }

  .prompt-grid article:last-child {
    grid-column: auto;
  }

  .parameters dl > div,
  .parameters dl > div:nth-child(odd) {
    border-right: 0;
    border-bottom: 1px solid var(--border);
  }

  .parameters dl > div:last-child {
    border-bottom: 0;
  }

  .retouch-link {
    grid-template-columns: 1fr;
    justify-items: start;
  }
}
</style>
