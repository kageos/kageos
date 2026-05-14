<!--
  WorkspaceView - 工作空间视图
  🔥 统一架构的展示层组件
  
  职责：
  - 纯 UI 展示，不包含业务逻辑
  - 通过事件与 Application Layer 通信
  - 从 StateManager 获取状态并渲染
-->

<template>
  <div class="workspace-container" data-testid="workspace-view">
    <!-- 顶部导航栏：工作空间切换、搜索与用户入口 -->
    <WorkspaceHeader
      ref="workspaceHeaderRef"
      :current-app="currentApp"
      :app-list="appList"
      :loading-apps="loadingApps"
      :service-tree="serviceTree"
      @switch-app="handleSwitchApp"
      @create-app="showCreateAppDialog"
      @update-app="handleUpdateApp"
      @delete-app="handleDeleteApp"
      @load-apps="loadAppList"
    />

    <div class="workspace-view">
      <!-- 左下角：隐藏/显示目录按钮 -->
      <div class="sidebar-toggle-bottom-left">
        <el-button
          link
          @click="toggleLeftSidebar"
          class="sidebar-toggle-btn"
          :title="showLeftSidebar ? '隐藏目录' : '显示目录'"
        >
          <el-icon>
            <ArrowLeft v-if="showLeftSidebar" />
            <ArrowRight v-else />
          </el-icon>
        </el-button>
      </div>

      <!-- 左侧：目录树 -->
      <div class="left-sidebar" :class="{ 'sidebar-collapsed': !showLeftSidebar }">
        <div class="left-sidebar-tree" data-testid="workspace-service-tree">
          <ServiceTreePanel
            ref="serviceTreePanelRef"
            :tree-data="serviceTree"
            :loading="loading"
            :current-node-id="currentFunction?.id || null"
            :current-function="currentFunction"
            :expanded-keys="expandedKeys"
            @node-click="handleNodeClick"
            @create-directory="handleCreateDirectory"
            @create-docs="handleCreateDocs"
            @create-board="handleCreateBoard"
            @delete-doc="handleDeleteDoc"
            @delete-board="handleDeleteBoard"
            @delete-function="handleDeleteFunction"
            @delete-directory="handleDeleteDirectory"
            @bulk-delete="handleBulkDeleteNodes"
            @refresh-tree="handleRefreshTree"
            @update-history="handleUpdateHistory"
          />
        </div>
      </div>

      <!-- 中间函数渲染区域 -->
      <div class="function-renderer" data-testid="workspace-function-renderer">
        <!-- 🔥 Create/Edit 模式：根据 queryTab 显示独立页面 -->
        <template v-if="queryTab === 'create' && currentFunction && currentFunctionDetail">
          <WorkspaceFormPage
            title="新增数据"
            :function-detail="currentFunctionDetail"
            :page-key="`form-create-${currentFunction.id}`"
            submit-text="提交"
            unsupported-message="该函数不支持新增操作"
            @back="backToList"
          />
        </template>
        
        <template v-else-if="queryTab === 'edit' && currentFunction && currentFunctionDetail">
          <WorkspaceFormPage
            title="编辑数据"
            :function-detail="editFunctionDetail || currentFunctionDetail"
            :initial-data="editInitialData"
            :page-key="`form-edit-${currentFunction.id}-${editRowId}`"
            submit-text="保存"
            unsupported-message="该函数不支持编辑操作"
            @back="backToList"
          />
        </template>
        
        <!-- 🔥 Detail 模式：显示详情抽屉（通过 URL 参数打开） -->
        <!-- 注意：detail 模式使用抽屉显示，不需要单独的页面 -->
        
        <!-- 🔥 文档详情页面（可滚动） -->
        <div v-if="currentFunction && currentFunction.type === 'docs'" class="main-content-scroll docs-content-scroll">
          <DocView
            :node="currentFunction"
            @deleted="handleDocDeleted"
          />
        </div>

        <!-- 🔥 版块/讨论区页面（可滚动） -->
        <div v-else-if="currentFunction && currentFunction.type === 'board'" class="main-content-scroll board-content-scroll">
          <BoardView
            :node="currentFunction"
          />
        </div>

        <!-- 🔥 服务目录详情页面（可滚动） -->
        <div v-else-if="currentFunction && currentFunction.type === 'package'" class="main-content-scroll package-content-scroll">
          <PackageDetailView
            :package-node="currentFunction"
            @refresh="handleRefreshTree"
            @open-session="openWorkspaceSession"
          />
        </div>
        
        <!-- 函数详情区域（正常模式 - 函数节点） -->
        <div v-else-if="currentFunction && currentFunction.type === 'function'" class="function-content-wrapper">
          <div class="function-content">
            <WorkspaceFunctionTabsPanel
              v-if="showFunctionTabsWrapper"
              :active-tab="functionActiveTab"
              :current-function="currentFunction"
              :current-function-detail="currentFunctionDetail"
              :show-form-operate-log-tab="showFormOperateLogTab"
              :show-scheduled-task-tab="showScheduledTaskTab"
              :show-scheduled-agent-task-tab="showScheduledAgentTaskTab"
              :function-form-view-ref="setFunctionFormViewRef"
              :form-operate-log-section-ref="setFormOperateLogSectionRef"
              :on-function-tab-change="handleFunctionTabChange"
              :on-apply-form-operate-log="handleApplyFormOperateLog"
              :on-scheduled-task-total-change="onScheduledTaskTotalChange"
              :on-scheduled-agent-task-total-change="onScheduledAgentTaskTotalChange"
              :on-open-workspace-session="openWorkspaceSession"
              :on-open-function-operate-log="openFunctionOperateLog"
              @update:active-tab="functionActiveTab = $event"
            />

            <!-- 没有函数 tabs 时，直接显示内容 -->
            <div v-else>
              <WorkspaceFunctionRenderer
                :current-function="currentFunction"
                :function-detail="currentFunctionDetail"
              />
            </div>
          </div>
        </div>
        <div v-else class="empty-state" data-testid="workspace-empty-state">
          <p>请在左侧选择功能或目录</p>
        </div>
      </div>
    </div>

    <WorkspaceCreateAppDialog
      v-model:visible="createAppDialogVisible"
      :form="createAppForm"
      :creating="creatingApp"
      @submit="submitCreateApp"
      @close="resetCreateAppForm"
    />

    <!-- 详情抽屉 -->
    <TableRowDetailDrawer
      v-model:visible="detailDrawerVisible"
      v-model:mode="detailDrawerMode"
      :title="detailDrawerTitle"
      :fields="detailFields"
      :row-data="detailRowData"
      :table-data="detailTableData"
      :current-index="currentDetailIndex"
      :supports-edit="supportsUpdateTable"
      :can-edit="canUpdateTable"
      :edit-function-detail="editFunctionDetail"
      :current-function-detail="currentFunctionDetail"
      :user-info-map="detailUserInfoMap"
      :submitting="drawerSubmitting"
      :current-function="currentFunction"
      ref="detailDrawerRef"
      @navigate="handleNavigateDetail"
      @submit="(formRendererRef) => submitDrawerEdit(formRendererRef)"
      @close="handleDetailDrawerClose"
    />

    <WorkspaceCreateDocsDialog
      v-model:visible="createDocsDialogVisible"
      :parent-node="currentDocsParentNode"
      :form="createDocsForm"
      :creating="creatingDocs"
      @submit="handleSubmitCreateDocs"
      @close="handleCloseCreateDocsDialog"
    />

    <!-- 创建讨论区（版块）对话框 - 封装组件 -->
    <CreateBoardDialog
      v-if="featureFlags.board && createBoardDialogVisible"
      v-model="createBoardDialogVisible"
      :current-app="currentApp"
      :parent-node="currentBoardParentNode"
      @success="afterCreateBoard"
    />

    <WorkspaceCreateDirectoryDialog
      v-model:visible="createDirectoryDialogVisible"
      :parent-node="currentParentNode"
      :form="createDirectoryForm"
      :creating="creatingDirectory"
      @submit="handleSubmitCreateDirectory"
      @close="handleCloseCreateDirectoryDialog"
    />

    <!-- 变更记录对话框 -->
    <DirectoryUpdateHistoryDialog
      v-if="featureFlags.operateLogs && updateHistoryDialogVisible"
      v-model="updateHistoryDialogVisible"
      :mode="updateHistoryMode"
      :app-id="updateHistoryAppId"
      :app-version="updateHistoryAppVersion"
      :full-code-path="updateHistoryFullCodePath"
    />

    <!-- 多个 Mini 浮动工作台 -->
    <button
      v-if="showMiniWorkstationLauncher"
      type="button"
      class="mini-workstation-launcher"
      :title="`打开工作台 (${MINI_WORKSTATION_TOGGLE_SHORTCUT_LABEL}) - ${miniWorkstationLauncherName}`"
      data-testid="mini-workstation-launcher"
      @click="openCurrentWorkstation"
    >
      <span class="mini-workstation-launcher-pulse"></span>
      <strong>工作台</strong>
      <span>{{ miniWorkstationLauncherSummary }}</span>
      <kbd class="mini-workstation-launcher-shortcut">{{ MINI_WORKSTATION_TOGGLE_SHORTCUT_LABEL }}</kbd>
      <span v-if="miniWorkstationLauncherCount > 1" class="mini-workstation-launcher-badge">
        {{ miniWorkstationLauncherCount }}
      </span>
    </button>

    <MiniWorkstation
      v-for="mini in miniWsList"
      :key="mini.id"
      :visible="mini.visible"
      :full-code-path="mini.fullCodePath"
      :dir-name="mini.dirName"
      :initial-session-id="mini.initialSessionId"
      :initial-offset="mini.offset"
      :initial-position="mini.initialPosition"
      :initial-expanded="mini.initialExpanded"
      :initial-maximized="mini.initialMaximized"
      :path-name-map="workspacePathNameMap"
      :toggle-shortcut-label="MINI_WORKSTATION_TOGGLE_SHORTCUT_LABEL"
      @minimize="handleMiniMinimize(mini.id)"
      @close="handleMiniRemove(mini.id)"
      @expanded-change="(payload) => handleMiniExpandedChange(mini.id, payload)"
      @maximize-change="(payload) => handleMiniMaximizeChange(mini.id, payload)"
      @task-started="(sessionId) => handleMiniTaskStarted(mini.id, sessionId)"
      @tool-call-ok="handleWorkstationToolCallOk"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, ArrowRight } from '@element-plus/icons-vue'
