/**
 * useWorkspaceTabs - Tab 管理 Composable
 * 
 * 职责：
 * - Tab 打开/关闭/激活
 * - Tab 持久化（localStorage）
 * - Tab 数据保存/恢复
 * - Tab 节点重新关联
 */

import { computed, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { serviceFactory } from '../../infrastructure/factories'
import type { ServiceTree } from '../../domain/services/WorkspaceDomainService'
import type { ServiceTree as ServiceTreeType } from '@/types'
import { FUNCTION_TYPE } from '@/utils/functionTypes'

export function useWorkspaceTabs() {
  const router = useRouter()
  const stateManager = serviceFactory.getWorkspaceStateManager()
  const applicationService = serviceFactory.getWorkspaceApplicationService()
  const serviceFactoryInstance = serviceFactory

  // Tab 状态
  const tabs = computed(() => {
    const stateTabs = stateManager.getState().tabs
    return Array.isArray(stateTabs) ? stateTabs : []
  })

  const activeTabId = computed({
    get: () => stateManager.getState().activeTabId || '',
    set: (val) => {
      if (!val) return
      
      // 🔥 如果是当前激活的 Tab，忽略（避免重复切换）
      if (val === stateManager.getState().activeTabId) {
        return
      }
      
      // 🔥 先更新路由，路由变化会触发 syncRouteToTab → activateTab
      const targetTab = tabs.value.find(t => t.id === val)
      if (targetTab && targetTab.path) {
        const tabPath = targetTab.path.startsWith('/') ? targetTab.path : `/${targetTab.path}`
        const targetPath = `/workspace${tabPath}`
        router.replace({ path: targetPath, query: {} }).catch(() => {})
      }
    }
  })

  // Tab 点击处理：直接切换路由，保存和恢复由 watch activeTabId 统一处理
  const handleTabClick = (tab: any) => {
    let tabId: string | undefined
    
    if (typeof tab === 'string') {
      tabId = tab
    } else if (tab && typeof tab === 'object') {
      tabId = tab.name || tab.paneName || (tab as any)?.props?.name
      if (!tabId && 'name' in tab) {
        try {
          tabId = String(tab.name)
        } catch (e) {
          // 忽略错误
        }
      }
    }
    
    if (!tabId) {
      return
    }
    
    // 🔥 如果点击的是当前激活的 Tab，忽略（避免重复切换）
    if (tabId === activeTabId.value) {
      return
    }
    
    const targetTab = tabs.value.find(t => t.id === tabId)
    if (!targetTab || !targetTab.path) {
      return
    }
    
    // 🔥 直接切换路由，保存和恢复由 watch activeTabId 统一处理
    const tabPath = targetTab.path.startsWith('/') ? targetTab.path : `/${targetTab.path}`
    const targetPath = `/workspace${tabPath}`
    
    router.replace({ path: targetPath, query: {} }).catch(() => {})
  }

  // Tab 编辑处理（添加/删除）
  const handleTabsEdit = (targetName: string | undefined, action: 'remove' | 'add') => {
    if (action === 'remove' && targetName) {
      applicationService.closeTab(targetName)
    }
  }

  // 保存旧 Tab 的状态
  const saveOldTabState = (oldId: string) => {
    const oldTab = tabs.value.find(t => t.id === oldId)
    if (!oldTab || !oldTab.node) return

    const detail = stateManager.getFunctionDetail(oldTab.node)
    if (detail?.template_type === FUNCTION_TYPE.TABLE) {
      // 从 TableStateManager 获取当前状态并保存
      const tableStateManager = serviceFactoryInstance.getTableStateManager()
      const currentState = tableStateManager.getState()
      
      oldTab.data = {
        searchForm: { ...currentState.searchForm },
        searchParams: { ...currentState.searchParams },
        sorts: [...currentState.sorts],
        hasManualSort: currentState.hasManualSort,
        pagination: { ...currentState.pagination },
        data: [...currentState.data],
        loading: false,
        sortParams: currentState.sortParams
      }
    } else if (detail?.template_type === FUNCTION_TYPE.FORM) {
      const currentState = serviceFactoryInstance.getFormStateManager().getState()
      oldTab.data = {
        data: Array.from(currentState.data.entries()),
        errors: Array.from(currentState.errors.entries()),
        submitting: currentState.submitting
      }
    }
  }

  // 恢复 Form 状态
  const restoreFormState = (savedState: any) => {
    if (savedState) {
      serviceFactoryInstance.getFormStateManager().setState({
        data: new Map(savedState.data),
        errors: new Map(savedState.errors),
        submitting: savedState.submitting
      })
    } else {
      // 新 Tab 没有保存的数据，重置为默认状态
      serviceFactoryInstance.getFormStateManager().setState({
        data: new Map(),
        errors: new Map(),
        submitting: false
      })
    }
  }

  // 恢复 Table 状态
  const restoreTableState = (savedState: any) => {
    if (savedState && savedState.searchForm !== undefined) {
      // 检查是否有有效的保存数据（searchForm 不为 undefined）
      serviceFactoryInstance.getTableStateManager().setState({
        searchForm: savedState.searchForm || {},
        searchParams: savedState.searchParams || {},
        sorts: savedState.sorts || [],
        hasManualSort: savedState.hasManualSort || false,
        pagination: savedState.pagination || {
          currentPage: 1,
          pageSize: 20,
          total: 0
        },
        data: savedState.data || [],
        loading: false,
        sortParams: savedState.sortParams || null
      })
    } else {
      // 新 Tab 没有有效的保存数据，必须重置为默认状态（避免状态污染）
      serviceFactoryInstance.getTableStateManager().setState({
        data: [],
        loading: false,
        searchParams: {},
        searchForm: {},
        sortParams: null,
        sorts: [],
        hasManualSort: false,
        pagination: {
          currentPage: 1,
          pageSize: 20,
          total: 0
        }
      })
    }
  }

  // 恢复新 Tab 的状态
  const restoreNewTabState = (newId: string) => {
    const newTab = tabs.value.find(t => t.id === newId)
    if (!newTab || !newTab.node) return

    const detail = stateManager.getFunctionDetail(newTab.node)
    
    if (detail?.template_type === FUNCTION_TYPE.FORM) {
      restoreFormState(newTab.data)
    } else if (detail?.template_type === FUNCTION_TYPE.TABLE) {
      restoreTableState(newTab.data)
    }
    
    // 检查函数详情是否已加载（刷新后切换 Tab 时可能需要加载）
    if (newTab.node && newTab.node.type === 'function') {
      const detail = stateManager.getFunctionDetail(newTab.node)
      if (!detail) {
        // 使用 handleNodeClick 加载函数详情
        applicationService.handleNodeClick(newTab.node)
      }
    }
  }

  // Tab 数据保存/恢复（watch activeTabId）
  const setupTabDataWatch = () => {
    watch(() => stateManager.getState().activeTabId, (newId, oldId) => {
      // 🔥 步骤 1：同步保存旧 Tab 的状态（必须在恢复新 Tab 之前）
      if (oldId) {
        saveOldTabState(oldId)
      }
      
      // 🔥 步骤 2：立即恢复新 Tab 的状态（在 TableView.onMounted 之前）
      // 🔥 重要：必须先清空状态，再恢复，避免状态污染
      if (newId) {
        restoreNewTabState(newId)
      }
    })
  }

  // 从 localStorage 恢复 Tabs
  const restoreTabsFromStorage = () => {
    try {
      const savedTabs = localStorage.getItem('workspace-tabs')
      const savedActiveTabId = localStorage.getItem('workspace-activeTabId')
      
      if (savedTabs) {
        const tabs = JSON.parse(savedTabs)
        const state = stateManager.getState()
        
        // 确保 tabs 是数组
        const tabsArray = Array.isArray(tabs) ? tabs : []
        
        // 恢复 tabs（注意：node 信息需要后续重新关联）
        stateManager.setState({
          ...state,
          tabs: tabsArray,
          activeTabId: savedActiveTabId || null
        })
      }
    } catch (error) {
      // 静默失败
    }
  }

  // 保存 Tabs 到 localStorage
  const saveTabsToStorage = () => {
    try {
      const state = stateManager.getState()
      
      // 确保 tabs 是数组
      if (!Array.isArray(state.tabs)) {
        return
      }
      
      const tabsToSave = state.tabs.map(tab => ({
        id: tab.id,
        title: tab.title,
        path: tab.path,
        data: tab.data
        // 注意：不保存 node，因为 node 是对象引用，刷新后需要重新关联
      }))
      
      localStorage.setItem('workspace-tabs', JSON.stringify(tabsToSave))
      localStorage.setItem('workspace-activeTabId', state.activeTabId || '')
    } catch (error) {
      // 静默失败
    }
  }

  // 设置自动保存到 localStorage
  const setupAutoSave = () => {
    watch(() => [stateManager.getState().tabs, stateManager.getState().activeTabId], () => {
      saveTabsToStorage()
    }, { deep: true })
  }

  // 重新关联 tabs 的 node 信息（服务树加载后调用）
  const restoreTabsNodes = (serviceTree: ServiceTreeType[], findNodeByPath: (tree: ServiceTreeType[], path: string) => ServiceTreeType | null) => {
    const state = stateManager.getState()
    
    if (serviceTree.length === 0) return
    
    // 确保 tabs 是数组
    if (!Array.isArray(state.tabs)) {
      return
    }
    
    let hasChanges = false
    const updatedTabs = state.tabs.map(tab => {
      if (tab.node) {
        // 已有 node，不需要更新
        return tab
      }
      
      // 根据 path 查找对应的 node
      const node = findNodeByPath(serviceTree, tab.path)
      if (node) {
        hasChanges = true
        return {
          ...tab,
          node: node as any
        }
      }
      
      return tab
    })
    
    if (hasChanges) {
      stateManager.setState({
        ...state,
        tabs: updatedTabs
      })
      
      // 重新关联 node 后，检查当前激活的 tab 是否需要加载函数详情
      nextTick(() => {
        const currentState = stateManager.getState()
        const activeTabId = currentState.activeTabId
        if (activeTabId) {
          const activeTab = updatedTabs.find(t => t.id === activeTabId)
          if (activeTab && activeTab.node && activeTab.node.type === 'function') {
            // 检查函数详情是否已加载
            const detail = stateManager.getFunctionDetail(activeTab.node)
            if (!detail) {
              // 使用 handleNodeClick 加载函数详情
              applicationService.handleNodeClick(activeTab.node)
            }
          }
        }
      })
    }
  }

  return {
    // 状态
    tabs,
    activeTabId,
    
    // 方法
    handleTabClick,
    handleTabsEdit,
    restoreTabsFromStorage,
    saveTabsToStorage,
    restoreTabsNodes,
    
    // 设置
    setupTabDataWatch,
    setupAutoSave
  }
}

