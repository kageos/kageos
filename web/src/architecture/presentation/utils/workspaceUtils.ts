/**
 * workspaceUtils - 工作空间工具函数
 */

import type { ServiceTree as ServiceTreeType } from '@/types'

/**
 * 递归查找节点
 */
export function findNodeByPath(tree: ServiceTreeType[], path: string): ServiceTreeType | null {
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

