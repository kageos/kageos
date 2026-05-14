import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'
import { createFieldValue } from '@/architecture/runtime/utils/createFieldValue'
import { buildContainerDisplayValue } from './containerValue'

function isSameRawValue(left: any, right: any): boolean {
  if (left === right) {
    return true
  }

  if (left === null || left === undefined || right === null || right === undefined) {
    return left === right
  }

  if (typeof left === 'object' || typeof right === 'object') {
    try {
      return JSON.stringify(left) === JSON.stringify(right)
    } catch {
      return false
    }
  }

  return String(left) === String(right)
}

function getDefaultDisplay(rawValue: any): string {
  if (rawValue === null || rawValue === undefined) {
    return ''
  }

  return typeof rawValue === 'object' ? JSON.stringify(rawValue) : String(rawValue)
}

export function createPersistedFieldValue(
  field: FieldConfig,
  rawValue: any,
  currentValue?: FieldValue | null
): FieldValue {
  const meta = { ...(currentValue?.meta || {}) }

  if (field.widget?.type === 'table' || field.widget?.type === 'form') {
    return createFieldValue(
      field,
      rawValue,
      buildContainerDisplayValue(field, rawValue),
      meta
    )
  }

  if (rawValue === null || rawValue === undefined) {
    return createFieldValue(field, null, '', meta)
  }

  const shouldReuseDisplay = !!currentValue?.display
    && isSameRawValue(currentValue.raw, rawValue)

  return createFieldValue(
    field,
    rawValue,
    shouldReuseDisplay ? currentValue.display : getDefaultDisplay(rawValue),
    meta
  )
}
