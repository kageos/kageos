import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'
import { createFieldValue } from '@/architecture/domain/utils/createFieldValue'

interface FieldTreeStore {
  getAllFieldPaths(): string[]
  deleteValue(fieldPath: string): void
}

export function isSubtreePath(rootPath: string, fieldPath: string): boolean {
  return fieldPath === rootPath
    || fieldPath.startsWith(`${rootPath}.`)
    || fieldPath.startsWith(`${rootPath}[`)
}

export function clearFieldSubtree(formDataStore: FieldTreeStore, rootPath: string): void {
  formDataStore.getAllFieldPaths().forEach((fieldPath) => {
    if (isSubtreePath(rootPath, fieldPath)) {
      formDataStore.deleteValue(fieldPath)
    }
  })
}

export function createClearedFieldValue(field: FieldConfig, meta?: Record<string, any>): FieldValue {
  if (field.widget?.type === 'table') {
    return createFieldValue(field, [], '共 0 条', meta)
  }

  if (field.widget?.type === 'form') {
    return createFieldValue(field, {}, '', meta)
  }

  return createFieldValue(field, null, '', meta)
}
