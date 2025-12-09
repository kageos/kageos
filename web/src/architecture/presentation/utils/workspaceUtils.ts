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
 * 返回这些子节点的 group_code（从 full_group_code 中提取最后一段）
 * 
 * 注意：多个 function 节点可能共享同一个 full_group_code，所以需要对结果去重
 * 
 * 例如：
 * - full_group_code: "/luobei/demo/crm/crm_ticket" → 返回 "crm_ticket"
 * - full_group_code: "/luobei/demo/crm/crm_meeting_room" → 返回 "crm_meeting_room"
 */
export function getDirectChildFunctionCodes(node: ServiceTreeType | null): string[] {
  if (!node || !node.children || node.children.length === 0) {
    return []
  }
  
  const codes = node.children
    .filter(child => child.type === 'function')
    .map(child => {
      // 优先使用 full_group_code，提取最后一段
      if (child.full_group_code) {
        // 去掉开头的斜杠，按斜杠分割，取最后一段
        const parts = child.full_group_code.replace(/^\/+/, '').split('/')
        return parts[parts.length - 1] || ''
      }
      
      // 回退到使用 code 字段（去掉 .go 后缀）
      if (child.code) {
        const code = child.code
        return code.endsWith('.go') ? code.slice(0, -3) : code
      }
      
      return ''
    })
    .filter(code => code !== '') // 过滤掉空字符串
  
  // 使用 Set 去重，然后转回数组
  return Array.from(new Set(codes))
}

