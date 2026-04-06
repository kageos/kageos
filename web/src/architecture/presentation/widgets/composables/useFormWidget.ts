/**
 * useFormWidget - FormWidget 组合式函数
 * 🔥 完全新增，不依赖旧代码
 * 
 * 功能：
 * - 提取 FormWidget 的共享逻辑
 * - 处理子字段的递归渲染
 * - 处理条件渲染
 */

import { computed } from 'vue'
import type { WidgetComponentProps } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { createAutoFieldValue, createEmptyRawFieldValue } from '@/core/utils/createFieldValue'

export function useFormWidget(props: WidgetComponentProps) {
  const formDataStore = useFormDataStore()
  
  // 子字段列表
  const subFields = computed(() => {
    return props.field.children || []
  })
  
  // 可见子字段（根据条件渲染规则过滤）
  const visibleSubFields = computed(() => {
    return subFields.value
  })
  
  // 获取子字段的值
  function getSubFieldValue(subFieldCode: string): any {
    // 🔥 响应模式 或 table-cell 抽屉（只读数据）：从 props.value.raw 读取
    // 表格里 form 列传的是 mode=table-cell，抽屉打开时也要用 value.raw，否则会去 formDataStore 取不到
    const isReadOnlyFromRaw =
      props.mode === 'response' ||
      (props.mode === 'table-cell' && props.value?.raw && typeof props.value.raw === 'object' && !Array.isArray(props.value.raw))
    if (isReadOnlyFromRaw) {
      const rawValue = props.value?.raw
      if (rawValue && typeof rawValue === 'object' && !Array.isArray(rawValue)) {
        const subValue = rawValue[subFieldCode]
        return createAutoFieldValue(subValue)
      }
      return createEmptyRawFieldValue()
    }

    // 编辑模式下，从 formDataStore 读取
    const subFieldPath = `${props.fieldPath}.${subFieldCode}`
    return formDataStore.getValue(subFieldPath)
  }
  
  // 更新子字段的值
  function updateSubFieldValue(subFieldCode: string, value: any): void {
    const subFieldPath = `${props.fieldPath}.${subFieldCode}`
    formDataStore.setValue(subFieldPath, value)
  }
  
  return {
    subFields,
    visibleSubFields,
    getSubFieldValue,
    updateSubFieldValue
  }
}
