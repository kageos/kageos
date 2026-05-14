<template>
  <el-dialog
    v-model="visible"
    class="workspace-list-dialog-shell"
    :title="forceSelect ? '请选择工作空间' : '工作空间列表'"
    width="900px"
    :append-to-body="true"
    :close-on-click-modal="false"
    :show-close="!forceSelect"
    :before-close="forceSelect ? handleBeforeClose : undefined"
    @close="handleClose"
  >
    <div class="workspace-list-dialog" data-testid="workspace-list-dialog">
      <p v-if="forceSelect" class="force-select-tip">请选择一个工作空间进入，或创建新工作空间。</p>
      <!-- 搜索栏 -->
      <div class="search-bar" data-testid="workspace-list-search-wrap">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索工作空间名称或代码"
          clearable
          data-testid="workspace-list-search"
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <!-- 标签页：我的工作空间 / 全部工作空间 / 系统工作空间 -->
      <el-tabs v-model="activeTab" data-testid="workspace-list-tabs" @tab-change="handleTabChange">
        <el-tab-pane label="我的工作空间" name="mine">
          <div class="workspace-list-container">
            <div v-if="loading" class="loading-state">
              <el-icon class="loading-icon"><Loading /></el-icon>
              <span>加载中...</span>
            </div>
            <div v-else-if="myWorkspaces.length === 0" class="empty-state">
              <el-empty description="暂无工作空间">
                <el-button type="primary" data-testid="workspace-list-create-empty" @click="$emit('create-app')">
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
                :data-testid="`workspace-card-${app.id}`"
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
                      :data-testid="`workspace-card-refresh-${app.id}`"
                      @click.stop="handleUpdateApp(app)"
                    >
                      <el-icon><RefreshRight /></el-icon>
                    </el-button>
                    <el-button
                      link
                      size="small"
                      title="删除"
                      :data-testid="`workspace-card-delete-${app.id}`"
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
                :data-testid="`workspace-card-public-${app.id}`"
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
                    <div class="workspace-name">
                      {{ app.name || app.code }}
                      <el-tooltip
                        v-if="app.type === 1"
                        content="官方认证工作空间"
                        placement="top"
                      >
                        <img 
                          src="/官方认证.svg" 
                          alt="官方认证" 
                          class="certified-badge-icon"
                        />
                      </el-tooltip>
                    </div>
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
        
        <el-tab-pane label="系统工作空间" name="system">
          <div class="workspace-list-container">
            <div v-if="loading" class="loading-state">
              <el-icon class="loading-icon"><Loading /></el-icon>
              <span>加载中...</span>
            </div>
            <div v-else-if="systemWorkspaces.length === 0" class="empty-state">
              <el-empty description="暂无系统工作空间" />
            </div>
            <div v-else class="workspace-grid">
              <div
                v-for="app in systemWorkspaces"
                :key="app.id"
                class="workspace-card"
                :data-testid="`workspace-card-system-${app.id}`"
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
                    <div class="workspace-name">
                      {{ app.name || app.code }}
                      <el-tooltip
                        content="官方认证工作空间"
                        placement="top"
                      >
                        <img 
                          src="/官方认证.svg" 
                          alt="官方认证" 
                          class="certified-badge-icon"
                        />
                      </el-tooltip>
                    </div>
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
                    <el-tag type="success" size="small">系统</el-tag>
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
        <el-button v-if="!forceSelect" data-testid="workspace-list-close" @click="handleClose">关闭</el-button>
        <el-button type="primary" data-testid="workspace-list-create" @click="$emit('create-app')">
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
import { getAppList } from '@/architecture/infrastructure/api/app'
import type { App } from '@/architecture/domain/types'
import { ElMessage } from 'element-plus'
import UserDisplay from '@/architecture/presentation/shared/components/UserDisplay.vue'

interface Props {
  modelValue: boolean
  currentApp: App | null
  /** 为 true 时不可关闭弹窗，必须选择或创建工作空间（如从 /workspace/:user 进入时） */
  forceSelect?: boolean
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
  set: (value: boolean) => emit('update:modelValue', value)
})

const activeTab = ref<'mine' | 'all' | 'system'>('mine')
const searchKeyword = ref('')
const loading = ref(false)
const myWorkspaces = ref<App[]>([])
const allWorkspaces = ref<App[]>([])
const systemWorkspaces = ref<App[]>([])

// 应用颜色映射
const appColors = [
  '#3C9AE8', '#52C41A', '#F5222D', '#FAAD14', '#1890FF', 
  '#722ED1', '#EB2F96', '#13C2C2', '#FA8C16', '#A0D911'
]

