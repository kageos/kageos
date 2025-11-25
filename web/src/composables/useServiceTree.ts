/**
 * useServiceTree - 服务目录树管理 Composable
 * 负责服务树的加载、节点定位、目录创建
 */

import { ref, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { getServiceTree, createServiceTree } from '@/api/service-tree'
import { Logger } from '@/core/utils/logger'
import type { App, ServiceTree, CreateServiceTreeRequest } from '@/types'

export function useServiceTree() {
  // 状态
  const serviceTree = ref<ServiceTree[]>([])
  const loading = ref(false)
  const currentNode = ref<ServiceTree | null>(null)

  /**
   * 加载服务目录树
   */
  const loadServiceTree = async (app: App): Promise<ServiceTree[]> => {
    if (!app || !app.user || !app.code) {
      serviceTree.value = []
      return []
    }

    try {
      loading.value = true
      const tree = await getServiceTree(app.user, app.code)
      serviceTree.value = tree || []
      return serviceTree.value
    } catch (error) {
      Logger.error('useServiceTree', '获取服务目录树失败', error)
      ElMessage.error('获取服务目录树失败')
      serviceTree.value = []
      return []
    } finally {
      loading.value = false
    }
  }

  /**
   * 查找节点（递归）
   */
  const findNodeByPath = (tree: ServiceTree[], path: string[]): ServiceTree | null => {
    if (path.length === 0) {
      return null
    }

    for (const node of tree) {
      // 匹配当前层级的节点
      if (node.code === path[0]) {
        // 如果是最后一个路径段，返回当前节点
        if (path.length === 1) {
          return node
        }
        
        // 否则在子节点中继续查找
        if (node.children && node.children.length > 0) {
          const found = findNodeByPath(node.children, path.slice(1))
          if (found) {
            return found
          }
        }
      }
    }

    return null
  }

  /**
   * 根据 tree_id 查找节点（递归）
   */
  const findNodeById = (tree: ServiceTree[], treeId: number): ServiceTree | null => {
    for (const node of tree) {
      if (node.id === treeId) {
        return node
      }
      if (node.children && node.children.length > 0) {
        const found = findNodeById(node.children, treeId)
        if (found) {
          return found
        }
      }
    }
    return null
  }

  /**
   * 根据路由路径定位节点
   */
  const locateNodeByRoute = (routePath?: string): ServiceTree | null => {
    // 如果没有传入路径，使用当前路径
    const currentPath = routePath || window.location.pathname
    
    const fullPath = currentPath.replace('/workspace/', '').replace(/^\/+|\/+$/g, '')
    if (!fullPath) {
      return null
    }

    const pathSegments = fullPath.split('/').filter(Boolean)
    
    // 路径至少需要 user/app/...
    if (pathSegments.length < 3) {
      return null
    }

    // 跳过 user 和 app，剩下的是服务树路径
    const treePath = pathSegments.slice(2)
    
    const node = findNodeByPath(serviceTree.value, treePath)
    if (node) {
      currentNode.value = node
    } else {
      currentNode.value = null
    }
    
    return node
  }

  /**
   * 创建服务目录
   */
  const handleCreateDirectory = async (request: CreateServiceTreeRequest): Promise<boolean> => {
    try {
      await createServiceTree(request)
      ElMessage.success('创建服务目录成功')
      return true
    } catch (error: any) {
      Logger.error('useServiceTree', '创建服务目录失败', error)
      const errorMessage = error?.response?.data?.message || error?.message || '创建服务目录失败'
      ElMessage.error(errorMessage)
      return false
    }
  }

  /**
   * 清空状态
   */
  const clear = () => {
    serviceTree.value = []
    currentNode.value = null
  }

  return {
    // 状态
    serviceTree,
    loading,
    currentNode,
    
    // 方法
    loadServiceTree,
    findNodeByPath,
    findNodeById,
    locateNodeByRoute,
    handleCreateDirectory,
    clear
  }
}

