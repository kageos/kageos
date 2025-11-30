<!--
  WorkspaceView - 工作空间视图
  🔥 新架构的展示层组件
  
  职责：
  - 纯 UI 展示，不包含业务逻辑
  - 通过事件与 Application Layer 通信
  - 从 StateManager 获取状态并渲染
-->

<template>
  <div class="workspace-container">
    <!-- 顶部导航栏 -->
    <WorkspaceHeader />

    <div class="workspace-view">
      <!-- 左侧服务目录树 -->
      <div class="left-sidebar">
        <ServiceTreePanel
          ref="serviceTreePanelRef"
          :tree-data="serviceTree"
          :loading="loading"
          :current-node-id="currentFunction?.id || null"
          :current-function="currentFunction"
          @node-click="handleNodeClick"
          @create-directory="handleCreateDirectory"
          @fork-group="handleForkGroup"
          @copy-link="handleCopyLink"
        />
      </div>

      <!-- 中间函数渲染区域 -->
      <div class="function-renderer">
        <!-- 标签页区域 -->
        <WorkspaceTabs
          :tabs="tabs"
          :active-tab-id="activeTabId"
          @update:active-tab-id="(val: string) => activeTabId = val"
          @tab-click="handleTabClick"
          @tab-edit="handleTabsEdit"
        />
        
        <!-- 🔥 Create/Edit 模式：根据 queryTab 显示独立页面 -->
        <template v-if="queryTab === 'create' && currentFunction && currentFunctionDetail">
          <div class="form-page">
            <div class="form-page-header">
              <el-button @click="backToList" :icon="ArrowLeft">返回列表</el-button>
              <h2 class="form-page-title">新增数据</h2>
            </div>
            <div class="form-page-content">
              <FormView
                v-if="currentFunctionDetail.template_type === 'form'"
                :key="`form-create-${currentFunction.id}`"
                :function-detail="currentFunctionDetail"
              />
              <div v-else class="empty-state">
                <p>该函数不支持新增操作</p>
              </div>
            </div>
            <div class="form-page-footer">
              <el-button @click="backToList">取消</el-button>
              <el-button type="primary" @click="handleCreateSubmit">提交</el-button>
            </div>
          </div>
        </template>
        
        <template v-else-if="queryTab === 'edit' && currentFunction && currentFunctionDetail">
          <div class="form-page">
            <div class="form-page-header">
              <el-button @click="backToList" :icon="ArrowLeft">返回列表</el-button>
              <h2 class="form-page-title">编辑数据</h2>
            </div>
            <div class="form-page-content">
              <FormView
                v-if="currentFunctionDetail.template_type === 'form'"
                :key="`form-edit-${currentFunction.id}-${editRowId}`"
                :function-detail="editFunctionDetail"
                :initial-data="editInitialData"
              />
              <div v-else class="empty-state">
                <p>该函数不支持编辑操作</p>
              </div>
            </div>
            <div class="form-page-footer">
              <el-button @click="backToList">取消</el-button>
              <el-button type="primary" @click="handleEditSubmit">保存</el-button>
            </div>
          </div>
        </template>
        
        <!-- 🔥 Detail 模式：显示详情抽屉（通过 URL 参数打开） -->
        <!-- 注意：detail 模式使用抽屉显示，不需要单独的页面 -->
        
        <!-- Tab 内容区域（正常模式） -->
        <div v-else-if="tabs.length > 0" class="tabs-content-wrapper">
          <div class="tab-content">
            <FormView
              v-if="currentFunctionDetail?.template_type === 'form'"
              :key="`form-${activeTabId}`"
              :function-detail="currentFunctionDetail"
            />
            <TableView
              v-else-if="currentFunctionDetail?.template_type === 'table'"
              :key="`table-${activeTabId}`"
              :function-detail="currentFunctionDetail"
            />
            <div v-else class="empty-state">
              <p>加载中...</p>
            </div>
          </div>
        </div>
        <div v-else class="empty-state">
          <p>请在左侧选择功能</p>
        </div>
      </div>
    </div>

    <!-- 应用切换器（底部固定） -->
    <!-- 始终显示，即使应用列表为空，让用户可以创建应用 -->
    <AppSwitcher
      :current-app="currentApp"
      :app-list="appList"
      :loading-apps="loadingApps"
      @switch-app="handleSwitchApp"
      @create-app="showCreateAppDialog"
      @update-app="handleUpdateApp"
      @delete-app="handleDeleteApp"
      @load-apps="loadAppList"
    />

    <!-- 创建应用对话框 -->
    <el-dialog
      v-model="createAppDialogVisible"
      title="创建新应用"
      width="520px"
      :close-on-click-modal="false"
      @close="resetCreateAppForm"
    >
      <el-form :model="createAppForm" label-width="90px">
        <el-form-item label="应用名称" required>
          <el-input
            v-model="createAppForm.name"
            placeholder="请输入应用名称（如：客户管理系统）"
            maxlength="100"
            show-word-limit
            clearable
          />
        </el-form-item>
        <el-form-item label="应用代码" required>
          <el-input
            v-model="createAppForm.code"
            placeholder="请输入应用代码（如：crm）"
            maxlength="50"
            show-word-limit
            clearable
            @input="createAppForm.code = createAppForm.code.toLowerCase()"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            应用代码只能包含小写字母、数字和下划线，长度 2-50 个字符
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createAppDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitCreateApp" :loading="creatingApp">
            创建
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 详情抽屉 -->
    <WorkspaceDetailDrawer
      v-model:visible="detailDrawerVisible"
      v-model:mode="detailDrawerMode"
      :title="detailDrawerTitle"
      :fields="detailFields"
      :row-data="detailRowData"
      :table-data="detailTableData"
      :current-index="currentDetailIndex"
      :can-edit="currentFunctionDetail?.callbacks?.includes('OnTableUpdateRow') || false"
      :edit-function-detail="editFunctionDetail"
      :user-info-map="detailUserInfoMap"
      :submitting="drawerSubmitting"
      ref="detailDrawerRef"
      @navigate="handleNavigateDetail"
      @submit="() => submitDrawerEdit(detailDrawerRef?.formRendererRef)"
      @close="handleDetailDrawerClose"
    />

    <!-- 创建服务目录对话框 -->
    <el-dialog
      v-model="createDirectoryDialogVisible"
      :title="currentParentNode ? `在「${currentParentNode.name || currentParentNode.code}」下创建服务目录` : '创建服务目录'"
      width="520px"
      :close-on-click-modal="false"
      @close="resetCreateDirectoryForm"
    >
      <el-form :model="createDirectoryForm" label-width="90px">
        <el-form-item label="目录名称" required>
          <el-input
            v-model="createDirectoryForm.name"
            placeholder="请输入目录名称（如：用户管理）"
            maxlength="50"
            show-word-limit
            clearable
          />
        </el-form-item>
        <el-form-item label="目录代码" required>
          <el-input
            v-model="createDirectoryForm.code"
            placeholder="请输入目录代码，如：user"
            maxlength="50"
            show-word-limit
            clearable
            @input="createDirectoryForm.code = createDirectoryForm.code.toLowerCase()"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            目录代码只能包含小写字母、数字和下划线
          </div>
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="createDirectoryForm.description"
            type="textarea"
            :rows="3"
            placeholder="请输入目录描述（可选）"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>
        <el-form-item label="标签">
          <el-input
            v-model="createDirectoryForm.tags"
            placeholder="请输入标签，多个标签用逗号分隔（可选）"
            maxlength="100"
            clearable
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createDirectoryDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmitCreateDirectory" :loading="creatingDirectory">
            创建
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- Fork 函数组对话框 -->
    <FunctionForkDialog
      v-model="forkDialogVisible"
      :source-full-group-code="forkSourceGroupCode || undefined"
      :source-group-name="forkSourceGroupName || undefined"
      :current-app="currentApp || undefined"
      @success="handleForkSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch, ref, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElNotification, ElDialog, ElForm, ElFormItem, ElInput, ElButton, ElIcon } from 'element-plus'
