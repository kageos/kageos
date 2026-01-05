<template>
  <el-dialog
    v-model="visible"
    title="工作空间列表"
    width="900px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div class="workspace-list-dialog">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索工作空间名称或代码"
          clearable
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <!-- 标签页：我的工作空间 / 全部工作空间 -->
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <el-tab-pane label="我的工作空间" name="mine">
          <div class="workspace-list-container">
            <div v-if="loading" class="loading-state">
              <el-icon class="loading-icon"><Loading /></el-icon>
              <span>加载中...</span>
            </div>
            <div v-else-if="myWorkspaces.length === 0" class="empty-state">
              <el-empty description="暂无工作空间">
                <el-button type="primary" @click="$emit('create-app')">
                  <el-icon><Plus /></el-icon>
                  创建工作空间
                </el-button>
              </el-empty>
            </div>
            <div v-else class="workspace-grid">
              <div
                v-for="app in myWorkspaces"
                :key="app.id"
                class="workspace-card"
                :class="{ 'is-active': currentApp && app.id === currentApp.id }"
                @click="handleSelectWorkspace(app)"
              >
                <div class="card-header">
                  <div class="workspace-avatar">
                    <div class="avatar-icon" :style="{ backgroundColor: getAppColor(app) }">
                      {{ getAppInitial(app.name || app.code) }}
                    </div>
                  </div>
                  <div class="workspace-info">
                    <div class="workspace-name">{{ app.name || app.code }}</div>
                    <div class="workspace-path">
                      <el-icon><FolderOpened /></el-icon>
                      {{ app.user }}/{{ app.code }}
                    </div>
                  </div>
                  <div v-if="currentApp && app.id === currentApp.id" class="active-badge">
                    <el-icon><Check /></el-icon>
                  </div>
                </div>
                <div class="card-footer">
                  <el-tag v-if="app.is_public" type="success" size="small">公开</el-tag>
                  <el-tag v-else type="info" size="small">私有</el-tag>
                  <div class="card-actions">
                    <el-button
                      link
                      size="small"
                      title="重新编译"
                      @click.stop="handleUpdateApp(app)"
                    >
                      <el-icon><RefreshRight /></el-icon>
                    </el-button>
                    <el-button
                      link
                      size="small"
                      title="删除"
                      @click.stop="handleDeleteApp(app)"
                    >
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>
        
        <el-tab-pane label="全部工作空间" name="all">
          <div class="workspace-list-container">
            <div v-if="loading" class="loading-state">
              <el-icon class="loading-icon"><Loading /></el-icon>
              <span>加载中...</span>
            </div>
            <div v-else-if="allWorkspaces.length === 0" class="empty-state">
              <el-empty description="暂无公开的工作空间" />
            </div>
            <div v-else class="workspace-grid">
              <div
                v-for="app in allWorkspaces"
                :key="app.id"
                class="workspace-card"
                :class="{ 'is-active': currentApp && app.id === currentApp.id }"
                @click="handleSelectWorkspace(app)"
              >
                <div class="card-header">
                  <div class="workspace-avatar">
                    <div class="avatar-icon" :style="{ backgroundColor: getAppColor(app) }">
                      {{ getAppInitial(app.name || app.code) }}
                    </div>
                  </div>
                  <div class="workspace-info">
                    <div class="workspace-name">{{ app.name || app.code }}</div>
                    <div class="workspace-path">
                      <el-icon><FolderOpened /></el-icon>
                      <span>{{ app.user }}/{{ app.code }}</span>
                    </div>
                  </div>
                  <div v-if="currentApp && app.id === currentApp.id" class="active-badge">
                    <el-icon><Check /></el-icon>
                  </div>
                </div>
                <div class="card-footer">
                  <div class="footer-left">
                    <el-tag type="success" size="small">公开</el-tag>
                    <UserDisplay
                      :username="app.user"
                      mode="card"
                      layout="horizontal"
                      size="small"
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">关闭</el-button>
        <el-button type="primary" @click="$emit('create-app')">
          <el-icon><Plus /></el-icon>
          创建新工作空间
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Search, Loading, Plus, Check, FolderOpened, RefreshRight, Delete } from '@element-plus/icons-vue'
import { getAppList } from '@/api/app'
import type { App } from '@/types'
import { ElMessage } from 'element-plus'
import UserDisplay from '@/architecture/presentation/widgets/UserDisplay.vue'

