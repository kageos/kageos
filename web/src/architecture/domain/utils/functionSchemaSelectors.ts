import { WidgetType } from '@/architecture/domain/types/api'
import type { FieldConfig, FieldScene, FunctionDetail, FunctionSchema } from '@/architecture/domain/types/field'

export function getFunctionSchema(functionDetail?: FunctionDetail | null): FunctionSchema | null {
  return functionDetail?.schema || null
}

export function getFunctionType(functionDetail?: FunctionDetail | null): string {
  return getFunctionSchema(functionDetail)?.type || functionDetail?.template_type || ''
}

export function getFunctionCallbacks(functionDetail?: FunctionDetail | null): string[] {
  return getFunctionSchema(functionDetail)?.callbacks || []
}

export function hasFunctionSchemaCallback(functionDetail: FunctionDetail | null | undefined, callback: string): boolean {
  return getFunctionCallbacks(functionDetail).includes(callback)
}

export function visibleInScene(field: FieldConfig, scene: FieldScene): boolean {
  if (scene === 'list' && isContainerWidget(field)) return false
  const hiddenScenes = field.hide?.scenes
  if (!Array.isArray(hiddenScenes)) return true
  return !hiddenScenes.includes(scene)
}

export function getFormRequestFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  return getFunctionSchema(functionDetail)?.form?.request || []
}

export function getFormResponseFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  return getFunctionSchema(functionDetail)?.form?.response || []
}

export function getChartRequestFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  return getFunctionSchema(functionDetail)?.chart?.request || []
}

export function getTableRequestFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  return getFunctionSchema(functionDetail)?.table?.request || []
}

export function getTableRawFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  return getFunctionSchema(functionDetail)?.table?.fields || []
}

export function getTableListFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  return getTableRawFields(functionDetail).filter((field) => visibleInScene(field, 'list'))
}

export function getTableDetailFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  return getTableListFields(functionDetail)
}

export function getTableCreateFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  return getTableRawFields(functionDetail)
    .filter((field) => visibleInScene(field, 'create'))
    .filter(isEditableTableField)
}

export function getTableUpdateFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  return getTableRawFields(functionDetail)
    .filter((field) => visibleInScene(field, 'update'))
    .filter(isEditableTableField)
}

export function getTableSearchFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  const fields = new Map<string, FieldConfig>()
  getTableRequestFields(functionDetail)
    .filter((field) => isRequestSearchEnabled(field))
    .forEach((field) => fields.set(field.code, field))
  return [...fields.values()]
}

export function getTableRequestSearchFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  return getTableRequestFields(functionDetail)
    .filter((field) => isRequestSearchEnabled(field))
}

export function getTableResponseSearchFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  return []
}

export function getTableIdField(functionDetail?: FunctionDetail | null): FieldConfig | null {
  return getTableListFields(functionDetail).find((field) =>
    field.widget?.type === WidgetType.ID || field.code === 'id' || field.code === '_id'
  ) || null
}

export function buildTableCreateFormDetail(functionDetail: FunctionDetail): FunctionDetail {
  return buildFormDetailFromFields(functionDetail, getTableCreateFields(functionDetail))
}

export function buildTableUpdateFormDetail(functionDetail: FunctionDetail): FunctionDetail {
  return buildFormDetailFromFields(functionDetail, getTableUpdateFields(functionDetail))
}

function buildFormDetailFromFields(functionDetail: FunctionDetail, fields: FieldConfig[]): FunctionDetail {
  return {
    ...functionDetail,
    id: 0,
    template_type: 'form',
    schema: {
      version: functionDetail.schema?.version || 1,
      type: 'form',
      callbacks: [],
      form: {
        request: fields,
        response: []
      }
    }
  }
}

function isRequestSearchEnabled(field: FieldConfig): boolean {
  return field.widget?.type !== WidgetType.TABLE && field.widget?.type !== WidgetType.FORM
}

function isContainerWidget(field: FieldConfig): boolean {
  const type = field.widget?.type?.toLowerCase()
  return type === WidgetType.TABLE || type === WidgetType.FORM
}

function isEditableTableField(field: FieldConfig): boolean {
  if (field.widget?.type === WidgetType.ID) return false
  if (field.widget?.config?.disabled === true) return false
  if (field.widget?.config?.readonly === true || field.widget?.config?.readOnly === true) return false
  return true
}