import { serviceFactory } from '../../infrastructure/factories'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import ServiceTreePanel from '@/architecture/presentation/components/ServiceTreePanel.vue'
import WorkspaceHeader from '../components/WorkspaceHeader.vue'
import TableRowDetailDrawer from '../components/TableRowDetailDrawer.vue'
import WorkspaceCreateAppDialog from '../components/WorkspaceCreateAppDialog.vue'
import WorkspaceFormPage from '../components/WorkspaceFormPage.vue'
import WorkspaceCreateDirectoryDialog from '../components/WorkspaceCreateDirectoryDialog.vue'
import WorkspaceCreateDocsDialog from '../components/WorkspaceCreateDocsDialog.vue'
import WorkspaceFunctionRenderer from '../components/WorkspaceFunctionRenderer.vue'
import WorkspaceFunctionTabsPanel from '../components/WorkspaceFunctionTabsPanel.vue'
import type { App } from '../../domain/types'
import type { FieldConfig, FunctionDetail } from '@/architecture/domain/types'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/architecture/domain/types'
// 🔥 导入 Composable
import { useWorkspaceRouting } from '../composables/useWorkspaceRouting'
import { useWorkspaceDetail } from '../composables/useWorkspaceDetail'
import { useWorkspaceApp } from '../composables/useWorkspaceApp'
import { useWorkspaceServiceTree } from '../composables/useWorkspaceServiceTree'
import { useWorkspaceFunctionTabs } from '../composables/useWorkspaceFunctionTabs'
import { useWorkspaceMiniWorkstations } from '../composables/useWorkspaceMiniWorkstations'
import { useWorkspaceNodeActions } from '../composables/useWorkspaceNodeActions'
import { useWorkspaceNodeNavigation } from '../composables/useWorkspaceNodeNavigation'
import { useWorkspaceNodeToolActions } from '../composables/useWorkspaceNodeToolActions'
import { useWorkspaceUiEffects } from '../composables/useWorkspaceUiEffects'
import { useWorkspaceViewLifecycle } from '../composables/useWorkspaceViewLifecycle'
import { findNodeByPath, findNodeById } from '../utils/workspaceUtils'
import { useAfterCreateNode } from '../composables/useAfterCreateNode'
import { getFormRequestFields, getFunctionCallbacks } from '@/architecture/runtime/utils/functionSchemaSelectors'
import type { WorkspaceSessionItem } from '@/architecture/infrastructure/api/workspace'
import { featureFlags } from '@/architecture/infrastructure/config/features'

