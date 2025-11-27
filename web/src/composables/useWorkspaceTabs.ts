/**
 * useWorkspaceTabs - 工作空间标签页管理
 * 
 * 功能：
 * - 管理多个函数标签页
 * - 标签页状态持久化（localStorage）
 * - 路由同步
 * - 组件缓存管理
 */

import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { ServiceTree } from '@/types'

/**
 * 标签页数据结构
 */
export interface WorkspaceTab {
  id: string  // 唯一标识（使用 full_code_path 或 id）
  name: string  // 标签显示名称
  path: string  // 路由路径（如：/workspace/user/app/path）
  functionId?: number  // 函数ID（如果有）
  fullCodePath?: string  // 完整代码路径
  function?: ServiceTree  // 函数节点信息（可选，用于缓存）
}

const STORAGE_KEY = 'workspace-tabs'
const MAX_TABS = 20  // 最大标签页数量

/**
 * 工作空间标签页管理 Composable
 */
export function useWorkspaceTabs() {
  const route = useRoute()
  const router = useRouter()
  
  // 标签页列表
  const tabs = ref<WorkspaceTab[]>([])
  // 当前激活的标签ID
  const activeTabId = ref<string | null>(null)

  /**
   * 从 localStorage 恢复标签页
   */
  function restoreTabs() {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored) {
        const data = JSON.parse(stored)
        if (data.tabs && Array.isArray(data.tabs)) {
          tabs.value = data.tabs
        }
        if (data.activeTabId) {
          activeTabId.value = data.activeTabId
        }
      }
    } catch (error) {
      console.error('[WorkspaceTabs] 恢复标签页失败:', error)
    }
  }

  /**
   * 保存标签页到 localStorage
   */
  function saveTabs() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({
        tabs: tabs.value,
        activeTabId: activeTabId.value
      }))
    } catch (error) {
      console.error('[WorkspaceTabs] 保存标签页失败:', error)
    }
  }

  /**
   * 根据节点创建标签
   */
  function createTabFromNode(node: ServiceTree): WorkspaceTab {
    const fullCodePath = node.full_code_path || ''
    const path = fullCodePath.startsWith('/') ? fullCodePath.substring(1) : fullCodePath
    const workspacePath = `/workspace/${path}`
    
    return {
      id: fullCodePath || `node-${node.id}`,
      name: node.name || node.code || '未命名',
      path: workspacePath,
      functionId: node.ref_id || node.id,
      fullCodePath: fullCodePath,
      function: node
    }
  }

  /**
   * 添加或激活标签
   */
  function addOrActivateTab(node: ServiceTree) {
    if (node.type !== 'function') {
      // 只对函数类型添加标签
      return
    }

    const fullCodePath = node.full_code_path || ''
    const tabId = fullCodePath || `node-${node.id}`
    
    // 检查标签是否已存在
    const existingTab = tabs.value.find(tab => tab.id === tabId)
    
    if (existingTab) {
      // 标签已存在，激活它
      activeTabId.value = tabId
      // 更新路由（如果路径不同）
      if (existingTab.path !== route.path) {
        router.push(existingTab.path)
      }
    } else {
      // 创建新标签
      const newTab = createTabFromNode(node)
      
      // 如果标签数量超过限制，删除最旧的标签
      if (tabs.value.length >= MAX_TABS) {
        tabs.value.shift()  // 删除第一个（最旧的）
      }
      
      tabs.value.push(newTab)
      activeTabId.value = tabId
      
      // 更新路由
      router.push(newTab.path)
    }
    
    saveTabs()
  }

  /**
   * 切换标签
   */
  function switchTab(tabId: string) {
    const tab = tabs.value.find(t => t.id === tabId)
    if (tab) {
      activeTabId.value = tabId
      router.push(tab.path)
      saveTabs()
    }
  }

  /**
   * 关闭标签
   */
  function closeTab(tabId: string) {
    const index = tabs.value.findIndex(t => t.id === tabId)
    if (index === -1) return
    
    const wasActive = activeTabId.value === tabId
    tabs.value.splice(index, 1)
    
    // 如果关闭的是当前激活的标签，切换到其他标签
    if (wasActive && tabs.value.length > 0) {
      // 优先切换到右侧的标签，如果没有则切换到左侧
      const nextTab = tabs.value[index] || tabs.value[index - 1] || tabs.value[0]
      if (nextTab) {
        switchTab(nextTab.id)
      } else {
        // 没有标签了，跳转到应用根路径
        activeTabId.value = null
        if (route.path.includes('/workspace/')) {
          const pathSegments = route.path.replace('/workspace/', '').split('/').filter(Boolean)
          if (pathSegments.length >= 2) {
            const [user, app] = pathSegments
            router.push(`/workspace/${user}/${app}`)
          }
        }
      }
    }
    
    saveTabs()
  }

  /**
   * 关闭其他标签（只保留当前）
   */
  function closeOtherTabs(tabId: string) {
    const tab = tabs.value.find(t => t.id === tabId)
    if (tab) {
      tabs.value = [tab]
      activeTabId.value = tabId
      saveTabs()
    }
  }

  /**
   * 关闭所有标签
   */
  function closeAllTabs() {
    tabs.value = []
    activeTabId.value = null
    saveTabs()
    
    // 跳转到应用根路径
    if (route.path.includes('/workspace/')) {
      const pathSegments = route.path.replace('/workspace/', '').split('/').filter(Boolean)
      if (pathSegments.length >= 2) {
        const [user, app] = pathSegments
        router.push(`/workspace/${user}/${app}`)
      }
    }
  }

  /**
   * 根据路由路径同步标签（用于刷新后恢复）
   */
  function syncTabFromRoute() {
    const currentPath = route.path
    if (!currentPath.startsWith('/workspace/')) {
      return
    }
    
    // 检查当前路径是否对应某个标签
    const matchingTab = tabs.value.find(tab => tab.path === currentPath)
    if (matchingTab) {
      activeTabId.value = matchingTab.id
    } else {
      // 如果路由路径不在标签中，可能是直接访问的URL，不自动添加标签
      // 用户点击树节点时会自动添加
      activeTabId.value = null
    }
  }

  /**
   * 清理无效标签（函数已删除等）
   */
  function cleanupInvalidTabs() {
    // 可以根据需要实现清理逻辑
    // 例如：检查标签对应的函数是否还存在
  }

  // 当前激活的标签
  const activeTab = computed(() => {
    return tabs.value.find(tab => tab.id === activeTabId.value) || null
  })

  // 初始化：从 localStorage 恢复
  restoreTabs()

  // 监听路由变化，同步标签
  watch(() => route.path, () => {
    syncTabFromRoute()
  }, { immediate: true })

  return {
    tabs,
    activeTabId,
    activeTab,
    addOrActivateTab,
    switchTab,
    closeTab,
    closeOtherTabs,
    closeAllTabs,
    syncTabFromRoute,
    cleanupInvalidTabs
  }
}

