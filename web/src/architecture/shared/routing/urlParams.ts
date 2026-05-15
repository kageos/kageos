/**
 * URL 参数常量
 */

/**
 * Table 相关的 URL 参数
 */
export const TABLE_PARAM_KEYS = ['page', 'page_size', 'sorts'] as const

/**
 * Table 筛选参数由 Request 字段 code 动态决定。
 */
export const SEARCH_PARAM_KEYS = [] as const

/**
 * 所有 URL 参数键的类型
 */
export type TableParamKey = typeof TABLE_PARAM_KEYS[number]
export type SearchParamKey = typeof SEARCH_PARAM_KEYS[number]

export function isTableParamKey(key: string): key is TableParamKey {
  return TABLE_PARAM_KEYS.some((tableKey) => tableKey === key)
}
