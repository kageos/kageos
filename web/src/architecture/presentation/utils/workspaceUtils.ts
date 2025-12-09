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

/**
 * 根据 ID 递归查找节点
 */
export function findNodeById(tree: ServiceTreeType[], id: number): ServiceTreeType | null {
  for (const node of tree) {
    if (node.id === id) {
      return node
    }
    if (node.children && node.children.length > 0) {
      const found = findNodeById(node.children, id)
      if (found) return found
    }
  }
  return null
}

/**
 * 获取节点的直接子节点（只收集一级子节点，type 为 'function' 的）
 * 返回这些子节点的 code（文件名，去掉 .go 后缀）
 */
export function getDirectChildFunctionCodes(node: ServiceTreeType | null): string[] {
  if (!node || !node.children || node.children.length === 0) {
    return []
  }
  
  return node.children
    .filter(child => child.type === 'function' && child.code)
    .map(child => {
      // 去掉 .go 后缀（如果有）
      const code = child.code
      return code.endsWith('.go') ? code.slice(0, -3) : code
    })
}

