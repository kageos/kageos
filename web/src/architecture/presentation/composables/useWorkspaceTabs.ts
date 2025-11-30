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
      if (val) applicationService.activateTab(val)
    }
  })

  // Tab 点击处理：只更新路由，路由变化会触发 Tab 状态更新
  const handleTabClick = (tab: any) => {
    // Element Plus 的 tab-click 事件传递的参数可能是：
    // 1. TabsPaneContext 对象（可能是 Proxy），包含 name 属性
    // 2. 或者直接是 tabId (字符串)
    // 3. 或者是包含 paneName 属性的对象
    
    let tabId: string | undefined
    
    // 尝试多种方式提取 tabId
    if (typeof tab === 'string') {
      tabId = tab
    } else if (tab && typeof tab === 'object') {
      // 尝试从对象中提取 name 或 paneName
      tabId = tab.name || tab.paneName || (tab as any)?.props?.name
      
      // 如果还是找不到，尝试直接访问 Proxy 对象的属性
      if (!tabId && 'name' in tab) {
        try {
          tabId = String(tab.name)
        } catch (e) {
          // 忽略错误
        }
      }
    }
    
    if (!tabId) {
      console.warn('[useWorkspaceTabs] handleTabClick: 无法提取 tabId', { 
        tab,
        tabType: typeof tab,
        tabKeys: tab && typeof tab === 'object' ? Object.keys(tab) : []
      })
      return
    }
    
    const targetTab = tabs.value.find(t => t.id === tabId)
    if (targetTab && targetTab.path) {
      const tabPath = targetTab.path.startsWith('/') ? targetTab.path : `/${targetTab.path}`
      const targetPath = `/workspace${tabPath}`
      const currentPath = router.currentRoute.value.path
      
      console.log('[useWorkspaceTabs] handleTabClick: 处理 Tab 点击', {
        tabId,
        tabPath: targetTab.path,
        targetPath,
        currentPath,
        pathMatches: currentPath === targetPath
      })
      
      // 🔥 路由优先策略：始终更新路由以清除 query 参数并触发路由变化
      // 路由变化会触发 syncRouteToTab → activateTab
      const pathMatches = currentPath === targetPath
      const hasQueryParams = Object.keys(router.currentRoute.value.query).length > 0
      const currentActiveTabId = activeTabId.value
      const stateNeedsSync = currentActiveTabId !== tabId
      
      console.log('[useWorkspaceTabs] handleTabClick: 处理 Tab 点击', {
        tabId,
        targetPath,
        currentPath,
        pathMatches,
        hasQueryParams,
        currentActiveTabId,
        stateNeedsSync
      })
      
      // 🔥 Tab 切换时：清空 URL 参数，然后从 Tab 的保存数据恢复状态
      // 这是 Tab 的核心功能：保持切换时的状态，切换回去时恢复切换前的参数
      // 注意：不保留当前 URL 的参数，因为那些是当前 Tab 的参数，不是目标 Tab 的参数
      // 目标 Tab 的状态应该从 Tab 的保存数据中恢复，而不是从当前 URL 中恢复
      
      // 🔥 强制触发路由更新：先添加临时参数，然后清空，确保路由变化被触发
      const tempQuery = { _refresh: Date.now().toString() }
      router.replace({ path: targetPath, query: tempQuery }).then(() => {
        // 清空 URL 参数，触发路由变化
        // Tab 的状态会从 Tab 的保存数据中恢复（通过 setupTabDataWatch）
        return router.replace({ path: targetPath, query: {} })
      }).then(() => {
        // 如果路径相同且没有 query 参数，Vue Router 可能不会触发路由变化
        // 此时需要检查状态是否同步，如果不同步则手动激活 Tab
        if (pathMatches && stateNeedsSync) {
          console.log('[useWorkspaceTabs] handleTabClick: 路径相同但状态不同步，手动激活 Tab', { 
            tabId, 
            currentActiveTabId 
          })
          // 手动激活 Tab 以确保状态同步
          applicationService.activateTab(tabId)
        }
      }).catch((err) => {
        console.error('[useWorkspaceTabs] handleTabClick: 路由更新失败', err)
      })
      
      // 注意：路由更新会触发 watch route.path → syncRouteToTab → activateTab
      // 但如果路径相同且无 query 参数，watch 可能不会触发，所以上面做了手动处理
    } else {
      console.warn('[useWorkspaceTabs] handleTabClick: 未找到对应的 tab', {
        tabId,
        availableTabs: tabs.value.map(t => ({ id: t.id, path: t.path }))
      })
    }
  }

  // Tab 编辑处理（添加/删除）
  const handleTabsEdit = (targetName: string | undefined, action: 'remove' | 'add') => {
    if (action === 'remove' && targetName) {
      applicationService.closeTab(targetName)
    }
  }

  // Tab 数据保存/恢复（watch activeTabId）
  const setupTabDataWatch = () => {
    watch(() => stateManager.getState().activeTabId, async (newId, oldId) => {
      console.log('[useWorkspaceTabs] watch activeTabId 触发', { oldId, newId })
      
      // 1. 保存旧 Tab 数据
      if (oldId) {
        const oldTab = tabs.value.find(t => t.id === oldId)
        if (oldTab && oldTab.node) {
          const detail = stateManager.getFunctionDetail(oldTab.node)
          if (detail?.template_type === 'form') {
            // 深度克隆，避免引用问题
            const currentState = serviceFactoryInstance.getFormStateManager().getState()
            oldTab.data = JSON.parse(JSON.stringify({
              data: Array.from(currentState.data.entries()), // Map 转 Array 以便序列化
              errors: Array.from(currentState.errors.entries()),
              submitting: currentState.submitting
            }))
          } else if (detail?.template_type === 'table') {
            const currentState = serviceFactoryInstance.getTableStateManager().getState()
            oldTab.data = JSON.parse(JSON.stringify(currentState))
          }
        }
      }

      // 2. 恢复新 Tab 数据
      if (newId) {
        const newTab = tabs.value.find(t => t.id === newId)
        if (newTab) {
          // 2.1 恢复 Tab 数据（如果有保存的数据）
          if (newTab.data && newTab.node) {
            const detail = stateManager.getFunctionDetail(newTab.node)
            if (detail?.template_type === 'form') {
              // 恢复 Form 数据
              const savedState = newTab.data
              serviceFactoryInstance.getFormStateManager().setState({
                data: new Map(savedState.data),
                errors: new Map(savedState.errors),
                submitting: savedState.submitting
              })
            } else if (detail?.template_type === 'table') {
              // 🔥 恢复 Table 数据：确保完全替换状态，避免残留上一个Tab的状态
              const savedState = newTab.data
              serviceFactoryInstance.getTableStateManager().setState({
                ...savedState,
                // 确保所有字段都被正确恢复，包括 searchForm
                searchForm: savedState.searchForm || {},
                sorts: savedState.sorts || [],
                hasManualSort: savedState.hasManualSort || false,
                pagination: savedState.pagination || {
                  currentPage: 1,
                  pageSize: 20,
                  total: 0
                },
                data: savedState.data || [],
                loading: false
              })
            }
          } else {
            // 🔥 Tab 没有保存的数据，清空状态，确保不会残留上一个Tab的状态
            const newTabNode = newTab?.node
            if (newTabNode) {
              const detail = stateManager.getFunctionDetail(newTabNode)
              if (detail?.template_type === 'table') {
                // 清空 Table 状态，避免残留上一个Tab的状态
                const defaultState = {
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
                }
                serviceFactoryInstance.getTableStateManager().setState(defaultState)
              }
            }
          }
          
          // 2.2 检查函数详情是否已加载（刷新后切换 Tab 时可能需要加载）
          if (newTab.node && newTab.node.type === 'function') {
            const detail = stateManager.getFunctionDetail(newTab.node)
            if (!detail) {
              console.log('[useWorkspaceTabs] Tab 切换但函数详情未加载，加载详情', {
                tabId: newId,
                path: newTab.path,
                nodeId: newTab.node.id,
                nodePath: newTab.node.full_code_path
              })
              // 使用 handleNodeClick 加载函数详情
              applicationService.handleNodeClick(newTab.node)
            }
          }
        }
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
        
        console.log('[useWorkspaceTabs] 从 localStorage 恢复 tabs', { 
          tabsCount: tabsArray.length, 
          activeTabId: savedActiveTabId 
        })
      }
    } catch (error) {
      console.error('[useWorkspaceTabs] 恢复 tabs 失败', error)
    }
  }

  // 保存 Tabs 到 localStorage
  const saveTabsToStorage = () => {
    try {
      const state = stateManager.getState()
      
      // 确保 tabs 是数组
      if (!Array.isArray(state.tabs)) {
        console.warn('[useWorkspaceTabs] state.tabs 不是数组，跳过保存', { tabs: state.tabs })
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
      
      console.log('[useWorkspaceTabs] 保存 tabs 到 localStorage', { 
        tabsCount: tabsToSave.length, 
        activeTabId: state.activeTabId 
      })
    } catch (error) {
      console.error('[useWorkspaceTabs] 保存 tabs 失败', error)
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
      console.warn('[useWorkspaceTabs] state.tabs 不是数组，跳过重新关联 node', { tabs: state.tabs })
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
      console.log('[useWorkspaceTabs] 重新关联 tabs 的 node 信息', { tabsCount: updatedTabs.length })
      
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
              console.log('[useWorkspaceTabs] 恢复 tab 后，加载函数详情', { 
                tabId: activeTabId, 
                path: activeTab.path,
                nodeId: activeTab.node.id,
                nodePath: activeTab.node.full_code_path
              })
              // 使用 handleNodeClick 加载函数详情
              applicationService.handleNodeClick(activeTab.node)
            } else {
              console.log('[useWorkspaceTabs] 恢复 tab 后，函数详情已存在', { 
                tabId: activeTabId, 
                detailId: detail.id 
              })
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

