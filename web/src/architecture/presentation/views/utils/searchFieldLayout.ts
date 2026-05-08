import type { FieldConfig } from '@/architecture/domain/types'

export const SearchFieldLayoutClass = {
  WIDE: 'search-field-layout--wide'
} as const

export function resolveSearchFieldLayoutClass(_field: FieldConfig): string {
  return ''
}
