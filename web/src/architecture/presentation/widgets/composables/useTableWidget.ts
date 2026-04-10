/**
 * useTableWidget - TableWidget 组合式函数（共享逻辑）
 * 🔥 完全新增，不依赖旧代码
 */

import { computed } from 'vue'
import type { WidgetComponentProps } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { syncTableContainerValue } from '@/core/widgetRuntime/containerValue'
import { clearScopedDependentFields } from '@/core/widgetRuntime/dependency'

function toFieldValue(rawValue: any) {
  return {
    raw: rawValue ?? null,
    display: rawValue !== null && rawValue !== undefined
      ? (typeof rawValue === 'object' ? JSON.stringify(rawValue) : String(rawValue))
      : '',
    meta: {}
  }
}

export function useTableWidget(props: WidgetComponentProps) {
  const formDataStore = useFormDataStore()
  
  // 表格数据
  const tableData = computed(() => {
    const storeValue = formDataStore.getValue(props.fieldPath)
    if (Array.isArray(storeValue?.raw)) {
      return storeValue.raw
    }

    return Array.isArray(props.value?.raw) ? props.value.raw : []
  })
  
  // 子字段列表（表格列）
  const itemFields = computed(() => {
    return props.field.children || []
  })
  
  // 获取行的字段值
  function getRowFieldValue(rowIndex: number, fieldCode: string): any {
    const fieldPath = `${props.fieldPath}[${rowIndex}].${fieldCode}`
    const storeValue = formDataStore.getValue(fieldPath)
    const hasStoredValue = formDataStore.data.has(fieldPath)

    if (hasStoredValue || (storeValue && (
      storeValue.raw !== null ||
      storeValue.display !== '' ||
      Object.keys(storeValue.meta || {}).length > 0
    ))) {
      return storeValue
    }

    const rawRow = tableData.value[rowIndex]
    if (rawRow && typeof rawRow === 'object' && fieldCode in rawRow) {
      return toFieldValue(rawRow[fieldCode])
    }

    return storeValue
  }

  function buildCurrentRowData(rowIndex: number): Record<string, any> {
    const currentRow = tableData.value[rowIndex]
    const rowData = currentRow && typeof currentRow === 'object' && !Array.isArray(currentRow)
      ? { ...currentRow }
      : {}

    itemFields.value.forEach((itemField) => {
      const fieldPath = `${props.fieldPath}[${rowIndex}].${itemField.code}`

      if (formDataStore.data.has(fieldPath)) {
        rowData[itemField.code] = formDataStore.getValue(fieldPath).raw
        return
      }

      if (itemField.widget?.type === 'form' && rowData[itemField.code] === undefined) {
        rowData[itemField.code] = {}
        return
      }

      if (itemField.widget?.type === 'table' && rowData[itemField.code] === undefined) {
        rowData[itemField.code] = []
      }
    })

    return rowData
  }
  
  // 更新行的字段值
  function updateRowFieldValue(rowIndex: number, fieldCode: string, value: any): void {
    const fieldPath = `${props.fieldPath}[${rowIndex}].${fieldCode}`
    formDataStore.setValue(fieldPath, value)
    props.formRenderer?.clearFieldErrors?.(fieldPath)

    const clearedFieldPaths = clearScopedDependentFields({
      formDataStore,
      fields: itemFields.value,
      changedFieldCode: fieldCode,
      scopePath: `${props.fieldPath}[${rowIndex}]`,
    })

    clearedFieldPaths.forEach((clearedFieldPath) => {
      props.formRenderer?.clearFieldErrors?.(clearedFieldPath, { includeSubtree: true })
    })

    const nextRows = [...tableData.value]
    nextRows[rowIndex] = buildCurrentRowData(rowIndex)
    syncTableContainerValue(formDataStore, props.field, props.fieldPath, nextRows, props.value)
    props.formRenderer?.clearFieldErrors?.(props.fieldPath)
  }
  
  // 获取所有行的数据（用于聚合计算）
  function getAllRowsData(): any[] {
    const rows: any[] = []
    
    tableData.value.forEach((row, index) => {
      const rowData: Record<string, any> = {}
      
      itemFields.value.forEach(itemField => {
        const fieldPath = `${props.fieldPath}[${index}].${itemField.code}`
        const itemValue = formDataStore.getValue(fieldPath)
        
        // 保存 raw 值
        rowData[itemField.code] = itemValue?.raw
        
        // 🔥 合并 displayInfo（来自 Select 回调）
        if (itemValue?.meta?.displayInfo && typeof itemValue.meta.displayInfo === 'object') {
          Object.assign(rowData, itemValue.meta.displayInfo)
        }
        
        // 🔥 合并行内聚合统计（来自 MultiSelect，场景 4 二层聚合）
        if (itemValue?.meta?.rowStatistics && typeof itemValue.meta.rowStatistics === 'object') {
          Object.assign(rowData, itemValue.meta.rowStatistics)
        }
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
