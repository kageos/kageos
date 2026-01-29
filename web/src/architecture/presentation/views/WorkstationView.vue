<template>
  <div class="workstation-view">
    <!-- 有有效目录时显示工作台对话；无目录或无效 query（如 [object PointerEvent]）时显示模式列表与配置 -->
    <WorkstationChat v-if="validFullCodePath" :full-code-path="validFullCodePath" :embedded="false" />
    <WorkstationModeManagement v-else />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import WorkstationChat from '../components/WorkstationChat.vue'
import WorkstationModeManagement from '../components/WorkstationModeManagement.vue'

const route = useRoute()

/** 有效的 full_code_path：非空、非 [object xxx]、且像目录路径（含 / 或 .） */
function isValidFullCodePath(q: string): boolean {
  if (!q || typeof q !== 'string') return false
  const s = q.trim()
  if (s.length < 2) return false
  if (s.startsWith('[object ') && s.endsWith(']')) return false
  return s.includes('/') || s.includes('.')
}

const validFullCodePath = computed(() => {
  const raw = (route.query.full_code_path as string) || ''
  return isValidFullCodePath(raw) ? raw.trim() : ''
})
</script>

<style scoped>
.workstation-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 100vh;
  background: var(--el-bg-color-page);
}
</style>