const route = useRoute()
const router = useRouter()
const isAppleShortcutPlatform = typeof navigator !== 'undefined'
  && /Mac|iPhone|iPad|iPod/i.test(navigator.platform)
const MINI_WORKSTATION_TOGGLE_SHORTCUT_LABEL = isAppleShortcutPlatform ? '⌘.' : 'Ctrl+.'
const DocView = defineAsyncComponent(() => import('../components/DocView.vue'))
const BoardView = defineAsyncComponent(() => import('../components/BoardView.vue'))
const PackageDetailView = defineAsyncComponent(() => import('../components/PackageDetailView.vue'))
const MiniWorkstation = defineAsyncComponent(() => import('../components/MiniWorkstation.vue'))
const CreateBoardDialog = defineAsyncComponent(() => import('../components/CreateBoardDialog.vue'))
const DirectoryUpdateHistoryDialog = defineAsyncComponent(() => import('@/architecture/presentation/shared/components/DirectoryUpdateHistoryDialog.vue'))

// 依赖注入（使用 IServiceProvider 接口，遵循依赖倒置原则）
const serviceProvider: IServiceProvider = serviceFactory
const applicationService = serviceProvider.getWorkspaceApplicationService()
const domainService = serviceProvider.getWorkspaceDomainService()

// 从状态管理器获取状态
const serviceTree = computed(() => domainService.getServiceTree())
const currentFunction = computed(() => domainService.getCurrentFunction())
const currentAppFromState = computed(() => domainService.getCurrentApp())
const workspacePathNameMap = computed<Record<string, string>>(() => {
  const map: Record<string, string> = {}
  fillWorkspacePathNameMap(serviceTree.value, map)
  return map
})

// ⭐ 需要自动展开的节点ID列表（从后端返回）
const expandedKeys = ref<number[]>([])

// 🔥 不再使用 Tab 功能，简化系统

function normalizeApp(app: Partial<AppType> & Pick<AppType, 'id' | 'user' | 'code' | 'name'>): AppType {
  return {
    id: app.id,
    user: app.user,
    code: app.code,
    name: app.name,
    nats_id: app.nats_id ?? 0,
    host_id: app.host_id ?? 0,
    status: app.status ?? 'enabled',
    type: app.type,
    version: app.version ?? '',
    is_public: app.is_public ?? false,
    admins: app.admins ?? '',
    show_only_permitted: app.show_only_permitted,
    permission_enforced: app.permission_enforced,
    created_at: app.created_at ?? '',
    updated_at: app.updated_at ?? ''
  }
}

