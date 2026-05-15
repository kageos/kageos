import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'

export function createFieldValue(
  field: FieldConfig,
  raw: unknown,
  display: string,
  meta?: Record<string, unknown>
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
  raw: unknown,
  display: string,
  meta?: Record<string, unknown>
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

export function createEmptyRawFieldValue(meta?: Record<string, unknown>): FieldValue {
  return createRawFieldValue(null, '', meta)
}

export function createAutoFieldValue(
  raw: unknown,
  field?: FieldConfig,
  meta?: Record<string, unknown>
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
