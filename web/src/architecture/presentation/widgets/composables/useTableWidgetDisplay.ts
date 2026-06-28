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

    // 智能推断：根据表头中文字符数给一个基础缓冲宽度 (每个汉字大约 13px + 表头 padding 24px)
    const nameStr = String(name)
    let charWidth = 0
    for (let i = 0; i < nameStr.length; i++) {
      charWidth += nameStr.charCodeAt(i) > 255 ? 13 : 8
    }
    const headerWidth = charWidth + 24 

    // 1. 智能推断：空值倾向很高的字段（如各类说明、原因、备注）
    // 即使它是 textarea，如果当前页没数据，它的极简宽度只要能装下表头即可
    if (/^(remark|desc|description|summary|reason|cause|degrade_reason|.*_reason|source_note)$/.test(code) || 
        /(备注|说明|描述|摘要|原因|理由)/.test(name)) {
      return headerWidth
    }

    // 2. 智能推断：系统级字段（ID、创建人、更新时间等）
    if (/^(id|created_at|updated_at|create_time|update_time|creator|updater|modifier|created_by)$/.test(code) || 
        /(ID|创建|更新|修改)(时间|人)/.test(name)) {
      if (type === WidgetType.DATETIME) return Math.max(headerWidth, 140)
      return Math.max(headerWidth, 60)
    }

    // 3. 其他常规字段，根据组件类型给一个相对合理的最小宽度保障
    let minWidth = 80
    if (type === WidgetType.DATETIME) {
      minWidth = 140
    } else if (type === WidgetType.SWITCH) {
      minWidth = 60
    } else if (type === WidgetType.INTEGER || type === WidgetType.FLOAT) {
      minWidth = Math.max(headerWidth, 60)
    } else if (type === WidgetType.PROGRESS || type === WidgetType.SLIDER) {
      minWidth = 120
    } else if (type === WidgetType.FILES) {
      minWidth = 100
    } else if (type === WidgetType.TEXT_AREA || type === WidgetType.RICH_TEXT) {
      // 常规文本区（非备注类）
      minWidth = 120
    } else if (type === WidgetType.SELECT || type === WidgetType.MULTI_SELECT) {
      // 选项通常带有 tag，需要稍微多一点点宽度
      minWidth = Math.max(headerWidth, 100)
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
