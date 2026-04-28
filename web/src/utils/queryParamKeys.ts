import { LINK_TYPE_QUERY_KEY } from '@/utils/linkNavigation'
import { SEARCH_PARAM_KEYS, TABLE_PARAM_KEYS } from '@/utils/urlParams'

/**
 * URL 参数键规则：
 * - 传给 sdk-app 用户函数的参数使用 schema 原始 field.code，
 *   必须和 sdk-app 的 json/form tag 对齐，例如 `genre=诗`。
 * - 仅前端/平台使用的状态参数才使用 `_` 开头，例如 `_tab`、`_mws`。
 * - 不要再引入 `s_`/`f_` 字段命名空间，也不要加 `_genre__display`
 *   这类显示值伴随参数；这些都会把平台状态泄漏到用户业务参数里。
 */

export const NODE_TYPE_QUERY_KEY = '_node_type'
export const LEGACY_FORM_DRAFT_QUERY_PREFIX = 'f_'
export const LEGACY_SEARCH_FIELD_QUERY_PREFIX = 's_'
export const FIELD_DISPLAY_QUERY_SUFFIX = '__display'

const LEGACY_FIELD_QUERY_PREFIXES = [
  LEGACY_FORM_DRAFT_QUERY_PREFIX,
  LEGACY_SEARCH_FIELD_QUERY_PREFIX
] as const

export const isPlatformStateQueryKey = (key: string): boolean => {
  return key.startsWith('_')
}

export const isLinkMarkerQueryKey = (key: string): boolean => {
  return key === LINK_TYPE_QUERY_KEY
}

export const isTableControlQueryKey = (key: string): boolean => {
  return TABLE_PARAM_KEYS.includes(key as typeof TABLE_PARAM_KEYS[number])
}

export const isBackendSearchOperatorQueryKey = (key: string): boolean => {
  return SEARCH_PARAM_KEYS.includes(key as typeof SEARCH_PARAM_KEYS[number])
}

export const isDisplayCompanionQueryKey = (key: string): boolean => {
  return key.endsWith(FIELD_DISPLAY_QUERY_SUFFIX)
}

export const isPersistentPlatformStateQueryKey = (key: string): boolean => {
  return isPlatformStateQueryKey(key) && !isLinkMarkerQueryKey(key) && !isDisplayCompanionQueryKey(key)
}

export const getLegacyFieldQueryKeys = (fieldCode: string): string[] => {
  return [
    `${LEGACY_FORM_DRAFT_QUERY_PREFIX}${fieldCode}`,
    `${LEGACY_SEARCH_FIELD_QUERY_PREFIX}${fieldCode}`,
    `${LEGACY_FORM_DRAFT_QUERY_PREFIX}${fieldCode}${FIELD_DISPLAY_QUERY_SUFFIX}`,
    `${LEGACY_SEARCH_FIELD_QUERY_PREFIX}${fieldCode}${FIELD_DISPLAY_QUERY_SUFFIX}`,
    `_${fieldCode}${FIELD_DISPLAY_QUERY_SUFFIX}`
  ]
}

export const isLegacyFieldNamespaceQueryKey = (
  key: string,
  fieldCodes: Set<string>
): boolean => {
  return LEGACY_FIELD_QUERY_PREFIXES.some(prefix => {
    if (!key.startsWith(prefix)) {
      return false
    }

    const fieldCode = key
      .slice(prefix.length)
      .replace(new RegExp(`${FIELD_DISPLAY_QUERY_SUFFIX}$`), '')

    return fieldCodes.has(fieldCode)
  })
}

export const isUnsupportedGeneratedFieldQueryKey = (
  key: string,
  fieldCodes?: Set<string>
): boolean => {
  if (isDisplayCompanionQueryKey(key)) {
    return true
  }

  if (!fieldCodes) {
    return false
  }

  return isLegacyFieldNamespaceQueryKey(key, fieldCodes)
}

export const deleteFieldQueryKey = (
  query: Record<string, any>,
  fieldCode: string,
  options?: {
    deleteRaw?: boolean
  }
): void => {
  // request 字段只拥有原始 field.code 这个 key。下面的 legacy key 只用于
  // 清理旧 URL，不允许在新逻辑中生成或读取。
  if (options?.deleteRaw !== false) {
    delete query[fieldCode]
  }

  getLegacyFieldQueryKeys(fieldCode).forEach(key => {
    delete query[key]
  })
}
