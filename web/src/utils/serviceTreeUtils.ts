/**
 * 服务树工具函数
 * 
 * 用于处理服务树节点的查找、展开等操作
 */

import type { ServiceTree } from '@/types'

/**
 * 查找从根节点到目标节点的路径
 * @param nodes 树节点数组
 * @param targetId 目标节点 ID
 * @returns 从根到目标的节点 ID 路径数组
 */
export function findPathToNode(nodes: ServiceTree[], targetId: number | string): number[] {
  const path: number[] = []
  const targetIdNum = Number(targetId)
  
  const findNode = (nodes: ServiceTree[], targetId: number): boolean => {
    for (const node of nodes) {
      // 处理分组节点（虚拟节点）
      if ((node as any).isGroup) {
        // 在分组节点的子节点中查找
        if (node.children && node.children.length > 0) {
          if (findNode(node.children, targetId)) {
            path.push(Number(node.id)) // 包含分组节点到路径中
            return true
          }
        }
        continue
      }
      
      const nodeIdNum = Number(node.id)
      path.push(nodeIdNum)
      
      if (nodeIdNum === targetId) {
        return true
      }
      
      if (node.children && node.children.length > 0) {
        if (findNode(node.children, targetId)) {
          return true
        }
      }
      
      path.pop()
    }
    return false
  }
  
  findNode(nodes, targetIdNum)
  return path
}

/**
 * 展开所有父节点（递归展开）
 * @param treeRef Element Plus Tree 组件的 ref
 * @param path 节点路径数组
 */
export function expandParentNodes(treeRef: any, path: number[]): void {
  if (path.length === 0 || !treeRef) return
  
  // 展开所有父节点（最后一个节点不需要展开，只需选中）
  const expandKeys = path.slice(0, -1)
  expandKeys.forEach((key: number) => {
    const node = treeRef.store.nodesMap[key]
    if (node && !node.expanded) {
      node.expand()
    }
  })
}

/**
 * 规范化服务树路径
 * - 移除开头的斜杠
 * - 如果是函数组节点，移除 __group__ 部分
 * 
 * @param path 原始路径
 * @param isGroup 是否是函数组节点
 * @returns 规范化后的路径
 */
export function normalizeServiceTreePath(path: string, isGroup?: boolean): string {
  let normalized = path.replace(/^\/+/, '')
  
  // 如果是分组节点，移除 __group__ 部分来匹配目录路径
  if (isGroup) {
    normalized = normalized.replace(/\/__group__[^/]+$/, '')
  }
  
  return normalized
}

/**
 * 根据 full_code_path 查找节点
 * @param nodes 树节点数组
 * @param targetPath 目标路径
 * @returns 找到的节点，如果没找到返回 null
 */
export function findNodeByPath(nodes: ServiceTree[], targetPath: string): ServiceTree | null {
  if (!nodes.length) {
    return null
  }
  
  // 规范化路径（移除开头的斜杠，确保格式一致）
  const normalizedPath = normalizeServiceTreePath(targetPath)
  
  const findNode = (nodes: ServiceTree[], path: string): ServiceTree | null => {
    for (const node of nodes) {
      // 规范化节点的 full_code_path
      const isGroup = (node as any).isGroup
      const nodePath = normalizeServiceTreePath(node.full_code_path, isGroup)
      
      // 检查当前节点是否匹配（精确匹配或目录匹配）
      if (nodePath === path || path.startsWith(nodePath + '/')) {
        // 如果是精确匹配，返回该节点
        if (nodePath === path) {
          return node
        }
        
        // 如果是目录匹配，继续在子节点中查找
        if (node.children && node.children.length > 0) {
          const found = findNode(node.children, path)
          if (found) return found
        }
      }
    }
    return null
  }
  
  return findNode(nodes, normalizedPath)
}

/**
 * 根据 full_group_code 查找函数组节点
 * @param nodes 树节点数组
 * @param fullGroupCode 函数组代码
 * @returns 找到的函数组节点，如果没找到返回 null
 */
export function findGroupByFullGroupCode(
  nodes: ServiceTree[], 
  fullGroupCode: string
): ServiceTree | null {
  if (!nodes.length) {
    return null
  }
  
  const findNode = (nodes: ServiceTree[]): ServiceTree | null => {
    for (const node of nodes) {
      // 检查是否是函数组节点且 full_group_code 匹配
      if ((node as any).isGroup && (node as any).full_group_code === fullGroupCode) {
        return node
      }
      // 递归查找子节点
      if (node.children && node.children.length > 0) {
        const found = findNode(node.children)
        if (found) return found
      }
    }
    return null
  }
  
  return findNode(nodes)
}

