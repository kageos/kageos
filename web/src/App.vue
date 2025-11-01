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
body,
#app {
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Helvetica Neue', sans-serif;
}
</style>
