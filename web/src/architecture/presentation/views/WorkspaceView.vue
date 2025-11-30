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
    <div class="workspace-header">
      <div class="header-left">
        <div class="logo">AI Agent OS</div>
      </div>
      <div class="header-right">
        <ThemeToggle />
        <el-dropdown @command="handleUserCommand">
          <span class="user-profile">
            <el-avatar :size="32" :src="userAvatar || undefined">{{ userInitials }}</el-avatar>
            <span class="username">{{ userName }}</span>
            <el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="settings">个人设置</el-dropdown-item>
              <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

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
        <div v-if="tabs.length > 0" class="workspace-tabs-container">
          <el-tabs
            v-model="activeTabId"
            type="card"
            editable
            class="workspace-tabs"
            @tab-click="handleTabClick"
            @edit="handleTabsEdit"
          >
            <el-tab-pane
              v-for="tab in tabs"
              :key="tab.id"
              :label="tab.title"
              :name="tab.id"
              :closable="tabs.length > 1"
            />
          </el-tabs>
        </div>
        
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
    <el-drawer
      v-model="detailDrawerVisible"
      :title="detailDrawerTitle"
      size="50%"
      destroy-on-close
      :modal="true"
      :close-on-click-modal="true"
      class="detail-drawer"
      :show-close="true"
      @close="handleDetailDrawerClose"
    >
      <template #header>
        <div class="drawer-header">
          <span class="drawer-title">{{ detailDrawerTitle }}</span>
          <div class="drawer-header-actions">
            <!-- 模式切换按钮 -->
            <div class="drawer-mode-actions">
              <el-button
                v-if="detailDrawerMode === 'read' && currentFunctionDetail?.callbacks?.includes('OnTableUpdateRow')"
                type="primary"
                size="small"
                @click="toggleDrawerMode('edit')"
              >
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>
              <el-button
                v-if="detailDrawerMode === 'edit'"
                size="small"
                @click="toggleDrawerMode('read')"
              >
                取消
              </el-button>
              <el-button
                v-if="detailDrawerMode === 'edit'"
                type="primary"
                size="small"
                :loading="drawerSubmitting"
                @click="submitDrawerEdit"
              >
                保存
              </el-button>
            </div>
            <!-- 导航按钮（上一个/下一个） -->
            <div class="drawer-navigation" v-if="detailTableData && detailTableData.length > 1 && detailDrawerMode === 'read'">
              <el-button
                size="small"
                :disabled="currentDetailIndex <= 0"
                @click="handleNavigateDetail('prev')"
              >
                <el-icon><ArrowLeft /></el-icon>
                上一个
              </el-button>
              <span class="nav-info">{{ (currentDetailIndex >= 0 ? currentDetailIndex + 1 : 0) }} / {{ detailTableData.length }}</span>
              <el-button
                size="small"
                :disabled="currentDetailIndex >= detailTableData.length - 1"
                @click="handleNavigateDetail('next')"
              >
                下一个
                <el-icon><ArrowRight /></el-icon>
              </el-button>
            </div>
          </div>
        </div>
      </template>

      <div class="detail-content">
        <!-- 详情模式 - 使用更美观的布局 -->
        <div v-if="detailDrawerMode === 'read'">
          <!-- 链接操作区域：收集所有 link 字段显示在顶部 -->
          <div v-if="detailLinkFields.length > 0" class="detail-links-section">
            <div class="links-section-title">相关链接</div>
            <div class="links-section-content">
              <LinkWidget
                v-for="linkField in detailLinkFields"
                :key="linkField.code"
                :field="linkField"
                :value="getDetailFieldValue(linkField.code)"
                :field-path="linkField.code"
                mode="detail"
                class="detail-link-item"
              />
            </div>
          </div>
          
          <!-- 字段网格（排除 link 字段） -->
          <div class="detail-fields-grid">
            <div
              v-for="field in detailFields.filter(f => f.widget?.type !== WidgetType.LINK)"
              :key="field.code"
              class="detail-field-row"
            >
              <div class="detail-field-label">
                {{ field.name }}
              </div>
              <div class="detail-field-value">
                <WidgetComponent
                  :field="field"
                  :value="getDetailFieldValue(field.code)"
                  mode="detail"
                  :user-info-map="detailUserInfoMap"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- 编辑模式（复用 FormRenderer，与旧版本一致） -->
        <div v-else class="edit-form-wrapper" v-loading="drawerSubmitting">
          <FormRenderer
            v-if="editFunctionDetail"
            ref="detailFormRendererRef"
            :key="`detail-edit-${detailRowData?.id || ''}-${detailDrawerMode}`"
            :function-detail="editFunctionDetail"
            :initial-data="detailRowData || {}"
            :show-submit-button="false"
            :show-reset-button="false"
            :show-share-button="false"
            :show-debug-button="false"
          />
          <el-empty v-else description="无法构建编辑表单" />
        </div>
      </div>

      <template #footer>
        <div class="drawer-footer">
          <template v-if="detailDrawerMode === 'read'">
            <el-button @click="detailDrawerVisible = false">关闭</el-button>
          </template>
          <template v-else>
            <el-button @click="toggleDrawerMode('read')">取消</el-button>
            <el-button type="primary" @click="submitDrawerEdit" :loading="drawerSubmitting">保存</el-button>
          </template>
        </div>
      </template>
    </el-drawer>

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
import { useRoute, useRouter, type LocationQueryValue } from 'vue-router'
import { extractWorkspacePath } from '@/utils/route'
import { ElMessage, ElMessageBox, ElNotification, ElDialog, ElForm, ElFormItem, ElInput, ElButton, ElIcon, ElTabs, ElTabPane, ElDrawer, ElDropdown, ElDropdownMenu, ElDropdownItem, ElAvatar, ElEmpty } from 'element-plus'
import { InfoFilled, ArrowDown, Edit, ArrowLeft, ArrowRight } from '@element-plus/icons-vue'
import { eventBus, WorkspaceEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import { apiClient } from '../../infrastructure/apiClient'
import { useAuthStore } from '@/stores/auth'
import ServiceTreePanel from '@/components/ServiceTreePanel.vue'
import AppSwitcher from '@/components/AppSwitcher.vue'
import ThemeToggle from '@/components/ThemeToggle.vue'
import FunctionForkDialog from '@/components/FunctionForkDialog.vue'
import type { ServiceTreePanel as ServiceTreePanelType } from '@/components/ServiceTreePanel.vue'
import FormView from './FormView.vue'
import TableView from './TableView.vue'
import WidgetComponent from '../widgets/WidgetComponent.vue'
import LinkWidget from '@/core/widgets-v2/components/LinkWidget.vue'
import { WidgetType } from '@/core/constants/widget'
import { convertToFieldValue } from '@/utils/field'
import FormRenderer from '@/core/renderers-v2/FormRenderer.vue'
import { createServiceTree } from '@/api/service-tree'
import type { ServiceTree, App } from '../../domain/services/WorkspaceDomainService'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'
import type { App as AppType, CreateAppRequest, ServiceTree as ServiceTreeType, CreateServiceTreeRequest } from '@/types'
import type { FieldConfig, FieldValue } from '../../domain/types'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

// 依赖注入（使用 ServiceFactory 简化）
const stateManager = serviceFactory.getWorkspaceStateManager()
const domainService = serviceFactory.getWorkspaceDomainService()
const applicationService = serviceFactory.getWorkspaceApplicationService()
const tableApplicationService = serviceFactory.getTableApplicationService()
const tableStateManager = serviceFactory.getTableStateManager()

// 从状态管理器获取状态
const serviceTree = computed(() => stateManager.getServiceTree())
const currentFunction = computed(() => stateManager.getCurrentFunction())
const currentAppFromState = computed(() => stateManager.getCurrentApp())
const tabs = computed(() => stateManager.getState().tabs)
const activeTabId = computed({
  get: () => stateManager.getState().activeTabId || '',
  set: (val) => {
    if (val) applicationService.activateTab(val)
  }
})

// 用户相关
const userName = computed(() => authStore.userName || 'User')
const userAvatar = computed(() => authStore.user?.avatar || '')
const userInitials = computed(() => {
  const name = userName.value
  return name ? name.substring(0, 2).toUpperCase() : 'US'
})

const handleUserCommand = (command: string) => {
  if (command === 'logout') {
    handleLogout()
  } else if (command === 'settings') {
    router.push('/user/settings')
  }
}

const handleLogout = async () => {
  try {
    await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await authStore.logout()
  } catch (error) {
    // 忽略取消操作
  }
}

// Tab 点击处理
const handleTabClick = (tab: any) => {
  console.log('[WorkspaceView] handleTabClick 开始', { tabName: tab.name, tab })
  if (tab.name) {
    console.log('[WorkspaceView] handleTabClick 调用 activateTab', { tabId: tab.name })
    applicationService.activateTab(tab.name as string)
  } else {
    console.warn('[WorkspaceView] handleTabClick tab.name 为空', { tab })
  }
}

// Tab 编辑处理（添加/删除）
const handleTabsEdit = (targetName: string | undefined, action: 'remove' | 'add') => {
  if (action === 'remove' && targetName) {
    applicationService.closeTab(targetName)
  }
}

// 状态保存与恢复
watch(() => stateManager.getState().activeTabId, async (newId, oldId) => {
  console.log('[WorkspaceView] watch activeTabId 触发', { oldId, newId, currentRoute: route.path, currentQuery: route.query })
  
  // 1. 保存旧 Tab 数据
  if (oldId) {
    const oldTab = tabs.value.find(t => t.id === oldId)
    if (oldTab && oldTab.node) {
       const detail = stateManager.getFunctionDetail(oldTab.node)
       if (detail?.template_type === 'form') {
         // 深度克隆，避免引用问题
         const currentState = serviceFactory.getFormStateManager().getState()
         oldTab.data = JSON.parse(JSON.stringify({
           data: Array.from(currentState.data.entries()), // Map 转 Array 以便序列化
           errors: Array.from(currentState.errors.entries()),
           submitting: currentState.submitting
         }))
       } else if (detail?.template_type === 'table') {
         const currentState = serviceFactory.getTableStateManager().getState()
         oldTab.data = JSON.parse(JSON.stringify(currentState))
       }
    }
  }

  // 2. 恢复新 Tab 数据
  if (newId) {
    const newTab = tabs.value.find(t => t.id === newId)
    if (newTab && newTab.data && newTab.node) {
       const detail = stateManager.getFunctionDetail(newTab.node)
       if (detail?.template_type === 'form') {
          // 恢复 Form 数据
          const savedState = newTab.data
          serviceFactory.getFormStateManager().setState({
            data: new Map(savedState.data),
            errors: new Map(savedState.errors),
            submitting: savedState.submitting
          })
       } else if (detail?.template_type === 'table') {
          // 恢复 Table 数据
          serviceFactory.getTableStateManager().setState(newTab.data)
       }
    } else {
      // 🔥 如果没有保存的数据，不要清空 FormState
      // 因为 FormView 会在 onMounted 时根据 URL 参数初始化表单
      // 如果这里清空了，会导致 URL 参数被覆盖
      // 让 FormView 自己处理初始化逻辑
    }
    
    // 更新路由参数（如果需要）
    // 🔥 注意：路由更新主要通过事件监听器（WorkspaceEvent.tabActivated）处理
    // 这里作为备用方案，确保路由更新
    if (newTab && newTab.path) {
      const path = newTab.path.startsWith('/') ? newTab.path : `/${newTab.path}`
      const targetPath = `/workspace${path}`
      // 🔥 检查当前路由是否已经是目标路由，避免重复导航
      // 同时检查 query 参数，如果有 _tab 参数需要清除
      const currentPath = route.path
      const hasQueryTab = !!route.query._tab
      const needsUpdate = currentPath !== targetPath || hasQueryTab
      
      console.log('[WorkspaceView] watch activeTabId 路由更新检查', {
        newTabPath: newTab.path,
        path,
        targetPath,
        currentPath,
        hasQueryTab,
        needsUpdate
      })
      
      if (needsUpdate) {
        console.log('[WorkspaceView] watch activeTabId 执行路由更新', { from: currentPath, to: targetPath })
        // 使用 replace 避免产生大量历史记录，并清除 query 参数
        router.replace({ path: targetPath, query: {} }).catch((err) => {
          console.error('[WorkspaceView] watch activeTabId 路由更新失败', err)
        })
      } else {
        console.log('[WorkspaceView] watch activeTabId 路由无需更新', { currentPath, targetPath })
      }
    } else {
      console.warn('[WorkspaceView] watch activeTabId newTab 或 path 不存在', { newTab })
    }
  }
})

// 将 App 类型转换为 AppType 类型（用于 AppSwitcher）
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
  const activeTab = tabs.value.find(t => t.id === activeTabIdValue)
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

// ...

const detailDrawerVisible = ref(false)
const detailDrawerTitle = ref('详情')
const detailRowData = ref<Record<string, any> | null>(null)
const detailFields = ref<FieldConfig[]>([])
const detailOriginalRow = ref<Record<string, any> | null>(null)
const detailDrawerMode = ref<'read' | 'edit'>('read')
const drawerSubmitting = ref(false)
const detailFormRendererRef = ref<InstanceType<typeof FormRenderer> | null>(null)
// 🔥 详情抽屉的用户信息映射（用于 UserWidget 批量查询优化）
const detailUserInfoMap = ref<Map<string, any>>(new Map())
// 🔥 详情抽屉的表格数据和索引（用于上一条下一条导航）
const detailTableData = ref<any[]>([])
const currentDetailIndex = ref<number>(-1)

// 🔥 queryTab：当前激活的Tab模式（用于路由查询参数，控制 create/edit 等模式）
// 🔥 使用 _tab 作为系统参数，避免与后端参数冲突
const queryTab = computed(() => (route.query._tab as string) || 'run')

// 🔥 编辑模式相关
const editRowId = computed(() => {
  const id = route.query.id || route.query._id
  return id ? Number(id) : null
})

// 🔥 详情模式相关
const detailRowId = computed(() => {
  // 🔥 使用 _id 作为系统参数，避免与后端参数冲突
  const id = route.query._id
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
          initialData[fieldCode] = value === 'true' || value === '1' || value === 1 || value === true
        } else {
          initialData[fieldCode] = value
        }
      }
    })
  }
  
  return initialData
})