function getCurrentAppForTreeLoad(): App | null {
  return currentApp.value ? normalizeApp(currentApp.value) : null
}

function normalizeFullCodePath(fullCodePath: string) {
  return (fullCodePath || '').trim().replace(/\/+$/g, '')
}

function setWorkspacePathName(map: Record<string, string>, fullCodePath: string, label: string) {
  const normalizedPath = normalizeFullCodePath(fullCodePath)
  if (!normalizedPath || !label) return
  map[normalizedPath] = label
  map[normalizedPath.replace(/^\/+/, '')] = label
}

function fillWorkspacePathNameMap(
  nodes: ServiceTreeType[],
  map: Record<string, string>
) {
  for (const node of nodes) {
    const fallbackName = normalizeFullCodePath(node.full_code_path).split('/').filter(Boolean).pop() || node.code || '工作台'
    const nodeName = node.name || fallbackName
    setWorkspacePathName(map, node.full_code_path, nodeName)
    if (node.children?.length) {
      fillWorkspacePathNameMap(node.children, map)
    }
  }
}

function resolveWorkspacePathName(fullCodePath: string) {
  const normalizedPath = normalizeFullCodePath(fullCodePath)
  return workspacePathNameMap.value[normalizedPath]
    || workspacePathNameMap.value[normalizedPath.replace(/^\/+/, '')]
}

const currentApp = computed<AppType | null>(() => {
  const app = currentAppFromState.value
  if (!app) return null
  // 从 appList 中查找对应的应用（确保使用最新的应用数据）
  const foundApp = appList.value.find((a: AppType) => a.id === app.id || (a.user === app.user && a.code === app.code))
  return foundApp ? normalizeApp(foundApp) : normalizeApp(app)
})

const {
  appList,
  loadingApps,
  createAppDialogVisible,
  creatingApp,
  createAppForm,
  loadAppList,
  handleSwitchApp: appHandleSwitchApp,
  showCreateAppDialog,
  resetCreateAppForm,
  submitCreateApp: appSubmitCreateApp,
  handleUpdateApp,
  handleDeleteApp: appHandleDeleteApp
} = useWorkspaceApp()

const {
  createDirectoryDialogVisible,
  creatingDirectory,
  currentParentNode,
  createDirectoryForm,
  handleCreateDirectory: serviceTreeHandleCreateDirectory,
  resetCreateDirectoryForm,
  handleSubmitCreateDirectory: serviceTreeHandleSubmitCreateDirectory,
  expandCurrentRoutePath: serviceTreeExpandCurrentRoutePath,
} = useWorkspaceServiceTree()

// 🔥 移除缓存后，通过事件获取函数详情
const currentFunctionDetail = ref<FunctionDetail | null>(null)

const {
  buildWorkspacePath,
  handleNodeClick,
  backToList
} = useWorkspaceNodeNavigation({
  route,
  currentFunction: () => currentFunction.value,
  triggerNodeClick: (node) => applicationService.triggerNodeClick(node)
})

const {
  detailDrawerVisible,
  detailDrawerTitle,
  detailRowData,
  detailFields,
  detailDrawerMode,
  drawerSubmitting,
  detailUserInfoMap,
  detailTableData,
  currentDetailIndex,
  editFunctionDetail,
  handleNavigateDetail,
  submitDrawerEdit,
  handleDetailDrawerClose,
  openDetailDrawer,
  setupUrlWatch
} = useWorkspaceDetail({
  currentFunctionDetail: () => currentFunctionDetail.value,
  currentFunction: () => currentFunction.value
})

const {
  loadAppFromRoute: routingLoadAppFromRoute,
  setupRouteWatch
} = useWorkspaceRouting({
  serviceTree: () => serviceTree.value,
  currentApp: () => currentApp.value,
  appList: () => appList.value,
  loadAppList,
  findNodeByPath,
  expandCurrentRoutePath: () => serviceTreeExpandCurrentRoutePath(
    () => serviceTree.value,
    () => serviceTreePanelRef.value,
    () => currentApp.value
  )
})

// 🔥 Tab 点击处理已移除（直接使用 v-model，避免双重触发）
// const handleTabClick = tabsHandleTabClick


// 🔥 queryTab：当前激活的Tab模式（用于路由查询参数，控制 create/edit 等模式）
const queryTab = computed(() => (route.query._tab as string) || 'run')

// 🔥 编辑模式相关
const editRowId = computed(() => {
  const id = route.query.id || route.query._id
  return id ? Number(id) : null
})

