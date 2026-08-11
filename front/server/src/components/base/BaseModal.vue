<script setup lang="ts">
import { computed } from 'vue'
import { X } from '@lucide/vue'
import IconButton from './IconButton.vue'
import { useOverlayLifecycle } from '@/composables/useOverlayLifecycle'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    description?: string
    width?: 'small' | 'medium' | 'large'
    closeOnBackdrop?: boolean
  }>(),
  {
    description: undefined,
    width: 'medium',
    closeOnBackdrop: true,
  },
)

const emit = defineEmits<{ close: [] }>()
const isOpen = computed(() => props.open)
const titleId = `modal-title-${Math.random().toString(36).slice(2)}`
const descriptionId = `modal-description-${Math.random().toString(36).slice(2)}`

function close(): void {
  emit('close')
}

function onBackdrop(): void {
  if (props.closeOnBackdrop) close()
}

useOverlayLifecycle(isOpen, close)
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="open"
        class="modal"
        role="presentation"
        @mousedown.self="onBackdrop"
      >
        <section
          class="modal__panel"
          :class="`modal__panel--${width}`"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="titleId"
          :aria-describedby="description ? descriptionId : undefined"
        >
          <header class="modal__header">
            <div>
              <h2 :id="titleId" class="modal__title">{{ title }}</h2>
              <p v-if="description" :id="descriptionId" class="modal__description">
                {{ description }}
              </p>
            </div>
            <IconButton label="关闭弹窗" @click="close">
              <X :size="20" />
            </IconButton>
          </header>
          <div class="modal__body"><slot></slot></div>
          <footer v-if="$slots.footer" class="modal__footer">
            <slot name="footer"></slot>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal {
  position: fixed;
  z-index: 80;
  inset: 0;
  display: grid;
  overflow: auto;
  padding: 24px;
  background: var(--scrim);
  place-items: center;
}

.modal__panel {
  width: min(100%, 600px);
  max-height: calc(100dvh - 48px);
  overflow: hidden;
  border: 1px solid rgb(255 255 255 / 35%);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-md);
}

.modal__panel--small {
  max-width: 440px;
}

.modal__panel--large {
  max-width: 860px;
}

.modal__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  padding: 20px 22px 16px;
  border-bottom: 1px solid var(--border);
}

.modal__title {
  font-size: 18px;
  line-height: 1.35;
}

.modal__description {
  margin-top: 5px;
  color: var(--ink-muted);
  font-size: 13px;
}

.modal__body {
  max-height: calc(100dvh - 240px);
  overflow: auto;
  padding: 22px;
}

.modal__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  padding: 14px 22px 18px;
  border-top: 1px solid var(--border);
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity var(--motion-normal) var(--ease-out);
}

.modal-enter-active .modal__panel,
.modal-leave-active .modal__panel {
  transition: transform var(--motion-normal) var(--ease-out);
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal__panel,
.modal-leave-to .modal__panel {
  transform: translateY(10px) scale(0.99);
}

@media (max-width: 640px) {
  .modal {
    align-items: end;
    padding: 0;
  }

  .modal__panel {
    width: 100%;
    max-height: calc(100dvh - env(safe-area-inset-top) - 12px);
    border-radius: 12px 12px 0 0;
  }

  .modal__header {
    padding: 16px 14px 13px;
  }

  .modal__body {
    max-height: calc(100dvh - 220px);
    padding: 16px 14px;
    overscroll-behavior: contain;
  }

  .modal__footer {
    flex-wrap: wrap;
    gap: 8px;
    padding: 10px 14px calc(10px + env(safe-area-inset-bottom));
    box-shadow: 0 -8px 24px rgb(22 30 28 / 5%);
  }

  .modal__footer :deep(.base-button) {
    flex: 1 1 120px;
  }

  .modal-enter-from .modal__panel,
  .modal-leave-to .modal__panel {
    transform: translateY(100%);
  }
}
</style>
