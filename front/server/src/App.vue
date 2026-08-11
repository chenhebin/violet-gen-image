<script setup lang="ts">
import { RouterView } from 'vue-router'
import ToastViewport from '@/components/base/ToastViewport.vue'
import { useRem } from '@/composables/useRem'

useRem()
</script>

<template>
  <RouterView />
  <ToastViewport />
</template>
