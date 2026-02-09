<template>
  <div class="workstation-view">
    <!-- 有有效目录时显示工作台对话；无目录时提示从工作空间打开 -->
    <WorkstationChat v-if="validFullCodePath" :full-code-path="validFullCodePath" :embedded="false" />
    <div v-else class="workstation-empty">
      <p>请从工作空间打开工作台</p>
      <p class="hint">在左侧服务目录对任意目录节点悬停，点击「打开工作台」即可开始对话</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import WorkstationChat from '../components/WorkstationChat.vue'

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

.workstation-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px;
  color: var(--el-text-color-secondary);
}
.workstation-empty .hint {
  margin-top: 8px;
  font-size: 13px;
  color: var(--el-text-color-placeholder);
}
</style>
