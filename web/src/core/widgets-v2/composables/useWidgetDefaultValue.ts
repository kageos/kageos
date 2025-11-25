/**
 * useWidgetDefaultValue - Widget 默认值处理组合式函数
 * 🔥 遵循依赖倒置原则：组件自己负责自己的默认值逻辑
 * 
 * 功能：
 * - 从字段配置中获取默认值
 * - 根据字段类型进行类型转换
 * - 支持组件特定的默认值处理逻辑
 */

import { computed } from 'vue'
import type { FieldConfig, FieldValue } from '../types/field'
import { DataType } from '../../constants/widget'
import { resolveDynamicDefaultValue } from '../utils/dynamicDefaultValue'

/**
 * 获取字段的默认值
 * 每个组件可以调用此函数来获取自己的默认值
 * 
 * @param field 字段配置
 * @param customConverter 自定义转换函数（可选，用于组件特定的转换逻辑）
 * @param getAuthStore 获取 authStore 的函数（可选，用于解析 $me）
 * @returns 默认的 FieldValue
 */
export function getWidgetDefaultValue(
  field: FieldConfig,
  customConverter?: (defaultValue: any, field: FieldConfig) => any,
  getAuthStore?: () => any
): FieldValue {
  // 1. 优先使用 widget.config.default
  const config = field.widget?.config
  if (config && typeof config === 'object' && 'default' in config) {
    let defaultValue = (config as Record<string, any>).default
    if (defaultValue !== undefined && defaultValue !== null && defaultValue !== '') {
      // 🔥 解析动态变量（如 $me, $now, $today 等）
      const widgetType = field.widget?.type || ''
      defaultValue = resolveDynamicDefaultValue(defaultValue, widgetType, getAuthStore)
      
      // 使用自定义转换函数（如果提供），否则使用默认转换
      const convertedValue = customConverter
        ? customConverter(defaultValue, field)
        : convertDefaultValueByType(defaultValue, field.data?.type || DataType.STRING)
      
      // 对于 select 组件，需要找到对应的 label
      if (field.widget?.type === 'select' && Array.isArray(config.options)) {
        const option = config.options.find((opt: any) => {
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
  
  // 2. 根据字段类型设置默认值
  const fieldType = field.data?.type || DataType.STRING
  return getDefaultValueByType(fieldType)
}

/**
 * 根据字段类型转换默认值
 */
function convertDefaultValueByType(defaultValue: any, fieldType: string): any {
  switch (fieldType.toLowerCase()) {
    case DataType.INT.toLowerCase():
    case 'number':
      const intValue = Number(defaultValue)
      return isNaN(intValue) ? defaultValue : intValue
    case DataType.FLOAT.toLowerCase():
      const floatValue = Number(defaultValue)
      return isNaN(floatValue) ? defaultValue : floatValue
    case DataType.BOOL.toLowerCase():
      return Boolean(defaultValue)
    case DataType.STRINGS.toLowerCase():
    case DataType.INTS.toLowerCase():
    case DataType.FLOATS.toLowerCase():
      if (Array.isArray(defaultValue)) {
        return defaultValue
      }
      if (typeof defaultValue === 'string') {
        // 尝试解析逗号分隔的字符串
        return defaultValue.split(',').map(s => s.trim()).filter(Boolean)
      }
      return defaultValue
    default:
      return defaultValue
  }
}

/**
 * 根据字段类型获取默认值
 */
function getDefaultValueByType(fieldType: string): FieldValue {
  switch (fieldType.toLowerCase()) {
    case DataType.INT.toLowerCase():
    case DataType.FLOAT.toLowerCase():
    case 'number':
    case DataType.TIMESTAMP.toLowerCase():
      return {
        raw: null,
        display: '',
        meta: {}
      }
    case DataType.BOOL.toLowerCase():
      return {
        raw: false,
        display: '否',
        meta: {}
      }
    case DataType.STRINGS.toLowerCase():
    case DataType.INTS.toLowerCase():
    case DataType.FLOATS.toLowerCase():
    case DataType.STRUCTS.toLowerCase():
      return {
        raw: [],
        display: '[]',
        meta: {}
      }
    case DataType.STRUCT.toLowerCase():
      return {
        raw: {},
        display: '{}',
        meta: {}
      }
    case DataType.FILES.toLowerCase():
      return {
        raw: null,
        display: '',
        meta: {}
      }
    case DataType.STRING.toLowerCase():
    default:
      return {
        raw: '',
        display: '',
        meta: {}
      }
  }
}

/**
 * 在组件中使用默认值的 composable
 * 
 * @param field 字段配置
 * @param customConverter 自定义转换函数（可选）
 * @returns 默认值（FieldValue）
 */
export function useWidgetDefaultValue(
  field: FieldConfig,
  customConverter?: (defaultValue: any, field: FieldConfig) => any
) {
  const defaultValue = computed(() => {
    return getWidgetDefaultValue(field, customConverter)
  })
  
  return {
    defaultValue
  }
}

