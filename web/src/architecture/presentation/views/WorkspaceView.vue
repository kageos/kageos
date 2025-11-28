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
          :tree-data="serviceTree"
          :loading="loading"
          :current-node-id="currentFunction?.id || null"
          :current-function="currentFunction"
          @node-click="handleNodeClick"
        />
      </div>

      <!-- 中间函数渲染区域 -->
      <div class="function-renderer">
        <el-tabs
          v-if="tabs.length > 0"
          v-model="activeTabId"
          type="card"
          closable
          class="workspace-tabs"
          @tab-remove="handleTabRemove"
        >
          <el-tab-pane
            v-for="tab in tabs"
            :key="tab.id"
            :label="tab.title"
            :name="tab.id"
          >
            <!-- 只渲染当前激活的 Tab 内容，确保切换时状态被保存后销毁/重建 -->
            <div v-if="activeTabId === tab.id" class="tab-content">
              <FormView
                v-if="currentFunctionDetail?.template_type === 'form'"
                :key="`form-${tab.id}`"
                :function-detail="currentFunctionDetail"
              />
              <TableView
                v-else-if="currentFunctionDetail?.template_type === 'table'"
                :key="`table-${tab.id}`"
                :function-detail="currentFunctionDetail"
              />
              <div v-else class="empty-state">
                <p>加载中...</p>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
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
      size="40%"
      destroy-on-close
      :modal="true"
      :close-on-click-modal="true"
      class="detail-drawer"
    >
      <div v-if="detailRowData" class="detail-content">
        <!-- 复用 FormView 但使用 detail 模式 -->
        <!-- 这里我们需要一个能渲染详情的组件，可以使用 WidgetComponent 遍历 response 字段 -->
        <el-form label-width="120px">
          <el-form-item
            v-for="field in detailFields"
            :key="field.code"
            :label="field.name"
          >
            <WidgetComponent
              :field="field"
              :value="getDetailFieldValue(field.code)"
              mode="detail"
            />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <div class="drawer-footer">
          <el-button @click="detailDrawerVisible = false">关闭</el-button>
          <!-- 只有当表格配置中有 Update 回调时才显示编辑按钮 -->
          <el-button 
            v-if="currentFunctionDetail?.callbacks?.includes('OnTableUpdateRow')" 
            type="primary" 
            @click="handleDrawerEdit"
          >
            编辑
          </el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch, ref, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElDialog, ElForm, ElFormItem, ElInput, ElButton, ElIcon, ElTabs, ElTabPane, ElDrawer, ElDropdown, ElDropdownMenu, ElDropdownItem, ElAvatar } from 'element-plus'
