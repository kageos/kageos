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
          @publish-to-hub="handlePublishToHub"
          @push-to-hub="handlePushToHub"
          @pull-from-hub="handlePullFromHub"
          @refresh-tree="handleRefreshTree"
          @update-history="handleUpdateHistory"
        />
      </div>

      <!-- 中间函数渲染区域 -->
      <div class="function-renderer">
        <!-- 面包屑导航（只在显示函数详情时显示） -->
        <FunctionBreadcrumb
          v-if="currentFunction && currentFunction.type === 'function'"
          :current-node="currentFunction"
          :service-tree="serviceTree"
          @node-click="handleBreadcrumbNodeClick"
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
                v-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.FORM"
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
                v-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.FORM"
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
        
        <!-- 🔥 服务目录详情页面 -->
        <PackageDetailView
          v-else-if="currentFunction && currentFunction.type === 'package' && !selectedAgent"
          :package-node="currentFunction"
          @generate-system="handlePackageGenerateSystem"
        />
        
        <!-- 🔥 点击目录节点时根据选择的智能体显示不同的聊天面板 -->
        <div v-else-if="currentFunction && currentFunction.type === 'package' && selectedAgent" class="ai-chat-wrapper">
          <!-- 根据 chat_type 选择不同的渲染方式 -->
          <AIChatPanel
            v-if="selectedAgent.chat_type === 'function_gen'"
            ref="aiChatPanelRef"
            :agent-id="selectedAgent.id"
            :tree-id="currentFunction.id"
            :package="currentFunction.code"
            :current-node-name="currentFunction.name"
            :existing-files="existingFilesInPackage"
            @close="handleCloseAIChat"
          />
          <!-- 可以在这里添加其他 chat_type 的渲染组件 -->
          <!-- 例如：<TaskChatPanel v-else-if="selectedAgent.chat_type === 'chat-task'" ... /> -->
        </div>
        
        <!-- 函数详情区域（正常模式 - 函数节点） -->
        <div v-else-if="currentFunction && currentFunction.type === 'function'" class="function-content-wrapper">
          <div class="function-content">
            <!-- ⭐ 如果函数详情已加载，显示对应的视图 -->
            <!-- ⚠️ 重要：只有当 currentFunctionDetail 的 id 或 router 与 currentFunction 匹配时才显示 -->
            <template v-if="currentFunctionDetail && 
                           currentFunction && 
                           (currentFunctionDetail.id === currentFunction.ref_id || 
                            currentFunctionDetail.router === currentFunction.full_code_path)">
              <!-- 🔥 移除 keep-alive，每次切换函数时重新渲染，保证数据一致性 -->
              <!-- 🔥 使用 full_code_path 作为 key，确保函数切换时组件正确重建 -->
              <FormView
                v-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.FORM"
                :key="`form-${currentFunction.full_code_path || currentFunction.id}`"
                :function-detail="currentFunctionDetail"
              />
              <TableView
                v-else-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.TABLE"
                :key="`table-${currentFunction.full_code_path || currentFunction.id}`"
                :function-detail="currentFunctionDetail"
              />
              <ChartView
                v-else-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.CHART"
                :key="`chart-${currentFunction.full_code_path || currentFunction.id}`"
                :function-detail="currentFunctionDetail"
              />
              <div v-else :key="`empty-${currentFunction.full_code_path || currentFunction.id}`" class="empty-state">
                <p>加载中...</p>
              </div>
            </template>
            <!-- 如果函数详情未加载且有权限错误，显示权限错误组件 -->
            <PermissionDeniedView
              v-else-if="hasPermissionError"
              :key="`permission-denied-${currentFunction.full_code_path || currentFunction.id}`"
            />
            <!-- 如果函数详情未加载且没有权限错误，显示加载中 -->
            <div v-else :key="`loading-${currentFunction.full_code_path || currentFunction.id}`" class="empty-state">
              <p>加载中...</p>
            </div>
          </div>
        </div>
        <div v-else class="empty-state">
          <p>请在左侧选择功能或目录</p>
        </div>
      </div>
    </div>

    <!-- 智能体选择对话框 -->
    <AgentSelectDialog
      v-model="agentSelectDialogVisible"
      :tree-id="currentFunction?.id || null"
      :package="currentFunction?.code || ''"
      :current-node-name="currentFunction?.name || ''"
      @confirm="handleAgentSelect"
    />

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

    <!-- 创建工作空间对话框 -->
    <el-dialog
      v-model="createAppDialogVisible"
      title="创建新工作空间"
      width="800px"
      :close-on-click-modal="false"
      @close="resetCreateAppForm"
    >
      <el-form :model="createAppForm" label-width="90px">
        <el-form-item label="名称" required>
          <el-input
            v-model="createAppForm.name"
            placeholder="请输入名称（如：清北大学、首都市政府、xxx图书馆、xxx医院、xxx银行、xxx科技公司）"
            maxlength="100"
            show-word-limit
            clearable
          />
        </el-form-item>
        <el-form-item label="英文标识" required>
          <el-input
            v-model="createAppForm.code"
            placeholder="请输入英文标识（如：tsinghua、pku_gsm）"
            maxlength="50"
            show-word-limit
            clearable
            @input="createAppForm.code = createAppForm.code.toLowerCase()"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            英文标识只能包含小写字母、数字和下划线，长度 2-50 个字符
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
    <TableRowDetailDrawer
      v-model:visible="detailDrawerVisible"
      v-model:mode="detailDrawerMode"
      :title="detailDrawerTitle"
      :fields="detailFields"
      :row-data="detailRowData"
      :table-data="detailTableData"
      :current-index="currentDetailIndex"
      :can-edit="(currentFunctionDetail?.callbacks?.includes('OnTableUpdateRow') || false) && canUpdateTable"
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

    <!-- 创建服务目录对话框 -->
    <el-dialog
      v-model="createDirectoryDialogVisible"
      :title="currentParentNode ? `在「${currentParentNode.name || currentParentNode.code}」下创建服务目录` : '创建服务目录'"
      width="520px"
      :close-on-click-modal="false"
      @close="handleCloseCreateDirectoryDialog"
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
          <el-button type="primary" @click="() => handleSubmitCreateDirectory(() => currentApp.value)" :loading="creatingDirectory">
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

    <!-- 发布到应用中心对话框 -->
    <PublishToHubDialog
      v-model="publishToHubDialogVisible"
      :selected-node="publishSelectedNode"
      :current-app="currentApp || undefined"
      @success="handlePublishSuccess"
    />
    <PushToHubDialog
      v-model="pushToHubDialogVisible"
      :selected-node="pushSelectedNode"
      :current-app="currentApp || undefined"
      @success="handlePushSuccess"
    />
    <PullFromHubDialog
      v-model="pullFromHubDialogVisible"
      :current-app="currentApp || undefined"
      :initial-hub-link="pastedHubLink"
      @success="handlePullSuccess"
    />

    <!-- 变更记录对话框 -->
    <DirectoryUpdateHistoryDialog
      v-model="updateHistoryDialogVisible"
      :mode="updateHistoryMode"
      :app-id="updateHistoryAppId"
      :app-version="updateHistoryAppVersion"
      :full-code-path="updateHistoryFullCodePath"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch, ref, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElNotification, ElDialog, ElForm, ElFormItem, ElInput, ElButton, ElIcon } from 'element-plus'