/**
 * 详情页的 Link 字段（用于顶部链接区域显示）
 */
const detailLinkFields = computed(() => {
  return detailFields.value.filter((field: FieldConfig) => field.widget?.type === WidgetType.LINK)
})

// 创建目录相关
const createDirectoryDialogVisible = ref(false)
const creatingDirectory = ref(false)
const currentParentNode = ref<ServiceTreeType | null>(null)
const createDirectoryForm = ref<CreateServiceTreeRequest>({
  user: '',
  app: '',
  name: '',
  code: '',
  parent_id: 0,
  description: '',
  tags: ''
})

// Fork 函数组相关
const forkDialogVisible = ref(false)
const forkSourceGroupCode = ref('')
const forkSourceGroupName = ref('')

// ServiceTreePanel 引用（用于展开路径）
const serviceTreePanelRef = ref<InstanceType<typeof ServiceTreePanel> | null>(null)

// 🔥 编辑模式的函数详情（从 response 字段中筛选可编辑的字段）
const editFunctionDetail = computed<FunctionDetail | null>(() => {
  const current = currentFunctionDetail.value
  if (!current) return null
  
  // 如果是 table 类型，从 response 字段中筛选可编辑的字段
  if (current.template_type === 'table') {
    const fields = (current.response || []) as FieldConfig[]
    const editableFields = fields.filter(field => {
      const permission = field.table_permission
      return !permission || permission === '' || permission === 'update'
    })
    return {
      ...current,
      template_type: 'form',
      request: editableFields,
      response: []
    }
  }
  
  // 如果是 form 类型，直接使用 request 字段
  if (current.template_type === 'form') {
    return current
  }
  
  return null
})