import { InfoFilled, ArrowDown } from '@element-plus/icons-vue'
import { eventBus, WorkspaceEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import { apiClient } from '../../infrastructure/apiClient'
import { useAuthStore } from '@/stores/auth'
import ServiceTreePanel from '@/components/ServiceTreePanel.vue'
import AppSwitcher from '@/components/AppSwitcher.vue'
import ThemeToggle from '@/components/ThemeToggle.vue'
import FormView from './FormView.vue'
import TableView from './TableView.vue'
import WidgetComponent from '../widgets/WidgetComponent.vue'
import type { ServiceTree, App } from '../../domain/services/WorkspaceDomainService'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'
import type { App as AppType, CreateAppRequest, ServiceTree as ServiceTreeType } from '@/types'
import type { FieldConfig, FieldValue } from '../../domain/types'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

// 依赖注入（使用 ServiceFactory 简化）
const stateManager = serviceFactory.getWorkspaceStateManager()
const domainService = serviceFactory.getWorkspaceDomainService()
const applicationService = serviceFactory.getWorkspaceApplicationService()

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

// Tab 关闭处理
const handleTabRemove = (targetName: string) => {
  applicationService.closeTab(targetName)
}

// 状态保存与恢复
watch(() => stateManager.getState().activeTabId, async (newId, oldId) => {
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
      // 如果没有数据，可能是新打开的（由 functionLoaded 初始化）
      // 或者是切换到一个未初始化的 Tab（需要清空残留数据）
      // 建议清空，以防万一
      if (newTab?.node) {
         const detail = stateManager.getFunctionDetail(newTab.node)
         if (detail?.template_type === 'form') {
             // 清空 FormState
             serviceFactory.getFormStateManager().setState({
               data: new Map(),
               errors: new Map(),
               submitting: false
             })
         }
      }
    }
    
    // 更新路由参数（如果需要）
    if (newTab) {
      const path = `/workspace-v2${newTab.path.startsWith('/') ? '' : '/'}${newTab.path}`
      if (route.path !== path) {
        // 使用 replace 避免产生大量历史记录
        router.replace(path).catch(() => {})
      }
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
  const node = currentFunction.value
  if (!node) return null
  return stateManager.getFunctionDetail(node)
})

const detailDrawerVisible = ref(false)
const detailDrawerTitle = ref('详情')
const detailRowData = ref<Record<string, any> | null>(null)
const detailFields = ref<FieldConfig[]>([])

// 监听表格详情事件
onMounted(() => {
  eventBus.on('table:detail-row', ({ row }: { row: Record<string, any> }) => {
    if (!currentFunctionDetail.value) return
    
    detailRowData.value = row
    detailDrawerTitle.value = currentFunctionDetail.value.name || '详情'
    // 使用响应参数作为详情字段
    detailFields.value = (currentFunctionDetail.value.response || []) as FieldConfig[]
    detailDrawerVisible.value = true
  })
})

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
  if (detailRowData.value) {
    // 先关闭详情抽屉，避免遮挡（或者根据需求保留）
    detailDrawerVisible.value = false
    // 触发表格的编辑逻辑
    // 由于编辑逻辑目前在 TableView 中，我们通过 EventBus 通知
    // 注意：这需要 TableView 监听此事件，或者我们将编辑逻辑提升到 Workspace 或 Application Service
    // 这里简单起见，我们发送一个事件让 TableView 处理
    eventBus.emit('table:edit-row', { row: detailRowData.value })
  }
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
    ElMessage.error('加载应用列表失败')
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
    const targetPath = `/workspace-v2/${app.user}/${app.code}`
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
    ElMessage.warning('请填写应用名称和应用代码')
    return
  }

  try {
    creatingApp.value = true
    await apiClient.post('/api/v1/app/create', createAppForm.value)
    ElMessage.success('应用创建成功')
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
    ElMessage.error(errorMessage)
  } finally {
    creatingApp.value = false
  }
}

// 更新应用（重新编译）
const handleUpdateApp = async (app: AppType): Promise<void> => {
  try {
    await apiClient.post(`/api/v1/app/update/${app.code}`, {})
    ElMessage.success('应用更新成功')
  } catch (error: any) {
    const errorMessage = error?.response?.data?.message || '更新应用失败'
    ElMessage.error(errorMessage)
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
    ElMessage.success('应用删除成功')
    
    // 刷新应用列表
    await loadAppList()
    
    // 如果删除的是当前应用，切换到第一个应用或清空
    if (currentApp.value && currentApp.value.id === app.id) {
      if (appList.value.length > 0) {
        await handleSwitchApp(appList.value[0])
      } else {
        await router.push('/workspace-v2')
      }
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      const errorMessage = error?.response?.data?.message || '删除应用失败'
      ElMessage.error(errorMessage)
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

// 从路由解析应用并加载
const loadAppFromRoute = async () => {
  // 支持 /workspace-v2 和 /workspace 两种路径
  const fullPath = route.path
    .replace('/workspace-v2/', '')
    .replace('/workspace/', '')
    .replace(/^\/+|\/+$/g, '')
  
  if (!fullPath) {
    return
  }

  const pathSegments = fullPath.split('/').filter(Boolean)
  if (pathSegments.length < 2) {
    return
  }

  const [user, appCode] = pathSegments
  
  try {
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
      
      // 如果刚刚切换了应用，需要等待服务树加载完成
      // 由于 appSwitched 事件是异步的，我们这里轮询检查 serviceTree 是否有值
      // 或者简单地等待一下（不是最优雅，但在 View 层简单有效）
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
  } catch (error) {
    console.error('[WorkspaceView] 加载应用失败', error)
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

// 监听路由变化
watch(() => route.path, async () => {
  await loadAppFromRoute()
})

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

.workspace-tabs {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.workspace-tabs :deep(.el-tabs__header) {
  margin: 0;
  background-color: var(--el-bg-color-overlay);
  border-bottom: 1px solid var(--el-border-color-light);
}

.workspace-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: auto;
  padding: 0;
}

.tab-content {
  height: 100%;
  overflow: auto;
}

.left-sidebar {
  width: 300px;
  border-right: 1px solid var(--el-border-color);
}

.function-renderer {
  flex: 1;
  padding: 20px;
  overflow: auto;
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

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 10px;
}
</style>
