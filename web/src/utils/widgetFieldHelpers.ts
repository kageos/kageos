import type { FieldConfig, FieldValue } from '@/architecture/runtime/types/field'
import { createFieldValue } from '@/architecture/runtime/utils/createFieldValue'

interface CreateWidgetFieldConfigOptions {
  code: string
  name: string
  widgetType: string
  widgetConfig?: Record<string, any>
  dataType?: string
  overrides?: Partial<FieldConfig>
}

interface CreateStringFieldValueOptions {
  delimiter?: string
  display?: string
  emptyRaw?: string | null
  meta?: Record<string, any>
}

export function createWidgetFieldConfig(options: CreateWidgetFieldConfigOptions): FieldConfig {
  const {
    code,
    name,
    widgetType,
    widgetConfig = {},
    dataType = 'string',
    overrides = {}
  } = options

  const overrideWidget = (overrides.widget || {}) as NonNullable<FieldConfig['widget']>
  const overrideData = overrides.data || {}

  return {
    ...overrides,
    code,
    name,
    widget: {
      ...overrideWidget,
      type: widgetType,
      config: {
        ...widgetConfig,
        ...(overrideWidget.config || {})
      }
    },
    data: {
      type: dataType,
      ...overrideData
    }
  }
}

export function normalizeDelimitedString(
  raw: string | string[] | null | undefined,
  delimiter = ','
): string {
  if (Array.isArray(raw)) {
    return raw.map(item => String(item).trim()).filter(Boolean).join(delimiter)
  }

  if (typeof raw !== 'string') {
    return ''
  }

  return raw.split(delimiter).map(item => item.trim()).filter(Boolean).join(delimiter)
}

export function createStringFieldValue(
  field: FieldConfig,
  raw: string | string[] | null | undefined,
  options: CreateStringFieldValueOptions = {}
): FieldValue {
  const delimiter = options.delimiter || ','
  const normalizedRaw = normalizeDelimitedString(raw, delimiter)

  if (!normalizedRaw) {
    return createFieldValue(field, options.emptyRaw ?? null, '', options.meta)
  }

  return createFieldValue(
    field,
    normalizedRaw,
    options.display ?? normalizedRaw,
    options.meta
  )
}

export function extractStringFieldRaw(
  value: FieldValue | null | undefined,
  delimiter = ','
): string {
  return normalizeDelimitedString(value?.raw as string | string[] | null | undefined, delimiter)
}
