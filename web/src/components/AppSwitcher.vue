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
      
      <!-- 设置按钮（仅管理员可见） -->
      <el-button
        v-if="currentApp && hasAdminPermission"
        class="settings-button"
        :icon="Setting"
        circle
        size="default"
        @click="handleOpenSettings"
        title="工作空间设置（需要 app:admin 权限）"
      />
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

    <!-- 工作空间设置弹窗 -->
    <WorkspaceSettingsDialog
      v-model="settingsDialogVisible"
      :current-app="currentApp"
      @saved="handleSettingsSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ArrowUp, FolderOpened, Setting } from '@element-plus/icons-vue'
import type { App, ServiceTree } from '@/types'
import { hasPermission } from '@/utils/permission'
import WorkspaceListDialog from './WorkspaceListDialog.vue'
import WorkspaceSettingsDialog from './WorkspaceSettingsDialog.vue'

interface Props {
  currentApp: App | null
  appList: App[]
  loadingApps: boolean
  serviceTree?: ServiceTree[]  // ⭐ 服务树（用于获取应用节点权限）
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
const settingsDialogVisible = ref(false)

// ⭐ 检查是否有 app:admin 权限
// 方案：从服务树的根节点获取应用权限（如果根节点有 app:admin 权限，说明用户有应用管理员权限）
const hasAdminPermission = computed(() => {
  if (!props.currentApp || !props.serviceTree || props.serviceTree.length === 0) {
    return false
  }
  
  // 应用节点路径：/{user}/{app}
  const appPath = `/${props.currentApp.user}/${props.currentApp.code}`
  
  // ⭐ 方案1：从服务树的第一个根节点获取权限（如果根节点有 app:admin 权限，说明用户有应用管理员权限）
  // 注意：服务树返回的是目录和函数节点，应用节点权限会继承到所有子节点
  // 如果根节点有 app:admin 权限，说明用户有应用管理员权限
  for (const node of props.serviceTree) {
    // 检查根节点是否有 app:admin 权限
    if (node.permissions && node.permissions['app:admin'] === true) {
      return true
    }
    // 递归检查子节点（应用权限会继承到所有子节点）
    const checkNode = (n: ServiceTree): boolean => {
      if (n.permissions && n.permissions['app:admin'] === true) {
        return true
      }
      if (n.children) {
        for (const child of n.children) {
          if (checkNode(child)) {
            return true
          }
        }
      }
      return false
    }
    if (checkNode(node)) {
      return true
    }
  }
  
  // ⭐ 方案2：如果服务树中没有权限信息，检查当前用户是否是应用管理员（从 admins 字段）
  // 注意：这只是临时方案，实际应该从后端获取权限
  // 但为了简化，我们可以检查 currentApp.admins 字段
  // 不过 currentApp 可能没有 admins 字段，所以这个方案不可靠
  
  return false
})

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

// 打开设置对话框
const handleOpenSettings = () => {
  settingsDialogVisible.value = true
}

// 设置保存后的回调
const handleSettingsSaved = () => {
  // 触发重新加载应用列表，以获取最新的管理员信息
  emit('load-apps')
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

.settings-button {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;

  &:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    transform: translateY(-2px);
  }
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
