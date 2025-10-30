<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import MainLayout from '@/views/layouts/MainLayout.vue'

const route = useRoute()

// 判断是否在认证页面
const isAuthPage = computed(() => {
  return ['/login', '/register'].includes(route.path)
})
</script>

<template>
  <!-- 认证页面不显示主布局 -->
  <div v-if="isAuthPage" class="auth-layout">
    <RouterView />
  </div>

  <!-- 主应用布局 - 使用MainLayout -->
  <MainLayout v-else>
    <RouterView />
  </MainLayout>
</template>

<style scoped>
.auth-layout {
  min-height: 100vh;
}
</style>
