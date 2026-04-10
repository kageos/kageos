import type { FieldConfig, FieldValue } from '@/core/types/field'
import type { FormDataStore } from '@/core/stores-v2/formData'
import { createFieldValue } from '@/core/utils/createFieldValue'

type ContainerStore = Pick<FormDataStore, 'data' | 'getValue' | 'setValue'>

function isPlainObject(value: unknown): value is Record<string, any> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function cloneMeta(meta?: Record<string, any>): Record<string, any> {
  return { ...(meta || {}) }
}

function getContainerMeta(
  currentValue: FieldValue | null | undefined,
  fallbackValue?: FieldValue | null
): Record<string, any> {
  if (currentValue?.meta && Object.keys(currentValue.meta).length > 0) {
    return cloneMeta(currentValue.meta)
  }

  return cloneMeta(fallbackValue?.meta)
}

export function buildContainerDisplayValue(field: FieldConfig, rawValue: any): string {
  if (rawValue === null || rawValue === undefined) {
    return ''
  }

  if (field.widget?.type === 'table' && Array.isArray(rawValue)) {
    return `共 ${rawValue.length} 条`
  }

  if (field.widget?.type === 'form' && isPlainObject(rawValue) && Object.keys(rawValue).length === 0) {
    return ''
  }

  if (typeof rawValue === 'object') {
    return JSON.stringify(rawValue)
  }

  return String(rawValue)
}

export function syncFormContainerValue(
  formDataStore: ContainerStore,
  field: FieldConfig,
  fieldPath: string,
  fallbackValue?: FieldValue | null
): FieldValue {
  const fallbackRaw = isPlainObject(fallbackValue?.raw) ? fallbackValue.raw : null
  const currentValue = formDataStore.getValue(fieldPath)
  const rawData: Record<string, any> = {}

  ;(field.children || []).forEach((subField) => {
    const subFieldPath = `${fieldPath}.${subField.code}`

    if (formDataStore.data.has(subFieldPath)) {
      rawData[subField.code] = formDataStore.getValue(subFieldPath).raw
      return
    }

    if (fallbackRaw && Object.prototype.hasOwnProperty.call(fallbackRaw, subField.code)) {
      rawData[subField.code] = fallbackRaw[subField.code]
      return
    }

    if (subField.widget?.type === 'form') {
      rawData[subField.code] = {}
      return
    }

    if (subField.widget?.type === 'table') {
      rawData[subField.code] = []
    }
  })

  const fieldValue = createFieldValue(
    field,
    rawData,
    buildContainerDisplayValue(field, rawData),
    getContainerMeta(currentValue, fallbackValue)
  )

  formDataStore.setValue(fieldPath, fieldValue)
  return fieldValue
}

export function syncTableContainerValue(
  formDataStore: ContainerStore,
  field: FieldConfig,
  fieldPath: string,
  rows: any[],
  fallbackValue?: FieldValue | null
): FieldValue {
  const currentValue = formDataStore.getValue(fieldPath)
  const normalizedRows = Array.isArray(rows) ? rows : []
  const fieldValue = createFieldValue(
    field,
    normalizedRows,
    buildContainerDisplayValue(field, normalizedRows),
    getContainerMeta(currentValue, fallbackValue)
  )

  formDataStore.setValue(fieldPath, fieldValue)
  return fieldValue
}
