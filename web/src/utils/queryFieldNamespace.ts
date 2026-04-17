export const FORM_DRAFT_QUERY_PREFIX = 'f_'
export const SEARCH_FIELD_QUERY_PREFIX = 's_'
export const SEARCH_FIELD_DISPLAY_QUERY_SUFFIX = '__display'

type FieldQueryScope = 'form' | 'search'

const getPrefixByScope = (scope: FieldQueryScope): string => {
  return scope === 'form' ? FORM_DRAFT_QUERY_PREFIX : SEARCH_FIELD_QUERY_PREFIX
}

export const getFormDraftQueryKey = (fieldCode: string): string => {
  return `${FORM_DRAFT_QUERY_PREFIX}${fieldCode}`
}

export const getSearchFieldQueryKey = (fieldCode: string): string => {
  return `${SEARCH_FIELD_QUERY_PREFIX}${fieldCode}`
}

export const getSearchFieldDisplayQueryKey = (fieldCode: string): string => {
  return `${getSearchFieldQueryKey(fieldCode)}${SEARCH_FIELD_DISPLAY_QUERY_SUFFIX}`
}

export const isFormDraftQueryKey = (key: string): boolean => {
  return key.startsWith(FORM_DRAFT_QUERY_PREFIX)
}

export const isSearchFieldQueryKey = (key: string): boolean => {
  return key.startsWith(SEARCH_FIELD_QUERY_PREFIX)
}

export const shouldUseRawFormQueryKeys = (query: Record<string, any>): boolean => {
  return query._tab === 'OnTableAddRow'
}

export const getScopedFieldQueryValue = (
  query: Record<string, any>,
  fieldCode: string,
  scope: FieldQueryScope,
  options?: {
    fallbackToLegacyRaw?: boolean
  }
): any => {
  const scopedKey = `${getPrefixByScope(scope)}${fieldCode}`
  if (query[scopedKey] !== undefined) {
    return query[scopedKey]
  }
  if (options?.fallbackToLegacyRaw === false) {
    return undefined
  }
  return query[fieldCode]
}

export const shouldAllowLegacyFormDraftFallback = (_query: Record<string, any>): boolean => {
  return true
}

export const deleteScopedFieldQueryKey = (
  query: Record<string, any>,
  fieldCode: string,
  scope: FieldQueryScope
): void => {
  delete query[`${getPrefixByScope(scope)}${fieldCode}`]
  if (scope === 'search') {
    delete query[getSearchFieldDisplayQueryKey(fieldCode)]
  }
  delete query[fieldCode]
}
