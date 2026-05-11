import type { FunctionDetail } from '@/architecture/domain/types'

export const DEFAULT_TABLE_PAGE_SIZE = 10
export const LEGACY_TABLE_DEFAULT_PAGE_SIZE = 20
export const TABLE_PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const

const TABLE_PAGE_SIZE_STORAGE_PREFIX = 'aos:table-page-size:'

export function normalizeTablePageSize(value: unknown, options: { requireKnownOption?: boolean } = {}): number | null {
  const rawValue = Array.isArray(value) ? value[0] : value
  if (rawValue === null || rawValue === undefined || rawValue === '') {
    return null
  }

  const pageSize = Number.parseInt(String(rawValue), 10)
  if (!Number.isFinite(pageSize) || pageSize <= 0) {
    return null
  }

  if (options.requireKnownOption && !TABLE_PAGE_SIZE_OPTIONS.includes(pageSize as typeof TABLE_PAGE_SIZE_OPTIONS[number])) {
    return null
  }

  return pageSize
}

export function getTablePageSizePreferenceKey(functionDetail: FunctionDetail): string {
  const rawKey = functionDetail.router || functionDetail.code || functionDetail.id || 'default'
  return `${TABLE_PAGE_SIZE_STORAGE_PREFIX}${String(rawKey).trim() || 'default'}`
}

export function readTablePageSizePreference(functionDetail: FunctionDetail): number | null {
  if (typeof window === 'undefined') {
    return null
  }

  try {
    return normalizeTablePageSize(window.localStorage.getItem(getTablePageSizePreferenceKey(functionDetail)), {
      requireKnownOption: true
    })
  } catch {
    return null
  }
}

export function writeTablePageSizePreference(functionDetail: FunctionDetail, pageSize: number): void {
  if (typeof window === 'undefined') {
    return
  }

  const normalizedPageSize = normalizeTablePageSize(pageSize, { requireKnownOption: true })
  if (!normalizedPageSize) {
    return
  }

  try {
    window.localStorage.setItem(getTablePageSizePreferenceKey(functionDetail), String(normalizedPageSize))
  } catch {
    // localStorage 不可用时不影响表格分页本身
  }
}

export function resolveTablePageSizeForRestore(options: {
  queryPageSize?: unknown
  preferredPageSize?: number | null
  isLinkNavigation: boolean
  defaultPageSize?: number
}): { pageSize: number; shouldSyncToURL: boolean } {
  const defaultPageSize = options.defaultPageSize || DEFAULT_TABLE_PAGE_SIZE
  const queryPageSize = normalizeTablePageSize(options.queryPageSize)
  const preferredPageSize = normalizeTablePageSize(options.preferredPageSize, { requireKnownOption: true })

  if (options.isLinkNavigation && queryPageSize) {
    return {
      pageSize: queryPageSize,
      shouldSyncToURL: false
    }
  }

  if (preferredPageSize) {
    return {
      pageSize: preferredPageSize,
      shouldSyncToURL: queryPageSize !== preferredPageSize
    }
  }

  if (queryPageSize && queryPageSize !== LEGACY_TABLE_DEFAULT_PAGE_SIZE) {
    return {
      pageSize: queryPageSize,
      shouldSyncToURL: false
    }
  }

  return {
    pageSize: defaultPageSize,
    shouldSyncToURL: queryPageSize !== defaultPageSize
  }
}
