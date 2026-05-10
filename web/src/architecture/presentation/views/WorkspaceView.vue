<!--
  WorkspaceView - 工作空间视图
  🔥 新架构的展示层组件
  
  职责：
  - 纯 UI 展示，不包含业务逻辑
  - 通过事件与 Application Layer 通信
  - 从 StateManager 获取状态并渲染
-->

<template>
  <div class="workspace-container" data-testid="workspace-view">
    <!-- 顶部导航栏：工作空间切换 + 应用中心 同一行 -->
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
            @import-go-files="handleImportGoFiles"
            @publish-to-hub="handlePublishToHub"
            @push-to-hub="handlePushToHub"
            @pull-from-hub="openPullFromHubDialog"
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
              :has-permission-error="hasPermissionError"
              :show-function-permission-tabs="showFunctionPermissionTabs"
              :show-form-operate-log-tab="showFormOperateLogTab"
              :show-scheduled-task-tab="showScheduledTaskTab"
              :show-scheduled-agent-task-tab="showScheduledAgentTaskTab"
              :function-form-view-ref="setFunctionFormViewRef"
              :function-permission-request-list-ref="setFunctionPermissionRequestListRef"
              :function-permission-manage-list-ref="setFunctionPermissionManageListRef"
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
                :has-permission-error="hasPermissionError"
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
      :admins-field="createAppAdminsField"
      :admins-field-value="createAppAdminsFieldValue"
      @update-admins="handleCreateAppAdminsChange"
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
      v-if="createBoardDialogVisible"
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
      :admins-field="adminsField"
      :admins-field-value="adminsFieldValue"
      @update-admins="handleAdminsChange"
      @submit="handleSubmitCreateDirectory"
      @close="handleCloseCreateDirectoryDialog"
    />

    <!-- 发布到应用中心对话框 -->
    <PublishToHubDialog
      v-if="publishToHubDialogVisible"
      v-model="publishToHubDialogVisible"
      :selected-node="publishSelectedNode"
      :current-app="currentApp || undefined"
      @success="handlePublishSuccess"
    />
    <PushToHubDialog
      v-if="pushToHubDialogVisible"
      v-model="pushToHubDialogVisible"
      :selected-node="pushSelectedNode"
      :current-app="currentApp || undefined"
      @success="handlePushSuccess"
    />
    <PullFromHubDialog
      v-if="pullFromHubDialogVisible"
      v-model="pullFromHubDialogVisible"
      :current-app="currentApp || undefined"
      :initial-hub-link="pastedHubLink"
      :initial-target-path="pullFromHubTargetPath"
      :initial-target-name="pullFromHubTargetName"
      @success="handlePullSuccess"
    />

    <!-- 变更记录对话框 -->
    <DirectoryUpdateHistoryDialog
      v-if="updateHistoryDialogVisible"
      v-model="updateHistoryDialogVisible"
      :mode="updateHistoryMode"
      :app-id="updateHistoryAppId"
      :app-version="updateHistoryAppVersion"
      :full-code-path="updateHistoryFullCodePath"
    />

    <!-- 导入代码文件：隐藏的 file input，选中的 .go 会写入当前目录 -->
    <input
      ref="importGoFileInputRef"
      type="file"
      accept=".go"
      multiple
      class="hidden"
      @change="onImportGoFilesSelected"
    />


    <!-- 多个 Mini 浮动工作台 -->
    <MiniWorkstation
      v-for="mini in miniWsList"
      :key="mini.id"
      :visible="mini.visible"
      :full-code-path="mini.fullCodePath"
      :dir-name="mini.dirName"
      :initial-session-id="mini.initialSessionId"
      :initial-offset="mini.offset"
      :initial-position="mini.initialPosition"
      :initial-maximized="mini.initialMaximized"
      @minimize="handleMiniMinimize(mini.id)"
      @close="handleMiniRemove(mini.id)"
      @maximize-change="handleMiniMaximizeChange"
      @tool-call-ok="handleWorkstationToolCallOk"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref, watch } from 'vue'
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
import type { App } from '../../domain/services/WorkspaceDomainService'
import type { FieldConfig, FieldValue, FunctionDetail } from '@/architecture/domain/types'
import { WidgetType } from '@/core/constants/widget'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/types'
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
import { hasPermission, TablePermission } from '@/utils/permission'
import { usePermissionErrorStore } from '@/stores/permissionError'
import { createStringFieldValue, createWidgetFieldConfig, extractStringFieldRaw } from '@/utils/widgetFieldHelpers'
import { getFormRequestFields, getFunctionCallbacks } from '@/utils/functionSchemaSelectors'
import type { WorkspaceSessionItem } from '@/api/workspace'