// 🔥 编辑模式的初始数据（从 URL 参数提取）
const editInitialData = computed(() => {
  const initialData: Record<string, unknown> = {}
  const query = route.query
  
  // 如果有 id 参数，添加到 initialData
  const requestFields = getFormRequestFields(editFunctionDetail.value || currentFunctionDetail.value)
  if (editRowId.value) {
    const idField = requestFields.find((f: FieldConfig) => 
      f.code.toLowerCase() === 'id' || f.widget?.type === 'number'
    )
    if (idField) {
      initialData[idField.code] = editRowId.value
    }
  }
  
  // 遍历所有查询参数，如果字段在 request 中，添加到 initialData
  if (requestFields.length > 0) {
    requestFields.forEach((field: FieldConfig) => {
      const fieldCode = field.code
      // 跳过 _ 开头的参数（系统参数）
      if (fieldCode.startsWith('_')) return
      
      const value = query[fieldCode]
      if (value !== undefined && value !== null && value !== '') {
        // 🔥 类型转换：根据字段类型转换值
        if (field.data?.type === 'int' || field.data?.type === 'integer') {
          const intValue = parseInt(String(value), 10)
          if (!isNaN(intValue)) {
            initialData[fieldCode] = intValue
          }
        } else if (field.data?.type === 'float' || field.data?.type === 'number') {
          const floatValue = parseFloat(String(value))
          if (!isNaN(floatValue)) {
            initialData[fieldCode] = floatValue
          }
        } else if (field.data?.type === 'bool' || field.data?.type === 'boolean') {
          const strValue = String(value)
          const numValue = typeof value === 'number' ? value : Number(strValue)
          const boolValue = typeof value === 'boolean' ? value : false
          initialData[fieldCode] = strValue === 'true' || strValue === '1' || numValue === 1 || boolValue
        } else {
          initialData[fieldCode] = value
        }
      }
    })
  }
  
  return initialData
})


// ServiceTreePanel 引用（用于展开路径）
const serviceTreePanelRef = ref<InstanceType<typeof ServiceTreePanel> | null>(null)
const workspaceHeaderRef = ref<InstanceType<typeof WorkspaceHeader> | null>(null)

// 左侧服务目录树显示状态
const showLeftSidebar = ref(true)

const {
  functionActiveTab,
  setFunctionFormViewRef,
  setFormOperateLogSectionRef,
  showScheduledTaskTab,
  showScheduledAgentTaskTab,
  showFormOperateLogTab,
  showFunctionTabsWrapper,
  handleFunctionTabChange,
  handleApplyFormOperateLog,
  openFunctionOperateLog,
  onScheduledTaskTotalChange,
  onScheduledAgentTaskTotalChange,
  activateScheduledTaskTab
} = useWorkspaceFunctionTabs({
  route,
  router,
  currentFunction,
  currentFunctionDetail
})

useWorkspaceViewLifecycle({
  route,
  router,
  currentFunction: () => currentFunction.value,
  currentFunctionDetail: () => currentFunctionDetail.value,
  setCurrentFunctionDetail: (detail) => {
    currentFunctionDetail.value = detail
  },
  expandedKeys,
  currentApp: () => currentApp.value,
  serviceTree: () => serviceTree.value,
  loadAppFromRoute: routingLoadAppFromRoute,
  setupRouteWatch,
  activateScheduledTaskTab,
  expandCurrentRoutePath: () => serviceTreeExpandCurrentRoutePath(
    () => serviceTree.value,
    () => serviceTreePanelRef.value,
    () => currentApp.value
  ),
  queryTab: () => queryTab.value,
  loadNodeDetail: (node) => applicationService.handleNodeClick(node),
  updateAppInfo: (app) => {
    const index = appList.value.findIndex((item: AppType) => item.code === app.code)
    if (index !== -1) {
      appList.value[index] = { ...appList.value[index], ...app }
    }
  },
  findNodeByPath,
  openWorkspaceListDialog: () => workspaceHeaderRef.value?.openWorkspaceListDialog(true)
})

// 切换左侧边栏显示
const toggleLeftSidebar = () => {
  showLeftSidebar.value = !showLeftSidebar.value
  // 保存到 localStorage 持久化
  localStorage.setItem('workspace-left-sidebar', String(showLeftSidebar.value))
}

/** 工作台上下文：点击什么节点就用什么节点的 full_code_path */
const workstationContext = computed(() => {
  const node = currentFunction.value
  if (!node?.full_code_path) return null
  const path = (node.full_code_path || '').replace(/\/+$/g, '')
  if (!path) return null
  const name = node.name || path.split('/').pop() || '工作台'
  return { fullCodePath: path, dirName: name }
})

const {
  miniWsList,
  openAmbientMiniWs,
  openNewMiniWs,
  handleMiniMinimize,
  hideVisibleMiniWs,
  handleMiniRemove,
  handleMiniMaximizeChange,
  handleMiniExpandedChange,
  handleMiniTaskStarted,
  handleWorkspaceOpenWorkstation,
  initializeFromRoute: initializeMiniWorkstationsFromRoute,
} = useWorkspaceMiniWorkstations({
  route,
  router,
  workstationContext,
  buildWorkspacePath: (fullCodePath: string) => buildWorkspacePath(fullCodePath),
  resolvePathName: resolveWorkspacePathName,
})

const showMiniWorkstationLauncher = computed(() => {
  return !!workstationContext.value?.fullCodePath && !miniWsList.value.some(mini => mini.visible)
})

const miniWorkstationLauncherName = computed(() => workstationContext.value?.dirName || '当前目录')

const miniWorkstationLauncherCount = computed(() => Math.max(miniWsList.value.length, 1))

const miniWorkstationLauncherSummary = computed(() => `${miniWorkstationLauncherCount.value} 个会话`)

