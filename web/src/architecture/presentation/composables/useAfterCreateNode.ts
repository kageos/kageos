/**
 * useAfterCreateNode - 创建节点后的统一处理：刷新树 + 定位并选中新节点
 * 用于文档、讨论区等创建成功后的导航，避免在视图里重复写相同逻辑
 */

import type { ServiceTree } from '@/architecture/domain/types'
import { Logger } from '@/architecture/runtime/utils/logger'

export function useAfterCreateNode(deps: {
  handleRefreshTree: () => Promise<void>
  serviceTree: () => ServiceTree[]
  findNodeById: (tree: ServiceTree[], id: number) => ServiceTree | null
  handleNodeClick: (node: ServiceTree) => void
}) {
  return async (response: ServiceTree) => {
    if (!response?.id) return
    try {
      await deps.handleRefreshTree()
      const newNode = deps.findNodeById(deps.serviceTree(), response.id)
      if (newNode) deps.handleNodeClick(newNode)
    } catch (err) {
      Logger.error('[useAfterCreateNode]', '刷新服务树失败', { error: err })
      const newNode = deps.findNodeById(deps.serviceTree(), response.id)
      if (newNode) deps.handleNodeClick(newNode)
    }
  }
}
