/**
 * useTableEditMode - TableWidget 编辑模式组合式函数
 * 🔥 完全新增，不依赖旧代码
 */

import { ref, computed } from 'vue'
import type { WidgetComponentProps } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'

export function useTableEditMode(props: WidgetComponentProps) {
  const formDataStore = useFormDataStore()
  
  // 编辑状态
  const editingIndex = ref<number | null>(null)
  const isAdding = ref(false)
  
  // 表格数据（可编辑）
  const tableData = computed({
    get: () => {
      return Array.isArray(props.value?.raw) ? props.value.raw : []
    },
    set: (newValue: any[]) => {
      const newFieldValue = {
        raw: newValue,
        display: `共 ${newValue.length} 条`,
        meta: {}
      }
      
      formDataStore.setValue(props.fieldPath, newFieldValue)
    }
  })
  
  // 开始编辑
  function startEditing(index: number): void {
    editingIndex.value = index
    isAdding.value = false
  }
  
  // 开始新增
  function startAdding(): void {
    // 先添加一个空行到表格数据
    const currentData = [...tableData.value]
    const newRow: Record<string, any> = {}
    
    // 初始化新行的所有字段为空值
    const itemFields = props.field.children || []
    const newIndex = currentData.length
    
    itemFields.forEach(itemField => {
      newRow[itemField.code] = null
      
      // 初始化 formDataStore 中的字段值
      const fieldPath = `${props.fieldPath}[${newIndex}].${itemField.code}`
      formDataStore.initializeField(fieldPath, {
        raw: null,
        display: '',
        meta: {}
      })
    })
    
    currentData.push(newRow)
    tableData.value = currentData
    
    // 设置编辑索引为新行的索引
    editingIndex.value = newIndex
    isAdding.value = true
  }
  
  // 取消编辑/新增
  function cancelEditing(): void {
    // 如果是新增模式且还没有保存，需要移除刚添加的空行
    if (isAdding.value && editingIndex.value !== null) {
      const currentData = [...tableData.value]
      const indexToRemove = editingIndex.value
      currentData.splice(indexToRemove, 1)
      tableData.value = currentData
      
      // 清理 formDataStore 中该行的数据
      const itemFields = props.field.children || []
      itemFields.forEach(itemField => {
        const fieldPath = `${props.fieldPath}[${indexToRemove}].${itemField.code}`
        // 注意：formDataStore 没有 delete 方法，这里先不清理，后续可以优化
      })
    }
    
    editingIndex.value = null
    isAdding.value = false
  }
  
  // 保存（新增或编辑）
  function saveRow(rowData: Record<string, any>): void {
    const currentData = [...tableData.value]
    
    if (isAdding.value) {
      // 新增
      currentData.push(rowData)
    } else if (editingIndex.value !== null) {
      // 编辑
      currentData[editingIndex.value] = rowData
    }
    
    tableData.value = currentData
    cancelEditing()
  }
  
  // 删除行
  function deleteRow(index: number): void {
    const currentData = [...tableData.value]
    currentData.splice(index, 1)
    tableData.value = currentData
  }
  
  return {
    editingIndex,
    isAdding,
    tableData,
    startEditing,
    startAdding,
    cancelEditing,
    saveRow,
    deleteRow
  }
}

