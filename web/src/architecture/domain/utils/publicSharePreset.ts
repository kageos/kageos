import type { FieldConfig, FunctionDetail } from '@/architecture/domain/types'

export function buildPublicSharePresetValues(data: Record<string, unknown>): Record<string, unknown> {
  const preset: Record<string, unknown> = {}
  Object.entries(data || {}).forEach(([key, value]) => {
    if (!key || isEmptyPublicSharePresetValue(value)) {
      return
    }
    preset[key] = value
  })
  return preset
}

export function lockFunctionDetailPresetFields(
  detail: FunctionDetail,
  presetValues?: Record<string, unknown> | null
): FunctionDetail {
  const preset = presetValues || {}
  const presetKeys = new Set(Object.keys(preset))
  if (presetKeys.size === 0 || !detail.schema?.form?.request) {
    return detail
  }

  return {
    ...detail,
    schema: {
      ...detail.schema,
      form: {
        ...detail.schema.form,
        request: detail.schema.form.request.map((field) => lockPresetField(field, presetKeys)),
      },
    },
  }
}

function lockPresetField(field: FieldConfig, presetKeys: Set<string>): FieldConfig {
  const children = field.children?.map((child) => lockPresetField(child, presetKeys))
  if (!presetKeys.has(field.code)) {
    return children ? { ...field, children } : field
  }

  return {
    ...field,
    children,
    widget: {
      ...field.widget,
      config: {
        ...(field.widget?.config || {}),
        disabled: true,
      },
    },
  }
}

function isEmptyPublicSharePresetValue(value: unknown): boolean {
  if (value === null || value === undefined || value === '') {
    return true
  }
  if (Array.isArray(value)) {
    return value.length === 0
  }
  if (typeof value === 'object') {
    return Object.keys(value as Record<string, unknown>).length === 0
  }
  return false
}
