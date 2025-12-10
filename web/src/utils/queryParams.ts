/**
 * URL 查询参数工具函数
 */

import { TABLE_PARAM_KEYS, SEARCH_PARAM_KEYS } from './urlParams'

/**
 * 保留查询参数
 * @param currentQuery 当前路由的查询参数
 * @param options 保留选项
 * @returns 保留后的查询参数
 */
export function preserveQueryParams(
  currentQuery: Record<string, any>,
  options: {
    preserveTableParams?: boolean  // 是否保留 table 参数（page, page_size, sorts）
    preserveSearchParams?: boolean // 是否保留搜索参数（eq, like, in 等）
    preserveStateParams?: boolean  // 是否保留状态参数（以 _ 开头）
    preserveCustomParams?: string[] // 自定义要保留的参数列表
  } = {}
): Record<string, string | string[]> {
  const {
    preserveTableParams = false,
    preserveSearchParams = false,
    preserveStateParams = true, // 默认保留状态参数
    preserveCustomParams = []
  } = options

  const preservedQuery: Record<string, string | string[]> = {}
  const tableParamKeys = TABLE_PARAM_KEYS
  const searchParamKeys = SEARCH_PARAM_KEYS

  Object.keys(currentQuery).forEach(key => {
    const value = currentQuery[key]
    if (value === null || value === undefined) {
      return
    }

    // 检查是否应该保留这个参数
    let shouldPreserve = false

    // 保留状态参数（以 _ 开头）
    if (preserveStateParams && key.startsWith('_')) {
      shouldPreserve = true
    }
    // 保留 table 参数
    else if (preserveTableParams && tableParamKeys.includes(key)) {
      shouldPreserve = true
    }
    // 保留搜索参数
    else if (preserveSearchParams && searchParamKeys.includes(key)) {
      shouldPreserve = true
    }
    // 保留自定义参数
    else if (preserveCustomParams.includes(key)) {
      shouldPreserve = true
    }

    if (shouldPreserve) {
      if (Array.isArray(value)) {
        preservedQuery[key] = value.filter(v => v !== null).map(v => String(v))
      } else {
        preservedQuery[key] = String(value)
      }
    }
  })

  return preservedQuery
}

/**
 * 为 table 函数保留查询参数
 * 保留：table 参数（page, page_size, sorts）+ 状态参数（_ 开头）
 * 不保留：搜索参数（避免状态污染）
 */
export function preserveQueryParamsForTable(
  currentQuery: Record<string, any>
): Record<string, string | string[]> {
  return preserveQueryParams(currentQuery, {
    preserveTableParams: true,
    preserveSearchParams: false,
    preserveStateParams: true
  })
}

/**
 * 为 form 函数保留查询参数
 * 只保留：状态参数（_ 开头）
 * 不保留：table 参数和搜索参数
 */
export function preserveQueryParamsForForm(
  currentQuery: Record<string, any>
): Record<string, string | string[]> {
  return preserveQueryParams(currentQuery, {
    preserveTableParams: false,
    preserveSearchParams: false,
    preserveStateParams: true
  })
}

