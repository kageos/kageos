/**
 * useServiceTree - 服务目录树管理 Composable
 * 负责服务树的加载、节点定位、目录创建
 */

import { ref, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { getServiceTree, createServiceTree } from '@/api/service-tree'
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
      console.log('[useServiceTree] 开始加载服务目录树:', app.user + '/' + app.code)
      loading.value = true
      const tree = await getServiceTree(app.user, app.code)
      serviceTree.value = tree || []
      console.log('[useServiceTree] 服务目录树加载完成，节点数:', serviceTree.value.length)
      return serviceTree.value
    } catch (error) {
      console.error('[useServiceTree] 获取服务目录树失败:', error)
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
    
    console.log('[useServiceTree] 定位节点，路径:', treePath.join('/'))
    
    const node = findNodeByPath(serviceTree.value, treePath)
    if (node) {
      console.log('[useServiceTree] 找到节点:', node.name, node.code)
      currentNode.value = node
    } else {
      console.log('[useServiceTree] 未找到节点')
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
      console.error('[useServiceTree] 创建服务目录失败:', error)
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
    locateNodeByRoute,
    handleCreateDirectory,
    clear
  }
}

