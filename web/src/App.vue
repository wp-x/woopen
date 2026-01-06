<template>
  <div class="noise-bg"></div>
  <div class="cyber-grid"></div>
  <router-view />
  <GlobalToast />
</template>

<script setup lang="ts">
import GlobalToast from './components/GlobalToast.vue'
// 根组件 - 全局背景特效已添加
</script>
