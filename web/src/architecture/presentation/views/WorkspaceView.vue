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
const currentApp = computed(() => stateManager.getCurrentApp())
const currentFunctionDetail = computed<FunctionDetail | null>(() => {
  const node = currentFunction.value
  if (!node) return null
  return stateManager.getFunctionDetail(node)
})

// 应用列表管理
const appList = ref<AppType[]>([])
const loadingApps = ref(false)

// 创建应用对话框
const createAppDialogVisible = ref(false)
const creatingApp = ref(false)
const createAppForm = ref<CreateAppRequest>({
  code: '',
  name: ''
})

// 加载状态
const loading = computed(() => false) // TODO: 从状态管理器获取

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
    const items = await apiClient.get<AppType[]>('/api/v1/app/list')
    appList.value = items || []
  } catch (error) {
    console.error('[WorkspaceView] 加载应用列表失败', error)
    ElMessage.error('加载应用列表失败')
  } finally {
    loadingApps.value = false
  }
}

// 切换应用
const handleSwitchApp = async (app: AppType): Promise<void> => {
  const appForService: App = {
    id: app.id,
    user: app.user,
    code: app.code,
    name: app.name
  }
  
  // 更新路由
  await router.push(`/workspace-v2/${app.user}/${app.code}`)
  
  // 切换应用（这会触发服务树加载）
  await applicationService.triggerAppSwitch(appForService)
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
    // 加载应用列表
    const appList = await apiClient.get<AppType[]>('/api/v1/app/list')
    const app = appList.find((a: AppType) => a.user === user && a.code === appCode)
    
    if (app) {
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
    }
  } catch (error) {
    console.error('[WorkspaceView] 加载应用失败', error)
  }
}

// 生命周期
let unsubscribeFunctionLoaded: (() => void) | null = null
let unsubscribeServiceTreeLoaded: (() => void) | null = null

onMounted(async () => {
  // 监听函数加载完成事件
  unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, () => {
    // 状态已通过 StateManager 自动更新
  })

  // 监听服务树加载完成事件
  unsubscribeServiceTreeLoaded = eventBus.on(WorkspaceEvent.serviceTreeLoaded, () => {
    // 状态已通过 StateManager 自动更新
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

