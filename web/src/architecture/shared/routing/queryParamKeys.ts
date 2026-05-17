import { LINK_TYPE_QUERY_KEY } from '@/architecture/shared/routing/linkNavigation'
import { TABLE_PARAM_KEYS } from '@/architecture/shared/routing/urlParams'

/**
 * URL 参数键规则：
 * - 传给 sdk-app 用户函数的参数使用 schema 原始 field.code，
 *   必须和 sdk-app 的 json/form tag 对齐，例如 `genre=诗`。
 * - 仅前端/平台使用的状态参数才使用 `_` 开头，例如 `_tab`、`_mws`、`_mws_expanded`。
 * - 不要再引入 `s_`/`f_` 字段命名空间，也不要加 `_genre__display`
 *   这类显示值伴随参数；这些都会把平台状态泄漏到用户业务参数里。
 */

export const NODE_TYPE_QUERY_KEY = '_node_type'
const STALE_TABLE_FILTER_QUERY_KEYS = 'eq,like,in,contains,gte,lte'.split(',')
const DISPLAY_COMPANION_QUERY_SUFFIX = '__display'
const GENERATED_FIELD_QUERY_PREFIXES = ['f_', 's_'] as const

export const isPlatformStateQueryKey = (key: string): boolean => {
  return key.startsWith('_')
}

export const isLinkMarkerQueryKey = (key: string): boolean => {
  return key === LINK_TYPE_QUERY_KEY
}

export const isTableControlQueryKey = (key: string): boolean => {
  return TABLE_PARAM_KEYS.includes(key as typeof TABLE_PARAM_KEYS[number])
}

export const isStaleTableFilterQueryKey = (key: string): boolean => {
  return STALE_TABLE_FILTER_QUERY_KEYS.includes(key)
}

export const isPersistentPlatformStateQueryKey = (key: string): boolean => {
  return isPlatformStateQueryKey(key) && !isLinkMarkerQueryKey(key) && !isDisplayCompanionQueryKey(key)
}

export const isDisplayCompanionQueryKey = (key: string): boolean => {
  return key.endsWith(DISPLAY_COMPANION_QUERY_SUFFIX)
}

export const isGeneratedFieldQueryKey = (key: string): boolean => {
  return isDisplayCompanionQueryKey(key) || GENERATED_FIELD_QUERY_PREFIXES.some(prefix => key.startsWith(prefix))
}

export const deleteFieldQueryKey = (
  query: Record<string, unknown>,
  fieldCode: string,
  options?: {
    deleteRaw?: boolean
  }
): void => {
  if (options?.deleteRaw !== false) {
    delete query[fieldCode]
  }

  GENERATED_FIELD_QUERY_PREFIXES.forEach(prefix => {
    delete query[`${prefix}${fieldCode}`]
    delete query[`${prefix}${fieldCode}${DISPLAY_COMPANION_QUERY_SUFFIX}`]
  })
  delete query[`_${fieldCode}${DISPLAY_COMPANION_QUERY_SUFFIX}`]
}
