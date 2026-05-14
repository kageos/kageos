/**
 * 搜索参数工具函数
 * 
 * 用于统一处理搜索参数的构建和转换，避免重复代码
 */

import type { FieldConfig } from '@/architecture/runtime/types/field'
import type { SearchParams } from '@/types'
import { getSearchFieldRawValue } from '@/utils/searchFieldValue'

const hasValue = (value: unknown): boolean => {
  return !(
    value === null ||
    value === undefined ||
    (Array.isArray(value) && value.length === 0) ||
    (typeof value === 'string' && value.trim() === '')
  )
}

const stringifyValue = (value: unknown): string => {
  if (Array.isArray(value)) return value.map(item => String(item)).join(',')
  if (typeof value === 'object' && value !== null) return JSON.stringify(value)
  return String(value)
}

export function buildSearchParamsString(
  searchForm: Record<string, any>,
  searchableFields: FieldConfig[]
): Partial<SearchParams> {
  const result: Record<string, string> = {}

  searchableFields.forEach(field => {
    const value = getSearchFieldRawValue(searchForm[field.code])
    if (!hasValue(value)) {
      return
    }
    result[field.code] = stringifyValue(value)
  })

  return result
}

export function buildURLSearchParams(
  searchForm: Record<string, any>,
  searchableFields: FieldConfig[]
): Record<string, string> {
  return buildSearchParamsString(searchForm, searchableFields) as Record<string, string>
}
