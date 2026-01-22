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
import type { FunctionDetail } from '../../domain/types'
import type { TableDomainService, SortItem } from '../../domain/services/TableDomainService'
import type { TableApplicationService } from '../../application/services/TableApplicationService'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import type { TableState } from '../../domain/services/TableDomainService'
import { extractWorkspacePath } from '@/utils/route'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { isLinkNavigation } from '@/utils/linkNavigation'

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
  /** 🔥 时机 1：预加载搜索表单中的用户信息（在 decideRestoreStrategy 完成后调用） */
  preloadUserInfoFromSearchForm?: (functionDetail: FunctionDetail, searchForm: Record<string, any>) => Promise<void>
  /** 🔥 时机 1：预加载搜索表单中的部门信息（在 decideRestoreStrategy 完成后调用） */
  preloadDepartmentInfoFromSearchForm?: (functionDetail: FunctionDetail, searchForm: Record<string, any>) => Promise<void>
}

export function useTableInitialization(options: UseTableInitializationOptions) {
  const {
    functionDetail,
    domainService,
    stateManager,
    syncToURL,
    loadTableData,
    isMounted,
    preloadUserInfoFromSearchForm
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
   * 检查路径是否匹配当前函数
   */
  const checkPathMatch = (router: string): boolean => {
    const currentPath = extractWorkspacePath(route.path)
    const expectedPath = (router || '').replace(/^\/+/, '')
    return currentPath === expectedPath || currentPath.startsWith(expectedPath + '?')
  }

  /**
   * 从 URL 恢复状态并同步到 URL（如果需要）
   */
  const restoreFromURLAndSync = async (): Promise<void> => {
    restoreFromURL()
    // 🔥 等待状态更新完成，确保 restoreFromURL 的状态已经应用到 stateManager
    // 注意：stateManager.setState() 是同步的，但 Vue 的响应式更新是异步的，需要一个 tick
    await nextTick()
    
    // 🔥 link 跳转场景：URL 已经有参数，不需要再同步到 URL（避免覆盖）
    // 只有在 URL 参数不完整时才同步（比如只有搜索参数，没有分页参数）
    // 🔥 修复：检查是否是 link 跳转，如果是 link 跳转，即使没有分页参数，也不要同步
    // 因为 link 跳转时，URL 中的参数是用户明确指定的，不应该被覆盖
    const isLinkNav = isLinkNavigation(route.query)
    const hasPaginationParams = route.query.page && route.query.page_size
    
    // 🔥 只有在非 link 跳转且没有分页参数时，才同步默认分页参数
    if (!isLinkNav && !hasPaginationParams) {
      // URL 中没有分页参数，需要同步默认分页参数
      if (!isSyncingToURL.value) {
        isSyncingToURL.value = true
        syncToURL() // 只同步分页和排序参数，保留 URL 中的搜索参数
        // syncToURL() 是同步的，路由更新是异步的，Vue Router 会自动处理
        await nextTick()
        isSyncingToURL.value = false
      }
    }
  }

  /**
   * 同步 Tab 状态到 URL
   */
  const syncTabStateToURL = async (): Promise<void> => {
    if (!isSyncingToURL.value) {
      isSyncingToURL.value = true
      syncToURL() // 完整同步所有参数（分页、排序、搜索）
      // syncToURL() 是同步的，路由更新是异步的，Vue Router 会自动处理
      await nextTick()
      isSyncingToURL.value = false
    }
  }

  /**
   * 决定恢复策略并执行
   * 使用早期返回优化条件判断
   */
  const decideRestoreStrategy = async (router: string): Promise<void> => {
    const currentState = stateManager.getState()
    const pathMatches = checkPathMatch(router)
    const hasTabState = currentState.searchForm && Object.keys(currentState.searchForm).length > 0
    const hasURLParams = pathMatches && Object.keys(route.query).length > 0
    
    // 🔥 检查是否是 link 跳转（通过 _link_type 参数）
    // link 跳转时，URL 中的参数是用户明确指定的（来自 link 值），应该优先从 URL 恢复
    const isLinkNav = isLinkNavigation(route.query)
    
    // 优先级 1：如果是 link 跳转，优先从 URL 恢复（即使 Tab 有状态也要覆盖）
    if (isLinkNav && hasURLParams) {
      await restoreFromURLAndSync()
      return
    }
    
    // 优先级 2：Tab 有保存的状态，优先使用 Tab 的状态（Tab 切换场景）
    if (hasTabState) {
      await syncTabStateToURL()
      return
    }
    
    // 优先级 3：Tab 没有保存的状态，且 URL 有参数，从 URL 恢复（link 跳转场景）
    if (hasURLParams) {
      await restoreFromURLAndSync()
      return
    }
    
    // 🔥 优先级 4：Tab 没有保存的状态，且 URL 没有参数（刚切换函数），清空状态
    // 这是关键修复：切换函数时，如果 URL 没有参数，说明是新的函数，应该清空旧的状态
    // 注意：状态清空已在 initializeTable 开始时处理，这里只需要同步默认参数到 URL
    if (!hasTabState && !hasURLParams && !isLinkNav) {
      // 清空状态后，同步默认参数到 URL
      await syncTabStateToURL()
      return
    }
    
    // 默认：同步 Tab 状态到 URL（即使没有状态，也需要同步默认参数）
    await syncTabStateToURL()
  }

  /**
   * 初始化表格（统一入口）
   */
  const initializeTable = async (): Promise<void> => {
    const functionDetailValue = 'value' in functionDetail ? functionDetail.value : functionDetail
    const router = functionDetailValue?.router

    if (isInitializing.value) {
      return
    }

    if (isMounted && !isMounted.value) {
      return
    }

    // 🔥 在初始化之前，先检查是否需要清空状态
    // 如果 URL 没有查询参数（刚切换函数），先清空 TableStateManager 的状态
    const hasQueryParams = Object.keys(route.query).length > 0
    const isLinkNav = isLinkNavigation(route.query)
    
    if (!hasQueryParams && !isLinkNav) {
      // 刚切换函数，清空 TableStateManager 的状态，避免旧函数的状态污染新函数
      const currentState = stateManager.getState()
      stateManager.setState({
        ...currentState,
        searchForm: {},
        sorts: [],
        hasManualSort: false,
        pagination: {
          currentPage: 1,
          pageSize: currentState.pagination.pageSize, // 保留分页大小
          total: 0
        }
      })
    }

    isInitializing.value = true

    try {
      // 🔥 步骤 1：决定恢复策略并执行
      // 优先级：Tab 保存的状态 > URL 参数
      // - 如果 Tab 有保存的状态（searchForm 不为空），说明是 Tab 切换，使用 Tab 的状态，不从 URL 恢复
      // - 如果 Tab 没有保存的状态（searchForm 为空），且 URL 有参数，说明是 link 跳转，从 URL 恢复
      await decideRestoreStrategy(router || '')
      
      // 🔥 时机 1：预加载搜索表单中的用户信息
      // 此时 searchForm 已经包含了从 URL 解析出来的所有搜索条件（如 in=create_by:luobei）
      if (preloadUserInfoFromSearchForm) {
        const currentState = stateManager.getState()
        await preloadUserInfoFromSearchForm(functionDetailValue, currentState.searchForm)
      }
      
      // 🔥 时机 1：预加载搜索表单中的部门信息
      // 此时 searchForm 已经包含了从 URL 解析出来的所有搜索条件（如 in=department:/dept1）
      if (preloadDepartmentInfoFromSearchForm) {
        const currentState = stateManager.getState()
        await preloadDepartmentInfoFromSearchForm(functionDetailValue, currentState.searchForm)
      }
      
      // 🔥 步骤 2：加载数据
      if (isMounted && !isMounted.value) {
        return
      }
      
      await loadTableData()
    } finally {
      isInitializing.value = false
    }
  }

  /**
   * 监听 URL 变化（用户操作浏览器前进/后退时）
   * 🔥 阶段4：改为监听 RouteEvent.queryChanged 事件，而不是直接 watch route.query
   * 这样可以避免程序触发的路由更新导致循环
   */
  const setupQueryWatch = () => {
    eventBus.on(RouteEvent.queryChanged, async (payload: { query: any, oldQuery: any, source: string }) => {
      // 🔥 只处理用户操作（浏览器前进/后退）或外部变化，不处理程序触发的更新
      if (payload.source === 'router-change') {
        const functionDetailValue = 'value' in functionDetail ? functionDetail.value : functionDetail
        const router = functionDetailValue?.router

        // 🔥 检查当前函数类型，如果是 form 函数，不应该处理 URL 变化
        if (functionDetailValue?.template_type !== TEMPLATE_TYPE.TABLE) {
          return
        }

        // 检查当前路由是否匹配当前函数的 router
        const currentPath = extractWorkspacePath(route.path)
        const expectedPath = (router || '').replace(/^\/+/, '')
        const pathMatches = currentPath === expectedPath || currentPath.startsWith(expectedPath + '?')

        if (!pathMatches) {
          return
        }

        // 🔥 检查组件是否还在挂载状态
        if (isMounted && !isMounted.value) {
          return
        }

        if (isSyncingToURL.value || isRestoringFromURL.value || isInitializing.value) {
          return
        }

        isRestoringFromURL.value = true
        try {
          restoreFromURL()

          if (isMounted && !isMounted.value) {
            return
          }

          const currentPathAfterRestore = extractWorkspacePath(route.path)
          const pathMatchesAfterRestore = currentPathAfterRestore === expectedPath || currentPathAfterRestore.startsWith(expectedPath + '?')
          if (!pathMatchesAfterRestore) {
            return
          }

          await loadTableData()
        } finally {
          isRestoringFromURL.value = false
        }
      }
    })
  }

  return {
    initializeTable,
    isInitializing,
    restoreFromURL,
    setupQueryWatch  // 🔥 阶段4：导出 setupQueryWatch，需要在组件中调用
  }
}
