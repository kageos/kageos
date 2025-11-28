<!--
  WorkspaceView - 工作空间视图
  🔥 新架构的展示层组件
  
  职责：
  - 纯 UI 展示，不包含业务逻辑
  - 通过事件与 Application Layer 通信
  - 从 StateManager 获取状态并渲染
-->

<template>
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
      <FormView
        v-if="currentFunctionDetail?.template_type === 'form'"
        :function-detail="currentFunctionDetail"
      />
      <TableView
        v-else-if="currentFunctionDetail?.template_type === 'table'"
        :function-detail="currentFunctionDetail"
      />
      <div v-else class="empty-state">
        <p>请选择一个函数</p>
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
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElDialog, ElForm, ElFormItem, ElInput, ElButton, ElIcon } from 'element-plus'
import { InfoFilled } from '@element-plus/icons-vue'
import { eventBus, WorkspaceEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import { apiClient } from '../../infrastructure/apiClient'
import ServiceTreePanel from '@/components/ServiceTreePanel.vue'
import AppSwitcher from '@/components/AppSwitcher.vue'
import FormView from './FormView.vue'
import TableView from './TableView.vue'
import type { ServiceTree, App } from '../../domain/services/WorkspaceDomainService'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'
import type { App as AppType, CreateAppRequest, ServiceTree as ServiceTreeType } from '@/types'

const route = useRoute()
const router = useRouter()

// 依赖注入（使用 ServiceFactory 简化）
const stateManager = serviceFactory.getWorkspaceStateManager()
const domainService = serviceFactory.getWorkspaceDomainService()
const applicationService = serviceFactory.getWorkspaceApplicationService()

// 从状态管理器获取状态
const serviceTree = computed(() => stateManager.getServiceTree())
const currentFunction = computed(() => stateManager.getCurrentFunction())
const currentAppFromState = computed(() => stateManager.getCurrentApp())

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
// const loading = computed(() => !!loadingTree.value) // 移除这行，直接使用 loadingTree

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

  // 🔥 检查是否正在切换到同一个应用
  if (String(pendingAppId.value) === String(targetAppId)) {
    console.log('[WorkspaceView] 正在切换到该应用，无需重复触发')
    return
  }
  
  // 记录正在切换的应用 ID
  pendingAppId.value = targetAppId
  
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
    pendingAppId.value = null // 失败时重置
  }
  // 注意：成功时不重置 pendingAppId，直到收到 appSwitched 事件或 serviceTreeLoaded 事件确认切换完成
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

    // 🔥 检查当前应用是否已经是目标应用
    const currentAppState = currentApp.value
    if (currentAppState && String(currentAppState.id) === String(targetAppId)) {
      // 即使应用相同，也可能需要处理子路径（定位节点）
      if (pathSegments.length > 2) {
        // TODO: 根据路径定位节点
      }
      return
    }

    // 🔥 检查是否正在切换到该应用
    if (String(pendingAppId.value) === String(targetAppId)) {
      console.log('[WorkspaceView] 路由变化检测：正在切换到该应用，跳过')
      return
    }
    
    // 需要切换应用
    pendingAppId.value = targetAppId
    
    try {
      const appForService: App = {
        id: app.id,
        user: app.user,
        code: app.code,
        name: app.name
      }
      
      // 切换应用
      await applicationService.triggerAppSwitch(appForService)
      
      // 如果路径中有更多段，尝试定位节点
      if (pathSegments.length > 2) {
        const functionPath = pathSegments.slice(2).join('/')
        // TODO: 根据路径定位节点
      }
    } catch (error) {
      console.error('[WorkspaceView] 路由加载应用失败', error)
      pendingAppId.value = null
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
    loadingTree.value = false
    pendingAppId.value = null // 加载完成，重置 pending 状态
  })
  
  // 监听应用切换事件，开始加载服务树
  unsubscribeAppSwitched = eventBus.on(WorkspaceEvent.appSwitched, (payload: { app: any }) => {
    console.log('[WorkspaceView] 收到 appSwitched 事件，目标应用:', payload.app?.user, payload.app?.code, 'ID:', payload.app?.id)
    console.log('[WorkspaceView] 当前状态 - currentApp:', currentApp.value?.id, 'pendingAppId:', pendingAppId.value)
    
    // 🔥 检查当前应用是否已经是目标应用
    const currentAppState = currentApp.value
    if (currentAppState && String(currentAppState.id) === String(payload.app?.id)) {
      console.log('[WorkspaceView] appSwitched: 当前应用已经是目标应用，跳过设置 loading')
      return
    }
    
    // 设置加载状态
    loadingTree.value = true
    // 确保 pendingAppId 被设置（如果是外部触发的切换）
    if (payload.app?.id) {
      pendingAppId.value = payload.app.id
    }
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
})
</script>

<style scoped>
.workspace-view {
  display: flex;
  height: 100%;
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

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--el-text-color-secondary);
}
</style>

