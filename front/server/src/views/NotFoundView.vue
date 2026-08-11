<script setup lang="ts">
import { FileQuestion } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
</script>

<template>
  <main class="not-found">
    <div class="not-found__mark"><FileQuestion :size="25" /></div>
    <p class="not-found__code">404</p>
    <h1>没有找到这个页面</h1>
    <p>地址可能已变更，或当前链接已失效。</p>
    <RouterLink to="/manage/dashboard">
      <BaseButton>返回运营概览</BaseButton>
    </RouterLink>
  </main>
</template>

<style scoped>
.not-found {
  display: grid;
  min-height: 100dvh;
  align-content: center;
  justify-items: center;
  padding: 40px 20px;
  background: var(--canvas);
  text-align: center;
}

.not-found__mark {
  display: grid;
  width: 54px;
  height: 54px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  color: var(--primary);
  place-items: center;
}

.not-found__code {
  margin-top: 20px;
  color: var(--ink-faint);
  font-family: var(--font-mono);
  font-size: 11px;
}

h1 {
  margin-top: 6px;
  font-family: var(--font-display);
  font-size: 27px;
}

p:not(.not-found__code) {
  margin: 8px 0 20px;
  color: var(--ink-muted);
  font-size: 13px;
}
</style>