// 监听 Tab 打开/激活事件，更新路由
onMounted(() => {
  eventBus.on(WorkspaceEvent.tabOpened, ({ tab, shouldUpdateRoute }: { tab: any, shouldUpdateRoute?: boolean }) => {
    if (shouldUpdateRoute && tab.path) {
      // 🔥 更新路由到新打开的 Tab
      const path = tab.path.startsWith('/') ? tab.path : `/${tab.path}`
      const targetPath = `/workspace${path}`
      router.push(targetPath).catch(() => {})
    }
  })

  eventBus.on(WorkspaceEvent.tabActivated, ({ tab, shouldUpdateRoute }: { tab: any, shouldUpdateRoute?: boolean }) => {
    console.log('[WorkspaceView] tabActivated 事件触发', { 
      tab, 
      shouldUpdateRoute, 
      tabPath: tab?.path,
      currentRoute: route.path,
      currentQuery: route.query
    })
    
    if (shouldUpdateRoute && tab && tab.path) {
      // 🔥 更新路由到激活的 Tab
      const path = tab.path.startsWith('/') ? tab.path : `/${tab.path}`
      const targetPath = `/workspace${path}`
      // 🔥 检查当前路由是否已经是目标路由，避免重复导航
      // 同时检查 query 参数，如果有 _tab 参数需要清除
      const currentPath = route.path
      const hasQueryTab = !!route.query._tab
      const needsUpdate = currentPath !== targetPath || hasQueryTab
      
      console.log('[WorkspaceView] tabActivated 路由更新检查', {
        tabPath: tab.path,
        path,
        targetPath,
        currentPath,
        hasQueryTab,
        needsUpdate
      })
      
      if (needsUpdate) {
        console.log('[WorkspaceView] tabActivated 执行路由更新', { from: currentPath, to: targetPath })
        // 使用 replace 避免产生大量历史记录，并清除 query 参数
        router.replace({ path: targetPath, query: {} }).catch((err) => {
          console.error('[WorkspaceView] tabActivated 路由更新失败', err)
        })
      } else {
        console.log('[WorkspaceView] tabActivated 路由无需更新', { currentPath, targetPath })
      }
    } else {
      console.warn('[WorkspaceView] tabActivated 跳过路由更新', { 
        shouldUpdateRoute, 
        hasTab: !!tab, 
        hasPath: !!tab?.path 
      })
    }
  })

  // 🔥 监听节点点击事件，直接更新路由（作为备用方案，确保路由更新）
  eventBus.on(WorkspaceEvent.nodeClicked, ({ node }: { node: any }) => {
    if (node && node.type === 'function' && node.full_code_path) {
      const targetPath = `/workspace${node.full_code_path}`
      // 🔥 检查当前路由是否已经是目标路由，避免重复导航
      if (route.path !== targetPath) {
        router.push(targetPath).catch(() => {})
      }
    }
  })

  // 监听表格详情事件
  eventBus.on('table:detail-row', async ({ row, index, tableData }: { row: Record<string, any>, index?: number, tableData?: any[] }) => {
    if (!currentFunctionDetail.value) return
    
    detailRowData.value = row
    detailOriginalRow.value = JSON.parse(JSON.stringify(row))
    detailDrawerTitle.value = currentFunctionDetail.value.name || '详情'
    detailFields.value = (currentFunctionDetail.value.response || []) as FieldConfig[]
    
    // 🔥 更新 URL 为 ?_tab=detail&_id=xxx（用于分享）
    // 🔥 使用 _tab 和 _id 作为系统参数，避免与后端参数冲突
    if (currentFunction.value) {
      const id = row.id || row._id
      if (id) {
        const query = { ...route.query, _tab: 'detail', _id: String(id) }
        router.replace({ query }).catch(() => {})
      }
    }
    
    // 🔥 保存表格数据和索引（用于上一条下一条导航）
    if (tableData && Array.isArray(tableData) && tableData.length > 0) {
      detailTableData.value = tableData
      if (typeof index === 'number' && index >= 0) {
        currentDetailIndex.value = index
      } else {
        // 如果没有传递 index，尝试从 tableData 中查找
        const idField = detailFields.value.find(f => f.code === 'id' || f.widget?.type === 'number')
        if (idField && row[idField.code]) {
          const foundIndex = tableData.findIndex((r: any) => r[idField.code] === row[idField.code])
          currentDetailIndex.value = foundIndex >= 0 ? foundIndex : -1
        } else {
          // 如果没有 id 字段，尝试通过对象匹配
          const foundIndex = tableData.findIndex((r: any) => JSON.stringify(r) === JSON.stringify(row))
          currentDetailIndex.value = foundIndex >= 0 ? foundIndex : -1
        }
      }
    } else {
      // 如果没有传递 tableData，尝试从 StateManager 获取
      try {
        const tableStateManager = serviceFactory.getTableStateManager()
        // 🔥 注意：TableStateManager 使用 data 字段存储表格数据，不是 tableData
        const tableData = tableStateManager.getData() || []
        if (tableData && Array.isArray(tableData) && tableData.length > 0) {
          detailTableData.value = tableData
          const idField = detailFields.value.find(f => f.code === 'id' || f.widget?.type === 'number')
          if (idField && row[idField.code]) {
            const foundIndex = tableData.findIndex((r: any) => r[idField.code] === row[idField.code])
            currentDetailIndex.value = foundIndex >= 0 ? foundIndex : -1
          } else {
            // 如果没有 id 字段，尝试通过对象匹配
            const foundIndex = tableData.findIndex((r: any) => JSON.stringify(r) === JSON.stringify(row))
            currentDetailIndex.value = foundIndex >= 0 ? foundIndex : -1
          }
        } else {
          detailTableData.value = []
          currentDetailIndex.value = -1
          console.warn('[WorkspaceView] StateManager 中没有表格数据')
        }
      } catch (error) {
        console.error('[WorkspaceView] 获取表格数据失败', error)
        detailTableData.value = []
        currentDetailIndex.value = -1
      }
    }
    
    // 🔥 收集详情中的用户字段，批量查询用户信息
    const userFields = detailFields.value.filter(f => f.widget?.type === 'user')
    if (userFields.length > 0) {
      const usernames: string[] = []
      userFields.forEach(field => {
        const value = row[field.code]
        if (value) {
          if (Array.isArray(value)) {
            usernames.push(...value.map(v => String(v)))
          } else {
            usernames.push(String(value))
          }
        }
      })
      
      if (usernames.length > 0) {
        try {
          const { useUserInfoStore } = await import('@/stores/userInfo')
          const userInfoStore = useUserInfoStore()
          const users = await userInfoStore.batchGetUserInfo([...new Set(usernames)])
          // 更新到 detailUserInfoMap
          detailUserInfoMap.value = new Map()
          users.forEach(user => {
            detailUserInfoMap.value.set(user.username, user)
          })
        } catch (error) {
          console.error('[WorkspaceView] 加载详情用户信息失败', error)
        }
      }
    }
    
    // 重置为只读模式
    detailDrawerMode.value = 'read'
    detailDrawerVisible.value = true
  })
  
  // 🔥 监听 URL 参数变化，自动打开详情抽屉（用于分享链接）
  // 🔥 使用 _tab 和 _id 作为系统参数，避免与后端参数冲突
  watch([() => route.query._tab, () => route.query._id, currentFunctionDetail], async ([tab, id, detail]: [any, any, any]) => {
    if (tab === 'detail' && id && detail && detail.template_type === 'table') {
      // 确保函数详情已加载
      if (!currentFunction.value) {
        console.log('[WorkspaceView] tab=detail 但当前函数不存在，等待函数加载')
        return
      }
      
      const rowId = Number(id)
      if (isNaN(rowId)) {
        console.warn('[WorkspaceView] tab=detail 但 id 无效:', id)
        return
      }
      
      // 从表格数据中查找对应 id 的记录
      try {
        const tableStateManager = serviceFactory.getTableStateManager()
        let tableData = tableStateManager.getData() || []
        
        // 尝试通过 id 字段查找
        let targetRow = tableData.find((r: any) => r.id === rowId || r._id === rowId)
        
        // 如果当前页没有找到，尝试通过搜索 id 来加载数据
        if (!targetRow) {
          console.log('[WorkspaceView] 当前页没有找到 id 为', rowId, '的记录，尝试通过搜索加载')
          
          // 先等待表格数据加载完成（如果表格正在加载）
          let retries = 0
          while (tableData.length === 0 && retries < 10) {
            await nextTick()
            await new Promise(resolve => setTimeout(resolve, 300))
            tableData = tableStateManager.getData() || []
            targetRow = tableData.find((r: any) => r.id === rowId || r._id === rowId)
            if (targetRow) break
            retries++
          }
          
          // 如果还是没有找到，尝试通过搜索 id 来加载
          if (!targetRow && currentFunctionDetail.value) {
            console.log('[WorkspaceView] 通过搜索 id 字段加载数据')
            try {
              const tableApplicationService = serviceFactory.getTableApplicationService()
              // 🔥 通过搜索 id 字段来加载数据
              // 查找 id 字段
              const idField = currentFunctionDetail.value.response?.find((f: FieldConfig) => 
                f.code === 'id' || f.code.toLowerCase() === 'id'
              )
              
              if (idField) {
                // 设置搜索条件为 id = rowId
                const searchParams: Record<string, any> = {}
                searchParams[idField.code] = rowId
                
                // 加载数据（使用搜索参数）
                await tableApplicationService.loadData(
                  currentFunctionDetail.value,
                  searchParams, // 搜索参数
                  undefined, // 排序参数
                  { page: 1, pageSize: 20 } // 分页参数
                )
                
                // 重新获取数据
                tableData = tableStateManager.getData() || []
                targetRow = tableData.find((r: any) => r.id === rowId || r._id === rowId)
              }
            } catch (error) {
              console.error('[WorkspaceView] 通过搜索加载数据失败', error)
            }
          }
        }
        
        if (targetRow) {
          // 找到记录，打开详情抽屉
          const index = tableData.findIndex((r: any) => r.id === rowId || r._id === rowId)
          detailRowData.value = targetRow
          detailOriginalRow.value = JSON.parse(JSON.stringify(targetRow))
          detailDrawerTitle.value = detail.name || '详情'
          detailFields.value = (detail.response || []) as FieldConfig[]
          detailTableData.value = tableData
          currentDetailIndex.value = index >= 0 ? index : -1
          
          // 收集用户字段信息
          const userFields = detailFields.value.filter(f => f.widget?.type === 'user')
          if (userFields.length > 0) {
            const usernames: string[] = []
            userFields.forEach(field => {
              const value = targetRow[field.code]
              if (value) {
                if (Array.isArray(value)) {
                  usernames.push(...value.map(v => String(v)))
                } else {
                  usernames.push(String(value))
                }
              }
            })
            
            if (usernames.length > 0) {
              try {
                const { useUserInfoStore } = await import('@/stores/userInfo')
                const userInfoStore = useUserInfoStore()
                const users = await userInfoStore.batchGetUserInfo([...new Set(usernames)])
                detailUserInfoMap.value = new Map()
                users.forEach(user => {
                  detailUserInfoMap.value.set(user.username, user)
                })
              } catch (error) {
                console.error('[WorkspaceView] 加载详情用户信息失败', error)
              }
            }
          }
          
          detailDrawerMode.value = 'read'
          detailDrawerVisible.value = true
        } else {
          console.warn('[WorkspaceView] 未找到 id 为', rowId, '的记录')
          ElNotification.warning({
            title: '提示',
            message: `未找到 id 为 ${rowId} 的记录，可能不在当前页`
          })
        }
      } catch (error) {
        console.error('[WorkspaceView] 打开详情失败', error)
      }
    }
  }, { immediate: false })
})

