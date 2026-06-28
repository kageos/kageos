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

    // 动态嗅探数据内容长度 (只嗅探 response 模式下的数据)
    let maxDataCharLength = 0
    let hasData = false
    if (responseMode.value && responseTableData.value && Array.isArray(responseTableData.value)) {
      const rows = responseTableData.value
      for (const row of rows) {
        let val = row[code]
        if (val !== undefined && val !== null && val !== '') {
          // 判断是否是真正的空标识
          const strVal = String(val).trim()
          if (strVal !== '-' && strVal !== '[]' && strVal !== '{}') {
            hasData = true
            let currLen = 0
            for (let i = 0; i < strVal.length; i++) {
               currLen += strVal.charCodeAt(i) > 255 ? 13 : 8
            }
            if (currLen > maxDataCharLength) {
              maxDataCharLength = currLen
            }
          }
        }
      }
    }

    // 如果嗅探到当前页这一列根本没有任何实际数据（全是 -、空值、[] 等），直接把它压死到表头宽度！
    if (responseMode.value && !hasData) {
       return Math.max(headerWidth, 60)
    }

    // 如果有数据，为数据预留一个合理的空间，最高不超过一个合理的上限（例如 300px），避免单列撑爆
    const dataWidth = hasData ? Math.min(maxDataCharLength + 32, 280) : 0

    // 根据综合计算出来的真实数据需求与组件本身的底限结合：
    const dataBasedWidth = Math.max(headerWidth, dataWidth)

    // 给组件一个保守的最小操作/展示空间保障
    let minWidth = 60
    if (type === WidgetType.DATETIME) {
      minWidth = 135
    } else if (type === WidgetType.SWITCH) {
      minWidth = 60
    } else if (type === WidgetType.INTEGER || type === WidgetType.FLOAT) {
      minWidth = Math.max(headerWidth, 60)
    } else if (type === WidgetType.PROGRESS || type === WidgetType.SLIDER) {
      minWidth = 120
    } else if (type === WidgetType.FILES) {
      minWidth = 100
    } else if (type === WidgetType.TEXT_AREA || type === WidgetType.RICH_TEXT) {
      minWidth = Math.max(headerWidth, 100)
    } else if (type === WidgetType.SELECT || type === WidgetType.MULTI_SELECT) {
      minWidth = Math.max(headerWidth, 80)
    }

    // 在编辑模式下，由于输入框本身需要操作空间，走组件底线与表头宽度
    if (!responseMode.value) {
       return Math.max(headerWidth, minWidth)
    }

    // 在响应模式下，结合真实数据宽度和组件特征来分配，如果真实数据极短，允许打破组件的固定最小宽度
    // 但必须大于等于表头
    return Math.max(headerWidth, Math.min(Math.max(minWidth, dataBasedWidth), 280))
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
