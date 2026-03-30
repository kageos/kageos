import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import type { WidgetInitContext } from '@/architecture/presentation/widgets/interfaces/IWidgetInitializer'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import { convertBasicType } from '@/architecture/presentation/widgets/utils/typeConverter'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { widgetInitializerRegistry } from './WidgetInitializerRegistry'

function isPlainObject(value: unknown): value is Record<string, any> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function cloneMeta(meta?: Record<string, any>): Record<string, any> {
  return { ...(meta || {}) }
}

function buildDisplayValue(field: FieldConfig, rawValue: any): string {
  if (rawValue === null || rawValue === undefined) {
    return ''
  }

  if (field.widget?.type === 'table' && Array.isArray(rawValue)) {
    return `共 ${rawValue.length} 条`
  }

  if (typeof rawValue === 'object') {
    return JSON.stringify(rawValue)
  }

  return String(rawValue)
}

function normalizeRawValue(field: FieldConfig, rawValue: any): any {
  if (field.widget?.type === 'form' && isPlainObject(rawValue)) {
    return rawValue
  }

  if (field.widget?.type === 'table' && Array.isArray(rawValue)) {
    return rawValue
  }

  return convertBasicType(rawValue, field.data?.type || 'string')
}

function createNestedFieldValue(field: FieldConfig, rawValue: any, meta?: Record<string, any>): FieldValue {
  const normalizedRawValue = normalizeRawValue(field, rawValue)
  return createFieldValue(
    field,
    normalizedRawValue,
    buildDisplayValue(field, normalizedRawValue),
    cloneMeta(meta)
  )
}

async function initializeChildField(
  parentContext: WidgetInitContext,
  childField: FieldConfig,
  childFieldPath: string,
  rawValue: any
): Promise<FieldValue> {
  const formDataStore = parentContext.formDataStore || useFormDataStore()

  const childContext: WidgetInitContext = {
    ...parentContext,
    field: childField,
    currentValue: createNestedFieldValue(childField, rawValue, parentContext.currentValue.meta),
    allFormData: formDataStore.getAllValues(),
    formDataStore,
    fieldPath: childFieldPath
  }

  let initializedValue: FieldValue

  // 容器字段的递归 hydrate 不能依赖外部 registry 注册时机，
  // 否则 form -> form -> table 这类深层结构会在某些入口下直接断掉。
  if (childField.widget?.type === 'form') {
    initializedValue = (await hydrateFormField(childContext)) || childContext.currentValue
  } else if (childField.widget?.type === 'table') {
    initializedValue = (await hydrateTableField(childContext)) || childContext.currentValue
  } else {
    initializedValue = await widgetInitializerRegistry.initialize(childContext)
  }

  formDataStore.setValue(childFieldPath, initializedValue)
  return initializedValue
}

export async function hydrateFormField(context: WidgetInitContext): Promise<FieldValue | null> {
  const { field, currentValue } = context
  if (!isPlainObject(currentValue.raw)) {
    return null
  }

  const subFields = field.children || []
  if (subFields.length === 0) {
    return null
  }

  const basePath = context.fieldPath || field.code
  const initializedFormData: Record<string, any> = {}

  for (const subField of subFields) {
    const subFieldPath = `${basePath}.${subField.code}`
    const initializedValue = await initializeChildField(
      context,
      subField,
      subFieldPath,
      currentValue.raw[subField.code]
    )
    initializedFormData[subField.code] = initializedValue.raw
  }

  return createFieldValue(
    field,
    initializedFormData,
    buildDisplayValue(field, initializedFormData),
    cloneMeta(currentValue.meta)
  )
}

export async function hydrateTableField(context: WidgetInitContext): Promise<FieldValue | null> {
  const { field, currentValue } = context
  if (!Array.isArray(currentValue.raw)) {
    return null
  }

  const itemFields = field.children || []
  if (itemFields.length === 0) {
    return null
  }

  const validRows = currentValue.raw.filter((row: any) => {
    if (!isPlainObject(row)) {
      return false
    }

    return Object.values(row).some((value: any) => value !== null && value !== undefined && value !== '')
  })

  const basePath = context.fieldPath || field.code
  const initializedRows: Record<string, any>[] = []

  for (const [rowIndex, row] of validRows.entries()) {
    const rowData: Record<string, any> = {}

    for (const itemField of itemFields) {
      const itemFieldPath = `${basePath}[${rowIndex}].${itemField.code}`
      const initializedValue = await initializeChildField(
        context,
        itemField,
        itemFieldPath,
        row[itemField.code]
      )
      rowData[itemField.code] = initializedValue.raw
    }

    initializedRows.push(rowData)
  }

  return createFieldValue(
    field,
    initializedRows,
    buildDisplayValue(field, initializedRows),
    cloneMeta(currentValue.meta)
  )
}