const route = useRoute()
const router = useRouter()
const DocView = defineAsyncComponent(() => import('../components/DocView.vue'))
const BoardView = defineAsyncComponent(() => import('../components/BoardView.vue'))
const PackageDetailView = defineAsyncComponent(() => import('../components/PackageDetailView.vue'))
const MiniWorkstation = defineAsyncComponent(() => import('../components/MiniWorkstation.vue'))
const CreateBoardDialog = defineAsyncComponent(() => import('../components/CreateBoardDialog.vue'))
const PublishToHubDialog = defineAsyncComponent(() => import('@/shared/components/PublishToHubDialog.vue'))
const PushToHubDialog = defineAsyncComponent(() => import('@/shared/components/PushToHubDialog.vue'))
const PullFromHubDialog = defineAsyncComponent(() => import('@/shared/components/PullFromHubDialog.vue'))
const DirectoryUpdateHistoryDialog = defineAsyncComponent(() => import('@/shared/components/DirectoryUpdateHistoryDialog.vue'))

// 依赖注入（使用 IServiceProvider 接口，遵循依赖倒置原则）
const serviceProvider: IServiceProvider = serviceFactory
const applicationService = serviceProvider.getWorkspaceApplicationService()
const domainService = serviceProvider.getWorkspaceDomainService()

// 从状态管理器获取状态
const serviceTree = computed(() => domainService.getServiceTree())
const currentFunction = computed(() => domainService.getCurrentFunction())
const currentAppFromState = computed(() => domainService.getCurrentApp())

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

const createAppAdminsField = createWidgetFieldConfig({
  code: 'create_app_admins',
  name: '管理员',
  widgetType: WidgetType.USERS
})

const createAppAdminsFieldValue = computed(() =>
  createStringFieldValue(createAppAdminsField, createAppForm.value.admins, {
    display: (createAppForm.value.admins || '').split(',').map(s => s.trim()).filter(Boolean).join(', ')
  })
)

function handleCreateAppAdminsChange(value: FieldValue) {
  createAppForm.value.admins = extractStringFieldRaw(value)
}

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

const adminsField = createWidgetFieldConfig({
  code: 'admins',
  name: '管理员',
  widgetType: WidgetType.USERS
})

const adminsFieldValue = computed(() =>
  createStringFieldValue(adminsField, createDirectoryForm.value.admins, {
    display: (createDirectoryForm.value.admins || '').split(',').map(s => s.trim()).filter(Boolean).join(', ')
  })
)

// 处理管理员字段变化
function handleAdminsChange(value: FieldValue) {
  createDirectoryForm.value.admins = extractStringFieldRaw(value)
}

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
  setFunctionPermissionRequestListRef,
  setFunctionPermissionManageListRef,
  setFormOperateLogSectionRef,
  showScheduledTaskTab,
  showScheduledAgentTaskTab,
  showFunctionPermissionTabs,
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
  clearPermissionError: () => permissionErrorStore.clearError(),
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
  handleMiniRemove,
  handleMiniMaximizeChange,
  handleWorkspaceOpenWorkstation,
  initializeFromRoute: initializeMiniWorkstationsFromRoute,
} = useWorkspaceMiniWorkstations({
  route,
  router,
  workstationContext,
  buildWorkspacePath: (fullCodePath: string) => buildWorkspacePath(fullCodePath),
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
    open_as_mini: true
  })
}

// ⭐ 权限检查：是否有表格更新权限
const canUpdateTable = computed(() => {
  const node = currentFunction.value
  if (!node) return true  // 如果没有节点信息，默认允许（向后兼容）
  return hasPermission(node, TablePermission.update)
})

const supportsUpdateTable = computed(() => {
  return getFunctionCallbacks(currentFunctionDetail.value).includes('OnTableUpdateRow')
})

// ⭐ 权限错误状态
const permissionErrorStore = usePermissionErrorStore()
const hasPermissionError = computed(() => {
  return permissionErrorStore.currentError !== null
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
  publishToHubDialogVisible,
  publishSelectedNode,
  pushToHubDialogVisible,
  pushSelectedNode,
  pullFromHubDialogVisible,
  pastedHubLink,
  pullFromHubTargetPath,
  pullFromHubTargetName,
  updateHistoryDialogVisible,
  updateHistoryMode,
  updateHistoryAppId,
  updateHistoryAppVersion,
  updateHistoryFullCodePath,
  importGoFileInputRef,
  handleImportGoFiles,
  onImportGoFilesSelected,
  handlePublishToHub,
  handlePushToHub,
  openPullFromHubDialog,
  handleUpdateHistory,
  handlePublishSuccess,
  handlePushSuccess,
  handlePullSuccess
} = useWorkspaceNodeToolActions({
  currentApp,
  handleRefreshTree
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
  currentApp: () => currentApp.value,
  showLeftSidebar,
  openPullFromHubDialog,
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
