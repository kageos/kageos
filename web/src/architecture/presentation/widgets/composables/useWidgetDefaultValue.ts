import { computed } from 'vue'
import type { FieldConfig } from '@/architecture/domain/types'
import { getWidgetDefaultValue } from '@/core/widgetRuntime/defaultValue'

export { getWidgetDefaultValue } from '@/core/widgetRuntime/defaultValue'

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