/**
 * 查找节点的父节点
 * @param nodes 树节点数组
 * @param targetId 目标节点 ID
 * @returns 找到的父节点，如果没找到返回 null
 */
export function findParentNode(nodes: ServiceTree[], targetId: number): ServiceTree | null {
  if (!nodes.length) {
    return null
  }
  
  const findParent = (nodes: ServiceTree[], targetId: number): ServiceTree | null => {
    for (const node of nodes) {
      // 检查当前节点的子节点中是否包含目标节点
      if (node.children && node.children.length > 0) {
        const hasTarget = node.children.some(child => Number(child.id) === targetId)
        if (hasTarget) {
          return node
        }
        // 递归查找
        const found = findParent(node.children, targetId)
        if (found) return found
      }
    }
    return null
  }
  
  return findParent(nodes, targetId)
}

/**
 * 查找函数组节点（优先查找 isGroup 节点，如果没有则查找匹配的函数）
 * @param nodes 树节点数组
 * @param fullGroupCode 函数组代码
 * @returns 函数组节点和函数列表
 */
export function findFunctionGroup(
  nodes: ServiceTree[], 
  fullGroupCode: string
): { groupNode: ServiceTree | null, functions: ServiceTree[] } {
  // 先尝试查找函数组节点
  const groupNode = findGroupByFullGroupCode(nodes, fullGroupCode)
  
  if (groupNode && (groupNode as any).isGroup) {
    // 如果找到函数组节点，获取其子函数
    const functions = (groupNode.children || []).filter(child => child.type === 'function')
    return { groupNode, functions }
  }
  
  // 如果没有找到函数组节点，查找所有匹配的函数（fallback 逻辑）
  const matchedFunctions: ServiceTree[] = []
  const findAllFunctions = (nodes: ServiceTree[]) => {
    for (const n of nodes) {
      if (n.type === 'function' && n.full_group_code === fullGroupCode) {
        matchedFunctions.push(n)
      }
      if (n.children) {
        findAllFunctions(n.children)
      }
    }
  }
  findAllFunctions(nodes)
  
  return { groupNode: null, functions: matchedFunctions }
}

/**
 * 等待节点展开完成
 * @param treeRef Element Plus Tree 组件的 ref
 * @param nodeId 节点 ID
 * @param timeout 超时时间（毫秒），默认 1000ms
 * @returns Promise，节点展开完成时 resolve
 */
export function waitForNodeExpansion(
  treeRef: any,
  nodeId: number,
  timeout = 1000
): Promise<boolean> {
  return new Promise((resolve) => {
    if (!treeRef) {
      resolve(false)
      return
    }
    
    const treeNode = treeRef.store.nodesMap[nodeId]
    if (!treeNode) {
      resolve(false)
      return
    }
    
    // 如果已经展开，直接返回
    if (treeNode.expanded) {
      resolve(true)
      return
    }
    
    // 如果没有子节点，不需要展开
    if (!treeNode.childNodes || treeNode.childNodes.length === 0) {
      resolve(true)
      return
    }
    
    // 展开节点
    treeNode.expand()
    
    // 等待展开完成（使用轮询检查）
    const startTime = Date.now()
    const checkInterval = setInterval(() => {
      if (treeNode.expanded || Date.now() - startTime > timeout) {
        clearInterval(checkInterval)
        resolve(treeNode.expanded)
      }
    }, 50)
  })
}

/**
 * 展开路径并选中节点
 * @param treeRef Element Plus Tree 组件的 ref
 * @param nodes 树节点数组
 * @param path 节点路径数组
 * @param targetNodeId 目标节点 ID
 * @returns Promise，展开和选中完成时 resolve
 */
export async function expandPathAndSelect(
  treeRef: any,
  nodes: ServiceTree[],
  path: number[],
  targetNodeId: number
): Promise<void> {
  if (!treeRef || path.length === 0) {
    return
  }
  
  // 展开所有父节点
  expandParentNodes(treeRef, path)
  
  // 等待展开完成
  await new Promise(resolve => setTimeout(resolve, 100))
  
  // 确保目标节点也被展开（如果它是可展开的）
  await waitForNodeExpansion(treeRef, targetNodeId, 200)
  
  // 选中目标节点
  treeRef.setCurrentKey(targetNodeId)
}

