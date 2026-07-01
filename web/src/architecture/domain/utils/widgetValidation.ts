import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'
import type { ValidationEngine, ValidationResult } from '@/architecture/domain/validation'
import { Logger } from '@/architecture/shared/logger'

export interface WidgetValueReader {
  getValue(fieldPath: string): FieldValue
}

export interface WidgetValidationContext {
  validationEngine: ValidationEngine | null
  allFields: FieldConfig[]
  fieldErrors: Map<string, ValidationResult[]>
  formDataStore: WidgetValueReader
}

export interface WidgetValidationResult {
  errors: ValidationResult[]
  nestedErrors: Map<string, ValidationResult[]>
  hasError: boolean
}

export function validateFieldValue(
  field: FieldConfig,
  fieldPath: string,
  context: WidgetValidationContext
): ValidationResult[] {
  if (!field.validation || !context.validationEngine) {
    return []
  }

  const value = context.formDataStore.getValue(fieldPath)

  try {
    return context.validationEngine.validateField(field, value, context.allFields, fieldPath)
  } catch (error) {
    Logger.error('[widgetValidation]', `验证字段 ${fieldPath} 失败`, error)
    return []
  }
}

export function validateWidget(
  field: FieldConfig,
  fieldPath: string,
  context: WidgetValidationContext
): WidgetValidationResult {
  const errors: ValidationResult[] = []
  const nestedErrors = new Map<string, ValidationResult[]>()

  const fieldErrors = validateFieldValue(field, fieldPath, context)
  if (fieldErrors.length > 0) {
    errors.push(...fieldErrors)
  }

  if (field.children && field.children.length > 0) {
    const nestedResult = validateNestedFields(field, fieldPath, context)
    nestedResult.forEach((nestedErrorsForPath, path) => {
      nestedErrors.set(path, nestedErrorsForPath)
    })
  }

  return {
    errors,
    nestedErrors,
    hasError: errors.length > 0 || nestedErrors.size > 0
  }
}

function validateNestedFields(
  _field: FieldConfig,
  _parentPath: string,
  _context: WidgetValidationContext
): Map<string, ValidationResult[]> {
  return new Map<string, ValidationResult[]>()
}

export function validateFormWidgetNestedFields(
  field: FieldConfig,
  parentPath: string,
  context: WidgetValidationContext
): Map<string, ValidationResult[]> {
  const nestedErrors = new Map<string, ValidationResult[]>()

  if (!field.children || field.children.length === 0) {
    return nestedErrors
  }

  field.children.forEach((subField: FieldConfig) => {
    const subFieldPath = `${parentPath}.${subField.code}`

    const subErrors = validateFieldValue(subField, subFieldPath, context)
    if (subErrors.length > 0) {
      nestedErrors.set(subFieldPath, subErrors)
    }

    if (subField.children && subField.children.length > 0) {
      if (subField.widget?.type === 'form') {
        const deeperErrors = validateFormWidgetNestedFields(subField, subFieldPath, context)
        deeperErrors.forEach((errors, path) => {
          nestedErrors.set(path, errors)
        })
      } else if (subField.widget?.type === 'table') {
        const deeperErrors = validateTableWidgetNestedFields(subField, subFieldPath, context)
        deeperErrors.forEach((errors, path) => {
          nestedErrors.set(path, errors)
        })
      } else {
        const deeperErrors = validateFormWidgetNestedFields(subField, subFieldPath, context)
        deeperErrors.forEach((errors, path) => {
          nestedErrors.set(path, errors)
        })
      }
    }
  })

  return nestedErrors
}

export function validateTableWidgetNestedFields(
  field: FieldConfig,
  parentPath: string,
  context: WidgetValidationContext
): Map<string, ValidationResult[]> {
  const nestedErrors = new Map<string, ValidationResult[]>()

  if (!field.children || field.children.length === 0) {
    return nestedErrors
  }

  const value = context.formDataStore.getValue(parentPath)
  const tableValue = value.raw

  if (!Array.isArray(tableValue)) {
    return nestedErrors
  }

  tableValue.forEach((_: any, index: number) => {
    field.children!.forEach((subField: FieldConfig) => {
      const subFieldPath = `${parentPath}[${index}].${subField.code}`

      const subErrors = validateFieldValue(subField, subFieldPath, context)
      if (subErrors.length > 0) {
        nestedErrors.set(subFieldPath, subErrors)
      }

      if (subField.children && subField.children.length > 0) {
        if (subField.widget?.type === 'form') {
          const deeperErrors = validateFormWidgetNestedFields(subField, subFieldPath, context)
          deeperErrors.forEach((errors, path) => {
            nestedErrors.set(path, errors)
          })
        } else if (subField.widget?.type === 'table') {
          const deeperErrors = validateTableWidgetNestedFields(subField, subFieldPath, context)
          deeperErrors.forEach((errors, path) => {
            nestedErrors.set(path, errors)
          })
        }
      }
    })
  })

  return nestedErrors
}
