import type { TableState } from '@/architecture/domain/services/TableDomainService'

export interface RestoredTableURLState {
  searchForm: Record<string, any>
  sorts: Array<{ field: string; order: 'asc' | 'desc' }>
  pagination: { page: number; pageSize: number }
}

export type TableRestoreStrategy = 'restore-from-url' | 'sync-tab-state'

export interface DecideTableRestoreStrategyOptions {
  pathMatches: boolean
  query: Record<string, any>
  searchForm: Record<string, any>
  isLinkNavigation: boolean
}

export interface TableRouteReloadGuardOptions {
  source: string
  pathMatches: boolean
  isMounted: boolean
  isSyncingToURL: boolean
  isRestoringFromURL: boolean
  isInitializing: boolean
  newQuery: Record<string, any>
  oldQuery: Record<string, any>
}

export function normalizeTableRouteQuery(
  query: Record<string, any>
): Record<string, string | string[]> {
  const normalized: Record<string, string | string[]> = {}

  Object.keys(query || {}).forEach(key => {
    const value = query[key]
    if (value === null || value === undefined) {
      return
    }

    if (Array.isArray(value)) {
      normalized[key] = value.filter(item => item !== null && item !== undefined).map(item => String(item))
      return
    }

    normalized[key] = String(value)
  })

  return normalized
}

export function buildRestoredTableState(
  currentState: TableState,
  restored: RestoredTableURLState
): TableState {
  return {
    ...currentState,
    searchForm: restored.searchForm,
    sorts: restored.sorts,
    hasManualSort: restored.sorts.length > 0,
    sortParams: restored.sorts[0]
      ? {
          field: restored.sorts[0].field,
          order: restored.sorts[0].order
        }
      : null,
    pagination: {
      ...currentState.pagination,
      currentPage: restored.pagination.page,
      pageSize: restored.pagination.pageSize
    }
  }
}

export function buildClearedTableState(currentState: TableState, pageSize = currentState.pagination.pageSize): TableState {
  return {
    ...currentState,
    searchParams: {},
    searchForm: {},
    sortParams: null,
    sorts: [],
    hasManualSort: false,
    pagination: {
      currentPage: 1,
      pageSize,
      total: 0
    }
  }
}

export function decideTableRestoreStrategy(
  options: DecideTableRestoreStrategyOptions
): TableRestoreStrategy {
  const hasTabState = !!options.searchForm && Object.keys(options.searchForm).length > 0
  const hasURLParams = options.pathMatches && Object.keys(options.query || {}).length > 0

  if (options.isLinkNavigation && hasURLParams) {
    return 'restore-from-url'
  }

  if (hasTabState) {
    return 'sync-tab-state'
  }

  if (hasURLParams) {
    return 'restore-from-url'
  }

  return 'sync-tab-state'
}

export function shouldSyncTableURLAfterRestore(options: {
  query: Record<string, any>
  isLinkNavigation: boolean
  shouldSyncPageSize?: boolean
}): boolean {
  const hasPaginationParams = !!(options.query.page && options.query.page_size)
  if (options.shouldSyncPageSize) {
    return !options.isLinkNavigation
  }
  return !options.isLinkNavigation && !hasPaginationParams
}

export function isDetailViewQuery(query: Record<string, any>): boolean {
  return !!(query && query._tab === 'detail' && query._id)
}

export function isOnlyDetailParamsChanged(
  oldQuery: Record<string, any>,
  newQuery: Record<string, any>
): boolean {
  const omitDetailParams = (value: Record<string, any>) => {
    const nextValue = { ...(value || {}) }
    delete nextValue._tab
    delete nextValue._id
    return nextValue
  }

  return JSON.stringify(omitDetailParams(oldQuery)) === JSON.stringify(omitDetailParams(newQuery))
}

export function shouldSkipTableReloadOnRouteChange(
  options: TableRouteReloadGuardOptions
): boolean {
  if (options.source !== 'router-change') {
    return true
  }

  if (!options.pathMatches || !options.isMounted) {
    return true
  }

  if (options.isSyncingToURL || options.isRestoringFromURL || options.isInitializing) {
    return true
  }

  if (isDetailViewQuery(options.newQuery)) {
    return true
  }

  if (isOnlyDetailParamsChanged(options.oldQuery, options.newQuery)) {
    return true
  }

  return false
}