// 获取应用颜色
const getAppColor = (app: App) => {
  const allApps = [...myWorkspaces.value, ...allWorkspaces.value, ...systemWorkspaces.value]
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
    allWorkspaces.value = allApps.filter((app: App) => app.user !== props.currentApp?.user || !myApps.some((my: App) => my.id === app.id))
    
    // 加载系统工作空间（type=1）
    const systemApps = await getAppList(200, searchKeyword.value || undefined, false, 1)
    systemWorkspaces.value = systemApps
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

// 强制选择模式下阻止关闭（ESC 等）：不调用 done() 弹窗不会关闭
const handleBeforeClose = (_done: () => void) => {
  // 不调用 done()，弹窗保持打开
}

// 监听弹窗显示状态
watch(visible, (newVal: boolean) => {
  if (newVal) {
    loadWorkspaces()
  }
})
</script>

<style scoped>
.workspace-list-dialog {
  min-height: 400px;
}

:deep(.workspace-list-dialog-shell) {
  border-radius: 28px;
  background: var(--app-auth-card-bg);
  border: 1px solid var(--app-auth-card-border);
  box-shadow: var(--app-auth-card-shadow);
  overflow: hidden;
}

:deep(.workspace-list-dialog-shell .el-dialog__header) {
  padding: 28px 32px 12px;
}

:deep(.workspace-list-dialog-shell .el-dialog__title) {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
}

:deep(.workspace-list-dialog-shell .el-dialog__body) {
  padding: 0 32px 24px;
  background: var(--app-auth-surface-bg);
}

:deep(.workspace-list-dialog-shell .el-dialog__footer) {
  padding: 0 32px 28px;
  background: var(--app-auth-surface-bg);
}

.force-select-tip {
  margin: 0 0 18px;
  padding: 12px 14px;
  background: rgba(var(--el-color-primary-rgb), 0.08);
  border: 1px solid rgba(var(--el-color-primary-rgb), 0.12);
  border-radius: 16px;
  font-size: 13px;
  color: var(--el-text-color-regular);
}

.search-bar {
  margin-bottom: 20px;
}

.search-bar :deep(.el-input__wrapper) {
  min-height: 46px;
  border-radius: 16px;
  background: var(--app-auth-input-bg);
  border: 1px solid var(--app-auth-input-border);
  box-shadow: none;
  transition: all 0.3s ease;
}

.search-bar :deep(.el-input__wrapper:hover) {
  border-color: rgba(var(--el-color-primary-rgb), 0.42);
  box-shadow: var(--app-auth-input-shadow-hover);
}

.search-bar :deep(.el-input__wrapper.is-focus) {
  border-color: var(--el-color-primary);
  box-shadow: var(--app-auth-input-shadow-focus);
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
  border: 1px solid var(--app-auth-card-border);
  border-radius: 22px;
  padding: 18px;
  cursor: pointer;
  transition: all 0.2s;
  background: var(--app-auth-card-bg-strong);
  box-shadow: var(--app-auth-card-shadow-soft);
  
  &:hover {
    border-color: rgba(var(--el-color-primary-rgb), 0.22);
    box-shadow: var(--app-auth-card-shadow);
    transform: translateY(-2px);
  }
  
  &.is-active {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.22), var(--app-auth-card-shadow-soft);
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
  border-radius: 12px;
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
  border-top: 1px solid var(--app-auth-card-border);
}

.footer-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.card-actions {
  display: flex;
  gap: 8px;
  opacity: 1;
  transition: opacity 0.2s;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.dialog-footer :deep(.el-button) {
  height: 44px;
  border-radius: 14px;
  font-weight: 600;
  border: 1px solid var(--app-auth-input-border);
  background: var(--app-auth-input-bg);
  box-shadow: none;
  transition: all 0.3s ease;
}

.dialog-footer :deep(.el-button:hover) {
  transform: translateY(-1px);
  border-color: rgba(var(--el-color-primary-rgb), 0.42);
  color: var(--el-color-primary);
  box-shadow: var(--app-auth-input-shadow-hover);
}

.dialog-footer :deep(.el-button--primary) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: #fff;
  box-shadow: var(--app-auth-primary-shadow);
}

.dialog-footer :deep(.el-button--primary:hover) {
  color: #fff;
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  box-shadow: var(--app-auth-primary-shadow-hover);
}

.workspace-name {
  display: flex;
  align-items: center;
  gap: 6px;
}

.certified-badge-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  vertical-align: middle;
}
</style>
