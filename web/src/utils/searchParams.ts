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
    
    // 🔥 检查值是否为空（包括空数组、空字符串、null、undefined）
    // 注意：空数组 [] 是 truthy，需要单独检查
    if (value === null || value === undefined || 
        (Array.isArray(value) && value.length === 0) || 
        (typeof value === 'string' && value.trim() === '')) {
      return
    }

    const searchType = field.search || ''

    // 精确匹配
    if (searchType.includes('eq')) {
      // 🔥 如果已有 eq 值，追加（支持多个字段）
      const valueStr = Array.isArray(value) ? value.join(',') : String(value)
      if (valueStr.trim()) {
        result.eq = result.eq ? `${result.eq},${field.code}:${valueStr}` : `${field.code}:${valueStr}`
      }
    }
    // 模糊查询
    else if (searchType.includes('like')) {
      // 🔥 如果已有 like 值，追加（支持多个字段）
      const valueStr = String(value).trim()
      if (valueStr) {
        result.like = result.like ? `${result.like},${field.code}:${valueStr}` : `${field.code}:${valueStr}`
      }
    }
    // 🔥 包含查询（用于多选场景，使用 FIND_IN_SET）
    // 注意：必须先检查 contains，再检查 in，因为 "contains" 包含 "in" 子字符串
    else if (searchType.includes('contains')) {
      // 🔥 contains 类型：如果 value 是数组，转换为逗号分隔的字符串
      // 注意：多个字段之间使用逗号 , 分隔，与 in 操作符保持一致
      const valueStr = Array.isArray(value) ? value.join(',') : String(value)
      // 🔥 确保值不为空
      if (valueStr.trim()) {
        // 🔥 如果已有 contains 值，使用逗号 , 追加（支持多个字段）
        // 格式：contains=tags:高,中,otherField:value1,value2（与 in 操作符格式一致）
        result.contains = result.contains ? `${result.contains},${field.code}:${valueStr}` : `${field.code}:${valueStr}`
      }
    }
    // 包含查询
    else if (searchType.includes('in')) {
      // 🔥 in 类型：如果 value 是数组，转换为逗号分隔的字符串
      const valueStr = Array.isArray(value) ? value.join(',') : String(value)
      // 🔥 确保值不为空
      if (valueStr.trim()) {
        // 🔥 如果已有 in 值，追加（支持多个字段）
        result.in = result.in ? `${result.in},${field.code}:${valueStr}` : `${field.code}:${valueStr}`
      }
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
 * 构建 URL 查询参数（用于 URL，格式：eq=field:value，与后端 API 格式一致）
 * 
 * @param searchForm 搜索表单数据
 * @param searchableFields 可搜索字段列表
 * @returns URL 查询参数字典（格式与后端 API 一致）
 */
export function buildURLSearchParams(
  searchForm: Record<string, any>,
  searchableFields: FieldConfig[]
): Record<string, string> {
  const result: Record<string, string> = {}

  searchableFields.forEach(field => {
    const value = searchForm[field.code]
    
    // 🔥 检查值是否为空（包括空数组、空字符串、null、undefined）
    // 注意：空数组 [] 是 truthy，需要单独检查
    if (value === null || value === undefined || 
        (Array.isArray(value) && value.length === 0) || 
        (typeof value === 'string' && value.trim() === '')) {
      return
    }

    const searchType = field.search || ''

    // 精确匹配
    if (searchType.includes('eq')) {
      // 🔥 如果已有 eq 值，追加（支持多个字段）
      const valueStr = Array.isArray(value) ? value.join(',') : String(value)
      if (valueStr.trim()) {
        result.eq = result.eq ? `${result.eq},${field.code}:${valueStr}` : `${field.code}:${valueStr}`
      }
    }
    // 模糊查询
    else if (searchType.includes('like')) {
      // 🔥 如果已有 like 值，追加（支持多个字段）
      const valueStr = String(value).trim()
      if (valueStr) {
        result.like = result.like ? `${result.like},${field.code}:${valueStr}` : `${field.code}:${valueStr}`
      }
    }
    // 🔥 包含查询（用于多选场景，使用 FIND_IN_SET）
    // 注意：必须先检查 contains，再检查 in，因为 "contains" 包含 "in" 子字符串
    else if (searchType.includes('contains')) {
      // 🔥 contains 类型：如果 value 是数组，转换为逗号分隔的字符串
      // 注意：多个字段之间使用逗号 , 分隔，与 in 操作符保持一致
      const valueStr = Array.isArray(value) ? value.join(',') : String(value)
      // 🔥 确保值不为空
      if (valueStr.trim()) {
        // 🔥 如果已有 contains 值，使用逗号 , 追加（支持多个字段）
        // 格式：contains=tags:高,中,otherField:value1,value2（与 in 操作符格式一致）
        result.contains = result.contains ? `${result.contains},${field.code}:${valueStr}` : `${field.code}:${valueStr}`
      }
    }
    // 包含查询
    else if (searchType.includes('in')) {
      // 🔥 in 类型：如果 value 是数组，转换为逗号分隔的字符串
      const valueStr = Array.isArray(value) ? value.join(',') : String(value)
      // 🔥 确保值不为空
      if (valueStr.trim()) {
        // 🔥 如果已有 in 值，追加（支持多个字段）
        result.in = result.in ? `${result.in},${field.code}:${valueStr}` : `${field.code}:${valueStr}`
      }
    }
    // 范围查询
    else if (searchType.includes('gte') && searchType.includes('lte')) {
      if (typeof value === 'object') {
        if (Array.isArray(value) && value.length === 2) {
          // 日期范围数组
          if (value[0]) result.gte = `${field.code}:${String(value[0])}`
          if (value[1]) result.lte = `${field.code}:${String(value[1])}`
        } else if (value.min !== undefined || value.max !== undefined) {
          // 数字范围对象
          if (value.min !== undefined && value.min !== null && value.min !== '') {
            result.gte = `${field.code}:${String(value.min)}`
          }
          if (value.max !== undefined && value.max !== null && value.max !== '') {
            result.lte = `${field.code}:${String(value.max)}`
          }
        }
      }
    }
  })

  return result
}

