import { computed, defineComponent } from 'vue'
import type { WidgetComponentProps } from '@/architecture/presentation/widgets/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { useFormDataStore } from '@/architecture/runtime/stores-v2/formData'
import { createEmptyFieldValue, createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import { renderTableCell } from '@/architecture/runtime/utils/tableCellRenderer'
import { widgetComponentFactory } from '@/architecture/infrastructure/widgetRegistry'
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

  function getCellContent(field: FieldConfig, rawValue: any): { content: any; isString: boolean } {
    return renderTableCell(field, rawValue, {
      mode: 'table-cell',
      fieldPath: field.code,
      formRenderer: props.formRenderer,
      formManager: props.formManager
    })
  }

  const CellRenderer = defineComponent({
    props: {
      vnode: {
        type: Object,
        required: true
      }
    },
    setup(rendererProps: { vnode: any }) {
      return () => rendererProps.vnode
    }
  })

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
    const type = field.widget?.type || 'input'

    if (type === 'datetime') {
      return 180
    }
    if (type === 'switch') {
      return 100
    }
    if (type === 'number' || type === 'float') {
      return 120
    }

    return 150
  }

  function getColumnAlign(field: any): 'left' | 'center' | 'right' {
    const configAlign = field.widget?.config?.align
    if (configAlign === 'left' || configAlign === 'center' || configAlign === 'right') {
      return configAlign
    }

    return 'left'
  }

  function getWidgetComponent(type: string, widgetMode: string = props.mode) {
    if (widgetMode === 'response') {
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
    getCellContent,
    CellRenderer,
    displayValue,
    handleTableCellConfirm,
    getColumnWidth,
    getColumnAlign,
    getWidgetComponent,
    isNestedContainerField
  }
}
