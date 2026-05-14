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
import type { TableDomainService } from '../../domain/services/TableDomainService'
import type { SortItem, TableState } from '../../domain/types'
import type { TableApplicationService } from '../../application/services/TableApplicationService'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import { extractWorkspacePath } from '@/architecture/runtime/utils/route'
import { TEMPLATE_TYPE } from '@/architecture/runtime/utils/functionTypes'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { isLinkNavigation } from '@/architecture/runtime/utils/linkNavigation'
import {
  buildClearedTableState,
  buildRestoredTableState,
  decideTableRestoreStrategy,
  normalizeTableRouteQuery,
  shouldSkipTableReloadOnRouteChange,
  shouldSyncTableURLAfterRestore
} from './utils/tableInitializationRuntime'
import {
  DEFAULT_TABLE_PAGE_SIZE,
  readTablePageSizePreference,
  resolveTablePageSizeForRestore
} from '../views/utils/tablePageSizePreference'

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
    preloadUserInfoFromSearchForm,
    preloadDepartmentInfoFromSearchForm
  } = options

  const route = useRoute()

  // 初始化标志位（防止循环调用）
  const isInitializing = ref(false)
  const isRestoringFromURL = ref(false)
  const isSyncingToURL = ref(false)

  /**
   * 从 URL 恢复状态
   */
  const restoreFromURL = (): { shouldSyncPageSizeToURL: boolean } => {
    const queryParams = normalizeTableRouteQuery(route.query as Record<string, any>)
    const functionDetailValue = 'value' in functionDetail ? functionDetail.value : functionDetail
    const pageSizeResolution = resolveTablePageSizeForRestore({
      queryPageSize: queryParams.page_size,
      preferredPageSize: readTablePageSizePreference(functionDetailValue),
      isLinkNavigation: isLinkNavigation(route.query)
    })

    // 使用 Domain Service 恢复状态
    const restored = domainService.restoreFromURL(functionDetailValue, queryParams)
    restored.pagination.pageSize = pageSizeResolution.pageSize

    // 🔥 更新 StateManager 中的状态
    const currentState = stateManager.getState()
    stateManager.setState(buildRestoredTableState(currentState, restored))

    return {
      shouldSyncPageSizeToURL: pageSizeResolution.shouldSyncToURL
    }
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
    const restoreResult = restoreFromURL()
    // 🔥 等待状态更新完成，确保 restoreFromURL 的状态已经应用到 stateManager
    // 注意：stateManager.setState() 是同步的，但 Vue 的响应式更新是异步的，需要一个 tick
    await nextTick()
    
    // 🔥 link 跳转场景：URL 已经有参数，不需要再同步到 URL（避免覆盖）
    // 只有在 URL 参数不完整时才同步（比如只有搜索参数，没有分页参数）
    // 🔥 修复：检查是否是 link 跳转，如果是 link 跳转，即使没有分页参数，也不要同步
    // 因为 link 跳转时，URL 中的参数是用户明确指定的，不应该被覆盖
    const isLinkNav = isLinkNavigation(route.query)
    
    // 🔥 非 link 跳转时，URL 缺少分页参数或 page_size 与默认/偏好不一致才同步
    if (shouldSyncTableURLAfterRestore({
      query: route.query as Record<string, any>,
      isLinkNavigation: isLinkNav,
      shouldSyncPageSize: restoreResult.shouldSyncPageSizeToURL
    })) {
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
    const strategy = decideTableRestoreStrategy({
      pathMatches,
      query: route.query as Record<string, any>,
      searchForm: currentState.searchForm,
      isLinkNavigation: isLinkNavigation(route.query)
    })

    if (strategy === 'restore-from-url') {
      await restoreFromURLAndSync()
      return
    }

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
      stateManager.setState(buildClearedTableState(
        currentState,
        readTablePageSizePreference(functionDetailValue) || DEFAULT_TABLE_PAGE_SIZE
      ))
    }

    // 🔥 立即显示加载状态（骨架屏），避免先出现「暂无数据」再出现 loading
    const stateForLoading = stateManager.getState()
    stateManager.setState({ ...stateForLoading, loading: true })

    isInitializing.value = true

    try {
      // 🔥 步骤 1：决定恢复策略并执行
      // 优先级：Tab 保存的状态 > URL 参数
      // - 如果 Tab 有保存的状态（searchForm 不为空），说明是 Tab 切换，使用 Tab 的状态，不从 URL 恢复
      // - 如果 Tab 没有保存的状态（searchForm 为空），且 URL 有参数，说明是 link 跳转，从 URL 恢复
      await decideRestoreStrategy(router || '')
      
      // 🔥 时机 1：预加载搜索表单中的用户信息
      // 此时 searchForm 已经包含了从 URL 解析出来的所有搜索条件（如 in=created_by:luobei）
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
   * 当前 URL 是否处于「详情已打开」状态（有 _tab=detail 且 _id）。
   * 此时列表数据已存在，不应再触发表格接口重载和骨架屏。
   */
  /**
   * 监听 URL 变化（用户操作浏览器前进/后退时）
   * 🔥 打开详情或仅 _tab/_id 变化时不重载表格，避免多余请求和骨架屏
   */
  const setupQueryWatch = (): (() => void) => {
    return eventBus.on(RouteEvent.queryChanged, async (payload: { query: any, oldQuery: any, source: string }) => {
      const functionDetailValue = 'value' in functionDetail ? functionDetail.value : functionDetail
      const router = functionDetailValue?.router

      if (functionDetailValue?.template_type !== TEMPLATE_TYPE.TABLE) {
        return
      }

      const currentPath = extractWorkspacePath(route.path)
      const expectedPath = (router || '').replace(/^\/+/, '')
      const pathMatches = currentPath === expectedPath || currentPath.startsWith(expectedPath + '?')

      const shouldSkipReload = shouldSkipTableReloadOnRouteChange({
        source: payload.source,
        pathMatches,
        isMounted: isMounted ? isMounted.value : true,
        isSyncingToURL: isSyncingToURL.value,
        isRestoringFromURL: isRestoringFromURL.value,
        isInitializing: isInitializing.value,
        newQuery: payload.query || {},
        oldQuery: payload.oldQuery || {}
      })

      if (shouldSkipReload) {
        return
      }

      isRestoringFromURL.value = true
      try {
        const restoreResult = restoreFromURL()
        if (restoreResult.shouldSyncPageSizeToURL && !isSyncingToURL.value) {
          isSyncingToURL.value = true
          syncToURL()
          await nextTick()
          isSyncingToURL.value = false
        }

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
    })
  }

  return {
    initializeTable,
    isInitializing,
    restoreFromURL,
    setupQueryWatch  // 🔥 阶段4：导出 setupQueryWatch，需要在组件中调用
  }
}
