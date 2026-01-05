<template>
  <div class="app-switcher">
    <div class="app-container">
      <div 
        class="app-current" 
        v-if="currentApp"
        @click="handleOpenDialog"
      >
          <div class="app-avatar">
            <div class="app-icon" :style="{ backgroundColor: getAppColor(currentApp) }">
              {{ getAppInitial(currentApp.name || currentApp.code) }}
            </div>
            <div class="status-indicator"></div>
          </div>
          <div class="app-info">
            <div class="app-name">{{ currentApp.name || currentApp.code }}</div>
            <div class="app-path">
              <el-icon class="path-icon"><FolderOpened /></el-icon>
              <span>{{ currentApp.user }}/{{ currentApp.code }}</span>
            </div>
          </div>
          <div class="expand-section">
            <el-icon class="expand-icon">
              <ArrowUp />
            </el-icon>
          </div>
        </div>
      <div 
        class="app-current" 
        v-else
        @click="handleOpenDialog"
      >
          <div class="app-avatar">
            <div class="app-icon" style="background-color: #909399;">
              ?
            </div>
          </div>
          <div class="app-info">
            <div class="app-name">请选择工作空间</div>
            <div class="app-path">
              <el-icon class="path-icon"><FolderOpened /></el-icon>
              <span>点击选择工作空间</span>
            </div>
          </div>
          <div class="expand-section">
            <el-icon class="expand-icon">
              <ArrowUp />
            </el-icon>
          </div>
        </div>
    </div>

    <!-- 工作空间列表弹窗 -->
    <WorkspaceListDialog
      v-model="dialogVisible"
      :current-app="currentApp"
      @switch-app="handleSwitchApp"
      @create-app="handleCreateApp"
      @update-app="handleUpdateApp"
      @delete-app="handleDeleteApp"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ArrowUp, FolderOpened } from '@element-plus/icons-vue'
import type { App } from '@/types'
import WorkspaceListDialog from './WorkspaceListDialog.vue'

interface Props {
  currentApp: App | null
  appList: App[]
  loadingApps: boolean
}

interface Emits {
  (e: 'switch-app', app: App): void
  (e: 'create-app'): void
  (e: 'update-app', app: App): void
  (e: 'delete-app', app: App): void
  (e: 'load-apps'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const dialogVisible = ref(false)

// 应用颜色映射
const appColors = [
  '#3C9AE8', '#52C41A', '#F5222D', '#FAAD14', '#1890FF', 
  '#722ED1', '#EB2F96', '#13C2C2', '#FA8C16', '#A0D911'
]

// 获取应用颜色
const getAppColor = (app: App) => {
  const index = props.appList.findIndex(a => a.id === app.id)
  return appColors[index % appColors.length] || appColors[0]
}

// 获取应用首字母
const getAppInitial = (text: string) => {
  if (!text) return 'A'
  return text.charAt(0).toUpperCase()
}

// 打开弹窗
const handleOpenDialog = () => {
  dialogVisible.value = true
  emit('load-apps')
}

// 切换应用
const handleSwitchApp = (app: App) => {
  emit('switch-app', app)
}

// 创建应用
const handleCreateApp = () => {
  dialogVisible.value = false
  emit('create-app')
}

// 更新应用
const handleUpdateApp = (app: App) => {
  emit('update-app', app)
}

// 删除应用
const handleDeleteApp = (app: App) => {
  emit('delete-app', app)
}
</script>

<style scoped>
.app-switcher {
  position: fixed;
  left: 20px;
  bottom: 20px;
  z-index: 1000;
}

.app-container {
  display: flex;
  align-items: center;
  gap: 12px;
}

.app-current {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  min-width: 200px;

  &:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    transform: translateY(-2px);
  }
}

.app-avatar {
  position: relative;
  flex-shrink: 0;
}

.app-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  font-size: 16px;
}

.status-indicator {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 12px;
  height: 12px;
  background: var(--el-color-success);
  border: 2px solid var(--el-bg-color);
  border-radius: 50%;
}

.app-info {
  flex: 1;
  min-width: 0;
}

.app-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-path {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  
  .path-icon {
    font-size: 12px;
  }
}

.expand-section {
  flex-shrink: 0;
}

.expand-icon {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  transition: transform 0.3s;
}
</style>
