<template>
  <div
    ref="switcherRootRef"
    class="app-switcher"
    :class="{ 'app-switcher-compact': compact }"
    data-testid="app-switcher"
  >
    <el-popover
      :visible="popoverVisible"
      trigger="click"
      placement="bottom-start"
      :width="760"
      :show-arrow="false"
      :offset="10"
      :popper-style="{ width: 'min(760px, calc(100vw - 32px))' }"
      :teleported="true"
      :disabled="workspaceListForceSelect"
      popper-class="workspace-switcher-popover"
    >
      <template #reference>
        <div class="app-container" data-testid="app-switcher-container">
          <div
            v-if="currentApp"
            class="app-current"
            data-testid="app-switcher-current"
            @click="handleTriggerClick"
          >
            <div class="app-avatar">
              <div class="app-icon" :style="{ backgroundColor: getAppColor(currentApp) }">
                {{ getAppInitial(getWorkspaceDisplayName(currentApp)) }}
              </div>
              <div class="status-indicator" v-if="!compact"></div>
            </div>
            <div class="app-info">
              <div class="app-name" :title="getWorkspaceDisplayName(currentApp)">
                {{ getWorkspaceDisplayName(currentApp) }}
              </div>
              <div class="app-path" v-if="!compact">
                <el-icon class="path-icon"><FolderOpened /></el-icon>
                <span :title="getWorkspaceRoute(currentApp)">{{ getWorkspaceMetaLine(currentApp) }}</span>
              </div>
            </div>
            <div class="expand-section" v-if="!compact">
              <el-icon class="expand-icon">
                <ArrowUp />
              </el-icon>
            </div>
            <el-icon v-else class="expand-icon-inline"><ArrowDown /></el-icon>
          </div>
          <div
            v-else
            class="app-current"
            data-testid="app-switcher-empty"
            @click="handleTriggerClick"
          >
            <div class="app-avatar">
              <div class="app-icon app-icon--placeholder">?</div>
            </div>
            <div class="app-info">
              <div class="app-name">选择工作空间</div>
              <div class="app-path" v-if="!compact">
                <el-icon class="path-icon"><FolderOpened /></el-icon>
                <span>点击选择</span>
              </div>
            </div>
            <el-icon v-if="compact" class="expand-icon-inline"><ArrowDown /></el-icon>
            <div v-else class="expand-section">
              <el-icon class="expand-icon"><ArrowUp /></el-icon>
            </div>
          </div>
        </div>
      </template>

      <WorkspaceListPanel
        v-if="popoverVisible"
        :current-app="currentApp"
        :visible="popoverVisible"
        surface="popover"
        show-header
        @switch-app="handleSwitchApp"
        @create-app="handleCreateApp"
        @update-app="handleUpdateApp"
        @delete-app="handleDeleteApp"
        @close="closeWorkspacePopover"
      />
    </el-popover>

    <WorkspaceListDialog
      v-if="dialogVisible || workspaceListForceSelect"
      v-model="dialogVisible"
      :current-app="currentApp"
      :force-select="workspaceListForceSelect"
      @switch-app="handleSwitchApp"
      @create-app="handleCreateApp"
      @update-app="handleUpdateApp"
      @delete-app="handleDeleteApp"
    />
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ArrowUp, ArrowDown, FolderOpened } from '@element-plus/icons-vue'
import type { App } from '@/architecture/domain/types'
import WorkspaceListDialog from './WorkspaceListDialog.vue'
import WorkspaceListPanel from './WorkspaceListPanel.vue'

