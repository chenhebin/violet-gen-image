<script setup lang="ts">
import { CheckCircle2, CircleAlert, Info, X } from '@lucide/vue'
import { useToast } from '@/composables/useToast'

const toast = useToast()
</script>

<template>
  <Teleport to="body">
    <div class="toast-viewport" aria-live="polite" aria-atomic="false">
      <TransitionGroup name="toast">
        <article
          v-for="item in toast.items.value"
          :key="item.id"
          class="toast-item"
          :class="`is-${item.tone}`"
        >
          <CheckCircle2 v-if="item.tone === 'success'" :size="20" />
          <CircleAlert v-else-if="item.tone === 'error'" :size="20" />
          <Info v-else :size="20" />
          <div>
            <strong>{{ item.title }}</strong>
            <p v-if="item.message">{{ item.message }}</p>
          </div>
          <button aria-label="关闭提示" @click="toast.remove(item.id)">
            <X :size="17" />
          </button>
        </article>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-viewport {
  position: fixed;
  z-index: 120;
  top: 78px;
  right: 18px;
  display: grid;
  width: min(380px, calc(100vw - 32px));
  gap: 10px;
}

.toast-item {
  display: grid;
  grid-template-columns: 22px 1fr 32px;
  gap: 12px;
  align-items: start;
  padding: 15px 12px 15px 16px;
  border: 1px solid var(--border);
  border-left-width: 3px;
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-md);
}

.toast-item.is-success {
  border-left-color: var(--success);
  color: var(--success);
}

.toast-item.is-error {
  border-left-color: var(--danger);
  color: var(--danger);
}

.toast-item.is-info {
  border-left-color: var(--primary);
  color: var(--primary);
}

.toast-item strong {
  display: block;
  color: var(--ink);
  font-size: 14px;
}

.toast-item p {
  margin-top: 2px;
  color: var(--ink-muted);
  font-size: 13px;
}

.toast-item button {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border-radius: 6px;
  background: transparent;
  color: var(--ink-faint);
}

.toast-enter-active,
.toast-leave-active {
  transition: all var(--motion-normal) var(--ease-out);
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateX(12px);
}

@media (max-width: 760px) {
  .toast-viewport {
    top: auto;
    right: 12px;
    bottom: calc(
      var(--mobile-nav-height) + env(safe-area-inset-bottom) + 12px
    );
    width: calc(100vw - 24px);
  }

  .toast-item {
    grid-template-columns: 22px minmax(0, 1fr) 36px;
    align-items: center;
    padding: 12px 8px 12px 14px;
    border-radius: 8px;
    box-shadow: 0 14px 38px rgb(23 25 29 / 16%);
  }

  .toast-item button {
    width: 36px;
    height: 36px;
  }

  .toast-enter-from,
  .toast-leave-to {
    transform: translateY(12px) scale(0.98);
  }
}
</style>
