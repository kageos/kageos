import type { FieldValue } from '@/architecture/domain/types/field'
import type { FormDataStore } from '@/architecture/infrastructure/stores/formData'

export interface FieldTreeSnapshot {
  rootPath: string
  values: Record<string, FieldValue>
}

function isSubtreePath(rootPath: string, fieldPath: string): boolean {
  return fieldPath === rootPath
    || fieldPath.startsWith(`${rootPath}.`)
    || fieldPath.startsWith(`${rootPath}[`)
}

function cloneFieldValue<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

export function captureFieldTreeSnapshot(
  formDataStore: Pick<FormDataStore, 'getAllFieldPaths' | 'getAllValues'>,
  rootPath: string
): FieldTreeSnapshot {
  const allValues = formDataStore.getAllValues()
  const values: Record<string, FieldValue> = {}

  formDataStore.getAllFieldPaths().forEach((fieldPath) => {
    if (!isSubtreePath(rootPath, fieldPath)) {
      return
    }
    const fieldValue = allValues[fieldPath]
    if (fieldValue) {
      values[fieldPath] = cloneFieldValue(fieldValue)
    }
  })

  return {
    rootPath,
    values,
  }
}

export function restoreFieldTreeSnapshot(
  formDataStore: Pick<FormDataStore, 'getAllFieldPaths' | 'deleteValue' | 'setValue'>,
  snapshot: FieldTreeSnapshot | null
): void {
  if (!snapshot) {
    return
  }

  formDataStore.getAllFieldPaths().forEach((fieldPath) => {
    if (isSubtreePath(snapshot.rootPath, fieldPath) && !(fieldPath in snapshot.values)) {
      formDataStore.deleteValue(fieldPath)
    }
  })

  Object.entries(snapshot.values).forEach(([fieldPath, fieldValue]) => {
    formDataStore.setValue(fieldPath, cloneFieldValue(fieldValue))
  })
}
