/**
 * URL 参数常量
 */

/**
 * Table 相关的 URL 参数
 */
export const TABLE_PARAM_KEYS = ['page', 'page_size', 'sorts'] as const

/**
 * 搜索相关的 URL 参数
 */
export const SEARCH_PARAM_KEYS = ['eq', 'like', 'in', 'contains', 'gte', 'lte'] as const

/**
 * 所有 URL 参数键的类型
 */
export type TableParamKey = typeof TABLE_PARAM_KEYS[number]
export type SearchParamKey = typeof SEARCH_PARAM_KEYS[number]

