<template>
  <div
    class="workspace-list-panel"
    :class="[`workspace-list-panel--${surface}`, { 'is-force-select': forceSelect }]"
    data-testid="workspace-list-dialog"
    :data-workspace-list-surface="surface"
  >
    <div v-if="showHeader" class="panel-header">
      <div class="panel-heading">
        <div class="panel-title">{{ forceSelect ? '请选择工作空间' : '工作空间' }}</div>
        <div class="panel-subtitle">{{ currentApp ? currentWorkspaceDisplayName : '选择一个工作空间进入' }}</div>
      </div>
      <el-button
        v-if="surface === 'popover' && !forceSelect"
        text
        circle
        class="panel-close-button"
        title="关闭"
        data-testid="workspace-list-popover-close"
        @click="$emit('close')"
      >
        <el-icon><Close /></el-icon>
      </el-button>
    </div>

    <p v-if="forceSelect" class="force-select-tip">请选择一个工作空间进入，或创建新工作空间。</p>

    <section
      v-if="surface === 'popover' && currentApp"
      class="current-workspace-card"
      :style="currentWorkspaceStyle"
      data-testid="workspace-current-summary"
    >
      <div class="current-workspace-main">
        <div class="current-workspace-avatar">
          <div class="current-workspace-avatar-icon">
            {{ getWorkspaceInitial(currentApp) }}
          </div>
        </div>
        <div class="current-workspace-info">
          <div class="current-workspace-eyebrow">当前工作空间</div>
          <div class="current-workspace-title-row">
            <span class="current-workspace-title" :title="currentWorkspaceDisplayName">
              {{ currentWorkspaceDisplayName }}
            </span>
            <el-tag :type="currentWorkspaceStatusType" size="small" effect="light">
              {{ currentWorkspaceStatusLabel }}
            </el-tag>
          </div>
          <div class="current-workspace-path" :title="currentWorkspaceRoute">
            <el-icon><FolderOpened /></el-icon>
            <span>{{ currentWorkspaceMetaLine }}</span>
          </div>
        </div>
      </div>

      <div class="current-workspace-actions">
        <el-button
          type="primary"
          plain
          :icon="RefreshRight"
          data-testid="workspace-current-update"
          @click.stop="handleUpdateCurrentWorkspace"
        >
          {{ t('workspace.updateCurrentWorkspace') }}
        </el-button>
      </div>

      <div class="current-workspace-meta-grid">
        <div class="current-workspace-meta-item">
          <div class="current-workspace-meta-icon">
            <el-icon><UserFilled /></el-icon>
          </div>
          <div class="current-workspace-meta-copy">
            <span class="current-workspace-meta-label">所有者</span>
            <span class="current-workspace-meta-value" :title="currentApp.user">{{ currentApp.user }}</span>
          </div>
        </div>
        <div class="current-workspace-meta-item">
          <div class="current-workspace-meta-icon">
            <el-icon><OfficeBuilding /></el-icon>
          </div>
          <div class="current-workspace-meta-copy">
            <span class="current-workspace-meta-label">类型</span>
            <span class="current-workspace-meta-value">{{ currentWorkspaceTypeLabel }}</span>
          </div>
        </div>
        <div class="current-workspace-meta-item">
          <div class="current-workspace-meta-icon">
            <el-icon>
              <Unlock v-if="currentApp.is_public" />
              <Lock v-else />
            </el-icon>
          </div>
          <div class="current-workspace-meta-copy">
            <span class="current-workspace-meta-label">可见性</span>
            <span class="current-workspace-meta-value">{{ currentWorkspaceVisibilityLabel }}</span>
          </div>
        </div>
        <div class="current-workspace-meta-item">
          <div class="current-workspace-meta-icon">
            <el-icon><PriceTag /></el-icon>
          </div>
          <div class="current-workspace-meta-copy">
            <span class="current-workspace-meta-label">版本</span>
            <span class="current-workspace-meta-value" :title="currentWorkspaceVersion">
              {{ currentWorkspaceVersion }}
            </span>
          </div>
        </div>
        <div class="current-workspace-meta-item current-workspace-meta-item--wide">
          <div class="current-workspace-meta-icon">
            <el-icon><Clock /></el-icon>
          </div>
          <div class="current-workspace-meta-copy">
            <span class="current-workspace-meta-label">最近更新</span>
            <span class="current-workspace-meta-value" :title="currentWorkspaceUpdatedAt">
              {{ currentWorkspaceUpdatedAt }}
            </span>
          </div>
        </div>
        <div class="current-workspace-meta-item">
          <div class="current-workspace-meta-icon">
            <el-icon><CollectionTag /></el-icon>
          </div>
          <div class="current-workspace-meta-copy">
            <span class="current-workspace-meta-label">标识</span>
            <span class="current-workspace-meta-value" :title="currentApp.code">{{ currentApp.code }}</span>
          </div>
        </div>
      </div>
    </section>

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
              :style="workspaceCardStyle(app)"
              @click="handleSelectWorkspace(app)"
            >
              <div class="card-header">
                <div class="workspace-avatar">
                  <div class="avatar-icon">
                    {{ getWorkspaceInitial(app) }}
                  </div>
                </div>
                <div class="workspace-info">
                  <div class="workspace-name">
                    <span class="workspace-name-text" :title="getWorkspaceDisplayName(app)">
                      {{ getWorkspaceDisplayName(app) }}
                    </span>
                  </div>
                  <div class="workspace-path">
                    <el-icon><FolderOpened /></el-icon>
                    <span class="workspace-path-text" :title="getWorkspaceRoute(app)">
                      {{ getWorkspaceMetaLine(app) }}
                    </span>
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
              :style="workspaceCardStyle(app)"
              @click="handleSelectWorkspace(app)"
            >
              <div class="card-header">
                <div class="workspace-avatar">
                  <div class="avatar-icon">
                    {{ getWorkspaceInitial(app) }}
                  </div>
                </div>
                <div class="workspace-info">
                  <div class="workspace-name">
                    <span class="workspace-name-text" :title="getWorkspaceDisplayName(app)">
                      {{ getWorkspaceDisplayName(app) }}
                    </span>
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
                    <span class="workspace-path-text" :title="getWorkspaceRoute(app)">
                      {{ getWorkspaceMetaLine(app) }}
                    </span>
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
              :style="workspaceCardStyle(app)"
              @click="handleSelectWorkspace(app)"
            >
              <div class="card-header">
                <div class="workspace-avatar">
                  <div class="avatar-icon">
                    {{ getWorkspaceInitial(app) }}
                  </div>
                </div>
                <div class="workspace-info">
                  <div class="workspace-name">
                    <span class="workspace-name-text" :title="getWorkspaceDisplayName(app)">
                      {{ getWorkspaceDisplayName(app) }}
                    </span>
                    <el-tooltip content="官方认证工作空间" placement="top">
                      <img
                        src="/官方认证.svg"
                        alt="官方认证"
                        class="certified-badge-icon"
                      />
                    </el-tooltip>
                  </div>
                  <div class="workspace-path">
                    <el-icon><FolderOpened /></el-icon>
                    <span class="workspace-path-text" :title="getWorkspaceRoute(app)">
                      {{ getWorkspaceMetaLine(app) }}
                    </span>
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

    <div class="panel-footer">
      <el-button
        v-if="surface === 'dialog' && !forceSelect"
        data-testid="workspace-list-close"
        @click="$emit('close')"
      >
        关闭
      </el-button>
      <el-button type="primary" data-testid="workspace-list-create" @click="$emit('create-app')">
        <el-icon><Plus /></el-icon>
        创建新工作空间
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, type CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Search,
  Loading,
  Plus,
  Check,
  FolderOpened,
  RefreshRight,
  Delete,
  Close,
  UserFilled,
  OfficeBuilding,
  Unlock,
  Lock,
  PriceTag,
  Clock,
  CollectionTag
} from '@element-plus/icons-vue'
import { getAppList } from '@/architecture/presentation/context/api/app'
import type { App } from '@/architecture/domain/types'
import { ElMessage, ElMessageBox } from 'element-plus'
import UserDisplay from '@/architecture/presentation/shared/components/UserDisplay.vue'

