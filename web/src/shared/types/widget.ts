import type { WidgetTypes } from '@/core/types/field'
import type { ReactiveFormDataManager } from '@/core/managers/ReactiveFormDataManager'
import type { FormRendererContext } from '@/core/types/widget'

export type WidgetMode = WidgetTypes.WidgetMode
export type FieldConfig = WidgetTypes.FieldConfig
export type FieldValue = WidgetTypes.FieldValue

export interface WidgetComponentProps {
  field: WidgetTypes.FieldConfig
  value: WidgetTypes.FieldValue
  mode: WidgetTypes.WidgetMode
  fieldPath: string
  formManager?: ReactiveFormDataManager | null
  formRenderer?: FormRendererContext | null
  depth?: number
  searchType?: string
  rowData?: any
  rowIndex?: number
  functionName?: string
  recordId?: string | number
  parentMode?: WidgetTypes.WidgetMode
}

export type WidgetComponentEmits = {
  (event: 'update:modelValue', value: WidgetTypes.FieldValue): void
  (event: 'statistics:updated', statistics: Record<string, unknown>): void
  (event: 'drawer:change', show: boolean): void
}
