import type { FieldConfig } from '@/architecture/runtime/types/field'
import type { FormDataStore } from '@/architecture/runtime/stores-v2/formData'
import { getFieldPresenceState } from '@/architecture/runtime/utils/conditionEvaluator'
import { clearFieldSubtree, createClearedFieldValue } from './fieldReset'

interface PresenceEffectStore extends Pick<FormDataStore, 'getValue' | 'setValue' | 'deleteValue' | 'getAllFieldPaths' | 'data'> {}

export function applyScopedPresenceEffects(options: {
  fields: FieldConfig[]
  formDataStore: PresenceEffectStore
  scopePath?: string
  clearFieldErrors?: (fieldPath: string, clearOptions?: { includeSubtree?: boolean }) => void
}): void {
  const scopePath = options.scopePath || ''
  const formManager = {
    getValue: (fieldPath: string) => options.formDataStore.getValue(fieldPath),
    hasValue: (fieldPath: string) => options.formDataStore.data.has(fieldPath),
  }

  options.fields.forEach((field) => {
    const fieldPath = scopePath ? `${scopePath}.${field.code}` : field.code
    const state = getFieldPresenceState(field, formManager, options.fields, fieldPath)

    if (!state.visible) {
      options.clearFieldErrors?.(fieldPath, { includeSubtree: true })
    }

    if (!state.excluded) {
      return
    }

    // excluded_* 命中时要同时清掉根字段和子树，避免表单 store 中残留旧值。
    const currentValue = options.formDataStore.getValue(fieldPath)
    clearFieldSubtree(options.formDataStore, fieldPath)
    options.formDataStore.setValue(fieldPath, createClearedFieldValue(field, currentValue?.meta || {}))
    options.clearFieldErrors?.(fieldPath, { includeSubtree: true })
  })
}
