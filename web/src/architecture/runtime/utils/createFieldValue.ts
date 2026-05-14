import type { FieldConfig, FieldValue } from '@/architecture/runtime/types/field'

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

export function createRawFieldValue(
  raw: any,
  display: string,
  meta?: Record<string, any>
): FieldValue {
  return {
    raw,
    display,
    meta: meta || {}
  }
}

export function createEmptyFieldValue(field: FieldConfig): FieldValue {
  return createFieldValue(field, null, '', {})
}

export function createEmptyRawFieldValue(meta?: Record<string, any>): FieldValue {
  return createRawFieldValue(null, '', meta)
}

export function createAutoFieldValue(
  raw: any,
  field?: FieldConfig,
  meta?: Record<string, any>
): FieldValue {
  const normalizedRaw = raw ?? null
  const display = normalizedRaw === null
    ? ''
    : (typeof normalizedRaw === 'object' ? JSON.stringify(normalizedRaw) : String(normalizedRaw))

  if (field) {
    return createFieldValue(field, normalizedRaw, display, meta)
  }

  return createRawFieldValue(normalizedRaw, display, meta)
}
