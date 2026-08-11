<script setup lang="ts">
import { Bot, Image as ImageIcon } from '@lucide/vue'
import type { DashboardMetric, DashboardData } from '@/types/domain'

defineProps<{
  metrics: DashboardMetric[]
  currentModels: DashboardData['currentModels']
  showModels: boolean
}>()
</script>

<template>
  <section class="status-rail" aria-labelledby="status-rail-title">
    <header class="status-rail__header">
      <div>
        <p>LIVE OPERATIONS</p>
        <h2 id="status-rail-title">运行状态脊线</h2>
      </div>
      <span>数据随管理接口刷新</span>
    </header>

    <div
      class="status-rail__track"
      :style="{
        gridTemplateColumns: `repeat(${Math.max(1, Math.min(metrics.length, 6))}, minmax(0, 1fr))`,
      }"
    >
      <article
        v-for="metric in metrics"
        :key="metric.key"
        class="status-rail__metric"
        :class="`status-rail__metric--${metric.tone}`"
      >
        <span class="status-rail__node" aria-hidden="true"></span>
        <strong>{{ metric.value.toLocaleString('zh-CN') }}</strong>
        <p>{{ metric.label }}</p>
      </article>
    </div>

    <div v-if="showModels" class="model-strip">
      <article>
        <Bot :size="17" aria-hidden="true" />
        <div>
          <span>提示词对话模型</span>
          <strong>{{ currentModels.chat?.displayName ?? '暂未配置' }}</strong>
          <small v-if="currentModels.chat">
            {{ currentModels.chat.providerName }} ·
            <span class="mono">{{ currentModels.chat.modelId }}</span>
          </small>
        </div>
      </article>
      <article>
        <ImageIcon :size="17" aria-hidden="true" />
        <div>
          <span>平台生图模型</span>
          <strong>{{ currentModels.image?.displayName ?? '暂未配置' }}</strong>
          <small v-if="currentModels.image">
            {{ currentModels.image.providerName }} ·
            <span class="mono">{{ currentModels.image.modelId }}</span>
          </small>
        </div>
      </article>
    </div>
  </section>
</template>

<style scoped>
.status-rail {
  padding: 22px 24px 20px;
  border: 1px solid #34413e;
  border-radius: var(--radius-md);
  background: var(--sidebar);
  color: #fff;
  box-shadow: 0 12px 32px rgb(25 34 32 / 12%);
}

.status-rail__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
}

.status-rail__header p {
  color: #77b9aa;
  font-size: 9px;
  font-weight: 750;
  letter-spacing: 0.12em;
}

.status-rail__header h2 {
  margin-top: 3px;
  font-family: var(--font-display);
  font-size: 19px;
}

.status-rail__header > span {
  color: rgb(255 255 255 / 42%);
  font-size: 10px;
}

.status-rail__track {
  position: relative;
  display: grid;
  margin-top: 27px;
  grid-template-columns: repeat(6, minmax(0, 1fr));
}

.status-rail__track::before {
  position: absolute;
  top: 5px;
  right: 4%;
  left: 4%;
  height: 1px;
  background: rgb(255 255 255 / 16%);
  content: '';
}

.status-rail__metric {
  position: relative;
  min-width: 0;
  padding-top: 17px;
}

.status-rail__node {
  position: absolute;
  z-index: 1;
  top: 0;
  left: 0;
  width: 11px;
  height: 11px;
  border: 3px solid var(--sidebar);
  border-radius: 50%;
  background: #89928f;
  box-shadow: 0 0 0 1px #89928f;
}

.status-rail__metric--positive .status-rail__node {
  background: #75b89d;
  box-shadow: 0 0 0 1px #75b89d;
}

.status-rail__metric--warning .status-rail__node {
  background: #d2a84f;
  box-shadow: 0 0 0 1px #d2a84f;
}

.status-rail__metric--danger .status-rail__node {
  background: #d8776b;
  box-shadow: 0 0 0 1px #d8776b;
}

.status-rail__metric strong {
  display: block;
  overflow: hidden;
  font-family: var(--font-mono);
  font-size: 25px;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
  text-overflow: ellipsis;
}

.status-rail__metric p {
  margin-top: 4px;
  color: rgb(255 255 255 / 54%);
  font-size: 11px;
}

.model-strip {
  display: grid;
  margin-top: 22px;
  border-top: 1px solid rgb(255 255 255 / 10%);
  grid-template-columns: 1fr 1fr;
}

.model-strip article {
  display: grid;
  min-width: 0;
  padding: 16px 0 0;
  color: #78b5a8;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 10px;
}

.model-strip article + article {
  padding-left: 22px;
  border-left: 1px solid rgb(255 255 255 / 10%);
}

.model-strip span,
.model-strip strong,
.model-strip small {
  display: block;
}

.model-strip span {
  color: rgb(255 255 255 / 46%);
  font-size: 10px;
}

.model-strip strong {
  margin-top: 2px;
  overflow: hidden;
  color: #fff;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-strip small {
  margin-top: 2px;
  overflow: hidden;
  color: rgb(255 255 255 / 42%);
  font-size: 9px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-strip small span {
  display: inline;
  color: inherit;
}

@media (max-width: 760px) {
  .status-rail {
    padding: 18px;
  }

  .status-rail__track {
    grid-template-columns: repeat(3, 1fr);
    row-gap: 22px;
  }

  .status-rail__track::before {
    display: none;
  }

  .status-rail__metric {
    padding-top: 14px;
  }

  .status-rail__metric strong {
    font-size: 21px;
  }

  .model-strip {
    grid-template-columns: 1fr;
  }

  .model-strip article + article {
    padding-left: 0;
    border-left: 0;
  }
}
</style>
