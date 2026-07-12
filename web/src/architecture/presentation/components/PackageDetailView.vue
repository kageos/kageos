<!--
  PackageDetailView - 服务目录详情页面

  职责：
  - 显示服务目录信息
-->
<template>
  <div class="package-detail-view">
    <div class="main-content">
      <PackageDetailContent
        :package-node="packageNode || null"
        @select-child="handleChildClick"
        @access-changed="emit('refresh')"
        @create-directory="emit('create-directory', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import type { ServiceTree } from '@/architecture/domain/types'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import PackageDetailContent from './PackageDetailContent.vue'

interface Props {
  packageNode?: ServiceTree | null
}

defineProps<Props>()

const emit = defineEmits<{
  'refresh': []
  'create-directory': [node: ServiceTree]
}>()

const route = useRoute()

// 处理子项点击（跳转到对应的目录或函数）
function handleChildClick(child: ServiceTree): void {
  const serviceProvider: IServiceProvider = serviceFactory
  const applicationService = serviceProvider.getWorkspaceApplicationService()

  if (child.type === 'function' && child.full_code_path) {
    // 函数节点：跳转到函数页面
    const targetPath = `/workspace${child.full_code_path}`

    if (route.path !== targetPath) {
      // 触发节点点击，加载函数详情
      applicationService.triggerNodeClick(child)

      const preserveParams = {
        table: false,
        search: false,
        state: false,
        linkNavigation: false
      }

      // 更新路由
      eventBus.emit(RouteEvent.updateRequested, {
        path: targetPath,
        query: {},
        replace: true,
        preserveParams,
        source: 'package-detail-child-click'
      })
    } else {
      // 路由已匹配，直接触发节点点击加载详情
      applicationService.triggerNodeClick(child)
    }
  } else if (child.type === 'package' && child.full_code_path) {
    // 目录节点：跳转到目录详情页面
    applicationService.triggerNodeClick(child)

    const targetPath = `/workspace${child.full_code_path}`
    if (route.path !== targetPath) {
      const preserveParams = {
        table: false,
        search: false,
        state: false,
        linkNavigation: false
      }

      eventBus.emit(RouteEvent.updateRequested, {
        path: targetPath,
        query: {},
        replace: true,
        preserveParams,
        source: 'package-detail-child-click-package'
      })
    }
  } else if (child.type === 'docs' && child.full_code_path) {
    // 文档节点：跳转到对应页面
    applicationService.triggerNodeClick(child)
    const targetPath = `/workspace${child.full_code_path}`
    if (route.path !== targetPath) {
      eventBus.emit(RouteEvent.updateRequested, {
        path: targetPath,
        query: {},
        replace: true,
        preserveParams: { table: false, search: false, state: false, linkNavigation: false },
        source: 'package-detail-child-click-docs'
      })
    } else {
      applicationService.triggerNodeClick(child)
    }
  }
}
</script>

<style scoped lang="scss">
.package-detail-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);

  .main-content {
    flex: 1;
    display: flex;
    overflow: hidden;
  }
}

// 响应式设计
@media (max-width: 768px) {
  .package-detail-view {
    .main-content {
      flex-direction: column;

      .detail-content {
        padding: 24px 20px;
      }
    }
  }
}

</style>
