import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'
import { DataType } from '@/architecture/runtime/constants/widget'
import { resolveDynamicDefaultValue } from './dynamicDefaultValue'

export function getRenderDefaultFromConfig(config: unknown): any {
  if (!config || typeof config !== 'object') {
    return undefined
  }

  const record = config as Record<string, any>
  if ('render_default' in record) {
    return record.render_default
  }
  return undefined
}

export function getWidgetDefaultValue(
  field: FieldConfig,
  customConverter?: (defaultValue: any, field: FieldConfig) => any,
  getAuthStore?: () => any
): FieldValue {
  const config = field.widget?.config
  const configRecord = config && typeof config === 'object' ? config as Record<string, any> : {}
  const configuredRenderDefault = getRenderDefaultFromConfig(config)
  if (configuredRenderDefault !== undefined) {
    let defaultValue = configuredRenderDefault

    if (defaultValue !== undefined && defaultValue !== null && defaultValue !== '') {
      const widgetType = field.widget?.type || ''
      defaultValue = resolveDynamicDefaultValue(defaultValue, widgetType, getAuthStore)

      const convertedValue = customConverter
        ? customConverter(defaultValue, field)
        : convertDefaultValueByType(defaultValue, field.data?.type || DataType.STRING)

      if (field.widget?.type === 'select' && Array.isArray(configRecord.options)) {
        const option = configRecord.options.find((opt: any) => {
          if (typeof opt === 'string') {
            return opt === convertedValue
          }
          return opt.value === convertedValue || opt.label === convertedValue
        })

        const display = option
          ? (typeof option === 'string' ? option : option.label || String(convertedValue))
          : String(convertedValue)

        return {
          raw: convertedValue,
          display,
          meta: {}
        }
      }

      return {
        raw: convertedValue,
        display: String(convertedValue),
        meta: {}
      }
    }
  }

  const widgetType = field.widget?.type
  if (widgetType === 'table') {
    return {
      raw: [],
      display: '',
      meta: {}
    }
  }
  if (widgetType === 'form') {
    return {
      raw: {},
      display: '',
      meta: {}
    }
  }

  const fieldType = field.data?.type || DataType.STRING
  return getDefaultValueByType(fieldType)
}

function convertDefaultValueByType(defaultValue: any, fieldType: string): any {
  switch (fieldType.toLowerCase()) {
    case DataType.INT.toLowerCase():
    case 'number': {
      const intValue = Number(defaultValue)
      return isNaN(intValue) ? defaultValue : intValue
    }
    case DataType.FLOAT.toLowerCase(): {
      const floatValue = Number(defaultValue)
      return isNaN(floatValue) ? defaultValue : floatValue
    }
    case DataType.BOOL.toLowerCase():
      if (typeof defaultValue === 'string') {
        return ['true', '1', 'yes', '是'].includes(defaultValue.trim().toLowerCase())
      }
      return Boolean(defaultValue)
    case DataType.STRINGS.toLowerCase():
    case DataType.INTS.toLowerCase():
    case DataType.FLOATS.toLowerCase():
      if (Array.isArray(defaultValue)) {
        return defaultValue
      }
      if (typeof defaultValue === 'string') {
        return defaultValue.split(',').map(s => s.trim()).filter(Boolean)
      }
      return defaultValue
    default:
      return defaultValue
  }
}

function getDefaultValueByType(fieldType: string): FieldValue {
  switch (fieldType.toLowerCase()) {
    case DataType.INT.toLowerCase():
    case DataType.FLOAT.toLowerCase():
    case 'number':
      return { raw: null, display: '', meta: {} }
    case DataType.BOOL.toLowerCase():
      return { raw: false, display: '否', meta: {} }
    case DataType.STRINGS.toLowerCase():
    case DataType.INTS.toLowerCase():
    case DataType.FLOATS.toLowerCase():
    case DataType.STRUCTS.toLowerCase():
      return { raw: [], display: '[]', meta: {} }
    case DataType.STRUCT.toLowerCase():
      return { raw: {}, display: '{}', meta: {} }
    case DataType.STRING.toLowerCase():
    default:
      return { raw: '', display: '', meta: {} }
  }
}
