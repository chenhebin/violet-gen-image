<script setup lang="ts">
import { ShieldX } from '@lucide/vue'
import BaseButton from '@/components/base/BaseButton.vue'
</script>

<template>
  <div class="state-page">
    <div class="state-page__mark"><ShieldX :size="25" /></div>
    <p class="state-page__code">403</p>
    <h1>当前账号无权访问</h1>
    <p>该模块需要平台管理权限。你仍可继续处理已授权的工作。</p>
    <RouterLink to="/manage/dashboard">
      <BaseButton>返回运营概览</BaseButton>
    </RouterLink>
  </div>
</template>

<style scoped>
.state-page {
  display: grid;
  min-height: 100%;
  align-content: center;
  justify-items: center;
  padding: 40px 20px;
  text-align: center;
}

.state-page__mark {
  display: grid;
  width: 54px;
  height: 54px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  color: var(--danger);
  place-items: center;
}

.state-page__code {
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

p:not(.state-page__code) {
  max-width: 460px;
  margin: 8px 0 20px;
  color: var(--ink-muted);
  font-size: 13px;
}
</style>