interface Props {
  currentApp: App | null
  appList: App[]
  loadingApps: boolean
  compact?: boolean  // 紧凑模式：用于左侧边栏控制区
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
const popoverVisible = ref(false)
const switcherRootRef = ref<HTMLElement | null>(null)
/** 本次打开工作空间列表是否强制必须选择/创建（从 /workspace/:user 进入时为 true） */
const workspaceListForceSelect = ref(false)

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

const getWorkspaceDisplayName = (app: App) => {
  const name = app.name?.trim()
  return name || app.code || '未命名工作空间'
}

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

const handleTriggerClick = () => {
  if (workspaceListForceSelect.value) {
    dialogVisible.value = true
    emit('load-apps')
    return
  }

  popoverVisible.value = !popoverVisible.value
  if (popoverVisible.value) {
    emit('load-apps')
  }
}

const closeWorkspacePopover = () => {
  popoverVisible.value = false
}

// 切换应用
const handleSwitchApp = (app: App) => {
  emit('switch-app', app)
  workspaceListForceSelect.value = false
  popoverVisible.value = false
  dialogVisible.value = false
}

// 创建应用
const handleCreateApp = () => {
  workspaceListForceSelect.value = false
  dialogVisible.value = false
  popoverVisible.value = false
  emit('create-app')
}

// 更新应用
const handleUpdateApp = (app: App) => {
  emit('update-app', app)
  popoverVisible.value = false
  dialogVisible.value = false
}

// 删除应用
const handleDeleteApp = (app: App) => {
  emit('delete-app', app)
}

// 供父组件（如 WorkspaceView）在进入 /workspace/:user 时打开选择工作空间弹窗
// forceSelect: 为 true 时弹窗不可关闭，必须选择或创建一个工作空间
function openWorkspaceListDialog(forceSelect = false) {
  workspaceListForceSelect.value = forceSelect
  popoverVisible.value = false
  dialogVisible.value = forceSelect
  if (!forceSelect) {
    popoverVisible.value = true
  }
  emit('load-apps')
}

const handleDocumentPointerDown = (event: PointerEvent) => {
  if (!popoverVisible.value) {
    return
  }

  const target = event.target
  if (!(target instanceof Node)) {
    return
  }

  const popoverEl = document.querySelector('.workspace-switcher-popover')
  const messageBoxEl = document.querySelector('.el-message-box')
  const messageBoxOverlayEl = document.querySelector('.el-overlay-message-box')
  if (
    switcherRootRef.value?.contains(target) ||
    popoverEl?.contains(target) ||
    messageBoxEl?.contains(target) ||
    messageBoxOverlayEl?.contains(target)
  ) {
    return
  }

  closeWorkspacePopover()
}

onMounted(() => {
  document.addEventListener('pointerdown', handleDocumentPointerDown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleDocumentPointerDown)
})

// 弹窗关闭后重置强制选择状态，避免下次从侧边栏打开时仍不可关闭
watch(dialogVisible, (val) => {
  if (!val) workspaceListForceSelect.value = false
})

defineExpose({
  openWorkspaceListDialog
})
</script>

<style scoped lang="scss">
.app-switcher {
  flex-shrink: 0;
}

.app-container {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 固定宽度：普通模式 240px，紧凑模式 220px */
.app-switcher .app-container {
  width: 240px;
  min-width: 240px;
}

.app-switcher-compact .app-container {
  width: 220px;
  min-width: 220px;
  gap: 8px;
}

.app-current {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: var(--el-fill-color-blank);
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: var(--box-shadow-sm);
  min-width: 0;

  &:hover {
    background: var(--el-fill-color-light);
    border-color: rgba(var(--el-color-primary-rgb, 69, 88, 200), 0.38);
    box-shadow: 0 10px 22px rgba(15, 23, 42, 0.08);
  }
}

.app-avatar {
  position: relative;
  flex-shrink: 0;
}

.app-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-color-white, white);
  font-weight: 600;
  font-size: 14px;
  box-shadow: 0 6px 16px rgba(15, 23, 42, 0.16);
}

.app-icon--placeholder {
  background: var(--el-fill-color-dark);
  color: var(--el-text-color-secondary);
}

.status-indicator {
  position: absolute;
  bottom: -1px;
  right: -1px;
  width: 10px;
  height: 10px;
  background: var(--el-color-success);
  border: 2px solid var(--el-bg-color);
  border-radius: 50%;
}

.app-info {
  flex: 1;
  min-width: 0;
}

.app-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-path {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  
  .path-icon {
    font-size: 11px;
  }
}

.expand-section {
  flex-shrink: 0;
}

.expand-icon {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  transition: transform 0.2s;
}

.expand-icon-inline {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}

/* 紧凑模式：顶部导航栏 */
.app-switcher-compact .app-current {
  padding: 8px 12px;
  border-radius: 8px;
}

.app-switcher-compact .app-icon {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  font-size: 12px;
}

.app-switcher-compact .app-name {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 0;
}

:global(.workspace-switcher-popover.el-popper) {
  max-width: calc(100vw - 32px) !important;
  padding: 0;
  overflow: hidden;
  border: 1px solid rgba(var(--el-color-primary-rgb), 0.18);
  border-radius: 16px;
  background:
    linear-gradient(180deg, rgba(var(--el-color-primary-rgb), 0.1), transparent 112px),
    var(--app-shell-panel-bg);
  box-shadow: 0 18px 42px rgba(2, 6, 23, 0.28);
  backdrop-filter: blur(18px);
}

:global(.workspace-switcher-popover.el-popper.is-light) {
  border-color: rgba(var(--el-color-primary-rgb), 0.18);
}
</style>
