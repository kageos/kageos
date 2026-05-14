import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'
import { clearFieldSubtree, createClearedFieldValue } from './fieldReset'

interface DependencyStore {
  getValue(fieldPath: string): FieldValue
  setValue(fieldPath: string, value: FieldValue): void
  deleteValue(fieldPath: string): void
  getAllFieldPaths(): string[]
}

interface ClearScopedDependentFieldsOptions {
  formDataStore: DependencyStore
  fields: FieldConfig[]
  changedFieldCode: string
  scopePath?: string
}

function resolveScopedFieldPath(scopePath: string | undefined, fieldCode: string): string {
  return scopePath ? `${scopePath}.${fieldCode}` : fieldCode
}

function clearDependentField(
  formDataStore: DependencyStore,
  field: FieldConfig,
  fieldPath: string
): void {
  const existingValue = formDataStore.getValue(fieldPath)
  clearFieldSubtree(formDataStore, fieldPath)
  formDataStore.setValue(fieldPath, createClearedFieldValue(field, existingValue?.meta || {}))
}

export function clearScopedDependentFields(options: ClearScopedDependentFieldsOptions): string[] {
  const clearedPaths: string[] = []
  const clearedFieldCodes = new Set<string>()
  const queue = [options.changedFieldCode]

  while (queue.length > 0) {
    const currentFieldCode = queue.shift()
    if (!currentFieldCode) {
      continue
    }

    options.fields.forEach((field) => {
      if (field.depend_on !== currentFieldCode || clearedFieldCodes.has(field.code)) {
        return
      }

      const dependentPath = resolveScopedFieldPath(options.scopePath, field.code)
      clearDependentField(options.formDataStore, field, dependentPath)
      clearedPaths.push(dependentPath)
      clearedFieldCodes.add(field.code)
      queue.push(field.code)
    })
  }

  return clearedPaths
}
