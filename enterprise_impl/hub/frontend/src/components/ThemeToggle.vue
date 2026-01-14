<template>
  <el-tooltip :content="tooltipText" placement="bottom">
    <el-button 
      :icon="themeIcon" 
      circle
      @click="handleToggle"
      class="theme-toggle-btn"
    />
  </el-tooltip>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Sunny, Moon } from '@element-plus/icons-vue'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()

// 当前主题图标
const themeIcon = computed(() => {
  return themeStore.currentTheme.mode === 'dark' ? Moon : Sunny
})

// 提示文本
const tooltipText = computed(() => {
  return themeStore.currentTheme.mode === 'dark' ? '切换到浅色模式' : '切换到深色模式'
})

// 切换主题
const handleToggle = () => {
  themeStore.toggleTheme()
}
</script>

<style scoped>
.theme-toggle-btn {
  transition: all 0.3s ease;
}

.theme-toggle-btn:hover {
  transform: rotate(180deg) scale(1.1);
}
</style>