function openCurrentWorkstation() {
  const ctx = workstationContext.value
  if (!ctx?.fullCodePath) return

  handleWorkspaceOpenWorkstation({
    full_code_path: ctx.fullCodePath,
    directory_name: ctx.dirName,
    open_as_mini: true
  })
}

function hideVisibleWorkstation() {
  return hideVisibleMiniWs()
}

function toggleCurrentWorkstationByShortcut() {
  if (hideVisibleWorkstation()) return
  openCurrentWorkstation()
}

function isMiniWorkstationToggleShortcut(event: KeyboardEvent) {
  const isPeriodKey = event.key === '.' || event.code === 'Period'
  const hasPlatformModifier = isAppleShortcutPlatform
    ? event.metaKey && !event.ctrlKey
    : event.ctrlKey && !event.metaKey
  return isPeriodKey
    && hasPlatformModifier
    && !event.altKey
    && !event.shiftKey
}

function isEventFromMiniWorkstation(event: KeyboardEvent) {
  const target = event.target instanceof Element
    ? event.target
    : document.activeElement instanceof Element ? document.activeElement : null
  return !!target?.closest('[data-testid="mini-workstation"]')
}

function handleWorkspaceShortcutKeydown(event: KeyboardEvent) {
  if (event.defaultPrevented) return

  if (isMiniWorkstationToggleShortcut(event)) {
    const canToggle = miniWsList.value.some(mini => mini.visible) || !!workstationContext.value?.fullCodePath
    if (!canToggle) return
    event.preventDefault()
    event.stopPropagation()
    toggleCurrentWorkstationByShortcut()
    return
  }

  if (event.key === 'Escape' && isEventFromMiniWorkstation(event)) {
    const hidden = hideVisibleWorkstation()
    if (!hidden) return
    event.preventDefault()
    event.stopPropagation()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleWorkspaceShortcutKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleWorkspaceShortcutKeydown)
})

watch(
  workstationContext,
  (ctx) => {
    if (!ctx?.fullCodePath) return
    openAmbientMiniWs(ctx.fullCodePath, ctx.dirName)
  },
  { immediate: true }
)

function openWorkspaceSession(session: WorkspaceSessionItem) {
  const sessionID = (session.session_id || '').trim()
  if (!sessionID) return

  const fullCodePath = (session.full_code_path || workstationContext.value?.fullCodePath || '').trim()
  if (!fullCodePath) {
    openNewMiniWs(sessionID)
    return
  }

  handleWorkspaceOpenWorkstation({
    full_code_path: fullCodePath,
    session_id: sessionID,
    directory_name: session.directory_name,
    open_as_mini: true
  })
}

const canUpdateTable = computed(() => {
  return true
})

const supportsUpdateTable = computed(() => {
  return getFunctionCallbacks(currentFunctionDetail.value).includes('OnTableUpdateRow')
})

// 转换 loadingTree 为 boolean (避免 computed 类型问题)
const loading = computed(() => domainService.isLoading())
// 🔥 处理创建目录（使用 Composable）
const handleCreateDirectory = (parentNode?: ServiceTreeType) => {
  serviceTreeHandleCreateDirectory(parentNode || null, () => currentApp.value)
}

const handleSubmitCreateDirectory = async () => {
  await serviceTreeHandleSubmitCreateDirectory(() => currentApp.value)
}

// 处理关闭创建目录对话框
const handleCloseCreateDirectoryDialog = () => {
  resetCreateDirectoryForm(() => currentApp.value)
}


// 处理刷新服务树（复制粘贴后需要刷新）
const handleRefreshTree = async () => {
  const app = getCurrentAppForTreeLoad()
  if (app) {
    await domainService.loadServiceTree(app)
  }
}

// 创建节点后的统一处理：刷新树 + 选中新节点（文档/讨论区复用，需在 handleRefreshTree 之后定义）
const afterCreateNode = useAfterCreateNode({
  handleRefreshTree,
  serviceTree: () => serviceTree.value,
  findNodeById,
  handleNodeClick
})
const afterCreateBoard = afterCreateNode

const {
  createDocsDialogVisible,
  creatingDocs,
  currentDocsParentNode,
  createDocsForm,
  createBoardDialogVisible,
  currentBoardParentNode,
  handleCreateDocs,
  handleSubmitCreateDocs,
  handleCloseCreateDocsDialog,
  handleCreateBoard,
  handleDeleteBoard,
  handleDeleteDoc,
  handleDocDeleted,
  handleDeleteDirectory,
  handleDeleteFunction,
  handleBulkDeleteNodes
} = useWorkspaceNodeActions({
  route,
  router,
  currentApp,
  currentFunction,
  domainService,
  handleRefreshTree,
  afterCreateNode
})

const {
  updateHistoryDialogVisible,
  updateHistoryMode,
  updateHistoryAppId,
  updateHistoryAppVersion,
  updateHistoryFullCodePath,
  handleUpdateHistory
} = useWorkspaceNodeToolActions({
  currentApp
})

// 会改变服务目录结构的工具名（创建目录、写文档、写代码、编译工作空间）
const TREE_AFFECTING_TOOLS = ['create_directory', 'write_doc', 'write_go_file', 'build_workspace']

