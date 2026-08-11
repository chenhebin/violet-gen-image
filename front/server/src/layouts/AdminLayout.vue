<script setup lang="ts">
import { ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppSidebar from '@/components/app/AppSidebar.vue'
import AppTopbar from '@/components/app/AppTopbar.vue'

const route = useRoute()
const navigationOpen = ref(false)

watch(
  () => route.fullPath,
  () => {
    navigationOpen.value = false
  },
)
</script>

<template>
  <div class="admin-layout">
    <a class="skip-link" href="#main-content">跳到主要内容</a>
    <AppSidebar
      :open="navigationOpen"
      @close="navigationOpen = false"
    />
    <div class="admin-layout__content">
      <AppTopbar @open-navigation="navigationOpen = true" />
      <main id="main-content" class="admin-layout__main" tabindex="-1">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<style scoped>
.admin-layout {
  min-height: 100dvh;
  background: var(--canvas);
}

.admin-layout__content {
  min-width: 0;
  min-height: 100dvh;
  margin-left: var(--sidebar-width);
}

.admin-layout__main {
  height: calc(100dvh - var(--topbar-height));
  overflow: auto;
}

.skip-link {
  position: fixed;
  z-index: 200;
  top: 8px;
  left: 8px;
  padding: 9px 12px;
  border-radius: var(--radius-sm);
  background: var(--surface);
  box-shadow: var(--shadow-md);
  color: var(--primary);
  font-weight: 700;
  transform: translateY(-150%);
}

.skip-link:focus {
  transform: translateY(0);
}

@media (max-width: 960px) {
  .admin-layout__content {
    margin-left: 0;
  }

  .admin-layout__main {
    height: auto;
    min-height: calc(100dvh - var(--topbar-height));
    overflow: visible;
  }
}
</style>
