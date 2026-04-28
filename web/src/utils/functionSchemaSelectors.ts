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
  if (scene === 'list' && isContainerWidget(field)) return false
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
  // table.request 字段是 sdk-app 入参，URL 保持原始 key（`genre=诗`）。
  // table.fields 字段是结果集搜索，URL 走后端操作符（`in=style:律诗`）。
  // 如果 code 重复，request 字段优先，避免同一个字段同时走两套协议。
  getTableRequestFields(functionDetail)
    .filter((field) => isRequestSearchEnabled(field))
    .forEach((field) => fields.set(field.code, field))
  getTableRawFields(functionDetail)
    .filter((field) => hasSearchConfig(field))
    .forEach((field) => {
      if (!fields.has(field.code)) fields.set(field.code, field)
    })
  return [...fields.values()]
}

export function getTableRequestSearchFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  return getTableRequestFields(functionDetail)
    .filter((field) => isRequestSearchEnabled(field))
}

export function getTableResponseSearchFields(functionDetail?: FunctionDetail | null): FieldConfig[] {
  const requestSearchFieldCodes = new Set(
    getTableRequestSearchFields(functionDetail).map((field) => field.code)
  )

  // 排除 request 搜索字段：这些参数归 sdk-app 入参所有，必须保持原始 key，
  // 不能再包进 `eq`/`in` 等操作符参数。
  return getTableRawFields(functionDetail)
    .filter((field) => hasSearchConfig(field))
    .filter((field) => !requestSearchFieldCodes.has(field.code))
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

function isRequestSearchEnabled(field: FieldConfig): boolean {
  return field.search?.trim() !== '-'
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
