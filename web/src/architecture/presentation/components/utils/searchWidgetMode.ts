import { SearchType, hasSearchType } from '@/core/constants/search'
import { WidgetType } from '@/core/constants/widget'
import type { FieldConfig } from '@/core/types/field'
import { parseCommaSeparatedString } from '@/utils/stringUtils'

export function resolveWidgetTypeForSearchRenderer(options: {
  widgetType?: string
  searchType?: string
}): string {
  const widgetType = options.widgetType || WidgetType.INPUT
  const searchType = options.searchType || ''

  // `select + in` 在搜索语义上是多值选择，更适合直接复用 multiselect 的稳定实现。
  if (widgetType === WidgetType.SELECT && hasSearchType(searchType, SearchType.IN)) {
    return WidgetType.MULTI_SELECT
  }

  // 搜索栏优先使用紧凑单行控件，多行/富文本在筛选区没有展示价值。
  if (widgetType === WidgetType.TEXT_AREA || widgetType === WidgetType.RICH_TEXT) {
    return WidgetType.INPUT
  }

  return widgetType
}

export function shouldUseWidgetSearchRenderer(options: {
  widgetType?: string
  searchType?: string
  hasRegisteredWidget: boolean
}): boolean {
  const widgetType = resolveWidgetTypeForSearchRenderer(options)
  const searchType = options.searchType || ''

  if (!options.hasRegisteredWidget) {
    return false
  }

  // 这些类型在搜索栏里有更合适的专用壳层或 fallback 控件。
  if (
    widgetType === WidgetType.USER ||
    widgetType === WidgetType.USERS ||
    widgetType === WidgetType.DEPARTMENT ||
    widgetType === WidgetType.DEPARTMENTS
  ) {
    return false
  }

  const hasInSearch = hasSearchType(searchType, SearchType.IN)
  const hasContainsSearch = hasSearchType(searchType, SearchType.CONTAINS)
  const hasLikeSearch = hasSearchType(searchType, SearchType.LIKE)
  const hasRangeSearch = hasSearchType(searchType, SearchType.GTE) && hasSearchType(searchType, SearchType.LTE)

  switch (widgetType) {
    case WidgetType.INPUT:
    case WidgetType.ID:
    case WidgetType.NUMBER:
    case WidgetType.FLOAT:
    case WidgetType.TEXT_AREA:
    case WidgetType.COLOR:
    case WidgetType.RICH_TEXT:
    case WidgetType.PROGRESS:
      return !hasInSearch && !hasContainsSearch && !hasRangeSearch
    case WidgetType.SELECT:
    case WidgetType.RADIO:
      return !hasInSearch && !hasLikeSearch && !hasRangeSearch
    case WidgetType.USER:
      return !hasInSearch && !hasLikeSearch
    case WidgetType.USERS:
    case WidgetType.DEPARTMENT:
    case WidgetType.DEPARTMENTS:
      return !hasLikeSearch
    case WidgetType.MULTI_SELECT:
      return !hasLikeSearch && !hasRangeSearch
    case WidgetType.CHECKBOX:
    case WidgetType.TIMESTAMP:
    case WidgetType.SLIDER:
    case WidgetType.RATE:
      return true
    default:
      return false
  }
}

export function buildSearchWidgetField(field: FieldConfig, searchType?: string): FieldConfig {
  const resolvedWidgetType = resolveWidgetTypeForSearchRenderer({
    widgetType: field.widget?.type,
    searchType
  })

  if ((field.widget?.type || WidgetType.INPUT) === resolvedWidgetType) {
    return field
  }

  return {
    ...field,
    widget: {
      ...(field.widget || {}),
      type: resolvedWidgetType
    }
  }
}

export function adaptSearchModelValueForWidget(value: any, widgetType?: string): any {
  if (value === null || value === undefined || value === '') {
    return value
  }

  if (widgetType === WidgetType.CHECKBOX && typeof value === 'string') {
    return parseCommaSeparatedString(value)
  }

  if (
    (widgetType === WidgetType.USERS ||
      widgetType === WidgetType.DEPARTMENT ||
      widgetType === WidgetType.DEPARTMENTS) &&
    Array.isArray(value)
  ) {
    return value.map(item => String(item).trim()).filter(Boolean).join(',')
  }

  return value
}