interface Props {
  currentApp: App | null
  forceSelect?: boolean
  surface?: 'dialog' | 'popover'
  visible?: boolean
  showHeader?: boolean
}

interface Emits {
  (e: 'switch-app', app: App): void
  (e: 'create-app'): void
  (e: 'update-app', app: App): void
  (e: 'delete-app', app: App): void
  (e: 'close'): void
}

const props = withDefaults(defineProps<Props>(), {
  forceSelect: false,
  surface: 'dialog',
  visible: true,
  showHeader: false
})

const emit = defineEmits<Emits>()
const { t } = useI18n()

const activeTab = ref<'mine' | 'all' | 'system'>('mine')
const searchKeyword = ref('')
const loading = ref(false)
const myWorkspaces = ref<App[]>([])
const allWorkspaces = ref<App[]>([])
const systemWorkspaces = ref<App[]>([])

const showHeader = computed(() => props.showHeader)
const currentWorkspaceDisplayName = computed(() => props.currentApp ? getWorkspaceDisplayName(props.currentApp) : '')
const currentWorkspaceRoute = computed(() => props.currentApp ? `/workspace/${props.currentApp.user}/${props.currentApp.code}` : '')
const currentWorkspaceMetaLine = computed(() => props.currentApp ? getWorkspaceMetaLine(props.currentApp) : '')
const currentWorkspaceStyle = computed(() => props.currentApp ? workspaceCardStyle(props.currentApp) : undefined)
const currentWorkspaceStatusLabel = computed(() => props.currentApp?.status === 'disabled' ? '已停用' : '运行中')
const currentWorkspaceStatusType = computed(() => props.currentApp?.status === 'disabled' ? 'danger' : 'success')
const currentWorkspaceTypeLabel = computed(() => props.currentApp?.type === 1 ? '系统工作空间' : '用户工作空间')
const currentWorkspaceVisibilityLabel = computed(() => props.currentApp?.is_public ? '公开' : '私有')
const currentWorkspaceVersion = computed(() => props.currentApp?.version || '未标记')
const currentWorkspaceUpdatedAt = computed(() => formatDateTime(props.currentApp?.updated_at))

