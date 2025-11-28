<template>
  <div class="app-switcher">
    <div class="app-container">
      <el-dropdown 
        trigger="click" 
        placement="top-start" 
        @command="handleSwitchApp"
        @visible-change="handleVisibleChange"
        popper-class="app-dropdown-popper"
      >
        <div class="app-current" v-if="currentApp">
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
        <div class="app-current" v-else>
          <div class="app-avatar">
            <div class="app-icon" style="background-color: #909399;">
              ?
            </div>
          </div>
          <div class="app-info">
            <div class="app-name">请选择应用</div>
            <div class="app-path">
              <el-icon class="path-icon"><FolderOpened /></el-icon>
              <span>点击选择应用</span>
            </div>
          </div>
          <div class="expand-section">
            <el-icon class="expand-icon">
              <ArrowUp />
            </el-icon>
          </div>
        </div>
      
      <template #dropdown>
        <div class="app-dropdown">
          <!-- 头部 -->
          <div class="dropdown-header">
            <div class="header-content">
              <div class="header-title">
                <el-icon class="header-icon"><Grid /></el-icon>
                应用列表
              </div>
              <div class="header-subtitle">选择或管理你的应用</div>
            </div>
          </div>
          
          <!-- 加载状态 -->
          <div v-if="loadingApps" class="loading-state">
            <div class="loading-content">
              <el-icon class="loading-icon"><Loading /></el-icon>
              <span class="loading-text">正在加载应用列表...</span>
            </div>
          </div>

          <!-- 应用列表 -->
          <div v-else class="app-list">
            <el-scrollbar v-if="appList.length > 0" max-height="300px" class="app-scrollbar">
              <div class="app-items">
                <div 
                  v-for="app in appList" 
                  :key="app.id" 
                  class="app-item"
                  :class="{ 'is-active': currentApp && app.id === currentApp.id }"
                  @click="handleSwitchApp(app)"
                >
                  <div class="item-avatar">
                    <div class="item-icon" :style="{ backgroundColor: getAppColor(app) }">
                      {{ getAppInitial(app.name || app.code) }}
                    </div>
                  </div>
                  <div class="item-content">
                    <div class="item-title">{{ app.name || app.code }}</div>
                    <div class="item-path">
                      <el-icon class="path-icon"><FolderOpened /></el-icon>
                      {{ app.user }}/{{ app.code }}
                    </div>
                  </div>
                  <div class="item-actions">
                    <el-button
                      link
                      size="small"
                      class="update-btn"
                      title="重新编译应用"
                      @click="(e) => handleUpdateApp(app, e)"
                    >
                      <el-icon><RefreshRight /></el-icon>
                    </el-button>
                    <el-button
                      link
                      size="small"
                      class="delete-btn"
                      title="删除应用"
                      @click="(e) => handleDeleteApp(app, e)"
                    >
                      <el-icon><Delete /></el-icon>
                    </el-button>
                    <div v-if="currentApp && app.id === currentApp.id" class="check-badge">
                      <el-icon class="check-icon"><Check /></el-icon>
                    </div>
                  </div>
                </div>
              </div>
            </el-scrollbar>
            <div v-else class="empty-app-list">
              <el-empty description="暂无应用" :image-size="80">
                <el-button type="primary" @click="$emit('create-app')">
                  <el-icon><Plus /></el-icon>
                  创建应用
                </el-button>
              </el-empty>
            </div>
          </div>
          
          <!-- 底部操作 -->
          <div class="dropdown-footer">
            <el-button 
              type="primary" 
              class="create-btn"
              @click="$emit('create-app')"
            >
              <el-icon class="create-icon"><Plus /></el-icon>
              创建新应用
            </el-button>
          </div>
        </div>
      </template>
    </el-dropdown>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowUp, Loading, Check, Plus, FolderOpened, Grid, RefreshRight, Delete } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'
import type { App } from '@/types'

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

// 应用颜色映射
const appColors = [
  '#3C9AE8', '#52C41A', '#F5222D', '#FAAD14', '#1890FF', 
  '#722ED1', '#EB2F96', '#13C2C2', '#FA8C16', '#A0D911'
]

// 获取应用颜色
const getAppColor = (app: App) => {
  const index = appList.value.findIndex(a => a.id === app.id)
  return appColors[index % appColors.length] || appColors[0]
}

// 获取应用首字母
const getAppInitial = (text: string) => {
  if (!text) return 'A'
  return text.charAt(0).toUpperCase()
}

// 切换应用
const handleSwitchApp = (app: App) => {
  if (app.id === props.currentApp?.id) return
  emit('switch-app', app)
}

// 更新应用（重新编译）
const handleUpdateApp = (app: App, event: Event) => {
  event.stopPropagation() // 阻止冒泡，避免触发切换应用
  emit('update-app', app)
}

// 删除应用
const handleDeleteApp = (app: App, event: Event) => {
  event.stopPropagation() // 阻止冒泡，避免触发切换应用
  emit('delete-app', app)
}

// 下拉框可见性变化
const handleVisibleChange = (visible: boolean) => {
  if (visible) {
    emit('load-apps')
  }
}

const appList = computed(() => props.appList)
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

.app-dropdown {
  min-width: 320px;
  background: var(--el-bg-color);
  border-radius: 12px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.15);
}

.dropdown-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--el-border-color-light);
}

.header-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  
  .header-icon {
    font-size: 18px;
    color: var(--el-color-primary);
  }
}

.header-subtitle {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.loading-state {
  padding: 40px 20px;
  text-align: center;
}

.loading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  
  .loading-icon {
    font-size: 24px;
    color: var(--el-color-primary);
    animation: rotate 1s linear infinite;
  }
  
  .loading-text {
    font-size: 14px;
    color: var(--el-text-color-secondary);
  }
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.app-list {
  max-height: 300px;
}

.app-items {
  padding: 8px 0;
}

.app-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 20px;
  cursor: pointer;
  transition: background-color 0.2s;

  &:hover {
    background: var(--el-fill-color-light);
  }

  &.is-active {
    background: var(--el-color-primary-light-9);
    
    .item-title {
      color: var(--el-color-primary);
      font-weight: 600;
    }
  }
}

.item-avatar {
  flex-shrink: 0;
}

.item-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  font-size: 14px;
}

.item-content {
  flex: 1;
  min-width: 0;
}

.item-title {
  font-size: 14px;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-path {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  
  .path-icon {
    font-size: 12px;
  }
}

.item-actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.update-btn,
.delete-btn {
  opacity: 0;
  transition: opacity 0.2s;
  color: var(--el-text-color-secondary);
  
  &:hover {
    color: var(--el-color-primary);
  }
}

.delete-btn:hover {
  color: var(--el-color-danger);
}

.app-item:hover .update-btn,
.app-item:hover .delete-btn {
  opacity: 1;
}

.check-badge {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--el-color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  
  .check-icon {
    color: white;
    font-size: 14px;
  }
}

.dropdown-footer {
  padding: 12px 20px;
  border-top: 1px solid var(--el-border-color-light);
}

.create-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

:deep(.app-dropdown-popper) {
  padding: 0 !important;
}
</style>
