import type { FunctionDetail } from '@/architecture/domain/types'
import type { SortItem, TableState } from '@/architecture/domain/services/TableDomainService'
import { buildURLSearchParams } from '@/utils/searchParams'
import { LINK_TYPE_QUERY_KEY } from '@/utils/linkNavigation'
import { TABLE_PARAM_KEYS, SEARCH_PARAM_KEYS } from '@/utils/urlParams'
import { getSearchFieldQueryKey, isSearchFieldQueryKey } from '@/utils/queryFieldNamespace'

interface BuildTableURLQueryParamsOptions {
  functionDetail: FunctionDetail
  state: TableState
  buildDefaultSorts: () => SortItem[]
}

interface PreserveExistingTableQueryParamsOptions {
  routeQuery: Record<string, any>
  requestFieldCodes: Set<string>
  isLinkNavigation: boolean
}

interface BuildNextTableSyncQueryOptions extends BuildTableURLQueryParamsOptions {
  routeQuery: Record<string, any>
  isLinkNavigation: boolean
}

const hasSearchValue = (value: unknown): boolean => {
  if (value === null || value === undefined) {
    return false
  }

  if (Array.isArray(value)) {
    return value.length > 0
  }

  if (typeof value === 'string') {
    return value.trim() !== ''
  }

  return true
}

const normalizeRouteQueryValue = (value: unknown): string | string[] | undefined => {
  if (value === null || value === undefined) {
    return undefined
  }

  if (Array.isArray(value)) {
    return value
      .filter(item => item !== null && item !== undefined)
      .map(item => String(item))
  }

  return String(value)
}

const getRequestFields = (functionDetail: FunctionDetail) => {
  return Array.isArray(functionDetail.request) ? functionDetail.request : []
}

export const getTableRequestFieldCodes = (functionDetail: FunctionDetail): Set<string> => {
  const fieldCodes = new Set<string>()

  getRequestFields(functionDetail).forEach(field => {
    fieldCodes.add(field.code)
  })

  return fieldCodes
}

export const buildTableURLQueryParams = (
  options: BuildTableURLQueryParamsOptions
): Record<string, string> => {
  const { functionDetail, state, buildDefaultSorts } = options
  const query: Record<string, string> = {}

  query.page = String(state.pagination.currentPage)
  query.page_size = String(state.pagination.pageSize)

  const finalSorts = state.sorts.length > 0
    ? state.sorts
    : (state.hasManualSort ? [] : buildDefaultSorts())

  if (finalSorts.length > 0) {
    query.sorts = finalSorts.map((item: SortItem) => `${item.field}:${item.order}`).join(',')
  }

  const requestFieldCodes = getTableRequestFieldCodes(functionDetail)
  const responseFields = (functionDetail.response || []).filter(field => {
    const search = field.search
    return search && search !== '-' && search !== '' && search.trim() !== ''
  })
  const responseFieldsForURL = responseFields.filter(field => !requestFieldCodes.has(field.code))

  Object.assign(query, buildURLSearchParams(state.searchForm, responseFieldsForURL))

  getRequestFields(functionDetail).forEach(field => {
    const value = state.searchForm[field.code]
    if (!hasSearchValue(value)) {
      return
    }

    query[getSearchFieldQueryKey(field.code)] = Array.isArray(value) ? value.join(',') : String(value)
  })

  Object.keys(query).forEach(key => {
    const value = query[key]
    if (value && !value.endsWith(':') && value.trim() !== '') {
      return
    }
    delete query[key]
  })

  return query
}

export const preserveExistingTableQueryParams = (
  options: PreserveExistingTableQueryParamsOptions
): Record<string, string | string[]> => {
  const result: Record<string, string | string[]> = {}
  const { routeQuery, requestFieldCodes, isLinkNavigation } = options

  Object.keys(routeQuery).forEach(key => {
    const normalizedValue = normalizeRouteQueryValue(routeQuery[key])
    if (normalizedValue === undefined) {
      return
    }

    if (key.startsWith('_')) {
      if (key !== LINK_TYPE_QUERY_KEY) {
        result[key] = normalizedValue
      }
      return
    }

    if (SEARCH_PARAM_KEYS.includes(key as any)) {
      if (isLinkNavigation) {
        result[key] = normalizedValue
      }
      return
    }

    if (isSearchFieldQueryKey(key)) {
      return
    }

    if (!TABLE_PARAM_KEYS.includes(key as any) && !requestFieldCodes.has(key)) {
      result[key] = normalizedValue
    }
  })

  return result
}

export const buildNextTableSyncQuery = (
  options: BuildNextTableSyncQueryOptions
): Record<string, string | string[]> => {
  const { routeQuery, functionDetail, state, buildDefaultSorts, isLinkNavigation } = options
  const nextTableQuery = buildTableURLQueryParams({ functionDetail, state, buildDefaultSorts })
  const hasQueryParams = Object.keys(routeQuery).length > 0

  if (!hasQueryParams && !isLinkNavigation) {
    return nextTableQuery
  }

  return {
    ...preserveExistingTableQueryParams({
      routeQuery,
      requestFieldCodes: getTableRequestFieldCodes(functionDetail),
      isLinkNavigation
    }),
    ...nextTableQuery
  }
}