// 切换抽屉模式
const toggleDrawerMode = (mode: 'read' | 'edit') => {
  if (mode === 'edit' && (!editFunctionDetail.value || !detailRowData.value)) {
    ElNotification.warning({
      title: '提示',
      message: '无法进入编辑模式'
    })
    return
  }
  detailDrawerMode.value = mode
}

// 导航详情（上一个/下一个）
const handleNavigateDetail = async (direction: 'prev' | 'next') => {
  if (detailTableData.value.length === 0) return

  let newIndex = currentDetailIndex.value
  if (direction === 'prev' && newIndex > 0) {
    newIndex--
  } else if (direction === 'next' && newIndex < detailTableData.value.length - 1) {
    newIndex++
  } else {
    return
  }

  currentDetailIndex.value = newIndex
  const row = detailTableData.value[newIndex]
  detailRowData.value = row
  detailOriginalRow.value = JSON.parse(JSON.stringify(row))
  detailDrawerMode.value = 'read'  // 切换记录时，重置为查看模式
  
  // 🔥 收集新行的用户字段并查询用户信息
  const userFields = detailFields.value.filter(f => f.widget?.type === 'user')
  if (userFields.length > 0) {
    const usernames: string[] = []
    userFields.forEach(field => {
      const value = row[field.code]
      if (value) {
        if (Array.isArray(value)) {
          usernames.push(...value.map(v => String(v)))
        } else {
          usernames.push(String(value))
        }
      }
    })
    
    if (usernames.length > 0) {
      try {
        const { useUserInfoStore } = await import('@/stores/userInfo')
        const userInfoStore = useUserInfoStore()
        const users = await userInfoStore.batchGetUserInfo([...new Set(usernames)])
        // 更新到 detailUserInfoMap
        detailUserInfoMap.value = new Map()
        users.forEach(user => {
          detailUserInfoMap.value.set(user.username, user)
        })
      } catch (error) {
        console.error('[WorkspaceView] 加载详情用户信息失败', error)
      }
    }
  }
}

