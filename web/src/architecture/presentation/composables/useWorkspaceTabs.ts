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
    
    console.log('[useWorkspaceTabs] handleTabClick: 处理 Tab 点击', {
      tabId,
      currentActiveTabId: activeTabId.value
    })
    
    // 🔥 直接切换路由，保存和恢复由 watch activeTabId 统一处理
    const tabPath = targetTab.path.startsWith('/') ? targetTab.path : `/${targetTab.path}`
    const targetPath = `/workspace${tabPath}`
    
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
      
      // 🔥 步骤 1：同步保存旧 Tab 的状态（必须在恢复新 Tab 之前）
      if (oldId) {
        const oldTab = tabs.value.find(t => t.id === oldId)
        if (oldTab && oldTab.node) {
          const detail = stateManager.getFunctionDetail(oldTab.node)
          if (detail?.template_type === 'table') {
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
            
            console.log('[useWorkspaceTabs] 保存旧 Tab 状态', {
              tabId: oldId,
              searchForm: oldTab.data.searchForm,
              searchFormKeys: Object.keys(oldTab.data.searchForm || {}),
              sorts: oldTab.data.sorts,
              pagination: oldTab.data.pagination
            })
          } else if (detail?.template_type === 'form') {
            const currentState = serviceFactoryInstance.getFormStateManager().getState()
            oldTab.data = {
              data: Array.from(currentState.data.entries()),
              errors: Array.from(currentState.errors.entries()),
              submitting: currentState.submitting
            }
          }
        }
      }
      
      // 🔥 步骤 2：立即恢复新 Tab 的状态（在 TableView.onMounted 之前）
      // 🔥 重要：必须先清空状态，再恢复，避免状态污染
      if (newId) {
        const newTab = tabs.value.find(t => t.id === newId)
        if (newTab && newTab.node) {
          const detail = stateManager.getFunctionDetail(newTab.node)
          
          if (detail?.template_type === 'form') {
            // 恢复 Form 数据
            if (newTab.data) {
              const savedState = newTab.data
              serviceFactoryInstance.getFormStateManager().setState({
                data: new Map(savedState.data),
                errors: new Map(savedState.errors),
                submitting: savedState.submitting
              })
              console.log('[useWorkspaceTabs] 恢复 Form 状态', { tabId: newId })
            } else {
              // 新 Tab 没有保存的数据，重置为默认状态
              serviceFactoryInstance.getFormStateManager().setState({
                data: new Map(),
                errors: new Map(),
                submitting: false
              })
              console.log('[useWorkspaceTabs] 新 Form Tab，重置状态', { tabId: newId })
            }
          } else if (detail?.template_type === 'table') {
            // 🔥 Table 类型：必须先清空状态，再恢复
            if (newTab.data) {
              // 有保存的数据，恢复到 TableStateManager
              const savedState = newTab.data
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
              
              console.log('[useWorkspaceTabs] 恢复 Table 状态', {
                tabId: newId,
                searchForm: savedState.searchForm,
                searchFormKeys: Object.keys(savedState.searchForm || {}),
                sorts: savedState.sorts,
                pagination: savedState.pagination
              })
            } else {
              // 🔥 新 Tab 没有保存的数据，必须重置为默认状态（避免状态污染）
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
              console.log('[useWorkspaceTabs] 新 Table Tab，重置状态', { tabId: newId })
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

