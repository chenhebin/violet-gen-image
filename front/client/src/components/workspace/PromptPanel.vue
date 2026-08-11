<script setup lang="ts">
import {
  Check,
  CheckCircle2,
  Sparkles,
} from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
import PromptSectionField from '@/components/workspace/PromptSectionField.vue'
import {
  PROMPT_CONFIG,
  PROMPT_SECTION_OPTIONS,
} from '@/config'
import { useWorkspaceStore } from '@/stores/workspace'

const workspace = useWorkspaceStore()

const promptPlaceholder =
  '例如：傍晚海岸边的女性时尚人像，保留真实五官和皮肤质感，参考图片的清透色调，整理碎发和背景杂物。'
</script>

<template>
  <aside class="prompt-panel">
    <header class="panel-heading">
      <div>
        <span>创作方案</span>
        <h2>需求与提示词</h2>
      </div>
      <div v-if="workspace.isConfirmed" class="confirmed-badge">
        <CheckCircle2 :size="15" />
        已确认
      </div>
    </header>

    <div class="panel-body">
      <section class="source-prompt">
        <div class="field-heading">
          <label for="source-prompt">你想得到什么画面</label>
          <span>
            {{ workspace.draft.sourcePrompt.length }}/{{ PROMPT_CONFIG.maxLength }}
          </span>
        </div>
        <textarea
          id="source-prompt"
          :value="workspace.draft.sourcePrompt"
          :placeholder="promptPlaceholder"
          :maxlength="PROMPT_CONFIG.maxLength"
          @input="
            workspace.setSourcePrompt(($event.target as HTMLTextAreaElement).value)
          "
        />
        <BaseButton
          variant="secondary"
          :loading="workspace.optimizing"
          :disabled="!workspace.canOptimize"
          @click="workspace.optimizePrompt"
        >
          <template #icon><Sparkles :size="17" /></template>
          {{
            workspace.draft.promptVersion ? '重新优化提示词' : '优化提示词'
          }}
        </BaseButton>
        <p class="free-note">提示词优化免费，不消耗次数</p>
      </section>

      <Transition name="prompt">
        <section v-if="workspace.draft.promptVersion" class="optimized">
          <div class="optimized-heading">
            <div>
              <p>优化方案</p>
              <h3>逐项检查后确认</h3>
            </div>
            <span v-if="!workspace.isConfirmed">未确认</span>
          </div>

          <div class="section-fields">
            <PromptSectionField
              v-for="section in PROMPT_SECTION_OPTIONS"
              :key="section.key"
              :field-key="section.key"
              :label="section.label"
              :value="workspace.draft.promptVersion.sections[section.key]"
              :unchanged="workspace.isPromptSectionUnchanged(section.key)"
              @update:value="workspace.updatePromptSection(section.key, $event)"
              @update:unchanged="
                workspace.setPromptSectionUnchanged(section.key, $event)
              "
            />
          </div>

          <BaseButton
            :variant="workspace.isConfirmed ? 'secondary' : 'primary'"
            :loading="workspace.confirming"
            :disabled="workspace.isConfirmed"
            @click="workspace.confirmPrompt"
          >
            <template #icon><Check :size="17" /></template>
            {{ workspace.isConfirmed ? '方案已确认' : '确认当前方案' }}
          </BaseButton>
          <p v-if="workspace.isConfirmed" class="confirmed-note">
            修改素材、提示词或关键设置后，需要重新确认。
          </p>
        </section>
      </Transition>
    </div>

    <slot name="quote" />
  </aside>
</template>

<style scoped>
.prompt-panel {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.panel-heading {
  display: flex;
  min-height: 70px;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border);
}

.panel-heading span,
.optimized-heading p {
  color: var(--primary);
  font-size: 10px;
  font-weight: 800;
}

.panel-heading h2 {
  margin-top: 2px;
  font-size: 16px;
}

.confirmed-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 7px;
  border-radius: 5px;
  background: var(--primary-soft);
  color: var(--primary);
  font-size: 10px;
  font-weight: 720;
}

.panel-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
  padding: 18px;
}

.source-prompt {
  display: grid;
  gap: 10px;
}

.field-heading {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.field-heading label {
  font-size: 13px;
  font-weight: 700;
}

.field-heading span {
  color: var(--ink-faint);
  font-size: 10px;
}

textarea {
  width: 100%;
  resize: vertical;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  outline: 0;
  background: var(--surface);
}

.source-prompt > textarea {
  min-height: 132px;
  padding: 12px;
  font-size: 13px;
  line-height: 1.65;
}

textarea:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px rgb(20 108 99 / 10%);
}

.free-note,
.confirmed-note {
  color: var(--ink-faint);
  font-size: 10px;
  text-align: center;
}

.optimized {
  padding-top: 20px;
  margin-top: 20px;
  border-top: 1px solid var(--border);
}

.optimized-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 13px;
}

.optimized-heading h3 {
  margin-top: 2px;
  font-size: 14px;
}

.optimized-heading > span {
  color: var(--coral);
  font-size: 10px;
  font-weight: 700;
}

.section-fields {
  display: grid;
  gap: 9px;
}

.optimized > .base-button {
  width: 100%;
  margin-top: 14px;
}

.confirmed-note {
  margin-top: 7px;
}

.prompt-enter-active,
.prompt-leave-active {
  transition: all var(--motion-normal) var(--ease-out);
}

.prompt-enter-from,
.prompt-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

@media (max-width: 900px) {
  .prompt-panel {
    min-height: auto;
    overflow: visible;
  }

  .panel-body {
    min-height: auto;
    overflow: visible;
    scrollbar-gutter: auto;
  }
}
</style>