// 提交编辑（复用 FormRenderer 逻辑）
const submitDrawerEdit = async () => {
  if (!currentFunctionDetail.value || !detailRowData.value || !detailFormRendererRef.value) {
    ElMessage.error('编辑表单未准备就绪')
    return
  }
  
  try {
    drawerSubmitting.value = true
    const submitData = detailFormRendererRef.value.prepareSubmitDataWithTypeConversion()
    const oldValues = detailOriginalRow.value
      ? JSON.parse(JSON.stringify(detailOriginalRow.value))
      : undefined
    const updatedRow = await tableApplicationService.updateRow(
      currentFunctionDetail.value,
      detailRowData.value.id,
      submitData,
      oldValues
    )
    if (updatedRow) {
      detailRowData.value = { ...updatedRow }
      detailOriginalRow.value = JSON.parse(JSON.stringify(updatedRow))
      await refreshDetailRowData()
      ElNotification.success({
        title: '成功',
        message: '更新成功'
      })
      detailDrawerMode.value = 'read'
      detailDrawerVisible.value = false
    }
  } catch (error: any) {
    console.error('更新失败:', error)
    ElNotification.error({
      title: '错误',
      message: error?.response?.data?.message || error?.message || '更新失败'
    })
  } finally {
    drawerSubmitting.value = false
  }
}

const refreshDetailRowData = async (): Promise<void> => {
  if (!detailRowData.value) return
  const currentId = detailRowData.value.id
  if (currentId === undefined || currentId === null) return
  const state = tableStateManager?.getState?.()
  const tableData = state?.tableData
  if (!Array.isArray(tableData)) {
    return
  }
  const updatedRow = tableData.find((row: any) => String(row.id) === String(currentId))
  if (updatedRow) {
    detailRowData.value = { ...updatedRow }
    detailOriginalRow.value = JSON.parse(JSON.stringify(updatedRow))
  }
}

// 获取详情字段值
const getDetailFieldValue = (fieldCode: string): FieldValue => {
  if (!detailRowData.value) return { raw: null, display: '', meta: {} }
  const value = detailRowData.value[fieldCode]
  return { 
    raw: value, 
    display: typeof value === 'object' ? JSON.stringify(value) : String(value ?? ''), 
    meta: {} 
  }
}

// 在详情抽屉中点击编辑
const handleDrawerEdit = () => {
  // 已废弃，改用 toggleDrawerMode('edit')
}

// 应用列表管理
const appList = ref<AppType[]>([])
const loadingApps = ref(false)

// 🔥 正在切换的目标应用 ID，用于解决路由和状态更新的竞态问题
const pendingAppId = ref<number | string | null>(null)

// 创建应用对话框
const createAppDialogVisible = ref(false)
const creatingApp = ref(false)
const createAppForm = ref<CreateAppRequest>({
  code: '',
  name: ''
})

// 转换 loadingTree 为 boolean (避免 computed 类型问题)
const loading = computed(() => stateManager.isLoading()) // 🔥 修复 loading 定义

// 事件处理
const handleNodeClick = (node: ServiceTreeType) => {
  // 转换为新架构的 ServiceTree 类型
  const serviceTree: ServiceTree = node as any
  applicationService.triggerNodeClick(serviceTree)
}

// 处理创建目录
const handleCreateDirectory = (parentNode?: ServiceTreeType) => {
  if (!currentApp.value) {
    ElNotification.warning({
      title: '提示',
      message: '请先选择一个应用'
    })
    return
  }
  currentParentNode.value = parentNode || null
  createDirectoryForm.value = {
    user: currentApp.value.user,
    app: currentApp.value.code,
    name: '',
    code: '',
    parent_id: parentNode ? Number(parentNode.id) : 0,
    description: '',
    tags: ''
  }
  createDirectoryDialogVisible.value = true
}

// 重置创建目录表单
const resetCreateDirectoryForm = () => {
  createDirectoryForm.value = {
    user: currentApp.value?.user || '',
    app: currentApp.value?.code || '',
    name: '',
    code: '',
    parent_id: 0,
    description: '',
    tags: ''
  }
  currentParentNode.value = null
}