// 工作台工具调用成功时：若为改树工具则刷新服务树
const handleWorkstationToolCallOk = (payload: { name: string }) => {
  if (payload?.name && TREE_AFFECTING_TOOLS.includes(payload.name)) {
    handleRefreshTree()
  }
}
// 🔥 切换应用（使用 Composable）
const handleSwitchApp = async (app: AppType): Promise<void> => {
  await appHandleSwitchApp(app, () => currentApp.value)
}

// 🔥 提交创建应用（使用 Composable）
const submitCreateApp = async (): Promise<void> => {
  await appSubmitCreateApp(() => currentApp.value)
}

// 🔥 删除应用（使用 Composable）
const handleDeleteApp = async (app: AppType): Promise<void> => {
  await appHandleDeleteApp(app, () => currentApp.value)
}

useWorkspaceUiEffects({
  showLeftSidebar,
  openDetailDrawer,
  setupUrlWatch,
  handleWorkspaceOpenWorkstation,
  initializeMiniWorkstationsFromRoute
})

</script>

<style scoped lang="scss">
.hidden {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
  pointer-events: none;
}

.workspace-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  padding: 16px 18px 18px;
  gap: 18px;
  box-sizing: border-box;
  background: var(--app-shell-bg);
  background-attachment: fixed;
}

.workspace-view {
  display: flex;
  flex: 1;
  overflow: hidden; /* 防止双滚动条 */
  position: relative;
  min-height: 0;
  gap: 18px;
}

.function-content-wrapper {
  flex: 1;
  overflow: hidden; /* 🔥 外层容器隐藏溢出，内层处理滚动 */
  display: flex;
  flex-direction: column;
  min-height: 0; /* 🔥 关键：允许 flex 子元素缩小 */
}

.function-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0; /* 🔥 关键：允许 flex 子元素缩小 */
  overflow: hidden; /* 🔥 外层容器隐藏溢出，内层处理滚动 */
  
  // 当有 tab 结构时，需要特殊处理
  .function-tabs-wrapper {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }
  
  // 当没有 tab 结构时，直接显示内容（允许滚动）
  > div:not(.function-tabs-wrapper) {
    flex: 1;
    min-height: 0;
    overflow-y: auto !important;
    overflow-x: hidden;
    -webkit-overflow-scrolling: touch;
    height: 0; /* 🔥 关键：配合 flex: 1 和 min-height: 0，让滚动容器正确计算高度 */
  }
}

/* 保留旧的类名以兼容（如果还有地方使用） */
.tabs-content-wrapper {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* 左下角：隐藏/显示目录按钮 */
.sidebar-toggle-bottom-left {
  position: absolute;
  bottom: 18px;
  left: 18px;
  z-index: 100;
}

  .sidebar-toggle-bottom-left .sidebar-toggle-btn {
    width: 40px;
    height: 40px;
    min-width: 40px;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    background: var(--app-shell-panel-bg-strong);
    border: 1px solid var(--app-shell-panel-border);
    border-radius: 16px;
    box-shadow: var(--app-shell-panel-shadow-soft);
    transition: all 0.2s;

  &:hover {
    color: var(--el-color-primary);
    border-color: var(--el-color-primary);
    background: var(--app-shell-panel-bg);
    box-shadow: var(--app-shell-panel-shadow);
  }

  .el-icon {
    font-size: 16px;
  }
}

.mini-workstation-launcher {
  position: fixed;
  left: 50%;
  bottom: 28px;
  z-index: 2400;
  height: 46px;
  max-width: min(360px, calc(100vw - 160px));
  display: inline-flex;
  align-items: center;
  gap: 9px;
  padding: 0 15px;
  transform: translateX(-50%);
  border: 1px solid rgba(130, 153, 190, 0.3);
  border-radius: 999px;
  background: rgba(10, 16, 29, 0.8);
  box-shadow:
    0 24px 70px rgba(0, 0, 0, 0.38),
    0 0 0 1px rgba(255, 255, 255, 0.06);
  color: #dce9fb;
  backdrop-filter: blur(24px) saturate(1.12);
  cursor: pointer;
  transition: transform 0.16s ease, border-color 0.16s ease, background 0.16s ease, box-shadow 0.16s ease, color 0.16s ease;
}

.mini-workstation-launcher:hover {
  transform: translateX(-50%) translateY(-2px);
  border-color: rgba(83, 174, 255, 0.5);
  background: rgba(10, 16, 29, 0.88);
  box-shadow:
    0 24px 70px rgba(0, 0, 0, 0.42),
    0 0 0 1px rgba(83, 174, 255, 0.16),
    0 0 26px rgba(83, 174, 255, 0.2);
  color: #ffffff;
}

.mini-workstation-launcher-pulse {
  width: 10px;
  height: 10px;
  flex: 0 0 auto;
  border-radius: 50%;
  background: #2bd59f;
  box-shadow: 0 0 18px rgba(43, 213, 159, 0.8);
}

.mini-workstation-launcher strong,
.mini-workstation-launcher span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-workstation-launcher strong {
  font-size: 13px;
  font-weight: 900;
}

.mini-workstation-launcher span {
  color: rgba(220, 235, 255, 0.72);
  font-size: 12px;
  font-weight: 720;
}

.mini-workstation-launcher-shortcut {
  flex: 0 0 auto;
  min-width: 34px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 7px;
  border: 1px solid rgba(130, 153, 190, 0.3);
  border-radius: 7px;
  background: rgba(30, 42, 68, 0.72);
  color: #e8f2ff;
  font-family: inherit;
  font-size: 12px;
  font-weight: 850;
}

.mini-workstation-launcher-badge {
  min-width: 18px;
  height: 18px;
  display: inline-grid;
  place-items: center;
  padding: 0 5px;
  border-radius: 999px;
  background: rgba(255, 109, 126, 0.9);
  color: #fff !important;
  font-size: 11px !important;
  font-weight: 900 !important;
}

@media (max-width: 720px) {
  .mini-workstation-launcher {
    bottom: 18px;
    max-width: calc(100vw - 32px);
  }
}

  .left-sidebar {
    width: 300px;
    min-width: 300px;
    transition: all 0.3s ease;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--app-shell-panel-border);
    border-radius: 22px;
    background: var(--app-shell-panel-bg);
    box-shadow: var(--app-shell-panel-shadow);

  &.sidebar-collapsed {
    width: 0;
    min-width: 0;
    overflow: hidden;
    border: none;
    background: transparent;
    box-shadow: none;
    margin-right: -18px;
  }
}

