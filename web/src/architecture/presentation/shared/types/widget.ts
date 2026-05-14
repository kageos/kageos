import type { FunctionDetail, WidgetTypes } from '@/architecture/domain/types/field'
import type { FormValueStorePort } from '@/architecture/domain/interfaces/IFormStateManager'

export type WidgetMode = WidgetTypes.WidgetMode
export type FieldConfig = WidgetTypes.FieldConfig
export type FieldValue = WidgetTypes.FieldValue

export interface FormRendererContext {
  getFunctionMethod: () => string
  getFunctionRouter: () => string
  getFunctionDetail?: () => FunctionDetail
  getSubmitData: () => Record<string, any>
  getFieldError?: (fieldPath: string) => string | null
  clearFieldErrors?: (fieldPath: string, options?: { includeSubtree?: boolean }) => void
}

export interface WidgetComponentProps {
  field: WidgetTypes.FieldConfig
  value: WidgetTypes.FieldValue
  mode: WidgetTypes.WidgetMode
  fieldPath: string
  formManager?: FormValueStorePort | null
  formRenderer?: FormRendererContext | null
  depth?: number
  searchType?: string
  rowData?: any
  rowIndex?: number
  parentMode?: WidgetTypes.WidgetMode
  functionMethod?: string
  functionRouter?: string
}

export type WidgetComponentEmits = {
  (event: 'update:modelValue', value: WidgetTypes.FieldValue): void
  (event: 'statistics:updated', statistics: Record<string, unknown>): void
  (event: 'drawer:change', show: boolean): void
}
