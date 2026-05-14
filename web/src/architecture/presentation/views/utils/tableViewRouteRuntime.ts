import { deleteFieldQueryKey } from '@/architecture/runtime/utils/queryParamKeys'

export interface TableViewRouteRequest {
  path?: string
  query?: Record<string, string | string[]>
  replace?: boolean
  preserveParams?: {
    table?: boolean
    search?: boolean
    state?: boolean
    linkNavigation?: boolean
  }
}

const normalizeRouteQuery = (query: Record<string, any>): Record<string, string | string[]> => {
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

export function buildTableLinkRouteRequest(finalUrl: string): TableViewRouteRequest {
  let path = finalUrl
  const query: Record<string, string> = {}

  const queryIndex = finalUrl.indexOf('?')
  if (queryIndex >= 0) {
    path = finalUrl.substring(0, queryIndex)
    const queryString = finalUrl.substring(queryIndex + 1)
    const params = new URLSearchParams(queryString)
    params.forEach((value, key) => {
      query[key] = value
    })
  }

  return {
    path,
    query,
    replace: false,
    preserveParams: {
      linkNavigation: true
    }
  }
}

export function buildTableAddDialogOpenRequest(routeQuery: Record<string, any>): TableViewRouteRequest {
  return {
    query: {
      ...normalizeRouteQuery(routeQuery),
      _tab: 'OnTableAddRow'
    },
    replace: true,
    preserveParams: {
      state: true
    }
  }
}

export function buildTableCreateDialogCloseRequest(options: {
  routeQuery: Record<string, any>
  responseFieldCodes: string[]
}): TableViewRouteRequest | null {
  const query = normalizeRouteQuery(options.routeQuery)
  if (query._tab !== 'OnTableAddRow') {
    return null
  }

  delete query._tab
  options.responseFieldCodes.forEach(fieldCode => {
    // 关闭新增弹窗只删除 raw form-field 参数；table 搜索操作符参数和 `_`
    // 平台状态由 preserveParams 保留。
    deleteFieldQueryKey(query, fieldCode)
  })

  return {
    query,
    replace: true,
    preserveParams: {
      table: true,
      search: true,
      state: true
    }
  }
}

export function buildTableDetailRowPayload(options: {
  row: Record<string, any>
  tableData: Record<string, any>[]
  initialMode: 'read' | 'edit'
}) {
  const index = options.tableData.findIndex(currentRow => {
    if (currentRow.id && options.row.id && currentRow.id === options.row.id) {
      return true
    }
    return JSON.stringify(currentRow) === JSON.stringify(options.row)
  })

  return {
    row: options.row,
    index: index >= 0 ? index : undefined,
    tableData: options.tableData.length > 0 ? options.tableData : undefined,
    initialMode: options.initialMode
  }
}

export function resolveTableAddDialogVisibility(options: {
  query: Record<string, any>
  hasAddCallback: boolean
  isMounted: boolean
  currentVisible: boolean
}): boolean {
  const tabParam = options.query._tab as string | undefined

  if (tabParam === 'OnTableAddRow' && options.hasAddCallback && options.isMounted) {
    return true
  }

  if (tabParam !== 'OnTableAddRow' && options.currentVisible) {
    return false
  }

  return options.currentVisible
}