interface Props {
  modelValue: boolean
  currentApp: App | null
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'switch-app', app: App): void
  (e: 'create-app'): void
  (e: 'update-app', app: App): void
  (e: 'delete-app', app: App): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const activeTab = ref<'mine' | 'all'>('mine')
const searchKeyword = ref('')
const loading = ref(false)
const myWorkspaces = ref<App[]>([])
const allWorkspaces = ref<App[]>([])

// 应用颜色映射
const appColors = [
  '#3C9AE8', '#52C41A', '#F5222D', '#FAAD14', '#1890FF', 
  '#722ED1', '#EB2F96', '#13C2C2', '#FA8C16', '#A0D911'
]

// 获取应用颜色
const getAppColor = (app: App) => {
  const allApps = [...myWorkspaces.value, ...allWorkspaces.value]
  const index = allApps.findIndex(a => a.id === app.id)
  return appColors[index % appColors.length] || appColors[0]
}

// 获取应用首字母
const getAppInitial = (text: string) => {
  if (!text) return 'A'
  return text.charAt(0).toUpperCase()
}

// 加载工作空间列表
const loadWorkspaces = async () => {
  try {
    loading.value = true
    
    // 加载我的工作空间
    const myApps = await getAppList(200, searchKeyword.value || undefined, false)
    myWorkspaces.value = myApps
    
    // 加载全部公开的工作空间
    const allApps = await getAppList(200, searchKeyword.value || undefined, true)
    // 过滤掉自己的，只显示其他人的公开工作空间
    allWorkspaces.value = allApps.filter(app => app.user !== props.currentApp?.user || !myApps.some(my => my.id === app.id))
  } catch (error: any) {
    console.error('加载工作空间列表失败:', error)
    ElMessage.error('加载工作空间列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  loadWorkspaces()
}

// 切换标签页
const handleTabChange = () => {
  // 切换标签页时重新加载（如果需要）
}

// 选择工作空间
const handleSelectWorkspace = (app: App) => {
  if (app.id === props.currentApp?.id) {
    handleClose()
    return
  }
  emit('switch-app', app)
  handleClose()
}

// 更新工作空间
const handleUpdateApp = (app: App) => {
  emit('update-app', app)
}

// 删除工作空间
const handleDeleteApp = (app: App) => {
  emit('delete-app', app)
}

// 关闭弹窗
const handleClose = () => {
  visible.value = false
  searchKeyword.value = ''
}

// 监听弹窗显示状态
watch(visible, (newVal) => {
  if (newVal) {
    loadWorkspaces()
  }
})
</script>

<style scoped>
.workspace-list-dialog {
  min-height: 400px;
}

.search-bar {
  margin-bottom: 20px;
}

.workspace-list-container {
  min-height: 300px;
  max-height: 500px;
  overflow-y: auto;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  gap: 12px;
  color: var(--el-text-color-secondary);
  
  .loading-icon {
    font-size: 32px;
    animation: rotate 1s linear infinite;
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

.empty-state {
  padding: 60px 20px;
  text-align: center;
}

.workspace-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 16px;
  padding: 8px 0;
}

.workspace-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.2s;
  background: var(--el-bg-color);
  
  &:hover {
    border-color: var(--el-color-primary);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    transform: translateY(-2px);
  }
  
  &.is-active {
    border-color: var(--el-color-primary);
    border-width: 2px;
    box-shadow: 0 0 0 1px var(--el-color-primary), 0 2px 8px rgba(64, 158, 255, 0.2);
  }
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.workspace-avatar {
  flex-shrink: 0;
}

.avatar-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  font-size: 16px;
}

.workspace-info {
  flex: 1;
  min-width: 0;
}

.workspace-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-path {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  
  .workspace-code {
    margin-left: 4px;
  }
}

.active-badge {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--el-color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.footer-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.card-actions {
  display: flex;
  gap: 8px;
  opacity: 0;
  transition: opacity 0.2s;
}

.workspace-card:hover .card-actions {
  opacity: 1;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>