import { InfoFilled, ArrowLeft } from '@element-plus/icons-vue'
import { eventBus, WorkspaceEvent, RouteEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import { RouteManager } from '../../infrastructure/routeManager'
import { useAuthStore } from '@/stores/auth'
import ServiceTreePanel from '@/components/ServiceTreePanel.vue'
import AppSwitcher from '@/components/AppSwitcher.vue'
import FunctionForkDialog from '@/components/FunctionForkDialog.vue'
import PublishToHubDialog from '@/components/PublishToHubDialog.vue'
import PushToHubDialog from '@/components/PushToHubDialog.vue'
import PullFromHubDialog from '@/components/PullFromHubDialog.vue'
import DirectoryUpdateHistoryDialog from '@/components/DirectoryUpdateHistoryDialog.vue'
import FormView from './FormView.vue'
import TableView from './TableView.vue'
import ChartView from './ChartView.vue'
import WorkspaceHeader from '../components/WorkspaceHeader.vue'
import FunctionBreadcrumb from '../components/FunctionBreadcrumb.vue'
import TableRowDetailDrawer from '../components/TableRowDetailDrawer.vue'
import PermissionDeniedView from '../components/PermissionDeniedView.vue'
import AIChatPanel from '../components/AIChatPanel.vue'
import AgentSelectDialog from '@/components/Agent/AgentSelectDialog.vue'
import PackageDetailView from '../components/PackageDetailView.vue'
import type { ServiceTree, App } from '../../domain/services/WorkspaceDomainService'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/types'
import type { FieldConfig, FieldValue } from '../../domain/types'
// 🔥 导入 Composable
import { useWorkspaceRouting } from '../composables/useWorkspaceRouting'
import { RouteSource } from '@/utils/routeSource'
import { useWorkspaceDetail } from '../composables/useWorkspaceDetail'
import { useWorkspaceApp } from '../composables/useWorkspaceApp'
import { useWorkspaceServiceTree } from '../composables/useWorkspaceServiceTree'
import { findNodeByPath, findNodeById, getDirectChildFunctionCodes } from '../utils/workspaceUtils'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { resolveWorkspaceUrl } from '@/utils/route'
import { getAgentList, type AgentInfo } from '@/api/agent'
import { isLinkNavigation as checkLinkNavigation, LINK_TYPE_QUERY_KEY } from '@/utils/linkNavigation'
import { hasPermission, TablePermissions, buildPermissionApplyURL } from '@/utils/permission'
import { usePermissionErrorStore } from '@/stores/permissionError'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

// 依赖注入（使用 ServiceFactory 简化）
const stateManager = serviceFactory.getWorkspaceStateManager()
const applicationService = serviceFactory.getWorkspaceApplicationService()
const domainService = serviceFactory.getWorkspaceDomainService()

// 从状态管理器获取状态
const serviceTree = computed(() => stateManager.getServiceTree())
const currentFunction = computed(() => stateManager.getCurrentFunction())
const currentAppFromState = computed(() => stateManager.getCurrentApp())

// 🔥 不再使用 Tab 功能，简化系统

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

// 🔥 移除缓存后，通过事件获取函数详情
const currentFunctionDetail = ref<FunctionDetail | null>(null)

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

// 发布到应用中心对话框
const publishToHubDialogVisible = ref(false)
const publishSelectedNode = ref<ServiceTreeType | null>(null)
const pushToHubDialogVisible = ref(false)
const pushSelectedNode = ref<ServiceTreeType | null>(null)
const pullFromHubDialogVisible = ref(false)
const pastedHubLink = ref('')  // 粘贴的 Hub 链接

// 变更记录对话框状态
const updateHistoryDialogVisible = ref(false)
const updateHistoryMode = ref<'app' | 'directory'>('app')
const updateHistoryAppId = ref(0)
const updateHistoryAppVersion = ref('')
const updateHistoryFullCodePath = ref('')

// ServiceTreePanel 引用（用于展开路径）
const serviceTreePanelRef = ref<InstanceType<typeof ServiceTreePanel> | null>(null)

// AI 对话框相关
const agentSelectDialogVisible = ref(false)
const selectedAgent = ref<AgentInfo | null>(null)
const aiChatPanelRef = ref<InstanceType<typeof AIChatPanel> | null>(null)

// 处理智能体选择
function handleAgentSelect(agent: AgentInfo) {
  selectedAgent.value = agent
  agentSelectDialogVisible.value = false
  
  // 选择智能体后，通知 AIChatPanel 创建新会话
  // 使用 nextTick 确保组件已渲染
  nextTick(() => {
    if (aiChatPanelRef.value && typeof (aiChatPanelRef.value as any).handleAgentSelect === 'function') {
      (aiChatPanelRef.value as any).handleAgentSelect(agent)
    }
  })
  
  // 如果路由不匹配，更新路由
  if (currentFunction.value?.full_code_path && currentApp.value) {
    const targetPath = buildWorkspacePath(currentFunction.value.full_code_path)
    if (route.path !== targetPath) {
      eventBus.emit(RouteEvent.updateRequested, {
        path: targetPath,
        query: {},
        replace: true,
        preserveParams: {
          state: false,
          table: false,
          search: false
        },
        source: RouteSource.AGENT_SELECT
      })
    }
  }
}

// 处理服务目录的生成系统按钮点击
function handlePackageGenerateSystem(agent: AgentInfo) {
  selectedAgent.value = agent
  // 设置当前函数（确保 AIChatPanel 能正确显示）
  if (currentFunction.value && currentFunction.value.type === 'package') {
    applicationService.triggerNodeClick(currentFunction.value)
  }
  // 触发 AIChatPanel 新建会话（使用 nextTick 确保组件已渲染）
  nextTick(() => {
    if (aiChatPanelRef.value && typeof (aiChatPanelRef.value as any).handleAgentSelect === 'function') {
      // 调用 handleAgentSelect 会创建新会话（清空 sessionId，显示欢迎消息）
      (aiChatPanelRef.value as any).handleAgentSelect(agent)
    }
  })
}

// 关闭 AI 聊天面板
function handleCloseAIChat() {
  selectedAgent.value = null
  // 如果当前是目录节点，清除当前函数选择
  if (currentFunction.value?.type === 'package') {
    applicationService.triggerNodeClick(null as any)
  }
}

// 获取当前 package 下的子节点文件名（用于确保生成的文件名唯一）
const existingFilesInPackage = computed(() => {
  if (!currentFunction.value || currentFunction.value.type !== 'package') {
    return []
  }
  
  // 从 serviceTree 中查找当前节点
  const currentNode = findNodeById(serviceTree.value, currentFunction.value.id)
  if (!currentNode) {
    return []
  }
  
  // 获取直接子节点（只收集一级子节点，type 为 'function' 的）
  return getDirectChildFunctionCodes(currentNode)
})

// ⭐ 权限检查：是否有表格更新权限
const canUpdateTable = computed(() => {
  const node = currentFunction.value
  if (!node) return true  // 如果没有节点信息，默认允许（向后兼容）
  return hasPermission(node, TablePermissions.update)
})

// ⭐ 权限错误状态
const permissionErrorStore = usePermissionErrorStore()
const hasPermissionError = computed(() => {
  return permissionErrorStore.currentError !== null
})

// 🔥 全局粘贴监听：检测 Hub 链接并自动打开安装对话框
const handleGlobalPaste = async (event: ClipboardEvent) => {
  // 如果当前焦点在输入框、文本域等可编辑元素上，不处理（让默认行为生效）
  const target = event.target as HTMLElement
  if (target && (
    target.tagName === 'INPUT' ||
    target.tagName === 'TEXTAREA' ||
    target.isContentEditable
  )) {
    return
  }

  const pastedText = event.clipboardData?.getData('text')
  if (pastedText && pastedText.trim().startsWith('hub://')) {
    // 阻止默认粘贴行为
    event.preventDefault()
    
    // 检查是否有当前应用
    if (!currentApp.value) {
      ElMessage.warning('请先选择应用')
      return
    }

    // 设置粘贴的 Hub 链接
    pastedHubLink.value = pastedText.trim()
    
    // 打开安装对话框
    pullFromHubDialogVisible.value = true
    
    ElMessage.info('检测到 Hub 链接，已打开安装对话框')
  }
}

onMounted(() => {
  // 🔥 监听表格详情事件（使用 Composable）
  eventBus.on('table:detail-row', async ({ row, index, tableData }: { row: Record<string, any>, index?: number, tableData?: any[] }) => {
    await openDetailDrawer(row, index, tableData)
  })
  
  // 🔥 Tab 功能已删除，相关事件监听已移除
  
  // 🔥 设置 URL 监听（使用 Composable）
  setupUrlWatch()
  
  // 🔥 添加全局粘贴监听
  document.addEventListener('paste', handleGlobalPaste)
})

onUnmounted(() => {
  // 🔥 移除全局粘贴监听
  document.removeEventListener('paste', handleGlobalPaste)
})



// 转换 loadingTree 为 boolean (避免 computed 类型问题)
const loading = computed(() => stateManager.isLoading())

/**
 * 构建工作空间路径
 */
const buildWorkspacePath = (fullCodePath: string): string => {
  return resolveWorkspaceUrl(fullCodePath.startsWith('/') ? fullCodePath : `/${fullCodePath}`)
}

/**
 * 判断是否是 table 函数
 */
const isTableFunction = (node: ServiceTree): boolean => {
  return node.template_type === TEMPLATE_TYPE.TABLE
}

/**
 * 判断是否是 link 跳转
 */
const isLinkNavigation = (): boolean => {
  return checkLinkNavigation(route.query as Record<string, any>)
}

/**
 * 构建 link 跳转的查询参数（保留所有参数，除了 _link_type）
 */
const buildLinkNavigationQuery = (): Record<string, string | string[]> => {
  const preservedQuery: Record<string, string | string[]> = {}
  Object.keys(route.query).forEach(key => {
    if (key !== LINK_TYPE_QUERY_KEY) {
      const value = route.query[key]
      if (value !== null && value !== undefined) {
        preservedQuery[key] = Array.isArray(value) 
          ? value.filter(v => v !== null).map(v => String(v))
          : String(value)
      }
    }
  })
  return preservedQuery
}

/**
 * 处理函数节点的路由更新
 * 🔥 切换函数时清空所有查询参数，避免参数污染
 */
const handleFunctionNodeRoute = (node: ServiceTree, source: string): void => {
  if (!node.full_code_path) {
    return
  }
  
  const targetPath = buildWorkspacePath(node.full_code_path)
  
  if (route.path === targetPath) {
    // 路由已匹配，直接触发节点点击加载详情（避免路由更新循环）
    applicationService.triggerNodeClick(node)
    return
  }
  
  const isLink = isLinkNavigation()
  
  // 🔥 构建查询参数
  // 只有 link 跳转时才保留参数，普通切换函数时清空所有参数
  const preservedQuery: Record<string, string | string[]> = isLink
    ? buildLinkNavigationQuery()  // link 跳转：保留所有参数（除了 _link_type）
    : {}                           // 普通切换函数：清空所有查询参数，避免参数污染
  
  const preserveParams = {
    table: false,      // 🔥 不再保留 table 参数
    search: false,     // 🔥 不再保留搜索参数
    state: false,      // 🔥 不再保留状态参数
    linkNavigation: isLink  // 只有 link 跳转时才保留参数
  }
  
  // 发出路由更新请求事件
  eventBus.emit(RouteEvent.updateRequested, {
    path: targetPath,
    query: preservedQuery,
    replace: true,
    preserveParams,
    source: source as any
  })
}

/**
 * 处理目录节点的路由更新
 */
const handlePackageNodeRoute = (node: ServiceTree, source: string): void => {
  if (!node.full_code_path) return
  
  const targetPath = buildWorkspacePath(node.full_code_path)
  if (route.path === targetPath) {
    applicationService.triggerNodeClick(node)
    return
  }
  
  eventBus.emit(RouteEvent.updateRequested, {
    path: targetPath,
    query: {},
    replace: true,
    preserveParams: {
      table: false,
      search: false,
      state: false,
      linkNavigation: false
    },
    source: source as any
  })
}

// 事件处理
const handleNodeClick = (node: ServiceTreeType) => {
  // 转换为新架构的 ServiceTree 类型
  const serviceTree: ServiceTree = node as any
  
  if (serviceTree.type === 'function') {
    handleFunctionNodeRoute(serviceTree, RouteSource.WORKSPACE_NODE_CLICK)
  } else if (serviceTree.type === 'package') {
    // 先设置当前函数，确保 PackageDetailView 能获取到数据
    applicationService.triggerNodeClick(serviceTree)
    handlePackageNodeRoute(serviceTree, RouteSource.WORKSPACE_NODE_CLICK_PACKAGE)
  } else {
    // 其他类型节点，只设置当前函数
    applicationService.triggerNodeClick(serviceTree)
  }
}

/**
 * 处理面包屑节点点击
 */
const handleBreadcrumbNodeClick = (node: ServiceTree) => {
  if (node.type === 'function') {
    handleFunctionNodeRoute(node, RouteSource.WORKSPACE_NODE_CLICK)
  } else if (node.type === 'package') {
    handlePackageNodeRoute(node, RouteSource.WORKSPACE_NODE_CLICK_PACKAGE)
  } else {
    applicationService.triggerNodeClick(node)
  }
}


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

// 处理发布到应用中心
const handlePublishToHub = (node: ServiceTreeType) => {
  publishSelectedNode.value = node
  publishToHubDialogVisible.value = true
}

// 处理推送到应用中心
const handlePushToHub = (node: ServiceTreeType) => {
  pushSelectedNode.value = node
  pushToHubDialogVisible.value = true
}

// 处理从应用中心拉取
const handlePullFromHub = () => {
  pastedHubLink.value = ''  // 清空之前的链接（手动打开对话框时）
  pullFromHubDialogVisible.value = true
}

// 处理刷新服务树（复制粘贴后需要刷新）
const handleRefreshTree = async () => {
  if (currentApp.value) {
    const app: App = {
      id: currentApp.value.id,
      user: currentApp.value.user,
      code: currentApp.value.code,
      name: currentApp.value.name
    }
    await domainService.loadServiceTree(app)
  }
}

// 处理变更记录
const handleUpdateHistory = (node?: ServiceTreeType) => {
  if (!currentApp.value) {
    ElMessage.warning('请先选择应用')
    return
  }
  
  // 🔥 修复：检查 appId 是否有效
  const appId = currentApp.value.id
  if (!appId || appId === 0) {
    console.error('[WorkspaceView] handleUpdateHistory: appId 无效', {
      currentApp: currentApp.value,
      appId
    })
    ElMessage.error('应用ID无效，无法加载变更记录。请刷新页面后重试。')
    return
  }
  
  if (node) {
    // 目录视角：显示指定目录的变更记录
    updateHistoryMode.value = 'directory'
    updateHistoryAppId.value = appId
    updateHistoryFullCodePath.value = node.full_code_path || ''
    updateHistoryAppVersion.value = ''
  } else {
    // App视角：显示工作空间的变更记录
    updateHistoryMode.value = 'app'
    updateHistoryAppId.value = appId
    updateHistoryAppVersion.value = '' // 空表示返回所有版本
    updateHistoryFullCodePath.value = ''
  }
  
  console.log('[WorkspaceView] 打开变更记录对话框', {
    mode: updateHistoryMode.value,
    appId: updateHistoryAppId.value,
    appVersion: updateHistoryAppVersion.value,
    fullCodePath: updateHistoryFullCodePath.value
  })
  
  updateHistoryDialogVisible.value = true
}

// 发布成功后的回调
const handlePublishSuccess = async () => {
  // 刷新服务目录树
  if (currentApp.value) {
    const app: App = {
      id: currentApp.value.id,
      user: currentApp.value.user,
      code: currentApp.value.code,
      name: currentApp.value.name
    }
    await domainService.loadServiceTree(app)
  }
}

// 推送成功后的回调
const handlePushSuccess = async () => {
  // 刷新服务目录树
  if (currentApp.value) {
    const app: App = {
      id: currentApp.value.id,
      user: currentApp.value.user,
      code: currentApp.value.code,
      name: currentApp.value.name
    }
    await domainService.loadServiceTree(app)
  }
}

// 拉取成功后的回调
const handlePullSuccess = async () => {
  // 清空粘贴的链接
  pastedHubLink.value = ''
  // 刷新服务目录树
  if (currentApp.value) {
    const app: App = {
      id: currentApp.value.id,
      user: currentApp.value.user,
      code: currentApp.value.code,
      name: currentApp.value.name
    }
    await domainService.loadServiceTree(app)
  }
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
// 🔥 阶段4：改为事件驱动，通过 RouteManager 统一处理路由更新
const backToList = () => {
  if (!currentFunction.value) return
  
  // 移除系统参数，保留其他参数
  const query: Record<string, string | string[]> = {}
  Object.keys(route.query).forEach(key => {
    if (key !== '_tab' && key !== '_id') {
      const value = route.query[key]
      if (value !== null && value !== undefined) {
        query[key] = Array.isArray(value) 
          ? value.filter(v => v !== null).map(v => String(v))
          : String(value)
      }
    }
  })
  
  const path = currentFunction.value.full_code_path 
    ? buildWorkspacePath(currentFunction.value.full_code_path)
    : ''
  
  // 🔥 发出路由更新请求事件
  eventBus.emit(RouteEvent.updateRequested, {
    path,
    query,
    replace: false,  // 返回列表使用 push，保留历史记录
    preserveParams: {
      state: true  // 保留状态参数
    },
    source: RouteSource.BACK_TO_LIST
  })
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
let unsubscribeAppInfoUpdated: (() => void) | null = null

// 🔥 重新关联 tabs 的 node 信息（使用 Composable）
// 🔥 不再使用 Tab，删除 restoreTabsNodes 函数

// 🔥 初始化 RouteManager（路由管理器）
let routeManager: RouteManager | null = null

onMounted(async () => {
  // 🔥 如果已存在 routeManager，先销毁（避免热更新时重复创建）
  if (routeManager) {
    routeManager.destroy()
    routeManager = null
  }
  
  // 🔥 初始化 RouteManager（不再使用 Tab）
  routeManager = new RouteManager(
    router,
    route,
    eventBus,
    () => null  // 🔥 Tab 功能已删除
  )
  
  // 🔥 开发环境下启用调试日志
  if (import.meta.env.DEV) {
    routeManager.setDebugLog(true)
  }
  
  // 监听函数加载完成事件
  // 🔥 监听函数加载完成事件，更新 currentFunctionDetail
  unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, (payload: { node: any, detail: FunctionDetail }) => {
    // 只有当加载的函数是当前函数时，才更新 currentFunctionDetail
    if (currentFunction.value && 
        (currentFunction.value.id === payload.node.id || 
         currentFunction.value.full_code_path === payload.node.full_code_path)) {
      currentFunctionDetail.value = payload.detail
      // 清除权限错误（因为函数已成功加载）
      permissionErrorStore.clearError()
    }
  })

  // 监听服务树加载完成事件
  unsubscribeServiceTreeLoaded = eventBus.on(WorkspaceEvent.serviceTreeLoaded, (payload: { app: any, tree: any[] }) => {
    // 状态已通过 StateManager 自动更新
  })
  
  // 监听应用切换事件，开始加载服务树
  unsubscribeAppSwitched = eventBus.on(WorkspaceEvent.appSwitched, (payload: { app: any }) => {
    // 应用切换事件处理
  })

  // 监听应用信息更新事件（用于更新应用列表中的 app.id）
  unsubscribeAppInfoUpdated = eventBus.on('workspace:app-info-updated' as any, (payload: { app: AppType }) => {
    // 更新应用列表中的 app 信息
    const index = appList.value.findIndex((a: AppType) => a.code === payload.app.code)
    if (index !== -1) {
      appList.value[index] = { ...appList.value[index], ...payload.app }
    }
  })

  // 从路由加载应用
  // 优化：如果路由中有应用信息，直接使用合并接口获取，不需要先加载整个应用列表
  await routingLoadAppFromRoute()
  
  // 注意：应用列表在用户点击应用切换器时才加载（AppSwitcher 的 handleVisibleChange 会触发 load-apps 事件）
  // 智能体列表在目录（package）节点时才加载（PackageDetailView 中处理）
  
  // 🔥 设置路由监听
  setupRouteWatch()
})

// 🔥 监听服务树变化，展开目录树
watch(() => serviceTree.value.length, (newLength: number) => {
  if (newLength > 0 && currentApp.value) {
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
    nextTick(() => {
      checkAndExpandForkedPaths()
    })
  }
})

// 🔥 监听当前函数变化，清除旧的函数详情和权限错误
watch(() => currentFunction.value?.id, (newId: number | undefined, oldId: number | undefined) => {
  // 当切换函数时，先清空旧的函数详情，避免显示上一个函数的详情
  if (newId !== oldId && oldId !== undefined) {
    // ⭐ 清空旧的函数详情，这样如果新函数加载失败，不会显示旧函数的详情
    currentFunctionDetail.value = null
    // 清除旧的权限错误（新的权限错误会在加载失败时重新设置）
    permissionErrorStore.clearError()
  }
})

// 🔥 监听 queryTab 变化，处理 create/edit/detail 模式
watch(queryTab, async (newTab: string, oldTab: string) => {
  if (newTab === 'create' || newTab === 'edit') {
    // create/edit 模式需要确保函数详情已加载
    if (!currentFunction.value) {
      return
    }
    
    // 如果函数详情未加载，触发加载
    if (!currentFunctionDetail.value) {
      await applicationService.handleNodeClick(currentFunction.value)
    }
  } else if (newTab === 'detail') {
    // detail 模式需要确保函数详情已加载，并且表格数据已加载
    if (!currentFunction.value) {
      return
    }
    
    // 如果函数详情未加载，触发加载
    if (!currentFunctionDetail.value) {
      await applicationService.handleNodeClick(currentFunction.value)
    }
    
    // detail 模式会在另一个 watch 中处理（监听 route.query.id）
  }
}, { immediate: false })

// 🔥 监听路由 query 变化，处理 _tab 参数
watch(() => route.query._tab, async (newTab: any) => {
  if (newTab === 'create' || newTab === 'edit') {
    // 确保当前函数已加载
    if (!currentFunction.value) {
      return
    }
    
    // 🔥 移除缓存后，切换函数时总是重新加载函数详情
    if (currentFunction.value && currentFunction.value.type === 'function') {
      await applicationService.handleNodeClick(currentFunction.value)
    }
  } else if (newTab === 'detail') {
    // detail 模式会在另一个 watch 中处理（监听 route.query.id）
    // 这里只需要确保函数详情已加载
    if (!currentFunction.value) {
      return
    }
    
    // 🔥 移除缓存后，切换函数时总是重新加载函数详情
    if (currentFunction.value && currentFunction.value.type === 'function') {
      await applicationService.handleNodeClick(currentFunction.value)
    }
  }
}, { immediate: false })


onUnmounted(() => {
  // 清理函数详情
  currentFunctionDetail.value = null
  
  if (unsubscribeFunctionLoaded) {
    unsubscribeFunctionLoaded()
  }
  if (unsubscribeServiceTreeLoaded) {
    unsubscribeServiceTreeLoaded()
  }
  if (unsubscribeAppSwitched) {
    unsubscribeAppSwitched()
  }
  if (unsubscribeAppInfoUpdated) {
    unsubscribeAppInfoUpdated()
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

.function-content-wrapper {
  flex: 1;
  overflow: hidden; /* 🔥 外层容器隐藏溢出，内层处理滚动 */
  display: flex;
  flex-direction: column;
  min-height: 0; /* 🔥 关键：允许 flex 子元素缩小 */
}

.function-content {
  flex: 1;
  overflow-y: auto !important; /* 🔥 强制允许垂直滚动，让搜索框和数据区一起滚动 */
  overflow-x: hidden;
  min-height: 0; /* 🔥 关键：允许 flex 子元素缩小 */
  height: 0; /* 🔥 关键：配合 flex: 1 和 min-height: 0，让滚动容器正确计算高度 */
  -webkit-overflow-scrolling: touch; /* 🔥 iOS 平滑滚动 */
}

/* 保留旧的类名以兼容（如果还有地方使用） */
.tabs-content-wrapper {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.tab-content {
  flex: 1;
  overflow-y: auto !important;
  overflow-x: hidden;
  min-height: 0;
  height: 0;
  -webkit-overflow-scrolling: touch;
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

.ai-chat-wrapper {
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
