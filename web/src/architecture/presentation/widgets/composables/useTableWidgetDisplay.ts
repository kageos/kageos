import { computed } from 'vue'
import type { WidgetComponentProps } from '@/architecture/presentation/widgets/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { useFormDataStore } from '@/architecture/presentation/context/formRuntimeContext'
import { createEmptyFieldValue, createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import { normalizeWidgetType, WidgetType } from '@/architecture/domain/constants/widget'
import { widgetComponentFactory } from '@/architecture/presentation/widgets/registry'
import {
  getTableRowFieldPresenceState,
  shouldShowTableRowField,
} from '@/architecture/presentation/widgets/utils/tableRowVisibility'
import { useTableResponseMode } from './useTableResponseMode'
import { useTableCellMode } from './useTableCellMode'

interface UseTableWidgetDisplayOptions {
  tableData: { value: any[] }
  itemFields: { value: FieldConfig[] }
}

export function useTableWidgetDisplay(
  props: WidgetComponentProps,
  { tableData, itemFields }: UseTableWidgetDisplayOptions
) {
  const formDataStore = useFormDataStore()
  const responseMode = useTableResponseMode()
  const tableCellMode = useTableCellMode(props)

  const responseTableData = computed(() => {
    if (props.mode === 'response') {
      return Array.isArray(props.value?.raw) ? props.value.raw : []
    }
    return []
  })

  function getResponseRowFieldValue(rowIndex: number, fieldCode: string): FieldValue {
    const itemField = itemFields.value.find((f) => f.code === fieldCode) || props.field

    if (props.mode !== 'response') {
      return createEmptyFieldValue(itemField)
    }

    const currentTableData = responseTableData.value
    if (!currentTableData || rowIndex < 0 || rowIndex >= currentTableData.length) {
      return createEmptyFieldValue(itemField)
    }

    const row = currentTableData[rowIndex]
    const rawValue = row?.[fieldCode]
    const display = rawValue !== null && rawValue !== undefined
      ? (typeof rawValue === 'object' ? JSON.stringify(rawValue) : String(rawValue))
      : ''

    return createFieldValue(itemField, rawValue ?? null, display)
  }

  function getEditRowSource(rowIndex: number): Record<string, any> | null {
    const row = tableData.value[rowIndex]
    return row && typeof row === 'object' && !Array.isArray(row) ? row : null
  }

  function getResponseRowSource(rowIndex: number): Record<string, any> | null {
    const row = responseTableData.value[rowIndex]
    return row && typeof row === 'object' && !Array.isArray(row) ? row : null
  }

  function isEditRowFieldVisible(rowIndex: number, field: FieldConfig): boolean {
    return shouldShowTableRowField(
      formDataStore,
      props.fieldPath,
      rowIndex,
      getEditRowSource(rowIndex),
      field,
      itemFields.value
    )
  }

  function isResponseRowFieldVisible(rowIndex: number, field: FieldConfig): boolean {
    return shouldShowTableRowField(
      formDataStore,
      props.fieldPath,
      rowIndex,
      getResponseRowSource(rowIndex),
      field,
      itemFields.value
    )
  }

  function getEditRowFieldPresenceState(rowIndex: number, field: FieldConfig) {
    return getTableRowFieldPresenceState(
      formDataStore,
      props.fieldPath,
      rowIndex,
      getEditRowSource(rowIndex),
      field,
      itemFields.value
    )
  }

  function getVisibleResponseDetailFields(rowIndex: number): FieldConfig[] {
    if (rowIndex < 0) {
      return []
    }

    return itemFields.value.filter((field) => isResponseRowFieldVisible(rowIndex, field))
  }

  const displayValue = computed(() => {
    const value = formDataStore.data.has(props.fieldPath)
      ? formDataStore.getValue(props.fieldPath)
      : props.value

    if (!value) {
      return '共 0 条记录'
    }

    const raw = value.raw
    if (raw === null || raw === undefined || raw === '') {
      return '共 0 条记录'
    }

    if (Array.isArray(raw)) {
      return `共 ${raw.length} 条记录`
    }

    if (typeof raw === 'object') {
      try {
        return JSON.stringify(raw)
      } catch {
        return '共 0 条记录'
      }
    }

    return String(raw)
  })

  function handleTableCellConfirm(): void {
    tableCellMode.confirmDrawer()
  }

  function getColumnWidth(field: any): number {
    const type = normalizeWidgetType(field.widget?.type)
    const code = (field.code || '').toLowerCase()
    const name = field.name || ''

    // 智能推断：根据表头中文字符数给一个基础缓冲宽度 (每个汉字大约 14px + 表头 padding 和 icon 大约 40px)
    const nameStr = String(name)
    // 粗略计算：中文字符算作 14px，英文字符算作 8px
    let charWidth = 0
    for (let i = 0; i < nameStr.length; i++) {
      charWidth += nameStr.charCodeAt(i) > 255 ? 14 : 8
    }
    const headerWidth = Math.max(70, charWidth + 40) // 确保至少给表头 70px 空间

    // 1. 智能推断：系统字段（创建人、更新时间等），降低基础宽度
    if (/^(created_at|updated_at|create_time|update_time|creator|updater|modifier)$/.test(code) || 
        /(创建|更新|修改)(时间|人)/.test(name)) {
      if (type === WidgetType.DATETIME) return Math.max(headerWidth, 160)
      return Math.max(headerWidth, 80)
    }

    // 2. 智能推断：备注/说明类长文本，降低下限，由 flex 或 ellipsis 处理
    if (/^(remark|desc|description|summary)$/.test(code) || /(备注|说明|描述|摘要)/.test(name)) {
      return Math.max(headerWidth, 110)
    }

    // 3. 其他常规字段，极限压缩空数据时的最小宽度，把空间交给 Element Plus 自动分配
    let minWidth = 100
    if (type === WidgetType.DATETIME) {
      minWidth = 160
    } else if (type === WidgetType.SWITCH) {
      minWidth = 75
    } else if (type === WidgetType.INTEGER || type === WidgetType.FLOAT) {
      minWidth = 80 // 数字通常很短，如果是 '-' 会被压得很窄
    } else if (type === WidgetType.PROGRESS || type === WidgetType.SLIDER) {
      minWidth = 140
    } else if (type === WidgetType.TEXT_AREA || type === WidgetType.RICH_TEXT) {
      minWidth = 140
    } else if (type === WidgetType.TEXT || type === WidgetType.INPUT || type === WidgetType.LINK) {
      minWidth = 90 // 极简下限，如果是短文本或空值就不会占地方
    } else if (type === WidgetType.FILES) {
      minWidth = 110
    } else if (type === WidgetType.DEPARTMENT || type === WidgetType.DEPARTMENTS) {
      minWidth = 100
    } else if (type === WidgetType.USER || type === WidgetType.USERS) {
      minWidth = 90
    } else {
      minWidth = 90
    }

    return Math.max(headerWidth, minWidth)
  }

  function getColumnAlign(field: any): 'left' | 'center' | 'right' {
    const configAlign = field.widget?.config?.align
    if (configAlign === 'left' || configAlign === 'center' || configAlign === 'right') {
      return configAlign
    }

    return 'left'
  }

  function getWidgetComponent(type: string, widgetMode: string = props.mode) {
    if (widgetMode === 'response' || props.mode === 'response') {
      return widgetComponentFactory.getResponseComponent(type)
    }
    return widgetComponentFactory.getRequestComponent(type)
  }

  function isNestedContainerField(field: FieldConfig): boolean {
    return field.widget?.type === 'form' || field.widget?.type === 'table'
  }

  return {
    responseMode,
    tableCellMode,
    responseTableData,
    getResponseRowFieldValue,
    getEditRowFieldPresenceState,
    isEditRowFieldVisible,
    isResponseRowFieldVisible,
    getVisibleResponseDetailFields,
    displayValue,
    handleTableCellConfirm,
    getColumnWidth,
    getColumnAlign,
    getWidgetComponent,
    isNestedContainerField
  }
}
