import type { FieldConfig, FieldValue } from '@/core/types/field'

export function createFieldValue(
  field: FieldConfig,
  raw: any,
  display: string,
  meta?: Record<string, any>
): FieldValue {
  return {
    raw,
    display,
    dataType: field.data?.type,
    widgetType: field.widget?.type,
    meta: meta || {}
  }
}

export function createEmptyFieldValue(field: FieldConfig): FieldValue {
  return createFieldValue(field, null, '', {})
}
