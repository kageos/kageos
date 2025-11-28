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
        :current-node="currentFunction"
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
import { computed, onMounted, onUnmounted } from 'vue'
import { WorkspaceApplicationService } from '../../application/services/WorkspaceApplicationService'
import { WorkspaceDomainService } from '../../domain/services/WorkspaceDomainService'
import { WorkspaceStateManager } from '../../infrastructure/stateManager/WorkspaceStateManager'
import { functionLoader } from '../../infrastructure/functionLoader'
import { eventBus, WorkspaceEvent } from '../../infrastructure/eventBus'
import ServiceTreePanel from '@/components/ServiceTreePanel.vue'
import FormView from './FormView.vue'
import TableView from './TableView.vue'
import type { ServiceTree } from '../../domain/services/WorkspaceDomainService'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'

// 依赖注入（在实际项目中可以使用依赖注入容器）
const stateManager = new WorkspaceStateManager()
const domainService = new WorkspaceDomainService(functionLoader, stateManager, eventBus)
const applicationService = new WorkspaceApplicationService(domainService, eventBus)

// 从状态管理器获取状态
const serviceTree = computed(() => stateManager.getServiceTree())
const currentFunction = computed(() => stateManager.getCurrentFunction())
const currentFunctionDetail = computed<FunctionDetail | null>(() => {
  const node = currentFunction.value
  if (!node) return null
  return stateManager.getFunctionDetail(node)
})

// 事件处理
const handleNodeClick = (node: ServiceTree) => {
  applicationService.triggerNodeClick(node)
}

// 生命周期
let unsubscribeFunctionLoaded: (() => void) | null = null

onMounted(() => {
  // 监听函数加载完成事件，更新 UI
  unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, () => {
    // 状态已通过 StateManager 自动更新，这里可以添加额外的 UI 更新逻辑
  })
})

onUnmounted(() => {
  // 取消事件监听
  if (unsubscribeFunctionLoaded) {
    unsubscribeFunctionLoaded()
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

