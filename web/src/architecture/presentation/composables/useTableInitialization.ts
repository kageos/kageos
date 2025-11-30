/**
 * useTableInitialization - 表格初始化 Composable
 * 
 * 职责：
 * - 统一处理表格初始化逻辑
 * - 从 URL 恢复状态
 * - 同步状态到 URL
 * - 加载数据
 * 
 * 优化目标：
 * - 减少 TableView.vue 中的重复代码
 * - 统一状态管理
 * - 简化 watch 逻辑
 */

import { ref, watch, nextTick, type Ref, type ComputedRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { extractWorkspacePath } from '@/utils/route'
import { Logger } from '@/core/utils/logger'
import { serviceFactory } from '../../infrastructure/factories'
import type { FunctionDetail } from '../../domain/types'
import type { TableDomainService, SortItem } from '../../domain/services/TableDomainService'
import type { TableApplicationService } from '../../application/services/TableApplicationService'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import type { TableState } from '../../domain/services/TableDomainService'

export interface UseTableInitializationOptions {
  functionDetail: Ref<FunctionDetail> | ComputedRef<FunctionDetail>
  domainService: TableDomainService
  applicationService: TableApplicationService
  stateManager: IStateManager<TableState>
  searchForm: ComputedRef<Record<string, any>>
  sorts: ComputedRef<SortItem[]>
  hasManualSort: ComputedRef<boolean>
  buildDefaultSorts: () => SortItem[]
  syncToURL: () => void
  loadTableData: () => Promise<void>
  isMounted?: Ref<boolean> // 组件挂载状态（可选，用于防止卸载后继续加载数据）
}

export function useTableInitialization(options: UseTableInitializationOptions) {
  const {
    functionDetail,
    domainService,
    applicationService,
    stateManager,
    searchForm,
    sorts,
    hasManualSort,
    buildDefaultSorts,
    syncToURL,
    loadTableData,
    isMounted
  } = options

  const route = useRoute()
  const router = useRouter()

  // 初始化标志位（防止循环调用）
  const isInitializing = ref(false)
  const isRestoringFromURL = ref(false)
  const isSyncingToURL = ref(false)

  /**
   * 从 URL 恢复状态
   */
  const restoreFromURL = (): void => {
    // 🔥 注意：在初始化时允许调用 restoreFromURL，因为需要从 URL 恢复状态
    // 只有在 watch 中调用时才需要检查 isRestoringFromURL，避免循环调用
    // if (isRestoringFromURL.value) return

    const query = route.query

    // 转换 query 类型为 Domain Service 期望的类型
    const queryParams: Record<string, string | string[]> = {}
    Object.keys(query).forEach(key => {
      const value = query[key]
      if (value !== null && value !== undefined) {
        if (Array.isArray(value)) {
          queryParams[key] = value.filter(v => v !== null).map(v => String(v))
        } else {
          queryParams[key] = String(value)
        }
      }
    })

    // 使用 Domain Service 恢复状态
    const functionDetailValue = 'value' in functionDetail ? functionDetail.value : functionDetail
    const restored = domainService.restoreFromURL(functionDetailValue, queryParams)

    // 🔥 更新 StateManager 中的状态
    const currentState = stateManager.getState()
    stateManager.setState({
      ...currentState,
      searchForm: restored.searchForm,
      sorts: restored.sorts,
      hasManualSort: restored.sorts.length > 0,
      pagination: {
        ...currentState.pagination,
        currentPage: restored.pagination.page,
        pageSize: restored.pagination.pageSize
      }
    })
  }

  /**
   * 初始化表格（统一入口）
   */
  const initializeTable = async (): Promise<void> => {
    const functionDetailValue = 'value' in functionDetail ? functionDetail.value : functionDetail
    const functionId = functionDetailValue?.id
    const router = functionDetailValue?.router
    
    Logger.debug('useTableInitialization', 'initializeTable 开始', {
      functionId,
      router,
      isInitializing: isInitializing.value,
      isMounted: isMounted?.value
    })
    
    if (isInitializing.value) {
      Logger.warn('useTableInitialization', '正在初始化中，跳过', { functionId, router })
      return
    }
    
    // 🔥 检查组件是否还在挂载状态
    if (isMounted && !isMounted.value) {
      Logger.warn('useTableInitialization', '组件已卸载，跳过初始化', { functionId, router })
      return
    }
    
    isInitializing.value = true

    try {
      // 🔥 检查 URL 中是否有 query 参数
      const hasQueryParams = Object.keys(route.query).length > 0
      
      if (hasQueryParams) {
        // URL 中有 query 参数，从 URL 恢复状态
        restoreFromURL()
        
        // 同步状态到 URL（确保 URL 和状态一致）
        if (!isSyncingToURL.value) {
          isSyncingToURL.value = true
          await nextTick()
          syncToURL()
          isSyncingToURL.value = false
        }
      } else {
        // 🔥 URL 中没有 query 参数（Tab 切换或服务目录切换时）
        // 简化逻辑：从 Tab 的保存数据恢复状态，然后重新调用接口获取数据
        const workspaceStateManager = serviceFactory.getWorkspaceStateManager()
        const workspaceState = workspaceStateManager.getState()
        const activeTabId = workspaceState.activeTabId
        const tabs = Array.isArray(workspaceState.tabs) ? workspaceState.tabs : []
        const activeTab = activeTabId ? tabs.find(t => t.id === activeTabId) : null
        
        const currentState = stateManager.getState()
        
        // 🔥 从 Tab 的保存数据恢复状态（如果有）
        if (activeTab && activeTab.data && activeTab.data.searchForm !== undefined) {
          // 🔥 Tab 有保存的数据，恢复 Tab 的状态（包括搜索参数）
          Logger.debug('useTableInitialization', '从 Tab 保存的数据恢复状态', {
            tabId: activeTabId,
            hasSearchForm: !!activeTab.data.searchForm,
            searchForm: activeTab.data.searchForm,
            hasSorts: !!activeTab.data.sorts,
            sorts: activeTab.data.sorts,
            hasPagination: !!activeTab.data.pagination,
            pagination: activeTab.data.pagination
          })
          
          // 🔥 恢复 Tab 保存的状态（包括搜索参数、排序、分页）
          const finalSorts = activeTab.data.sorts || []
          stateManager.setState({
            searchForm: activeTab.data.searchForm || {},
            searchParams: activeTab.data.searchParams || {},
            sorts: finalSorts,
            hasManualSort: activeTab.data.hasManualSort || false,
            sortParams: finalSorts.length > 0 ? {
              field: finalSorts[0].field,
              order: finalSorts[0].order
            } : null,
            pagination: activeTab.data.pagination || {
              currentPage: 1,
              pageSize: 20,
              total: 0
            },
            data: [], // 🔥 清空数据，强制重新加载
            loading: false
          })
          
          // 同步状态到 URL（确保 URL 参数和接口请求参数对齐）
          if (!isSyncingToURL.value) {
            isSyncingToURL.value = true
            await nextTick()
            syncToURL() // 完整同步所有参数（分页、排序、搜索）
            await nextTick()
            isSyncingToURL.value = false
          }
        } else {
          // 🔥 Tab 没有保存的数据，清空状态，避免残留上一个函数的状态
          const defaultSorts = buildDefaultSorts()
          stateManager.setState({
            ...currentState,
            searchForm: {}, // 🔥 清空搜索表单，避免状态污染
            sorts: defaultSorts.length > 0 ? defaultSorts : [],
            hasManualSort: false,
            pagination: {
              ...currentState.pagination,
              currentPage: 1
            },
            data: [], // 🔥 清空数据，强制重新加载
            loading: false
          })
          
          // 同步状态到 URL（确保 URL 参数和接口请求参数对齐）
          if (!isSyncingToURL.value) {
            isSyncingToURL.value = true
            await nextTick()
            syncToURL()
            await nextTick()
            isSyncingToURL.value = false
          }
        }
      }

            // 🔥 再次检查组件是否还在挂载状态
            if (isMounted && !isMounted.value) {
              Logger.warn('useTableInitialization', '组件在初始化过程中已卸载，取消加载数据', { functionId, router })
              return
            }

            // 🔥 简化逻辑：每次初始化都重新调用接口获取数据
            Logger.debug('useTableInitialization', '开始加载数据', { functionId, router })
            await loadTableData()
            Logger.debug('useTableInitialization', '数据加载完成', { functionId, router })
    } finally {
      isInitializing.value = false
      Logger.debug('useTableInitialization', 'initializeTable 完成', { functionId, router })
    }
  }

  /**
   * 监听 URL 变化
   */
  watch(() => route.query, async (newQuery, oldQuery) => {
    const functionDetailValue = 'value' in functionDetail ? functionDetail.value : functionDetail
    const functionId = functionDetailValue?.id
    const router = functionDetailValue?.router
    
    // 检查当前路由是否匹配当前函数的 router
    // 如果路由已经切换到其他函数，这个 watch 不应该处理
    const currentPath = extractWorkspacePath(route.path)
    // 🔥 统一路径格式：移除前导斜杠，确保格式一致
    const expectedPath = (router || '').replace(/^\/+/, '')
    const pathMatches = currentPath === expectedPath || currentPath.startsWith(expectedPath + '?')
    
    Logger.debug('useTableInitialization', 'URL query 变化', {
      functionId,
      router,
      currentPath,
      expectedPath,
      pathMatches,
      newQuery,
      oldQuery,
      isMounted: isMounted?.value,
      isSyncingToURL: isSyncingToURL.value,
      isRestoringFromURL: isRestoringFromURL.value,
      isInitializing: isInitializing.value
    })
    
    // 🔥 如果路由不匹配当前函数，直接返回（可能是其他函数的路由变化）
    if (!pathMatches) {
      Logger.debug('useTableInitialization', '路由不匹配当前函数，忽略 URL 变化', {
        functionId,
        router,
        currentPath,
        expectedPath
      })
      return
    }
    
    // 🔥 检查组件是否还在挂载状态
    if (isMounted && !isMounted.value) {
      Logger.warn('useTableInitialization', '组件已卸载，忽略 URL 变化', { functionId, router })
      return
    }
    
    if (isSyncingToURL.value || isRestoringFromURL.value || isInitializing.value) {
      Logger.debug('useTableInitialization', '正在同步或初始化中，忽略 URL 变化', {
        functionId,
        router,
        isSyncingToURL: isSyncingToURL.value,
        isRestoringFromURL: isRestoringFromURL.value,
        isInitializing: isInitializing.value
      })
      return
    }

    isRestoringFromURL.value = true
    try {
      restoreFromURL()
      const hasQueryParams = Object.keys(route.query).length > 0
      if (!hasQueryParams) {
        // URL 中没有 query 参数，同步默认状态到 URL
        isSyncingToURL.value = true
        await nextTick()
        syncToURL()
        isSyncingToURL.value = false
      }
      
      // 🔥 再次检查组件是否还在挂载状态和路由是否匹配
      if (isMounted && !isMounted.value) {
        Logger.warn('useTableInitialization', '组件在 URL 恢复过程中已卸载，取消加载数据', { functionId, router })
        return
      }
      
      // 再次检查路由是否匹配（可能在异步操作期间路由又变化了）
      const currentPathAfterRestore = extractWorkspacePath(route.path)
      const pathMatchesAfterRestore = currentPathAfterRestore === expectedPath || currentPathAfterRestore.startsWith(expectedPath + '?')
      if (!pathMatchesAfterRestore) {
        Logger.debug('useTableInitialization', '路由在恢复过程中已变化，取消加载数据', {
          functionId,
          router,
          currentPathAfterRestore,
          expectedPath
        })
        return
      }
      
      Logger.debug('useTableInitialization', 'URL 变化后开始加载数据', { functionId, router })
      await loadTableData()
    } finally {
      isRestoringFromURL.value = false
    }
  }, { deep: true })

  /**
   * 监听函数变化
   * 🔥 防止重复初始化：只在函数 ID 或 router 变化时才初始化
   * 🔥 重要：不要在 watch 中调用 initializeTable，因为 onMounted 已经调用了
   * 如果 watch 也调用，会导致重复初始化
   */
  // 移除 watch，因为 onMounted 已经调用了 initializeTable
  // 如果需要在函数变化时重新初始化，应该在 WorkspaceView 中处理

  return {
    initializeTable,
    restoreFromURL,
    isInitializing,
    isRestoringFromURL,
    isSyncingToURL
  }
}

