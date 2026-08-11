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
    size?: 'medium' | 'large'
  }>(),
  {
    description: undefined,
    size: 'large',
  },
)

const emit = defineEmits<{ close: [] }>()
const isOpen = computed(() => props.open)
const titleId = `drawer-title-${Math.random().toString(36).slice(2)}`

function close(): void {
  emit('close')
}

useOverlayLifecycle(isOpen, close)
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="open" class="drawer" role="presentation" @mousedown.self="close">
        <aside
          class="drawer__panel"
          :class="`drawer__panel--${size}`"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="titleId"
        >
          <header class="drawer__header">
            <div>
              <h2 :id="titleId" class="drawer__title">{{ title }}</h2>
              <p v-if="description" class="drawer__description">
                {{ description }}
              </p>
            </div>
            <IconButton label="关闭详情" @click="close">
              <X :size="20" />
            </IconButton>
          </header>
          <div class="drawer__body"><slot></slot></div>
          <footer v-if="$slots.footer" class="drawer__footer">
            <slot name="footer"></slot>
          </footer>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.drawer {
  position: fixed;
  z-index: 75;
  inset: 0;
  display: flex;
  justify-content: flex-end;
  background: var(--scrim);
}

.drawer__panel {
  display: grid;
  width: clamp(620px, 52vw, 920px);
  max-width: 100%;
  height: 100dvh;
  grid-template-rows: auto minmax(0, 1fr) auto;
  border-left: 1px solid rgb(255 255 255 / 25%);
  background: var(--surface);
  box-shadow: var(--shadow-md);
}

.drawer__panel--large {
  width: clamp(760px, 66.666vw, 1280px);
}

.drawer__header {
  display: flex;
  min-height: 76px;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding: 17px 22px;
  border-bottom: 1px solid var(--border);
}

.drawer__title {
  font-size: 18px;
}

.drawer__description {
  margin-top: 4px;
  color: var(--ink-muted);
  font-size: 13px;
}

.drawer__body {
  min-height: 0;
  overflow: auto;
  padding: 22px;
}

.drawer__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  padding: 14px 22px;
  border-top: 1px solid var(--border);
  background: var(--surface);
}

.drawer-enter-active,
.drawer-leave-active {
  transition: opacity var(--motion-normal) var(--ease-out);
}

.drawer-enter-active .drawer__panel,
.drawer-leave-active .drawer__panel {
  transition: transform var(--motion-normal) var(--ease-out);
}

.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}

.drawer-enter-from .drawer__panel,
.drawer-leave-to .drawer__panel {
  transform: translateX(100%);
}

@media (max-width: 760px) {
  .drawer__panel,
  .drawer__panel--large {
    width: 100%;
  }

  .drawer__body {
    padding: 18px 14px;
    overscroll-behavior: contain;
  }

  .drawer__header {
    min-height: 68px;
    padding: calc(12px + env(safe-area-inset-top)) 14px 12px;
    background: rgb(255 255 255 / 94%);
    backdrop-filter: blur(14px);
  }

  .drawer__title {
    font-size: 17px;
  }

  .drawer__description {
    max-width: 270px;
    font-size: 11px;
  }

  .drawer__footer {
    flex-wrap: wrap;
    gap: 8px;
    padding: 10px 14px calc(10px + env(safe-area-inset-bottom));
    box-shadow: 0 -8px 24px rgb(22 30 28 / 5%);
  }

  .drawer__footer :deep(.base-button) {
    flex: 1 1 120px;
  }

  .drawer-enter-from .drawer__panel,
  .drawer-leave-to .drawer__panel {
    transform: translateY(24px);
  }
}
</style>
