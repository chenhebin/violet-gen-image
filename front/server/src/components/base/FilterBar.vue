<script setup lang="ts">
import { ref } from 'vue'
import { ChevronDown, SlidersHorizontal } from '@lucide/vue'

const expanded = ref(false)
</script>

<template>
  <section class="filter-bar" aria-label="筛选条件">
    <button
      class="filter-bar__mobile-toggle"
      type="button"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      <span>
        <SlidersHorizontal :size="17" />
        筛选条件
      </span>
      <ChevronDown :size="17" :class="{ rotated: expanded }" />
    </button>
    <div class="filter-bar__content" :class="{ expanded }">
      <div class="filter-bar__fields"><slot></slot></div>
      <div v-if="$slots.actions" class="filter-bar__actions">
        <slot name="actions"></slot>
      </div>
    </div>
  </section>
</template>

<style scoped>
.filter-bar {
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.filter-bar__content {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
}

.filter-bar__mobile-toggle {
  display: none;
}

.filter-bar__fields {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 12px;
}

.filter-bar__actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
}

@media (max-width: 760px) {
  .filter-bar {
    padding: 8px;
  }

  .filter-bar__mobile-toggle {
    display: flex;
    width: 100%;
    min-height: 44px;
    align-items: center;
    justify-content: space-between;
    padding: 0 8px;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--ink);
    font-size: 13px;
    font-weight: 700;
  }

  .filter-bar__mobile-toggle > span {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  .filter-bar__mobile-toggle > svg {
    color: var(--ink-muted);
    transition: transform var(--motion-normal) var(--ease-out);
  }

  .filter-bar__mobile-toggle > svg.rotated {
    transform: rotate(180deg);
  }

  .filter-bar__content {
    display: none;
    align-items: stretch;
    flex-direction: column;
    gap: 12px;
    padding: 8px 4px 4px;
  }

  .filter-bar__content.expanded {
    display: flex;
    animation: filters-in var(--motion-normal) var(--ease-out) both;
  }

  .filter-bar__fields {
    display: grid;
    grid-template-columns: 1fr;
  }

  .filter-bar__actions {
    justify-content: stretch;
  }

  .filter-bar__actions :deep(.base-button) {
    width: 100%;
  }
}

@keyframes filters-in {
  from {
    opacity: 0;
    transform: translateY(-6px);
  }
}
</style>
