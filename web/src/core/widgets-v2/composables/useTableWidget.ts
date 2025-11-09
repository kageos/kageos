/**
 * useTableWidget - TableWidget 组合式函数（共享逻辑）
 * 🔥 完全新增，不依赖旧代码
 */

import { computed } from 'vue'
import type { WidgetComponentProps } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'

export function useTableWidget(props: WidgetComponentProps) {
  const formDataStore = useFormDataStore()
  
  // 表格数据
  const tableData = computed(() => {
    return Array.isArray(props.value?.raw) ? props.value.raw : []
  })
  
  // 子字段列表（表格列）
  const itemFields = computed(() => {
    return props.field.children || []
  })
  
  // 获取行的字段值
  function getRowFieldValue(rowIndex: number, fieldCode: string): any {
    const fieldPath = `${props.fieldPath}[${rowIndex}].${fieldCode}`
    return formDataStore.getValue(fieldPath)
  }
  
  // 更新行的字段值
  function updateRowFieldValue(rowIndex: number, fieldCode: string, value: any): void {
    const fieldPath = `${props.fieldPath}[${rowIndex}].${fieldCode}`
    formDataStore.setValue(fieldPath, value)
  }
  
  // 获取所有行的数据（用于聚合计算）
  function getAllRowsData(): any[] {
    const rows: any[] = []
    
    tableData.value.forEach((row, index) => {
      const rowData: Record<string, any> = {}
      
      itemFields.value.forEach(itemField => {
        const fieldPath = `${props.fieldPath}[${index}].${itemField.code}`
        const itemValue = formDataStore.getValue(fieldPath)
        rowData[itemField.code] = itemValue?.raw
      })
      
      rows.push(rowData)
    })
    
    return rows
  }
  
  return {
    tableData,
    itemFields,
    getRowFieldValue,
    updateRowFieldValue,
    getAllRowsData
  }
}

