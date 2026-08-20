<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { ChevronLeft, ChevronRight, X } from '@lucide/vue'
import IconButton from './IconButton.vue'
import { useOverlayLifecycle } from '@/composables/useOverlayLifecycle'

export interface LightboxImage {
  id: string
  src: string
  alt: string
  label: string
  meta?: string
}

const props = defineProps<{
  open: boolean
  images: LightboxImage[]
  selectedId: string
}>()

const emit = defineEmits<{
  close: []
  select: [id: string]
  imageError: [id: string]
}>()

const closeButton = ref<HTMLElement | null>(null)
const isOpen = computed(() => props.open)
const currentIndex = computed(() => {
  const index = props.images.findIndex((item) => item.id === props.selectedId)
  return index >= 0 ? index : 0
})
const current = computed(() => props.images[currentIndex.value])
const hasMultiple = computed(() => props.images.length > 1)

function close(): void { emit('close') }
function selectOffset(offset: number): void {
  if (!props.images.length) return
  const index = (currentIndex.value + offset + props.images.length) % props.images.length
  emit('select', props.images[index].id)
}

function onKeydown(event: KeyboardEvent): void {
  if (!props.open) return
  if (event.key === 'ArrowLeft' && hasMultiple.value) {
    event.preventDefault()
    selectOffset(-1)
  } else if (event.key === 'ArrowRight' && hasMultiple.value) {
    event.preventDefault()
    selectOffset(1)
  }
}

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    await nextTick()
    closeButton.value?.querySelector('button')?.focus()
  },
)

useOverlayLifecycle(isOpen, close)
</script>

<template>
  <Teleport to="body">
    <Transition name="lightbox">
      <div
        v-if="open && current"
        class="lightbox"
        role="dialog"
        aria-modal="true"
        :aria-label="`图片预览：${current.label}`"
        tabindex="-1"
        @keydown="onKeydown"
        @mousedown.self="close"
      >
        <header>
          <div>
            <strong>{{ current.label }}</strong>
            <span>
              {{ currentIndex + 1 }} / {{ images.length }}
              <template v-if="current.meta"> · {{ current.meta }}</template>
            </span>
          </div>
          <span ref="closeButton">
            <IconButton label="关闭图片预览" @click="close"><X :size="22" /></IconButton>
          </span>
        </header>

        <button
          v-if="hasMultiple"
          class="nav-button is-previous"
          type="button"
          aria-label="查看上一张"
          @click="selectOffset(-1)"
        >
          <ChevronLeft :size="25" />
        </button>

        <div class="image-stage" @mousedown.self="close">
          <img
            :key="`${current.id}-${current.src}`"
            :src="current.src"
            :alt="current.alt"
            @error="emit('imageError', current.id)"
          />
        </div>

        <button
          v-if="hasMultiple"
          class="nav-button is-next"
          type="button"
          aria-label="查看下一张"
          @click="selectOffset(1)"
        >
          <ChevronRight :size="25" />
        </button>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.lightbox {
  position: fixed;
  z-index: 120;
  inset: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  padding: 20px;
  background: rgb(11 15 14 / 95%);
  color: #fff;
  backdrop-filter: blur(10px);
}

header {
  display: flex;
  min-height: 52px;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 0 4px 12px;
}

header div { display: grid; gap: 2px; }
header strong { font-size: 14px; }
header span { color: rgb(255 255 255 / 62%); font-size: 11px; }
header :deep(.icon-button) {
  border-color: rgb(255 255 255 / 16%);
  background: rgb(255 255 255 / 8%);
  color: #fff;
}

.image-stage {
  display: grid;
  min-width: 0;
  min-height: 0;
  place-items: center;
  overflow: hidden;
}

.image-stage img {
  display: block;
  max-width: min(100%, 1440px);
  max-height: calc(100dvh - 96px);
  object-fit: contain;
  border-radius: 4px;
  box-shadow: 0 22px 60px rgb(0 0 0 / 34%);
}

.nav-button {
  position: absolute;
  z-index: 1;
  top: 50%;
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  transform: translateY(-50%);
  border: 1px solid rgb(255 255 255 / 16%);
  border-radius: var(--radius-md);
  background: rgb(255 255 255 / 8%);
  color: #fff;
}
.nav-button:hover,
.nav-button:focus-visible { background: rgb(255 255 255 / 16%); }
.is-previous { left: 22px; }
.is-next { right: 22px; }

.lightbox-enter-active,
.lightbox-leave-active { transition: opacity var(--motion-normal) var(--ease-out); }
.lightbox-enter-active .image-stage,
.lightbox-leave-active .image-stage { transition: transform var(--motion-normal) var(--ease-out); }
.lightbox-enter-from,
.lightbox-leave-to { opacity: 0; }
.lightbox-enter-from .image-stage,
.lightbox-leave-to .image-stage { transform: scale(0.985); }

@media (max-width: 640px) {
  .lightbox { padding: calc(10px + env(safe-area-inset-top)) 10px calc(10px + env(safe-area-inset-bottom)); }
  header { padding: 0 0 8px; }
  .image-stage img { max-height: calc(100dvh - 88px - env(safe-area-inset-top) - env(safe-area-inset-bottom)); }
  .nav-button { top: auto; bottom: calc(18px + env(safe-area-inset-bottom)); transform: none; }
  .is-previous { left: 14px; }
  .is-next { right: 14px; }
}

@media (prefers-reduced-motion: reduce) {
  .lightbox-enter-active,
  .lightbox-leave-active,
  .lightbox-enter-active .image-stage,
  .lightbox-leave-active .image-stage { transition: none; }
}
</style>
