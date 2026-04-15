import type { FieldConfig, FieldValue } from '@/core/types/field'
import type { FormDataStore } from '@/core/stores-v2/formData'
import { createAutoFieldValue, createEmptyRawFieldValue } from '@/core/utils/createFieldValue'
import { getFieldPresenceState } from '@/core/utils/conditionEvaluator'
import {
  clearFieldSubtree as clearFieldSubtreeInStore,
  createClearedFieldValue as createClearedStoreFieldValue,
} from '@/core/widgetRuntime/fieldReset'

function isPlainObject(value: unknown): value is Record<string, any> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function hasMeaningfulFieldValue(value: FieldValue | null | undefined): boolean {
  if (!value) {
    return false
  }

  return (value.raw !== null && value.raw !== undefined)
    || value.display !== ''
    || Object.keys(value.meta || {}).length > 0
}

function readObjectPath(source: Record<string, any>, path: string): any {
  if (!path) {
    return source
  }

  const segments = path.match(/([^[.\]]+)|\[(\d+)\]/g) || []
  let current: any = source

  for (const segment of segments) {
    if (current === null || current === undefined) {
      return undefined
    }

    if (segment.startsWith('[')) {
      const index = Number(segment.slice(1, -1))
      current = Array.isArray(current) ? current[index] : undefined
    } else {
      current = current[segment]
    }
  }

  return current
}

function resolveRowScopedPath(tablePath: string, rowIndex: number, fieldCodeOrPath: string): string {
  const rowPrefix = `${tablePath}[${rowIndex}]`

  if (!fieldCodeOrPath) {
    return rowPrefix
  }

  if (fieldCodeOrPath === rowPrefix) {
    return fieldCodeOrPath
  }

  if (fieldCodeOrPath.startsWith(`${rowPrefix}.`) || fieldCodeOrPath.startsWith(`${rowPrefix}[`)) {
    return fieldCodeOrPath
  }

  return `${rowPrefix}.${fieldCodeOrPath}`
}

function resolveRowRelativePath(tablePath: string, rowIndex: number, fieldCodeOrPath: string): string {
  const rowPrefix = `${tablePath}[${rowIndex}]`

  if (!fieldCodeOrPath) {
    return fieldCodeOrPath
  }

  if (fieldCodeOrPath.startsWith(`${rowPrefix}.`)) {
    return fieldCodeOrPath.slice(rowPrefix.length + 1)
  }

  if (fieldCodeOrPath.startsWith(`${rowPrefix}[`)) {
    return fieldCodeOrPath.slice(rowPrefix.length)
  }

  return fieldCodeOrPath
}

export function getTableRowScopedFieldValue(
  formDataStore: Pick<FormDataStore, 'getValue' | 'data'>,
  tablePath: string,
  rowIndex: number,
  rowData: Record<string, any> | null | undefined,
  fieldCodeOrPath: string
): FieldValue {
  const scopedPath = resolveRowScopedPath(tablePath, rowIndex, fieldCodeOrPath)
  const storeValue = formDataStore.getValue(scopedPath)
  const hasStoredValue = formDataStore.data.has(scopedPath)

  if (hasStoredValue || hasMeaningfulFieldValue(storeValue)) {
    return storeValue
  }

  if (rowData && isPlainObject(rowData)) {
    const rawValue = readObjectPath(rowData, resolveRowRelativePath(tablePath, rowIndex, fieldCodeOrPath))
    if (rawValue !== undefined) {
      return createAutoFieldValue(rawValue)
    }
  }

  return storeValue || createEmptyRawFieldValue()
}

export function shouldShowTableRowField(
  formDataStore: Pick<FormDataStore, 'getValue' | 'data'>,
  tablePath: string,
  rowIndex: number,
  rowData: Record<string, any> | null | undefined,
  field: FieldConfig,
  allFields: FieldConfig[]
): boolean {
  const scopedFormManager = {
    getValue: (fieldCodeOrPath: string) =>
      getTableRowScopedFieldValue(formDataStore, tablePath, rowIndex, rowData, fieldCodeOrPath),
    hasValue: (fieldCodeOrPath: string) =>
      formDataStore.data.has(resolveRowScopedPath(tablePath, rowIndex, fieldCodeOrPath)),
  }

  return getFieldPresenceState(
    field,
    scopedFormManager as any,
    allFields,
    `${tablePath}[${rowIndex}].${field.code}`
  ).visible
}

export function getTableRowFieldPresenceState(
  formDataStore: Pick<FormDataStore, 'getValue' | 'data'>,
  tablePath: string,
  rowIndex: number,
  rowData: Record<string, any> | null | undefined,
  field: FieldConfig,
  allFields: FieldConfig[]
) {
  const scopedFormManager = {
    getValue: (fieldCodeOrPath: string) =>
      getTableRowScopedFieldValue(formDataStore, tablePath, rowIndex, rowData, fieldCodeOrPath),
    hasValue: (fieldCodeOrPath: string) =>
      formDataStore.data.has(resolveRowScopedPath(tablePath, rowIndex, fieldCodeOrPath)),
  }

  return getFieldPresenceState(
    field,
    scopedFormManager as any,
    allFields,
    `${tablePath}[${rowIndex}].${field.code}`
  )
}

export function clearFieldSubtree(
  formDataStore: Pick<FormDataStore, 'getAllFieldPaths' | 'deleteValue'>,
  rootPath: string
): void {
  clearFieldSubtreeInStore(formDataStore, rootPath)
}

export function createClearedFieldValue(field: FieldConfig, meta?: Record<string, any>): FieldValue {
  return createClearedStoreFieldValue(field, meta)
}
