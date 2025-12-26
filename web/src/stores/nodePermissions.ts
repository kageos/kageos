/**
 * 节点权限缓存 Store
 * 用于缓存服务树节点和函数详情的权限信息
 */

import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ServiceTree } from '@/types'

/**
 * 权限缓存项
 */
interface PermissionCacheItem {
  nodeId: number
  fullCodePath: string
  permissions: Record<string, boolean>
  timestamp: number  // 缓存时间戳
}

/**
 * 权限缓存 Store
 */
export const useNodePermissionsStore = defineStore('nodePermissions', () => {
  // 权限缓存：key 为 nodeId 或 fullCodePath
  const permissionCache = ref<Map<string, PermissionCacheItem>>(new Map())
  
  // 缓存过期时间（5分钟）
  const CACHE_TTL = 5 * 60 * 1000

  /**
   * 获取缓存键（优先使用 fullCodePath，如果没有则使用 nodeId）
   */
  function getCacheKey(node: ServiceTree): string {
    return node.full_code_path || `node:${node.id}`
  }

  /**
   * 设置权限缓存
   */
  function setPermissions(node: ServiceTree, permissions: Record<string, boolean>): void {
    const key = getCacheKey(node)
    permissionCache.value.set(key, {
      nodeId: node.id,
      fullCodePath: node.full_code_path,
      permissions,
      timestamp: Date.now()
    })
  }

  /**
   * 获取权限缓存
   */
  function getPermissions(node: ServiceTree): Record<string, boolean> | null {
    const key = getCacheKey(node)
    const cached = permissionCache.value.get(key)
    
    if (!cached) {
      return null
    }

    // 检查缓存是否过期
    if (Date.now() - cached.timestamp > CACHE_TTL) {
      permissionCache.value.delete(key)
      return null
    }

    return cached.permissions
  }

  /**
   * 清除权限缓存
   */
  function clearPermissions(node: ServiceTree): void {
    const key = getCacheKey(node)
    permissionCache.value.delete(key)
  }

  /**
   * 清除所有权限缓存
   */
  function clearAllPermissions(): void {
    permissionCache.value.clear()
  }

  /**
   * 更新节点的权限信息（合并到节点对象中）
   */
  function updateNodePermissions(node: ServiceTree): ServiceTree {
    const permissions = getPermissions(node)
    if (permissions) {
      return {
        ...node,
        permissions
      }
    }
    return node
  }

  return {
    permissionCache,
    setPermissions,
    getPermissions,
    clearPermissions,
    clearAllPermissions,
    updateNodePermissions
  }
})