const appColors = [
  '#3C9AE8', '#52C41A', '#F5222D', '#FAAD14', '#1890FF',
  '#722ED1', '#EB2F96', '#13C2C2', '#FA8C16', '#A0D911'
]

const getAppColor = (app: App) => {
  const allApps = [...myWorkspaces.value, ...allWorkspaces.value, ...systemWorkspaces.value]
  const index = allApps.findIndex(a => a.id === app.id)
  return appColors[index % appColors.length] || appColors[0]
}

const getAppInitial = (text: string) => {
  if (!text) return 'A'
  return text.charAt(0).toUpperCase()
}

const getWorkspaceDisplayName = (app: App) => {
  const name = app.name?.trim()
  return name || app.code || '未命名工作空间'
}

const getWorkspaceInitial = (app: App) => getAppInitial(getWorkspaceDisplayName(app))

const getWorkspaceRoute = (app: App) => `/workspace/${app.user}/${app.code}`

const getWorkspaceMetaLine = (app: App) => {
  const parts = []
  if (app.code) {
    parts.push(`标识：${app.code}`)
  }
  if (app.user) {
    parts.push(`所有者：${app.user}`)
  }
  return parts.join(' · ')
}

const workspaceCardStyle = (app: App): CSSProperties => ({
  '--workspace-color': getAppColor(app)
} as CSSProperties)

const formatDateTime = (value?: string) => {
  if (!value) {
    return '暂无记录'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  }).format(date).replace(/\//g, '-')
}

const loadWorkspaces = async () => {
  try {
    loading.value = true

    const myApps = await getAppList(200, searchKeyword.value || undefined, false)
    myWorkspaces.value = myApps

    const allApps = await getAppList(200, searchKeyword.value || undefined, true)
    allWorkspaces.value = allApps.filter((app: App) => app.user !== props.currentApp?.user || !myApps.some((my: App) => my.id === app.id))

    const systemApps = await getAppList(200, searchKeyword.value || undefined, false, 1)
    systemWorkspaces.value = systemApps
  } catch (error: any) {
    console.error('加载工作空间列表失败:', error)
    ElMessage.error('加载工作空间列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  loadWorkspaces()
}

const handleTabChange = () => {
  // 当前三个标签页共用同一批加载结果，切换时无需重新请求。
}

const handleSelectWorkspace = (app: App) => {
  if (app.id !== props.currentApp?.id) {
    emit('switch-app', app)
  }
  emit('close')
}

async function confirmUpdateWorkspace(app: App) {
  try {
    await ElMessageBox.confirm(
      t('workspace.updateWorkspaceConfirm', { name: app.name || app.code }),
      t('workspace.updateWorkspaceConfirmTitle'),
      {
        type: 'warning',
        confirmButtonText: t('workspace.updateWorkspaceConfirmButton'),
        cancelButtonText: t('common.cancel')
      }
    )
    return true
  } catch {
    return false
  }
}

const handleUpdateApp = async (app: App) => {
  if (await confirmUpdateWorkspace(app)) {
    emit('update-app', app)
  }
}

const handleUpdateCurrentWorkspace = async () => {
  if (!props.currentApp) {
    return
  }
  if (await confirmUpdateWorkspace(props.currentApp)) {
    emit('update-app', props.currentApp)
  }
}

const handleDeleteApp = (app: App) => {
  emit('delete-app', app)
}

watch(() => props.visible, (newVal: boolean) => {
  if (newVal) {
    loadWorkspaces()
    return
  }

  searchKeyword.value = ''
}, { immediate: true })
</script>

<style scoped lang="scss">
.workspace-list-panel {
  min-height: 400px;
  min-width: 0;
}

.workspace-list-panel--popover {
  width: min(760px, calc(100vw - 32px));
  min-height: 0;
  padding: 20px;
}

.panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin: -20px -20px 16px;
  padding: 18px 20px 14px;
  border-bottom: 1px solid rgba(var(--el-color-primary-rgb), 0.12);
  background: rgba(var(--el-color-primary-rgb), 0.04);
}

.panel-heading {
  min-width: 0;
}

.panel-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  line-height: 1.2;
}

