import type {
  TableSearchParams,
  SortItem,
  SortParams,
  TableState
} from '@/architecture/domain/types'
import type { FunctionDetail } from '@/architecture/domain/types'

export type TableLoadGuardResult = 'skip-unmounted' | 'skip-next-load' | 'proceed'

export interface BuildTableLoadRequestOptions {
  functionDetail: FunctionDetail
  state: TableState
  buildDefaultSorts: () => SortItem[]
  buildSearchParams: (functionDetail: FunctionDetail, searchForm: Record<string, any>) => TableSearchParams
}

export function decideTableLoadGuard(options: {
  isMounted: boolean
  skipNextTableLoad: boolean
}): TableLoadGuardResult {
  if (!options.isMounted) {
    return 'skip-unmounted'
  }

  if (options.skipNextTableLoad) {
    return 'skip-next-load'
  }

  return 'proceed'
}

export function buildTableLoadingState(
  currentState: TableState,
  loading: boolean
): TableState {
  return {
    ...currentState,
    loading
  }
}

export function buildTableLoadRequest(
  options: BuildTableLoadRequestOptions
): {
  searchParams: TableSearchParams
  sortParams: SortParams | undefined
  pagination: { page: number; pageSize: number }
} {
  const { functionDetail, state, buildDefaultSorts, buildSearchParams } = options
  const searchParams = buildSearchParams(functionDetail, state.searchForm)
  const finalSorts = state.sorts.length > 0
    ? state.sorts
    : (state.hasManualSort ? [] : buildDefaultSorts())

  const firstSort = finalSorts[0]

  return {
    searchParams,
    sortParams: firstSort
      ? {
          field: firstSort.field,
          order: firstSort.order
        }
      : undefined,
    pagination: {
      page: state.pagination.currentPage,
      pageSize: state.pagination.pageSize
    }
  }
}

export function getTableLoadErrorMessage(error: any): string {
  if (error?.response?.data?.msg) {
    return error.response.data.msg
  }

  if (error?.message) {
    return error.message
  }

  return '加载数据失败，请稍后重试'
}
