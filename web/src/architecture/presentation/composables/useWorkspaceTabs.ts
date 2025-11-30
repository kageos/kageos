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

  // 🔥 保存当前 Tab 的状态到 localStorage（单独函数，可以在多处调用）
  const saveCurrentTabState = () => {
    const currentActiveTabId = activeTabId.value
    if (!currentActiveTabId) return
    
    const currentTab = tabs.value.find(t => t.id === currentActiveTabId)
    if (!currentTab || !currentTab.node) return
    
    const detail = stateManager.getFunctionDetail(currentTab.node)
    if (!detail) return
    
    if (detail.template_type === 'form') {
      const currentState = serviceFactoryInstance.getFormStateManager().getState()
      currentTab.data = {
        data: Array.from(currentState.data.entries()),
        errors: Array.from(currentState.errors.entries()),
        submitting: currentState.submitting
      }
    } else if (detail.template_type === 'table') {
      const tableStateManager = serviceFactoryInstance.getTableStateManager()
      const currentState = tableStateManager.getState()
      
      currentTab.data = {
        searchForm: { ...currentState.searchForm },
        searchParams: { ...currentState.searchParams },
        sorts: [...currentState.sorts],
        hasManualSort: currentState.hasManualSort,
        pagination: { ...currentState.pagination },
        data: [...currentState.data],
        loading: false,
        sortParams: currentState.sortParams
      }
      
      console.log('[useWorkspaceTabs] 保存当前 Tab 状态', {
        tabId: currentActiveTabId,
        searchForm: currentTab.data.searchForm,
        searchFormKeys: Object.keys(currentTab.data.searchForm || {}),
        sorts: currentTab.data.sorts,
        pagination: currentTab.data.pagination
      })
    }
    
    // 保存到 localStorage
    saveTabsToLocalStorage()
  }

  // Tab 点击处理：保存当前状态，切换路由，恢复目标状态
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
      console.warn('[useWorkspaceTabs] handleTabClick: 无法提取 tabId', { 
        tab,
        tabType: typeof tab,
        tabKeys: tab && typeof tab === 'object' ? Object.keys(tab) : []
      })
      return
    }
    
    const targetTab = tabs.value.find(t => t.id === tabId)
    if (!targetTab || !targetTab.path) {
      console.warn('[useWorkspaceTabs] handleTabClick: 未找到对应的 tab', {
        tabId,
        availableTabs: tabs.value.map(t => ({ id: t.id, path: t.path }))
      })
      return
    }
    
    const currentActiveTabId = activeTabId.value
    
    console.log('[useWorkspaceTabs] handleTabClick: 处理 Tab 点击', {
      tabId,
      currentActiveTabId,
      needSwitch: currentActiveTabId !== tabId
    })
    
    // 🔥 步骤1：保存当前 Tab 的状态
    if (currentActiveTabId && currentActiveTabId !== tabId) {
      saveCurrentTabState()
    }
    
    // 🔥 步骤2：切换到目标 Tab（路由优先）
    const tabPath = targetTab.path.startsWith('/') ? targetTab.path : `/${targetTab.path}`
    const targetPath = `/workspace${tabPath}`
    
    console.log('[useWorkspaceTabs] handleTabClick: 切换到目标 Tab', {
      tabId,
      targetPath,
      hasSavedData: !!targetTab.data
    })
    
    // 🔥 步骤3：更新路由（不带 query，状态通过 watch activeTabId 恢复）
    // 清空 query 确保路由变化能触发 initializeTable
    router.replace({ path: targetPath, query: {} }).catch((err) => {
      console.error('[useWorkspaceTabs] handleTabClick: 路由更新失败', err)
    })
  }

  // Tab 编辑处理（添加/删除）
  const handleTabsEdit = (targetName: string | undefined, action: 'remove' | 'add') => {
    if (action === 'remove' && targetName) {
      applicationService.closeTab(targetName)
    }
  }

  // Tab 数据保存/恢复（watch activeTabId）
  const setupTabDataWatch = () => {
    watch(() => stateManager.getState().activeTabId, (newId, oldId) => {
      console.log('[useWorkspaceTabs] watch activeTabId 触发', { oldId, newId })
      
      // 🔥 注意：保存逻辑已移至 handleTabClick，这里只负责恢复
      // 不在这里保存的原因：watch 触发时，TableStateManager 的状态可能已被新 Tab 覆盖

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
              console.log('[useWorkspaceTabs] 恢复 Tab 数据', {
                tabId: newId,
                savedState,
                hasSearchForm: !!savedState.searchForm,
                searchForm: savedState.searchForm,
                hasSorts: !!savedState.sorts,
                sorts: savedState.sorts,
                hasPagination: !!savedState.pagination,
                pagination: savedState.pagination,
                hasData: !!(savedState.data && savedState.data.length > 0)
              })
              
              // 🔥 确保所有字段都被正确恢复，包括 searchForm
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
                loading: false
              })
              
              console.log('[useWorkspaceTabs] Tab 数据恢复完成', {
                tabId: newId,
                restoredState: serviceFactoryInstance.getTableStateManager().getState(),
                searchForm: serviceFactoryInstance.getTableStateManager().getState().searchForm
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

