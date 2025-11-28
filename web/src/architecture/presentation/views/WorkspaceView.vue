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
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { eventBus, WorkspaceEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import { apiClient } from '../../infrastructure/apiClient'
import ServiceTreePanel from '@/components/ServiceTreePanel.vue'
import FormView from './FormView.vue'
import TableView from './TableView.vue'
import type { ServiceTree, App } from '../../domain/services/WorkspaceDomainService'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'
import type { App as AppType } from '@/types'

const route = useRoute()

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

// 加载状态
const loading = computed(() => false) // TODO: 从状态管理器获取

// 事件处理
const handleNodeClick = (node: ServiceTreeType) => {
  // 转换为新架构的 ServiceTree 类型
  const serviceTree: ServiceTree = node as any
  applicationService.triggerNodeClick(serviceTree)
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