// 提交创建目录
const handleSubmitCreateDirectory = async () => {
  if (!currentApp.value) {
    ElNotification.warning({
      title: '提示',
      message: '请先选择一个应用'
    })
    return
  }
  
  if (!createDirectoryForm.value.name || !createDirectoryForm.value.code) {
    ElNotification.warning({
      title: '提示',
      message: '请输入目录名称和代码'
    })
    return
  }
  
  // 验证代码格式
  if (!/^[a-z0-9_]+$/.test(createDirectoryForm.value.code)) {
    ElNotification.warning({
      title: '提示',
      message: '目录代码只能包含小写字母、数字和下划线'
    })
    return
  }

  try {
    creatingDirectory.value = true
    const requestData: CreateServiceTreeRequest = {
      user: currentApp.value.user,
      app: currentApp.value.code,
      name: createDirectoryForm.value.name,
      code: createDirectoryForm.value.code,
      parent_id: createDirectoryForm.value.parent_id || 0,
      description: createDirectoryForm.value.description || '',
      tags: createDirectoryForm.value.tags || ''
    }
    
    await createServiceTree(requestData)
    ElNotification.success({
      title: '成功',
      message: '创建服务目录成功'
    })
    createDirectoryDialogVisible.value = false
    resetCreateDirectoryForm()
    
    // 🔥 刷新服务目录树
    if (currentApp.value) {
      await applicationService.triggerAppSwitch({
        id: currentApp.value.id,
        user: currentApp.value.user,
        code: currentApp.value.code,
        name: currentApp.value.name
      })
    }
  } catch (error: any) {
    console.error('[WorkspaceView] 创建服务目录失败', error)
    const errorMessage = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '创建服务目录失败'
    ElNotification.error({
      title: '错误',
      message: errorMessage
    })
  } finally {
    creatingDirectory.value = false
  }
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
    applicationService.triggerAppSwitch({
      id: currentApp.value.id,
      user: currentApp.value.user,
      code: currentApp.value.code,
      name: currentApp.value.name
    })
  }
  ElNotification.success({
    title: '成功',
    message: '克隆完成！请刷新页面查看新功能'
  })
}

// 🔥 检查并展开 forked 路径
const checkAndExpandForkedPaths = () => {
  const forkedParam = route.query._forked as string
  if (!forkedParam) return
  
  console.log('[WorkspaceView] 检查 forked 参数:', forkedParam)
  console.log('[WorkspaceView] 当前应用:', currentApp.value ? `${currentApp.value.user}/${currentApp.value.code}` : 'null')
  console.log('[WorkspaceView] serviceTree 长度:', serviceTree.value.length)
  console.log('[WorkspaceView] serviceTreePanelRef:', serviceTreePanelRef.value)
  
  // 检查当前应用是否匹配 URL 中的应用
  const pathSegments = extractWorkspacePath(route.path).split('/').filter(Boolean)
  if (pathSegments.length >= 2) {
    const [urlUser, urlApp] = pathSegments
    if (currentApp.value && (currentApp.value.user !== urlUser || currentApp.value.code !== urlApp)) {
      console.log('[WorkspaceView] ⚠️ 应用不匹配，等待应用切换完成')
      console.log('[WorkspaceView]    URL 应用:', `${urlUser}/${urlApp}`)
      console.log('[WorkspaceView]    当前应用:', `${currentApp.value.user}/${currentApp.value.code}`)
      return // 应用不匹配，不展开
    }
  }
  
  if (forkedParam && serviceTree.value.length > 0 && serviceTreePanelRef.value && currentApp.value) {
    const forkedPaths = decodeURIComponent(forkedParam).split(',').filter(Boolean)
    console.log('[WorkspaceView] 解析后的路径列表:', forkedPaths)
    
    // 验证路径是否属于当前应用
    const validPaths = forkedPaths.filter(path => {
      const pathMatch = path.match(/^\/([^/]+)\/([^/]+)/)
      if (pathMatch) {
        const [, pathUser, pathApp] = pathMatch
        const isValid = pathUser === currentApp.value?.user && pathApp === currentApp.value?.code
        if (!isValid) {
          console.log('[WorkspaceView] ⚠️ 路径不属于当前应用，跳过:', path)
        }
        return isValid
      }
      return false
    })
    
    if (validPaths.length > 0) {
      console.log('[WorkspaceView] 有效路径列表:', validPaths)
      nextTick(() => {
        setTimeout(() => {
          if (serviceTreePanelRef.value && serviceTreePanelRef.value.expandPaths) {
            console.log('[WorkspaceView] 调用 expandPaths')
            serviceTreePanelRef.value.expandPaths(validPaths)
          } else {
            console.log('[WorkspaceView] ⚠️ serviceTreePanelRef 或 expandPaths 不存在')
          }
        }, 500) // 延迟确保树完全渲染
      })
    } else {
      console.log('[WorkspaceView] ⚠️ 没有有效的路径可以展开')
    }
  }
}

