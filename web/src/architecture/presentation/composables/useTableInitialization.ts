/**
 * useTableInitialization - 表格初始化 Composable
 * 
 * 职责：
 * - 从 TableStateManager 获取状态（由 watch activeTabId 恢复）
 * - 同步状态到 URL
 * - 加载表格数据
 * - 监听 URL 变化并重新加载数据
 */

import { ref, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import type { FunctionDetail } from '../../../domain/types'
import type { TableDomainService, SortItem } from '../../../domain/services/TableDomainService'
import type { TableApplicationService } from '../../../application/services/TableApplicationService'
import type { IStateManager } from '../../../domain/interfaces/IStateManager'
import type { TableState } from '../../../domain/services/TableDomainService'
import { Logger } from '@/core/utils/logger'
import { extractWorkspacePath } from '@/utils/route'

export interface UseTableInitializationOptions {
  functionDetail: FunctionDetail | { value: FunctionDetail }
  domainService: TableDomainService
  applicationService: TableApplicationService
  stateManager: IStateManager<TableState>
  searchForm: any
  sorts: any
  hasManualSort: any
  buildDefaultSorts: () => SortItem[]
  syncToURL: () => void
  loadTableData: () => Promise<void>
  isMounted?: { value: boolean }
}

export function useTableInitialization(options: UseTableInitializationOptions) {
  const {
    functionDetail,
    domainService,
    stateManager,
    syncToURL,
    loadTableData,
    isMounted
  } = options

  const route = useRoute()

  // 初始化标志位（防止循环调用）
  const isInitializing = ref(false)
  const isRestoringFromURL = ref(false)
  const isSyncingToURL = ref(false)

  /**
   * 从 URL 恢复状态
   */
  const restoreFromURL = (): void => {
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
      searchParams: restored.searchParams, // 确保 searchParams 也被恢复
      sorts: restored.sorts,
      hasManualSort: restored.sorts.length > 0,
      sortParams: restored.sorts.length > 0 ? {
        field: restored.sorts[0].field,
        order: restored.sorts[0].order
      } : null,
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

    if (isInitializing.value) {
      Logger.warn('useTableInitialization', '正在初始化中，跳过', { functionId, router })
      return
    }

    if (isMounted && !isMounted.value) {
      Logger.warn('useTableInitialization', '组件已卸载，跳过初始化', { functionId, router })
      return
    }

    isInitializing.value = true

    try {
      // 🔥 步骤 1：从 TableStateManager 获取状态（已由 watch activeTabId 恢复）
      const currentState = stateManager.getState()
      
      Logger.debug('useTableInitialization', '开始初始化', {
        functionId,
        router,
        searchForm: currentState.searchForm,
        searchFormKeys: Object.keys(currentState.searchForm || {}),
        sorts: currentState.sorts,
        pagination: currentState.pagination
      })
      
      // 🔥 步骤 2：同步状态到 URL
      if (!isSyncingToURL.value) {
        isSyncingToURL.value = true
        await nextTick()
        syncToURL() // 完整同步所有参数（分页、排序、搜索）
        await nextTick()
        isSyncingToURL.value = false
      }
      
      // 🔥 步骤 3：加载数据
      if (isMounted && !isMounted.value) {
        Logger.warn('useTableInitialization', '组件在初始化过程中已卸载，取消加载数据', { functionId, router })
        return
      }
      
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
  watch(() => route.query, async (newQuery: any, oldQuery: any) => {
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

      // 🔥 再次检查组件是否还在挂载状态
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

  return {
    initializeTable,
    isInitializing,
    restoreFromURL
  }
}
