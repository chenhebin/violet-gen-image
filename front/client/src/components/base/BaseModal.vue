<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { X } from '@lucide/vue'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    description?: string
    size?: 'small' | 'medium' | 'wide'
  }>(),
  {
    description: '',
    size: 'medium',
  },
)

const emit = defineEmits<{
  close: []
}>()

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape' && props.open) emit('close')
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => document.removeEventListener('keydown', onKeydown))
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="modal-layer" role="presentation">
        <button class="modal-scrim" aria-label="关闭窗口" @click="$emit('close')" />
        <section
          class="modal-panel"
          :class="`is-${size}`"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="`${title}-heading`"
        >
          <header class="modal-header">
            <div>
              <h2 :id="`${title}-heading`">{{ title }}</h2>
              <p v-if="description">{{ description }}</p>
            </div>
            <button class="icon-button" aria-label="关闭" @click="$emit('close')">
              <X :size="20" />
            </button>
          </header>
          <div class="modal-body">
            <slot />
          </div>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-layer {
  position: fixed;
  z-index: 80;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 20px;
}

.modal-scrim {
  position: absolute;
  inset: 0;
  width: 100%;
  background: var(--scrim);
  cursor: default;
}

.modal-panel {
  position: relative;
  width: min(100%, 520px);
  max-height: calc(100dvh - 40px);
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-md);
}

.modal-panel.is-small {
  width: min(100%, 420px);
}

.modal-panel.is-wide {
  width: min(100%, 900px);
}

.modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  padding: 24px 24px 18px;
  border-bottom: 1px solid var(--border);
}

.modal-header h2 {
  font-size: 20px;
  font-weight: 720;
}

.modal-header p {
  margin-top: 4px;
  color: var(--ink-muted);
  font-size: 14px;
}

.modal-body {
  padding: 24px;
}

.icon-button {
  display: grid;
  width: 44px;
  height: 44px;
  flex: 0 0 44px;
  place-items: center;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--ink-muted);
}

.icon-button:hover {
  background: var(--surface-soft);
  color: var(--ink);
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity var(--motion-normal) var(--ease-out);
}

.modal-enter-active .modal-panel,
.modal-leave-active .modal-panel {
  transition: transform var(--motion-normal) var(--ease-out);
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal-panel,
.modal-leave-to .modal-panel {
  transform: translateY(10px) scale(0.985);
}

@media (max-width: 600px) {
  .modal-layer {
    align-items: end;
    padding: 0;
  }

  .modal-panel,
  .modal-panel.is-small,
  .modal-panel.is-wide {
    width: 100%;
    max-height: 88dvh;
    border-right: 0;
    border-bottom: 0;
    border-left: 0;
    border-radius: 8px 8px 0 0;
  }
}
</style>
