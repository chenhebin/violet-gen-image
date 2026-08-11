<script setup lang="ts">
import { Image, MessageSquareText, Unplug } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import type { AIModel, ModelType } from '@/types'

const props = defineProps<{
  chatModel?: AIModel
  imageModel?: AIModel
  loading?: boolean
}>()

const emit = defineEmits<{
  unbind: [type: ModelType]
  locate: [model: AIModel]
}>()
</script>

<template>
  <section class="binding-spine" aria-label="平台当前模型">
    <div class="spine-heading">
      <span>平台路由</span>
      <strong>Client 当前使用</strong>
    </div>

    <article :class="{ missing: !props.chatModel }">
      <div class="type-icon chat" aria-hidden="true">
        <MessageSquareText :size="18" />
      </div>
      <div class="binding-copy">
        <span>提示词优化</span>
        <strong>{{ props.chatModel?.displayName || '未配置对话模型' }}</strong>
        <small class="data-mono">
          {{ props.chatModel?.modelId || 'Client 暂无法优化提示词' }}
        </small>
      </div>
      <div v-if="props.chatModel" class="binding-actions">
        <button type="button" @click="emit('locate', props.chatModel)">
          查看配置
        </button>
        <BaseButton
          variant="ghost"
          size="sm"
          :disabled="props.loading"
          @click="emit('unbind', 'chat')"
        >
          <Unplug :size="14" aria-hidden="true" />
          解除
        </BaseButton>
      </div>
    </article>

    <span class="spine-connector" aria-hidden="true" />

    <article :class="{ missing: !props.imageModel }">
      <div class="type-icon image" aria-hidden="true">
        <Image :size="18" />
      </div>
      <div class="binding-copy">
        <span>图片生成与编辑</span>
        <strong>{{ props.imageModel?.displayName || '未配置生图模型' }}</strong>
        <small class="data-mono">
          {{ props.imageModel?.modelId || 'Client 暂无法提交生成任务' }}
        </small>
      </div>
      <div v-if="props.imageModel" class="binding-actions">
        <button type="button" @click="emit('locate', props.imageModel)">
          查看配置
        </button>
        <BaseButton
          variant="ghost"
          size="sm"
          :disabled="props.loading"
          @click="emit('unbind', 'image')"
        >
          <Unplug :size="14" aria-hidden="true" />
          解除
        </BaseButton>
      </div>
    </article>

    <div class="route-note">
      <span>后端路由</span>
      <strong>/api</strong>
      <small>服务商切换不会改变 Client 协议</small>
    </div>
  </section>
</template>

<style scoped>
.binding-spine {
  display: grid;
  grid-template-columns: 135px minmax(240px, 1fr) 38px minmax(240px, 1fr) 145px;
  gap: 12px;
  align-items: center;
  min-height: 104px;
  padding: 15px 17px;
  overflow: hidden;
  color: #fff;
  background: #202625;
  border-radius: 8px;
}

.spine-heading {
  display: grid;
  gap: 5px;
}

.spine-heading span,
.binding-copy span,
.route-note span,
.route-note small {
  color: #9eaaa7;
  font-size: 9px;
}

.spine-heading strong {
  font-family: var(--font-display, serif);
  font-size: 15px;
  font-weight: 600;
}

article {
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  min-width: 0;
  min-height: 70px;
  padding: 10px;
  background: #2a302f;
  border: 1px solid #39403e;
  border-radius: 7px;
}

article.missing {
  border-color: #765b2a;
}

.type-icon {
  display: grid;
  width: 38px;
  height: 38px;
  color: #dbe7e4;
  background: #35403e;
  border-radius: 6px;
  place-items: center;
}

article.missing .type-icon {
  color: #e4c98d;
  background: #4b402d;
}

.binding-copy {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.binding-copy strong,
.binding-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.binding-copy strong {
  font-size: 12px;
}

.binding-copy small {
  color: #aeb8b6;
  font-size: 9px;
}

.binding-actions {
  display: flex;
  align-items: center;
}

.binding-actions > button:first-child {
  min-height: 32px;
  padding: 0 7px;
  color: #aeb8b6;
  font-size: 9px;
  background: transparent;
  border: 0;
  cursor: pointer;
}

.spine-connector {
  position: relative;
  height: 1px;
  background: #52605d;
}

.spine-connector::before,
.spine-connector::after {
  position: absolute;
  top: -2px;
  width: 5px;
  height: 5px;
  background: #71827e;
  border-radius: 50%;
  content: '';
}

.spine-connector::before {
  left: 0;
}

.spine-connector::after {
  right: 0;
}

.route-note {
  display: grid;
  gap: 3px;
  padding-left: 3px;
  border-left: 1px solid #414947;
}

.route-note strong {
  color: #a8d1c9;
  font-family: var(--font-mono, monospace);
  font-size: 15px;
}

@media (max-width: 1220px) {
  .binding-spine {
    grid-template-columns: 110px 1fr 1fr;
  }

  .spine-connector,
  .route-note {
    display: none;
  }
}

@media (max-width: 760px) {
  .binding-spine {
    grid-template-columns: 1fr;
  }

  article {
    grid-template-columns: 38px minmax(0, 1fr);
  }

  .binding-actions {
    grid-column: 1 / -1;
    justify-content: flex-end;
  }
}
</style>