.panel-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.panel-close-button {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);

  &:hover {
    color: var(--el-color-primary);
    background: rgba(var(--el-color-primary-rgb), 0.1);
  }
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

.current-workspace-card {
  position: relative;
  min-width: 0;
  margin-bottom: 14px;
  padding: 16px;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--workspace-color, var(--el-color-primary)) 24%, var(--app-auth-card-border));
  border-radius: 16px;
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--workspace-color, var(--el-color-primary)) 16%, transparent), transparent 46%),
    var(--app-auth-card-bg-strong);
  box-shadow: var(--app-auth-card-shadow-soft);

  &::before {
    content: '';
    position: absolute;
    inset: 12px auto 12px 0;
    width: 3px;
    border-radius: 0 999px 999px 0;
    background: var(--workspace-color, var(--el-color-primary));
  }
}

.current-workspace-main {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}

.current-workspace-avatar {
  flex-shrink: 0;
}

.current-workspace-avatar-icon {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--workspace-color, var(--el-color-primary)) 84%, #fff), var(--workspace-color, var(--el-color-primary)));
  font-size: 18px;
  font-weight: 700;
  box-shadow: 0 12px 24px color-mix(in srgb, var(--workspace-color, var(--el-color-primary)) 26%, transparent);
}

.current-workspace-info {
  flex: 1;
  min-width: 0;
}

.current-workspace-eyebrow {
  margin-bottom: 3px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 650;
  line-height: 1.2;
}

.current-workspace-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.current-workspace-title {
  min-width: 0;
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 18px;
  font-weight: 750;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.current-workspace-title-row :deep(.el-tag) {
  flex-shrink: 0;
}

.current-workspace-path {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;

  .el-icon {
    flex-shrink: 0;
  }

  span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.current-workspace-actions {
  position: relative;
  z-index: 1;
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}

.current-workspace-meta-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.current-workspace-meta-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 9px 10px;
  border: 1px solid rgba(var(--el-color-primary-rgb), 0.1);
  border-radius: 12px;
  background: color-mix(in srgb, var(--app-auth-input-bg) 82%, transparent);
}

.current-workspace-meta-icon {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--workspace-color, var(--el-color-primary));
  background: color-mix(in srgb, var(--workspace-color, var(--el-color-primary)) 12%, transparent);
}

.current-workspace-meta-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.current-workspace-meta-label {
  color: var(--el-text-color-secondary);
  font-size: 11px;
  line-height: 1.1;
}

.current-workspace-meta-value {
  min-width: 0;
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 12px;
  font-weight: 650;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.search-bar {
  margin-bottom: 14px;
}

.search-bar :deep(.el-input__wrapper) {
  min-height: 42px;
  border-radius: 12px;
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
  overflow-x: hidden;
  scrollbar-gutter: stable;
}

.workspace-list-panel--popover .workspace-list-container {
  min-height: 220px;
  max-height: clamp(220px, calc(100vh - 410px), 360px);
}

.workspace-list-panel--popover :deep(.el-tabs__header) {
  margin-bottom: 12px;
}

.workspace-list-panel--popover :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
  background: rgba(var(--el-color-primary-rgb), 0.16);
}

