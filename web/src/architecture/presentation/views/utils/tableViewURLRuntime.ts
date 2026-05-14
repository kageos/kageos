import type { FunctionDetail } from '@/architecture/domain/types'
import type { SortItem, TableState } from '@/architecture/domain/services/TableDomainService'
import {
  getSearchFieldRawValue,
  hasSearchFieldValue
} from '@/architecture/runtime/utils/searchFieldValue'
import {
  getTableRequestFields,
  getTableRequestSearchFields
} from '@/architecture/runtime/utils/functionSchemaSelectors'
import {
  isPersistentPlatformStateQueryKey,
  isPlatformStateQueryKey,
  isStaleTableFilterQueryKey,
  isTableControlQueryKey,
  isUnsupportedGeneratedFieldQueryKey
} from '@/architecture/runtime/utils/queryParamKeys'

interface BuildTableURLQueryParamsOptions {
  functionDetail: FunctionDetail
  state: TableState
  buildDefaultSorts: () => SortItem[]
}

interface PreserveExistingTableQueryParamsOptions {
  routeQuery: Record<string, any>
  requestFieldCodes: Set<string>
  generatedFieldCodes?: Set<string>
  isLinkNavigation: boolean
}

interface BuildNextTableSyncQueryOptions extends BuildTableURLQueryParamsOptions {
  routeQuery: Record<string, any>
  isLinkNavigation: boolean
}

const hasSearchValue = (value: unknown): boolean => {
  return hasSearchFieldValue(value)
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
  return getTableRequestFields(functionDetail)
}

const getRequestSearchFields = (functionDetail: FunctionDetail) => {
  return getTableRequestSearchFields(functionDetail)
}

const serializeSortsForURL = (items: SortItem[]): string => {
  return JSON.stringify(items.map(item => ({
    field: item.field,
    order: item.order
  })))
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
    query.sorts = serializeSortsForURL(finalSorts)
  }

  getRequestSearchFields(functionDetail).forEach(field => {
    const value = state.searchForm[field.code]
    if (!hasSearchValue(value)) {
      return
    }

    const rawValue = getSearchFieldRawValue(value)
    // request 字段是 sdk-app 入参，key 必须等于 schema 的 field.code。
    // 不要加 `s_`/`f_`、`_` 前缀，也不要加显示值伴随参数。
    query[field.code] = Array.isArray(rawValue) ? rawValue.join(',') : String(rawValue)
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
  const generatedFieldCodes = options.generatedFieldCodes || requestFieldCodes

  Object.keys(routeQuery).forEach(key => {
    const normalizedValue = normalizeRouteQueryValue(routeQuery[key])
    if (normalizedValue === undefined) {
      return
    }

    if (isUnsupportedGeneratedFieldQueryKey(key, generatedFieldCodes)) {
      return
    }

    if (isPersistentPlatformStateQueryKey(key)) {
      // `_` key 是前端/平台状态，例如 `_tab`、`_id`、`_mws`。
      // 这类 key 不会和 sdk-app 业务参数冲突；临时态 key 由 helper 过滤。
      result[key] = normalizedValue
      return
    }

    if (isPlatformStateQueryKey(key)) {
      return
    }

    if (isStaleTableFilterQueryKey(key)) {
      return
    }

    if (!isTableControlQueryKey(key) && !requestFieldCodes.has(key)) {
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
      generatedFieldCodes: new Set([
        ...getTableRequestSearchFields(functionDetail).map(field => field.code)
      ]),
      isLinkNavigation
    }),
    ...nextTableQuery
  }
}
