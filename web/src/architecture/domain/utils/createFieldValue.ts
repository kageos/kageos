import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'

function isRecord(value: unknown): value is Record<string, any> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function firstDisplayText(value: Record<string, any>): string {
  for (const key of ['display', 'label', 'name', 'title', 'text']) {
    const candidate = value[key]
    if (candidate !== null && candidate !== undefined && candidate !== '') {
      return String(candidate)
    }
  }
  return ''
}

function optionDisplayInfo(value: Record<string, any>): unknown {
  return value.displayInfo ?? value.display_info
}

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

export function createDisplayAwareFieldValue(
  raw: unknown,
  field?: FieldConfig,
  meta?: Record<string, unknown>
): FieldValue {
  if (isRecord(raw) && 'raw' in raw && 'display' in raw) {
    const existing = raw as FieldValue
    return {
      raw: existing.raw,
      display: existing.display || (existing.raw === null || existing.raw === undefined ? '' : String(existing.raw)),
      dataType: field?.data?.type || existing.dataType,
      widgetType: field?.widget?.type || existing.widgetType,
      meta: {
        ...(existing.meta || {}),
        ...(meta || {}),
      },
    }
  }

  if (Array.isArray(raw) && raw.every(isRecord) && raw.some((item) => 'value' in item)) {
    const values = raw.map((item) => item.value)
    const labels = raw.map((item) => firstDisplayText(item) || String(item.value ?? '')).filter(Boolean)
    const displayInfos = raw.map(optionDisplayInfo).filter((item) => item !== undefined && item !== null)
    return createFieldValue(field || ({ code: '', name: '', widget: { type: '' } } as FieldConfig), values, labels.join(', '), {
      ...(meta || {}),
      ...(displayInfos.length > 0 ? { displayInfo: displayInfos } : {}),
    })
  }

  if (isRecord(raw) && 'value' in raw) {
    const display = firstDisplayText(raw) || String(raw.value ?? '')
    const displayInfo = optionDisplayInfo(raw)
    return createFieldValue(field || ({ code: '', name: '', widget: { type: '' } } as FieldConfig), raw.value, display, {
      ...(meta || {}),
      ...(displayInfo ? { displayInfo } : {}),
    })
  }

  return createAutoFieldValue(raw, field, meta)
}
