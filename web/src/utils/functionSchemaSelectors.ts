import { WidgetType } from '@/core/constants/widget'
import type { DisplayScene, FieldConfig, FunctionDetail, FunctionSchema } from '@/types/field'

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

export function visibleInScene(field: FieldConfig, scene: DisplayScene): boolean {
  const scenes = field.display?.scenes
  if (!Array.isArray(scenes)) return true
  return scenes.includes(scene)
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
  getTableRawFields(functionDetail)
    .filter((field) => hasSearchConfig(field))
    .forEach((field) => fields.set(field.code, field))
  getTableRequestFields(functionDetail)
    .filter((field) => field.search !== '-')
    .forEach((field) => {
      if (!fields.has(field.code)) fields.set(field.code, field)
    })
  return [...fields.values()]
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

function hasSearchConfig(field: FieldConfig): boolean {
  const search = field.search?.trim()
  return !!search && search !== '-'
}

function isEditableTableField(field: FieldConfig): boolean {
  if (field.widget?.type === WidgetType.ID) return false
  if (field.widget?.config?.disabled === true) return false
  if (field.widget?.config?.readonly === true || field.widget?.config?.readOnly === true) return false
  return true
}
