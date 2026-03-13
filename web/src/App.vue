<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <!-- 🔥 移除 MainLayout，所有页面自己管理布局 -->
  <RouterView />
</template>

<style>
/* 全局样式 */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html,
body {
  height: 100%;
  overflow-y: auto;
}

#app {
  min-height: 100%;
}
</style>
