import { SearchType, hasSearchType } from '@/architecture/runtime/constants/search'
import { WidgetType } from '@/architecture/runtime/constants/widget'
import type { FieldConfig } from '@/architecture/domain/types'
import { parseCommaSeparatedString } from '@/utils/stringUtils'

export function resolveWidgetTypeForSearchRenderer(options: {
  widgetType?: string
  searchType?: string
}): string {
  const widgetType = options.widgetType || WidgetType.INPUT
  const searchType = options.searchType || ''

  if (widgetType === WidgetType.USER) {
    return hasSearchType(searchType, SearchType.IN) ? WidgetType.USERS : WidgetType.USER
  }

  if (widgetType === WidgetType.USERS) {
    return hasSearchType(searchType, SearchType.EQ) ? WidgetType.USER : WidgetType.USERS
  }

  if (widgetType === WidgetType.DEPARTMENT) {
    return hasSearchType(searchType, SearchType.IN) ? WidgetType.DEPARTMENTS : WidgetType.DEPARTMENT
  }

  if (widgetType === WidgetType.DEPARTMENTS) {
    return hasSearchType(searchType, SearchType.EQ) ? WidgetType.DEPARTMENT : WidgetType.DEPARTMENTS
  }

  // `select + in` 在搜索语义上是多值选择，更适合直接复用 multiselect 的稳定实现。
  if (widgetType === WidgetType.SELECT && hasSearchType(searchType, SearchType.IN)) {
    return WidgetType.MULTI_SELECT
  }

  // `radio` 在搜索栏里更适合统一降级为下拉：
  // - `radio + eq` => 单选下拉
  // - `radio + in` => 多选下拉
  if (widgetType === WidgetType.RADIO) {
    return hasSearchType(searchType, SearchType.IN) ? WidgetType.MULTI_SELECT : WidgetType.SELECT
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

  const hasInSearch = hasSearchType(searchType, SearchType.IN)
  const hasContainsSearch = hasSearchType(searchType, SearchType.CONTAINS)
  const hasLikeSearch = hasSearchType(searchType, SearchType.LIKE)
  const hasRangeSearch = hasSearchType(searchType, SearchType.GTE) && hasSearchType(searchType, SearchType.LTE)
  const hasDefaultSearch = searchType.trim() === ''

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
    case WidgetType.USERS:
    case WidgetType.DEPARTMENT:
    case WidgetType.DEPARTMENTS:
      return hasDefaultSearch || hasSearchType(searchType, SearchType.EQ) || hasInSearch || hasContainsSearch
    case WidgetType.MULTI_SELECT:
    case WidgetType.LIST:
      return !hasLikeSearch && !hasRangeSearch
    case WidgetType.CHECKBOX:
    case WidgetType.DATETIME:
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
