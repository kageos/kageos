import type { FieldConfig } from '@/architecture/domain/types'
import type { FunctionDetail } from '@/architecture/domain/types'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { deleteScopedFieldQueryKey } from '@/utils/queryFieldNamespace'
import {
  buildTableUpdateFormDetail,
  getFormRequestFields,
  getTableIdField
} from '@/utils/functionSchemaSelectors'

export type DetailEditFormReadiness =
  | 'missing-edit-detail'
  | 'missing-row-data'
  | 'no-editable-fields'
  | 'missing-edit-values'
  | 'ready'

export interface DetailEditFormState {
  readiness: DetailEditFormReadiness
  editableFieldCodes: string[]
  initialData: Record<string, any>
}

export interface DetailRouteRequest {
  rowId: string
  key: string
}

export interface DetailRowMatch<T extends Record<string, any> = Record<string, any>> {
  row: T
  index: number
}

export type DetailRestoreTrigger = 'setup' | 'route-change' | 'function-loaded' | 'table-data-loaded'

export interface DetailLookupSearchRequest {
  url: string
  params: Record<string, any>
}

export function buildEditFunctionDetail(current: FunctionDetail | null): FunctionDetail | null {
  if (!current) return null

  if (current.template_type === TEMPLATE_TYPE.TABLE) {
    return buildTableUpdateFormDetail(current)
  }

  if (current.template_type === TEMPLATE_TYPE.FORM) {
    return current
  }

  return null
}

export function getEditableFieldCodes(editFunctionDetail: FunctionDetail | null): string[] {
  return getFormRequestFields(editFunctionDetail).map((field: FieldConfig) => field.code)
}

export function filterDetailInitialData(options: {
  rowData: Record<string, any> | null
  editFunctionDetail: FunctionDetail | null
}): Record<string, any> {
  const { rowData, editFunctionDetail } = options
  const requestFields = getFormRequestFields(editFunctionDetail)
  if (!rowData || requestFields.length === 0) {
    return {}
  }

  const editableFieldCodes = new Set(
    requestFields.map((field: FieldConfig) => field.code)
  )

  const filtered: Record<string, any> = {}
  Object.keys(rowData).forEach(key => {
    if (editableFieldCodes.has(key)) {
      filtered[key] = rowData[key]
    }
  })

  return filtered
}

export function buildDetailEditFormState(options: {
  rowData: Record<string, any> | null
  editFunctionDetail: FunctionDetail | null
}): DetailEditFormState {
  const editableFieldCodes = getEditableFieldCodes(options.editFunctionDetail)

  if (!options.editFunctionDetail) {
    return {
      readiness: 'missing-edit-detail',
      editableFieldCodes,
      initialData: {}
    }
  }

  if (!options.rowData) {
    return {
      readiness: 'missing-row-data',
      editableFieldCodes,
      initialData: {}
    }
  }

  if (editableFieldCodes.length === 0) {
    return {
      readiness: 'no-editable-fields',
      editableFieldCodes,
      initialData: {}
    }
  }

  const initialData = filterDetailInitialData(options)

  if (Object.keys(initialData).length === 0) {
    return {
      readiness: 'missing-edit-values',
      editableFieldCodes,
      initialData
    }
  }

  return {
    readiness: 'ready',
    editableFieldCodes,
    initialData
  }
}

function normalizeQueryValue(value: unknown): string | null {
  if (Array.isArray(value)) {
    return normalizeQueryValue(value[0])
  }

  if (value === undefined || value === null || value === '') {
    return null
  }

  return String(value)
}

export function resolveDetailRouteRequest(query: Record<string, any>): DetailRouteRequest | null {
  const tab = normalizeQueryValue(query._tab)
  const rowId = normalizeQueryValue(query._id)

  if (tab !== 'detail' || !rowId) {
    return null
  }

  return {
    rowId,
    key: `detail:${rowId}`
  }
}

export function findDetailRowMatch<T extends Record<string, any>>(
  tableData: T[] | null | undefined,
  rowId: string
): DetailRowMatch<T> | null {
  if (!Array.isArray(tableData) || tableData.length === 0) {
    return null
  }

  const index = tableData.findIndex(row => {
    const candidateId = row.id ?? row._id
    return candidateId !== undefined && candidateId !== null && String(candidateId) === rowId
  })

  if (index < 0) {
    return null
  }

  const row = tableData[index]
  if (!row) {
    return null
  }

  return {
    row,
    index
  }
}

export function shouldWaitForDetailTableData(options: {
  loading: boolean
  dataLength: number
  trigger: DetailRestoreTrigger
}): boolean {
  if (options.loading) {
    return true
  }

  return options.dataLength === 0 && (
    options.trigger === 'setup' ||
    options.trigger === 'function-loaded'
  )
}

export function findDetailIdField(detail: FunctionDetail | null): FieldConfig | null {
  return getTableIdField(detail)
}

export function buildDetailLookupSearchRequest(options: {
  detail: FunctionDetail
  idFieldCode: string
  rowId: string
}): DetailLookupSearchRequest {
  const fullCodePath = options.detail.router?.startsWith('/')
    ? options.detail.router
    : `/${options.detail.router || ''}`

  return {
    url: `/workspace/api/v1/table/search${fullCodePath}`,
    params: {
      [options.idFieldCode]: options.rowId,
      page: 1,
      page_size: 20
    }
  }
}

export function buildDetailBaseQuery(options: {
  query: Record<string, any>
  editableFieldCodes: string[]
}): Record<string, string | string[]> {
  const nextQuery: Record<string, string | string[]> = {}

  Object.keys(options.query).forEach(key => {
    if (key === '_tab' || key === '_id') {
      return
    }

    const value = options.query[key]
    if (value !== null && value !== undefined) {
      nextQuery[key] = Array.isArray(value)
        ? value.filter(v => v !== null).map(v => String(v))
        : String(value)
    }
  })

  options.editableFieldCodes.forEach(fieldCode => {
    deleteScopedFieldQueryKey(nextQuery, fieldCode, 'form')
  })

  return nextQuery
}