// 处理复制链接
const handleCopyLink = (node: ServiceTreeType) => {
  const link = `${window.location.origin}/workspace${node.full_code_path}`
  navigator.clipboard.writeText(link).then(() => {
    ElNotification.success({
      title: '成功',
      message: '链接已复制到剪贴板'
    })
  }).catch(() => {
    ElNotification.error({
      title: '错误',
      message: '复制链接失败'
    })
  })
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

// 🔥 处理详情抽屉关闭（移除 URL 参数）
const handleDetailDrawerClose = () => {
  // 如果当前 URL 有 _tab=detail 参数，移除它
  if (route.query._tab === 'detail') {
    const query = { ...route.query }
    delete query._tab
    delete query._id
    router.replace({ query }).catch(() => {})
  }
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

// 加载应用列表
const loadAppList = async (): Promise<void> => {
  try {
    loadingApps.value = true
    const response = await apiClient.get<any>('/api/v1/app/list', {
      page_size: 200,
      page: 1
    })
    
    // API 返回的是分页对象 { page, page_size, total_count, items: App[] }
    // 需要提取 items 数组
    if (response && typeof response === 'object') {
      if (Array.isArray(response)) {
        appList.value = response
      } else if ('items' in response && Array.isArray(response.items)) {
        appList.value = response.items
      } else {
        appList.value = []
      }
    } else {
      appList.value = []
    }
  } catch (error) {
    console.error('[WorkspaceView] 加载应用列表失败', error)
    ElNotification.error({
      title: '错误',
      message: '加载应用列表失败'
    })
    appList.value = []
  } finally {
    loadingApps.value = false
  }
}

// 切换应用
const handleSwitchApp = async (app: AppType): Promise<void> => {
  const targetAppId = app.id
  
  // 🔥 检查当前应用是否已经是目标应用，避免重复切换
  const currentAppState = currentApp.value
  if (currentAppState && String(currentAppState.id) === String(targetAppId)) {
    console.log('[WorkspaceView] 当前应用已经是目标应用，无需切换')
    return
  }

  try {
    const appForService: App = {
      id: app.id,
      user: app.user,
      code: app.code,
      name: app.name
    }
    
    // 切换应用（这会触发服务树加载）
    await applicationService.triggerAppSwitch(appForService)
    
    // 更新路由
    const targetPath = `/workspace/${app.user}/${app.code}`
    if (route.path !== targetPath) {
      await router.push(targetPath)
    }
  } catch (error) {
    console.error('[WorkspaceView] 切换应用失败', error)
  }
}

// 显示创建应用对话框
const showCreateAppDialog = (): void => {
  resetCreateAppForm()
  createAppDialogVisible.value = true
}

// 重置创建应用表单
const resetCreateAppForm = (): void => {
  createAppForm.value = {
    code: '',
    name: ''
  }
}

// 提交创建应用
const submitCreateApp = async (): Promise<void> => {
  if (!createAppForm.value.name || !createAppForm.value.code) {
    ElNotification.warning({
      title: '提示',
      message: '请填写应用名称和应用代码'
    })
    return
  }

  try {
    creatingApp.value = true
    await apiClient.post('/api/v1/app/create', createAppForm.value)
    ElNotification.success({
      title: '成功',
      message: '应用创建成功'
    })
    createAppDialogVisible.value = false
    
    // 刷新应用列表
    await loadAppList()
    
    // 如果应用列表中有新创建的应用，自动切换
    const newApp = appList.value.find(
      (a: AppType) => a.code === createAppForm.value.code
    )
    if (newApp) {
      await handleSwitchApp(newApp)
    }
  } catch (error: any) {
    const errorMessage = error?.response?.data?.message || '创建应用失败'
    ElNotification.error({
      title: '错误',
      message: errorMessage
    })
  } finally {
    creatingApp.value = false
  }
}

// 更新应用（重新编译）
const handleUpdateApp = async (app: AppType): Promise<void> => {
  try {
    await apiClient.post(`/api/v1/app/update/${app.code}`, {})
    ElNotification.success({
      title: '成功',
      message: '应用更新成功'
    })
  } catch (error: any) {
    const errorMessage = error?.response?.data?.message || '更新应用失败'
    ElNotification.error({
      title: '错误',
      message: errorMessage
    })
  }
}

// 删除应用
const handleDeleteApp = async (app: AppType): Promise<void> => {
  try {
    await ElMessageBox.confirm(
      `确定要删除应用 "${app.name}" 吗？此操作不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    await apiClient.delete(`/api/v1/app/delete/${app.code}`)
    ElNotification.success({
      title: '成功',
      message: '应用删除成功'
    })
    
    // 刷新应用列表
    await loadAppList()
    
    // 如果删除的是当前应用，切换到第一个应用或清空
    if (currentApp.value && currentApp.value.id === app.id) {
      if (appList.value.length > 0) {
        await handleSwitchApp(appList.value[0])
      } else {
        await router.push('/workspace')
      }
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      const errorMessage = error?.response?.data?.message || '删除应用失败'
      ElNotification.error({
        title: '错误',
        message: errorMessage
      })
    }
  }
}

// 递归查找节点
const findNodeByPath = (tree: ServiceTreeType[], path: string): ServiceTreeType | null => {
  for (const node of tree) {
    // 移除路径开头的斜杠进行比较
    const nodePath = (node.full_code_path || '').replace(/^\/+/, '')
    const targetPath = path.replace(/^\/+/, '')
    
    if (nodePath === targetPath && node.type === 'function') {
      return node
    }
    if (node.children && node.children.length > 0) {
      const found = findNodeByPath(node.children, path)
      if (found) return found
    }
  }
  return null
}

// 防重复调用保护
let isLoadingAppFromRoute = false
let lastProcessedPath = ''

// 从路由解析应用并加载
const loadAppFromRoute = async () => {
  // 🔥 防止重复调用
  if (isLoadingAppFromRoute) {
    return
  }
  
  // 提取路径
  const fullPath = extractWorkspacePath(route.path)
  
  // 🔥 如果路径没有变化，不重复处理
  if (fullPath === lastProcessedPath) {
    return
  }
  
  if (!fullPath) {
    return
  }

  const pathSegments = fullPath.split('/').filter(Boolean)
  if (pathSegments.length < 2) {
    return
  }

  const [user, appCode] = pathSegments
  
  try {
    isLoadingAppFromRoute = true
    
    // 确保应用列表已加载
    if (appList.value.length === 0) {
      await loadAppList()
    }
    
    // 从已加载的应用列表中查找
    const app = appList.value.find((a: AppType) => a.user === user && a.code === appCode)
    
    if (!app) {
      console.warn('[WorkspaceView] 未找到应用:', user, appCode)
      return
    }
    
    const targetAppId = app.id
    let appSwitched = false

    // 🔥 检查当前应用是否已经是目标应用
    const currentAppState = currentApp.value
    if (!currentAppState || String(currentAppState.id) !== String(targetAppId)) {
        // 需要切换应用
        if (String(pendingAppId.value) !== String(targetAppId)) {
           pendingAppId.value = targetAppId
           try {
             const appForService: App = {
               id: app.id,
               user: app.user,
               code: app.code,
               name: app.name
             }
             await applicationService.triggerAppSwitch(appForService)
             appSwitched = true
           } catch (error) {
             console.error('[WorkspaceView] 路由加载应用失败', error)
             pendingAppId.value = null
             return
           }
        }
    }

    // 处理子路径（打开 Tab）
    if (pathSegments.length > 2) {
      const functionPath = '/' + pathSegments.join('/') // 构造完整路径，如 /luobei/demo/crm/list
      
    // 🔥 检查是否有 _tab 参数（create/edit/detail 模式）
    // 🔥 使用 _tab 作为系统参数，避免与后端参数冲突
    const tabParam = route.query._tab as string
    if (tabParam === 'create' || tabParam === 'edit' || tabParam === 'detail') {
        // create/edit 模式不需要打开 Tab，直接加载函数详情
        // 尝试查找节点并加载函数详情
        const tryLoadFunction = () => {
          const tree = serviceTree.value
          if (tree && tree.length > 0) {
            const node = findNodeByPath(tree as ServiceTreeType[], functionPath)
            if (node) {
              const serviceNode: ServiceTree = node as any
              // 设置当前函数，但不打开 Tab
              applicationService.handleNodeClick(serviceNode)
            }
          }
        }
        
        if (appSwitched) {
          let retries = 0
          const interval = setInterval(() => {
            if (serviceTree.value.length > 0 || retries > 10) {
              clearInterval(interval)
              tryLoadFunction()
            }
            retries++
          }, 200)
        } else {
          tryLoadFunction()
        }
        
        // 🔥 检查 _forked 参数，自动展开路径
        if (route.query._forked) {
          nextTick(() => {
            checkAndExpandForkedPaths()
          })
        }
        
        // 🔥 记录已处理的路径
        lastProcessedPath = fullPath
        return // create/edit 模式不打开 Tab
      }
      
      // 如果刚刚切换了应用，需要等待服务树加载完成
      // 由于 appSwitched 事件是异步的，我们这里轮询检查 serviceTree 是否有值
      // 或者简单地等待一下（不是最优雅，但在 View 层简单有效）
      // 🔥 检查 _forked 参数，自动展开路径
      if (route.query._forked) {
        nextTick(() => {
          checkAndExpandForkedPaths()
        })
      }
      
      // 更好的方式是 watch serviceTree，但这会变得复杂
      
      // 尝试查找节点
      const tryOpenTab = () => {
         const tree = serviceTree.value
         if (tree && tree.length > 0) {
            const node = findNodeByPath(tree as ServiceTreeType[], functionPath)
            if (node) {
               // 转换为新架构类型
               const serviceNode: ServiceTree = node as any
               // 如果当前没有激活这个 Tab，才去点击
               if (activeTabId.value !== serviceNode.full_code_path && activeTabId.value !== String(serviceNode.id)) {
                  // 检查是否存在该路径的 Tab
                  const existingTab = tabs.value.find(t => t.path === serviceNode.full_code_path || t.path === String(serviceNode.id))
                  if (existingTab) {
                     applicationService.activateTab(existingTab.id)
                  } else {
                     applicationService.triggerNodeClick(serviceNode)
                  }
               }
            }
         }
      }

      if (appSwitched) {
         // 等待服务树加载（通过 watch serviceTree 或者 简单的 timeout）
         // 这里使用简单的重试机制
         let retries = 0
         const interval = setInterval(() => {
            if (serviceTree.value.length > 0 || retries > 10) {
               clearInterval(interval)
               tryOpenTab()
            }
            retries++
         }, 200)
      } else {
         tryOpenTab()
      }
    }
    
    // 🔥 记录已处理的路径
    lastProcessedPath = fullPath
  } catch (error) {
    console.error('[WorkspaceView] 加载应用失败', error)
  } finally {
    isLoadingAppFromRoute = false
  }
}

// 生命周期
let unsubscribeFunctionLoaded: (() => void) | null = null
let unsubscribeServiceTreeLoaded: (() => void) | null = null
let unsubscribeAppSwitched: (() => void) | null = null

onMounted(async () => {
  // 监听函数加载完成事件
  unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, () => {
    // 状态已通过 StateManager 自动更新
  })

  // 监听服务树加载完成事件
  unsubscribeServiceTreeLoaded = eventBus.on(WorkspaceEvent.serviceTreeLoaded, (payload: { app: any, tree: any[] }) => {
    // 状态已通过 StateManager 自动更新
    console.log('[WorkspaceView] 收到 serviceTreeLoaded 事件，节点数:', payload.tree?.length || 0)
  })
  
  // 监听应用切换事件，开始加载服务树
  unsubscribeAppSwitched = eventBus.on(WorkspaceEvent.appSwitched, (payload: { app: any }) => {
    console.log('[WorkspaceView] 收到 appSwitched 事件，目标应用:', payload.app?.user, payload.app?.code, 'ID:', payload.app?.id)
  })

  // 加载应用列表
  await loadAppList()

  // 从路由加载应用
  await loadAppFromRoute()
})

// 监听路由变化（添加防抖，避免频繁调用）
let routeWatchTimer: ReturnType<typeof setTimeout> | null = null
// 🔥 监听服务树变化，检查 _forked 参数
watch(() => serviceTree.value.length, (newLength: number) => {
  if (newLength > 0 && currentApp.value && route.query._forked) {
    console.log('[WorkspaceView] 服务树加载完成，检查 _forked 参数')
    checkAndExpandForkedPaths()
  }
})

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

watch(() => route.path, async () => {
  // 🔥 防抖：如果路径相同，不重复处理
  if (routeWatchTimer) {
    clearTimeout(routeWatchTimer)
  }
  routeWatchTimer = setTimeout(() => {
    loadAppFromRoute()
  }, 100) // 100ms 防抖
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

.workspace-header {
  height: 48px;
  border-bottom: 1px solid var(--el-border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  background-color: var(--el-bg-color);
  flex-shrink: 0;
}

.header-left .logo {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.user-profile {
  display: flex;
  align-items: center;
  cursor: pointer;
  gap: 8px;
}

.username {
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.workspace-view {
  display: flex;
  flex: 1;
  overflow: hidden; /* 防止双滚动条 */
}

/* 标签页样式 */
.workspace-tabs-container {
  border-bottom: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  position: relative;
  z-index: 1; /* 🔥 确保标签页在弹窗下方 */
}

.workspace-tabs {
  margin: 0;
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

.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.drawer-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.drawer-header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.drawer-mode-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.drawer-navigation {
  display: flex;
  align-items: center;
  gap: 12px;
}

.drawer-navigation .nav-info {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  min-width: 60px;
  text-align: center;
  background: var(--el-fill-color-light);
  padding: 6px 12px;
  border-radius: 4px;
  border: 1px solid var(--el-border-color-lighter);
  font-weight: 500;
}

.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  padding-right: 20px;
}

.drawer-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.drawer-header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.drawer-mode-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.drawer-navigation {
  display: flex;
  align-items: center;
  gap: 12px;
}

.drawer-navigation .nav-info {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  min-width: 60px;
  text-align: center;
  background: var(--el-fill-color-light);
  padding: 6px 12px;
  border-radius: 4px;
  border: 1px solid var(--el-border-color-lighter);
  font-weight: 500;
}

.detail-drawer :deep(.el-drawer__header) {
  margin-bottom: 0;
  padding: 16px 20px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.detail-drawer :deep(.el-drawer__body) {
  padding: 20px;
  overflow: auto;
}

.detail-content {
  height: 100%;
}

/* 详情字段网格布局 */
.detail-fields-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 4px;
}

.detail-field-row {
  display: grid;
  grid-template-columns: 140px 1fr;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  align-items: start;
  min-height: auto;
  transition: all 0.2s ease;
  border-radius: 4px;
  background: transparent;
}

.detail-field-row:hover {
  background: var(--el-fill-color-light);
  border-color: var(--el-border-color);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.detail-field-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  display: flex;
  align-items: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.detail-field-value {
  font-size: 14px;
  color: var(--el-text-color-primary);
  word-break: break-word;
  line-height: 1.6;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-height: 24px;
  /* 🔥 确保子组件可以接收点击事件 */
  pointer-events: auto;
  position: relative;
  z-index: 1;
}

/* 详情页链接区域 */
.detail-links-section {
  margin-bottom: 24px;
  padding: 16px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}

.links-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
}

.links-section-content {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.detail-link-item {
  flex-shrink: 0;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 10px;
}
</style>