.left-sidebar-tree {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
}

.function-renderer {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
  position: relative;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 24px;
  background: var(--app-shell-panel-bg);
  box-shadow: var(--app-shell-panel-shadow);
}

.function-renderer::before {
  content: '';
  position: absolute;
  top: 0;
  left: 28px;
  right: 28px;
  height: 1px;
  background: var(--app-shell-panel-highlight);
  opacity: 0.7;
  pointer-events: none;
  z-index: 1;
}

/* 讨论区/文档/目录主内容区：可滚动；右侧留白避免被「板块说明」按钮挡住 */
.main-content-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  -webkit-overflow-scrolling: touch;
}

.board-content-scroll {
  padding-right: 130px; /* 为右上角「板块说明」按钮留出空间，避免挡住发帖等操作 */
}

.ai-chat-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

/* 函数加载骨架屏样式 */
.function-loading {
  padding: 24px;
  width: 100%;
}

.left-sidebar :deep(.service-tree-panel) {
  background: transparent;
}

.left-sidebar :deep(.tree-header) {
  padding: 16px 16px 12px;
  border-bottom-color: var(--app-shell-panel-border);
  background: transparent;
}

.left-sidebar :deep(.tree-search-input .el-input__wrapper) {
  min-height: 42px;
  background: var(--app-shell-panel-muted-bg) !important;
  border: 1px solid var(--app-shell-panel-border) !important;
  border-radius: 14px !important;
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight) !important;
}

.left-sidebar :deep(.tree-content) {
  padding: 12px;
}

.left-sidebar :deep(.el-tree),
.left-sidebar :deep(.el-tree-node),
.left-sidebar :deep(.el-tree-node__content) {
  background: transparent;
}

.left-sidebar :deep(.el-tree-node__content) {
  height: 36px;
  margin-bottom: 2px;
  border-radius: 12px;
  transition: background-color 0.2s ease, transform 0.2s ease;
}

.left-sidebar :deep(.el-tree-node__content:hover) {
  background-color: rgba(var(--el-color-primary-rgb), 0.06);
}

.left-sidebar :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background-color: rgba(var(--el-color-primary-rgb), 0.12) !important;
  border-left: none;
  box-shadow: inset 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.14);
}

.function-renderer :deep(.package-detail-view) {
  background: transparent;
}

.function-renderer :deep(.package-detail-view .hero-section) {
  background: transparent;
  border-bottom-color: var(--app-shell-panel-border);
  padding: 30px 34px 20px;
}

.function-renderer :deep(.package-detail-view .hero-content) {
  max-width: none;
}

.function-renderer :deep(.package-detail-view .back-button) {
  background: var(--app-shell-panel-muted-bg);
  border: 1px solid var(--app-shell-panel-border);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
}

.function-renderer :deep(.package-detail-view .hero-description) {
  background: var(--app-shell-panel-muted-bg);
  border-left-color: rgba(var(--el-color-primary-rgb), 0.78);
}

.function-renderer :deep(.package-detail-view .detail-content) {
  padding: 26px 34px 34px;
}

.function-renderer :deep(.package-detail-view .overview-item),
.function-renderer :deep(.package-detail-view .child-card) {
  border-color: var(--app-shell-panel-border);
  box-shadow: 0 16px 32px rgba(15, 23, 42, 0.07);
}

.function-renderer :deep(.package-detail-view .overview-item:hover),
.function-renderer :deep(.package-detail-view .child-card:hover) {
  box-shadow: 0 20px 38px rgba(15, 23, 42, 0.11);
}

.function-renderer :deep(.package-detail-view .detail-tabs .el-tabs__nav-wrap::after) {
  display: none;
  height: 0;
  background-color: transparent;
}

.function-renderer :deep(.package-detail-view .detail-tabs .el-tabs__item.is-active) {
  font-weight: 600;
}
</style>
