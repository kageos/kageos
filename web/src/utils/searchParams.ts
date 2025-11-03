/**
 * 搜索参数工具函数
 * 
 * 用于统一处理搜索参数的构建和转换，避免重复代码
 */

import type { FieldConfig } from '@/core/types/field'
import type { SearchParams } from '@/types'

/**
 * 构建搜索参数字符串（用于 SearchParams，格式：eq=field:value）
 * 
 * @param searchForm 搜索表单数据
 * @param searchableFields 可搜索字段列表
 * @returns SearchParams 格式的搜索参数对象
 */
export function buildSearchParamsString(
  searchForm: Record<string, any>,
  searchableFields: FieldConfig[]
): Partial<SearchParams> {
  const result: Partial<SearchParams> = {}

  searchableFields.forEach(field => {
    const value = searchForm[field.code]
    if (!value) return

    const searchType = field.search || ''

    // 精确匹配
    if (searchType.includes('eq')) {
      result.eq = `${field.code}:${value}`
    }
    // 模糊查询
    else if (searchType.includes('like')) {
      result.like = `${field.code}:${value}`
    }
    // 包含查询
    else if (searchType.includes('in')) {
      result.in = `${field.code}:${value}`
    }
    // 范围查询
    else if (searchType.includes('gte') && searchType.includes('lte')) {
      if (typeof value === 'object') {
        if (Array.isArray(value) && value.length === 2) {
          // 日期范围数组
          if (value[0]) result.gte = `${field.code}:${value[0]}`
          if (value[1]) result.lte = `${field.code}:${value[1]}`
        } else if (value.min !== undefined || value.max !== undefined) {
          // 数字范围对象
          if (value.min !== undefined && value.min !== null && value.min !== '') {
            result.gte = `${field.code}:${value.min}`
          }
          if (value.max !== undefined && value.max !== null && value.max !== '') {
            result.lte = `${field.code}:${value.max}`
          }
        }
      }
    }
  })

  return result
}

/**
 * 构建 URL 查询参数（用于 URL，格式：eq_fieldCode=value）
 * 
 * @param searchForm 搜索表单数据
 * @param searchableFields 可搜索字段列表
 * @returns URL 查询参数字典
 */
export function buildURLSearchParams(
  searchForm: Record<string, any>,
  searchableFields: FieldConfig[]
): Record<string, string> {
  const result: Record<string, string> = {}

  searchableFields.forEach(field => {
    const value = searchForm[field.code]
    if (!value) return

    const searchType = field.search || ''

    // 精确匹配
    if (searchType.includes('eq')) {
      result[`eq_${field.code}`] = String(value)
    }
    // 模糊查询
    else if (searchType.includes('like')) {
      result[`like_${field.code}`] = String(value)
    }
    // 包含查询
    else if (searchType.includes('in')) {
      result[`in_${field.code}`] = String(value)
    }
    // 范围查询
    else if (searchType.includes('gte') && searchType.includes('lte')) {
      if (typeof value === 'object') {
        if (Array.isArray(value) && value.length === 2) {
          // 日期范围数组
          if (value[0]) result[`gte_${field.code}`] = String(value[0])
          if (value[1]) result[`lte_${field.code}`] = String(value[1])
        } else if (value.min !== undefined || value.max !== undefined) {
          // 数字范围对象
          if (value.min !== undefined && value.min !== null && value.min !== '') {
            result[`gte_${field.code}`] = String(value.min)
          }
          if (value.max !== undefined && value.max !== null && value.max !== '') {
            result[`lte_${field.code}`] = String(value.max)
          }
        }
      }
    }
  })

  return result
}