import { InfoFilled, ArrowLeft } from '@element-plus/icons-vue'
import { eventBus, WorkspaceEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import { useAuthStore } from '@/stores/auth'
import ServiceTreePanel from '@/components/ServiceTreePanel.vue'
import AppSwitcher from '@/components/AppSwitcher.vue'
import FunctionForkDialog from '@/components/FunctionForkDialog.vue'
import FormView from './FormView.vue'
import TableView from './TableView.vue'
import WorkspaceHeader from '../components/WorkspaceHeader.vue'
import WorkspaceTabs from '../components/WorkspaceTabs.vue'
import WorkspaceDetailDrawer from '../components/WorkspaceDetailDrawer.vue'
import type { ServiceTree, App } from '../../domain/services/WorkspaceDomainService'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/types'
import type { FieldConfig, FieldValue } from '../../domain/types'
// 🔥 导入 Composable
import { useWorkspaceTabs } from '../composables/useWorkspaceTabs'
import { useWorkspaceRouting } from '../composables/useWorkspaceRouting'
import { useWorkspaceDetail } from '../composables/useWorkspaceDetail'
import { useWorkspaceApp } from '../composables/useWorkspaceApp'
import { useWorkspaceServiceTree } from '../composables/useWorkspaceServiceTree'
import { findNodeByPath } from '../utils/workspaceUtils'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

// 依赖注入（使用 ServiceFactory 简化）
const stateManager = serviceFactory.getWorkspaceStateManager()
const applicationService = serviceFactory.getWorkspaceApplicationService()

// 从状态管理器获取状态
const serviceTree = computed(() => stateManager.getServiceTree())
const currentFunction = computed(() => stateManager.getCurrentFunction())
const currentAppFromState = computed(() => stateManager.getCurrentApp())

// 🔥 初始化 Composable
const {
  tabs,
  activeTabId,
  handleTabClick: tabsHandleTabClick,
  handleTabsEdit,
  restoreTabsFromStorage,
  restoreTabsNodes: tabsRestoreTabsNodes,
  setupTabDataWatch,
  setupAutoSave
} = useWorkspaceTabs()

const currentApp = computed<AppType | null>(() => {
  const app = currentAppFromState.value
  if (!app) return null
  // 从 appList 中查找对应的应用（确保使用最新的应用数据）
  const foundApp = appList.value.find((a: AppType) => a.id === app.id || (a.user === app.user && a.code === app.code))
  return foundApp || {
    id: app.id,
    user: app.user,
    code: app.code,
    name: app.name,
    nats_id: 0,
    host_id: 0,
    status: 'enabled' as const,
    version: '',
    created_at: '',
    updated_at: ''
  }
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
  checkAndExpandForkedPaths: serviceTreeCheckAndExpandForkedPaths,
  handleCopyLink
} = useWorkspaceServiceTree()

const currentFunctionDetail = computed<FunctionDetail | null>(() => {
  const tabsCount = tabs.value.length
  const activeTabIdValue = activeTabId.value
  
  // 🔥 如果没有标签页，不返回 functionDetail，避免渲染旧的组件
  if (tabsCount === 0) {
    console.log('[WorkspaceView] currentFunctionDetail: 没有标签页，返回 null')
    return null
  }
  
  const node = currentFunction.value
  if (!node) {
    console.log('[WorkspaceView] currentFunctionDetail: 没有当前函数节点，返回 null')
    return null
  }
  
  // 🔥 检查当前函数是否属于当前激活的 tab
  const activeTab = tabs.value.find((t: any) => t.id === activeTabIdValue)
  if (activeTab && activeTab.node) {
    const activeTabNode = activeTab.node
    // 检查 node 是否匹配当前激活的 tab
    const nodeId = node.full_code_path || String(node.id)
    const activeTabNodeId = activeTab.node.full_code_path || String(activeTab.node.id)
    if (nodeId !== activeTabNodeId) {
      // 如果不匹配，返回 null，避免渲染错误的组件
      console.log('[WorkspaceView] currentFunctionDetail: 节点不匹配当前激活的 tab', {
        nodeId,
        activeTabNodeId,
        activeTabId: activeTabIdValue
      })
      return null
    }
  }
  
  const detail = stateManager.getFunctionDetail(node)
  console.log('[WorkspaceView] currentFunctionDetail: 返回详情', {
    functionId: detail?.id,
    router: detail?.router,
    templateType: detail?.template_type,
    activeTabId: activeTabIdValue,
    tabsCount
  })
  
  return detail
})

const {
  detailDrawerVisible,
  detailDrawerTitle,
  detailRowData,
  detailFields,
  detailOriginalRow,
  detailDrawerMode,
  drawerSubmitting,
  detailFormRendererRef,
  detailUserInfoMap,
  detailTableData,
  currentDetailIndex,
  editFunctionDetail,
  toggleDrawerMode,
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
  syncRouteToTab,
  loadAppFromRoute: routingLoadAppFromRoute,
  setupRouteWatch
} = useWorkspaceRouting({
  tabs: () => tabs.value,
  activeTabId: () => activeTabId.value,
  serviceTree: () => serviceTree.value,
  currentApp: () => currentApp.value,
  appList: () => appList.value,
  loadAppList,
  findNodeByPath,
  checkAndExpandForkedPaths: () => serviceTreeCheckAndExpandForkedPaths(
    () => serviceTree.value,
    () => serviceTreePanelRef.value,
    () => currentApp.value
  ),
  expandCurrentRoutePath: () => serviceTreeExpandCurrentRoutePath(
    () => serviceTree.value,
    () => serviceTreePanelRef.value,
    () => currentApp.value
  )
})


// 🔥 Tab 点击处理（使用 Composable）
const handleTabClick = tabsHandleTabClick


// 🔥 queryTab：当前激活的Tab模式（用于路由查询参数，控制 create/edit 等模式）
const queryTab = computed(() => (route.query._tab as string) || 'run')

// 🔥 编辑模式相关
const editRowId = computed(() => {
  const id = route.query.id || route.query._id
  return id ? Number(id) : null
})

// 🔥 编辑模式的初始数据（从 URL 参数提取）
const editInitialData = computed(() => {
  const initialData: Record<string, any> = {}
  const query = route.query
  
  // 如果有 id 参数，添加到 initialData
  if (editRowId.value) {
    const idField = currentFunctionDetail.value?.request?.find((f: FieldConfig) => 
      f.code.toLowerCase() === 'id' || f.widget?.type === 'number'
    )
    if (idField) {
      initialData[idField.code] = editRowId.value
    }
  }
  
  // 遍历所有查询参数，如果字段在 request 中，添加到 initialData
  if (currentFunctionDetail.value?.request) {
    currentFunctionDetail.value.request.forEach((field: FieldConfig) => {
      const fieldCode = field.code
      // 跳过 _ 开头的参数（系统参数）
      if (fieldCode.startsWith('_')) return
      
      if (query[fieldCode] !== undefined && query[fieldCode] !== null && query[fieldCode] !== '') {
        const value = query[fieldCode]
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


// Fork 函数组相关
const forkDialogVisible = ref(false)
const forkSourceGroupCode = ref('')
const forkSourceGroupName = ref('')

// ServiceTreePanel 引用（用于展开路径）
const serviceTreePanelRef = ref<InstanceType<typeof ServiceTreePanel> | null>(null)

onMounted(() => {
  // 🔥 监听表格详情事件（使用 Composable）
  eventBus.on('table:detail-row', async ({ row, index, tableData }: { row: Record<string, any>, index?: number, tableData?: any[] }) => {
    await openDetailDrawer(row, index, tableData)
  })
  
  // 🔥 注意：不再监听 tabActivated 事件来更新路由
  // 路由应该由 handleTabClick 直接更新（路由优先策略）
  // tabActivated 事件只用于状态同步，不用于路由更新
  // 这样可以与服务目录切换的逻辑保持一致
  
  // 🔥 设置 URL 监听（使用 Composable）
  setupUrlWatch()
})



// 转换 loadingTree 为 boolean (避免 computed 类型问题)
const loading = computed(() => stateManager.isLoading())

// 事件处理
const handleNodeClick = (node: ServiceTreeType) => {
  // 转换为新架构的 ServiceTree 类型
  const serviceTree: ServiceTree = node as any
  
  // 🔥 路由优先策略：先更新路由，路由变化会触发 Tab 状态更新
  if (serviceTree.type === 'function' && serviceTree.full_code_path) {
    const targetPath = `/workspace${serviceTree.full_code_path}`
    if (route.path !== targetPath) {
      // 路由不同，更新路由，保留当前 URL 的 query 参数（分页、排序、搜索等）
      // 🔥 服务目录切换时保留 URL 参数，这样切换回去时能恢复之前的状态
      const currentQuery = route.query
      const preservedQuery: Record<string, string | string[]> = {}
      
      // 保留所有参数（分页、排序、搜索等）
      Object.keys(currentQuery).forEach(key => {
        const value = currentQuery[key]
        if (value !== null && value !== undefined) {
          if (Array.isArray(value)) {
            preservedQuery[key] = value.filter(v => v !== null).map(v => String(v))
          } else {
            preservedQuery[key] = String(value)
          }
        }
      })
      
      router.replace({ path: targetPath, query: preservedQuery }).catch(() => {})
    } else {
      // 路由已匹配，直接触发节点点击加载详情（避免路由更新循环）
      applicationService.triggerNodeClick(serviceTree)
    }
  } else {
    // 目录节点，不更新路由，只设置当前函数
    applicationService.triggerNodeClick(serviceTree)
  }
}

// 🔥 处理创建目录（使用 Composable）
const handleCreateDirectory = (parentNode?: ServiceTreeType) => {
  serviceTreeHandleCreateDirectory(parentNode || null, () => currentApp.value)
}

const handleSubmitCreateDirectory = async () => {
  await serviceTreeHandleSubmitCreateDirectory(() => currentApp.value)
}

// 处理 Fork 函数组
const handleForkGroup = (node: ServiceTreeType | null) => {
  // 如果传入了节点，使用它；否则打开对话框让用户选择
  if (node) {
    if (!node.full_group_code) {
      ElNotification.warning({
        title: '提示',
        message: '该节点没有函数组代码，无法克隆'
      })
      return
    }
    forkSourceGroupCode.value = node.full_group_code
    forkSourceGroupName.value = node.group_name || node.name || ''
  } else {
    // 没有传入节点，清空预设值，让用户在对话框中选择
    forkSourceGroupCode.value = ''
    forkSourceGroupName.value = ''
  }
  forkDialogVisible.value = true
}

// Fork 成功后的回调
const handleForkSuccess = () => {
  // 刷新服务目录树
  if (currentApp.value) {
    const appForService: App = {
      id: currentApp.value.id,
      user: currentApp.value.user,
      code: currentApp.value.code,
      name: currentApp.value.name,
      nats_id: currentApp.value.nats_id || 0,
      host_id: currentApp.value.host_id || 0,
      status: currentApp.value.status || 'enabled',
      version: currentApp.value.version || '',
      created_at: currentApp.value.created_at || '',
      updated_at: currentApp.value.updated_at || ''
    }
    applicationService.triggerAppSwitch(appForService)
  }
  ElNotification.success({
    title: '成功',
    message: '克隆完成！请刷新页面查看新功能'
  })
}

// 🔥 展开当前路由对应的路径（使用 Composable）
const expandCurrentRoutePath = () => {
  serviceTreeExpandCurrentRoutePath(
    () => serviceTree.value,
    () => serviceTreePanelRef.value,
    () => currentApp.value
  )
}

// 🔥 检查并展开 forked 路径（使用 Composable）
const checkAndExpandForkedPaths = () => {
  serviceTreeCheckAndExpandForkedPaths(
    () => serviceTree.value,
    () => serviceTreePanelRef.value,
    () => currentApp.value
  )
}

// 🔥 返回列表（从 create/edit 模式返回）
const backToList = () => {
  if (!currentFunction.value) return
  
  // 移除系统参数，保留其他参数
  const query = { ...route.query }
  delete query._tab
  delete query._id
  
  const path = `/workspace${currentFunction.value.full_code_path || ''}`
  router.push({ path, query }).catch(() => {})
}


// 🔥 处理新增提交（通过 FormView 的提交按钮，这里只是占位）
const handleCreateSubmit = async () => {
  // FormView 内部已经有提交逻辑，这里不需要额外处理
  // 如果需要，可以通过 ref 或事件总线来触发 FormView 的提交
  ElNotification.info({
    title: '提示',
    message: '请使用表单内的提交按钮提交数据'
  })
}

// 🔥 处理编辑提交（通过 FormView 的提交按钮，这里只是占位）
const handleEditSubmit = async () => {
  // FormView 内部已经有提交逻辑，这里不需要额外处理
  // 如果需要，可以通过 ref 或事件总线来触发 FormView 的提交
  ElNotification.info({
    title: '提示',
    message: '请使用表单内的提交按钮提交数据'
  })
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


// 生命周期
let unsubscribeFunctionLoaded: (() => void) | null = null
let unsubscribeServiceTreeLoaded: (() => void) | null = null
let unsubscribeAppSwitched: (() => void) | null = null

// 🔥 重新关联 tabs 的 node 信息（使用 Composable）
const restoreTabsNodes = () => {
  tabsRestoreTabsNodes(serviceTree.value, findNodeByPath)
}

onMounted(async () => {
  // 🔥 首先从 localStorage 恢复 tabs
  restoreTabsFromStorage()
  
  // 🔥 设置 Tab 数据监听和自动保存
  setupTabDataWatch()
  setupAutoSave()
  
  // 监听函数加载完成事件
  unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, () => {
    // 状态已通过 StateManager 自动更新
  })

  // 监听服务树加载完成事件
  unsubscribeServiceTreeLoaded = eventBus.on(WorkspaceEvent.serviceTreeLoaded, (payload: { app: any, tree: any[] }) => {
    // 状态已通过 StateManager 自动更新
    console.log('[WorkspaceView] 收到 serviceTreeLoaded 事件，节点数:', payload.tree?.length || 0)
    
    // 🔥 服务树加载后，重新关联 tabs 的 node 信息
    nextTick(() => {
      restoreTabsNodes()
    })
  })
  
  // 监听应用切换事件，开始加载服务树
  unsubscribeAppSwitched = eventBus.on(WorkspaceEvent.appSwitched, (payload: { app: any }) => {
    console.log('[WorkspaceView] 收到 appSwitched 事件，目标应用:', payload.app?.user, payload.app?.code, 'ID:', payload.app?.id)
  })

  // 加载应用列表
  await loadAppList()

  // 从路由加载应用（会激活对应的 Tab）
  await routingLoadAppFromRoute()
  
  // 🔥 设置路由监听
  setupRouteWatch()
})

// 🔥 监听服务树变化，重新关联 tabs 的 node 并展开目录树
watch(() => serviceTree.value.length, (newLength: number) => {
  if (newLength > 0 && currentApp.value) {
    // 重新关联 tabs 的 node 信息（会检查并加载函数详情）
    restoreTabsNodes()
    
    // 展开目录树
    if (route.query._forked) {
    checkAndExpandForkedPaths()
    } else {
      expandCurrentRoutePath()
  }
  }
}, { immediate: true })

// 🔥 监听当前应用变化，检查 _forked 参数
watch(currentApp, () => {
  if (serviceTree.value.length > 0 && currentApp.value && route.query._forked) {
    console.log('[WorkspaceView] 应用变化，检查 _forked 参数')
    nextTick(() => {
      checkAndExpandForkedPaths()
    })
  }
})

// 🔥 监听 queryTab 变化，处理 create/edit/detail 模式
watch(queryTab, async (newTab: string, oldTab: string) => {
  if (newTab === 'create' || newTab === 'edit') {
    // create/edit 模式需要确保函数详情已加载
    if (!currentFunction.value) {
      console.log('[WorkspaceView] queryTab 变化但当前函数不存在，等待函数加载')
      return
    }
    
    // 如果函数详情未加载，触发加载
    if (!currentFunctionDetail.value) {
      console.log('[WorkspaceView] queryTab 变化，加载函数详情')
      await applicationService.handleNodeClick(currentFunction.value)
    }
  } else if (newTab === 'detail') {
    // detail 模式需要确保函数详情已加载，并且表格数据已加载
    if (!currentFunction.value) {
      console.log('[WorkspaceView] queryTab=detail 但当前函数不存在，等待函数加载')
      return
    }
    
    // 如果函数详情未加载，触发加载
    if (!currentFunctionDetail.value) {
      console.log('[WorkspaceView] queryTab=detail，加载函数详情')
      await applicationService.handleNodeClick(currentFunction.value)
    }
    
    // detail 模式会在另一个 watch 中处理（监听 route.query.id）
  }
}, { immediate: false })

// 🔥 监听路由 query 变化，处理 _tab 参数
watch(() => route.query._tab, async (newTab: any) => {
  if (newTab === 'create' || newTab === 'edit') {
    // 确保当前函数和函数详情已加载
    if (!currentFunction.value) {
      console.log('[WorkspaceView] tab 参数变化但当前函数不存在')
      return
    }
    
    if (!currentFunctionDetail.value) {
      console.log('[WorkspaceView] tab 参数变化，加载函数详情')
      await applicationService.handleNodeClick(currentFunction.value)
    }
  } else if (newTab === 'detail') {
    // detail 模式会在另一个 watch 中处理（监听 route.query.id）
    // 这里只需要确保函数详情已加载
    if (!currentFunction.value) {
      console.log('[WorkspaceView] tab=detail 但当前函数不存在')
      return
    }
    
    if (!currentFunctionDetail.value) {
      console.log('[WorkspaceView] tab=detail，加载函数详情')
      await applicationService.handleNodeClick(currentFunction.value)
    }
  }
}, { immediate: false })


onUnmounted(() => {
  if (unsubscribeFunctionLoaded) {
    unsubscribeFunctionLoaded()
  }
  if (unsubscribeServiceTreeLoaded) {
    unsubscribeServiceTreeLoaded()
  }
  if (unsubscribeAppSwitched) {
    unsubscribeAppSwitched()
  }
})
</script>

<style scoped>
.workspace-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.workspace-view {
  display: flex;
  flex: 1;
  overflow: hidden; /* 防止双滚动条 */
}

.tabs-content-wrapper {
  flex: 1;
  overflow: hidden; /* 🔥 外层容器隐藏溢出，内层处理滚动 */
  display: flex;
  flex-direction: column;
  min-height: 0; /* 🔥 关键：允许 flex 子元素缩小 */
}

.tab-content {
  flex: 1;
  overflow-y: auto !important; /* 🔥 强制允许垂直滚动，让搜索框和数据区一起滚动 */
  overflow-x: hidden;
  min-height: 0; /* 🔥 关键：允许 flex 子元素缩小 */
  height: 0; /* 🔥 关键：配合 flex: 1 和 min-height: 0，让滚动容器正确计算高度 */
  -webkit-overflow-scrolling: touch; /* 🔥 iOS 平滑滚动 */
}

.left-sidebar {
  width: 300px;
  border-right: 1px solid var(--el-border-color);
}

.function-renderer {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

/* 新增/编辑页面样式 */
.form-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
  overflow-y: auto;
}

.form-page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.form-page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.form-page-content {
  flex: 1;
  min-height: 0;
}

.form-page-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}
</style>
