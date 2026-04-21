import { SearchType, hasSearchType } from '@/core/constants/search'
import { WidgetType } from '@/core/constants/widget'
import type { FieldConfig } from '@/architecture/domain/types'

export const SearchFieldLayoutClass = {
  WIDE: 'search-field-layout--wide'
} as const

export function resolveSearchFieldLayoutClass(field: FieldConfig): string {
  const widgetType = field.widget?.type || WidgetType.INPUT
  const searchType = field.search || ''
  const isRangeDateTime =
    widgetType === WidgetType.DATETIME &&
    hasSearchType(searchType, SearchType.GTE) &&
    hasSearchType(searchType, SearchType.LTE)

  if (isRangeDateTime) {
    return SearchFieldLayoutClass.WIDE
  }

  return ''
}