.workspace-list-panel--popover :deep(.el-tabs__item) {
  height: 38px;
  padding: 0 18px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  font-weight: 650;
}

.workspace-list-panel--popover :deep(.el-tabs__item.is-active) {
  color: var(--el-color-primary);
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

.workspace-list-panel--popover .workspace-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  padding: 4px 4px 8px 0;
}

.workspace-card {
  position: relative;
  min-width: 0;
  border: 1px solid var(--app-auth-card-border);
  border-radius: 22px;
  padding: 18px;
  cursor: pointer;
  transition: all 0.2s;
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--workspace-color, var(--el-color-primary)) 12%, transparent), transparent 48%),
    var(--app-auth-card-bg-strong);
  box-shadow: var(--app-auth-card-shadow-soft);
  overflow: hidden;

  &::before {
    content: '';
    position: absolute;
    top: 12px;
    bottom: 12px;
    left: 0;
    width: 3px;
    border-radius: 0 999px 999px 0;
    background: var(--workspace-color, var(--el-color-primary));
    opacity: 0.82;
  }

  &:hover {
    border-color: color-mix(in srgb, var(--workspace-color, var(--el-color-primary)) 42%, var(--app-auth-card-border));
    box-shadow: var(--app-auth-card-shadow);
    transform: translateY(-2px);
  }

  &.is-active {
    border-color: color-mix(in srgb, var(--workspace-color, var(--el-color-primary)) 68%, var(--el-color-primary));
    box-shadow:
      0 0 0 1px color-mix(in srgb, var(--workspace-color, var(--el-color-primary)) 38%, transparent),
      0 14px 34px rgba(2, 6, 23, 0.22);
  }
}

.workspace-list-panel--popover .workspace-card {
  border-radius: 14px;
  padding: 13px 12px 12px 14px;
}

.card-header {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
  margin-bottom: 11px;
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
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--workspace-color, var(--el-color-primary)) 86%, #fff), var(--workspace-color, var(--el-color-primary)));
  font-weight: 600;
  font-size: 16px;
  box-shadow: 0 10px 22px color-mix(in srgb, var(--workspace-color, var(--el-color-primary)) 24%, transparent);
}

.workspace-info {
  flex: 1;
  min-width: 0;
}

.workspace-name {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
}

.workspace-name-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-path {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);

  .el-icon {
    flex-shrink: 0;
  }
}

.workspace-path-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.active-badge {
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--workspace-color, var(--el-color-primary));
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  box-shadow: 0 8px 18px color-mix(in srgb, var(--workspace-color, var(--el-color-primary)) 28%, transparent);
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--app-auth-card-border);
}

.footer-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.card-actions {
  display: flex;
  gap: 8px;
  opacity: 1;
  transition: opacity 0.2s;
}

.panel-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin: 14px -20px -20px;
  padding: 14px 20px 18px;
  border-top: 1px solid rgba(var(--el-color-primary-rgb), 0.12);
  background: rgba(var(--el-color-primary-rgb), 0.035);
}

.panel-footer :deep(.el-button) {
  height: 44px;
  border-radius: 14px;
  font-weight: 600;
  border: 1px solid var(--app-auth-input-border);
  background: var(--app-auth-input-bg);
  box-shadow: none;
  transition: all 0.3s ease;
}

.panel-footer :deep(.el-button:hover) {
  transform: translateY(-1px);
  border-color: rgba(var(--el-color-primary-rgb), 0.42);
  color: var(--el-color-primary);
  box-shadow: var(--app-auth-input-shadow-hover);
}

.panel-footer :deep(.el-button--primary) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: #fff;
  box-shadow: var(--app-auth-primary-shadow);
}

.panel-footer :deep(.el-button--primary:hover) {
  color: #fff;
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  box-shadow: var(--app-auth-primary-shadow-hover);
}

.certified-badge-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  vertical-align: middle;
}

.workspace-list-panel--popover .panel-footer :deep(.el-button) {
  max-width: 100%;
}

.workspace-list-panel--popover .footer-left :deep(.user-display-wrapper) {
  min-width: 0;
  overflow: hidden;
}

@media (max-width: 820px) {
  .workspace-list-panel--popover {
    width: calc(100vw - 32px);
  }

  .current-workspace-meta-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .workspace-list-panel--popover .workspace-grid {
    grid-template-columns: 1fr;
  }

  .current-workspace-meta-grid {
    grid-template-columns: 1fr;
  }
}
</style>
