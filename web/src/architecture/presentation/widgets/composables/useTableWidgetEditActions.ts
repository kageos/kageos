import { computed } from 'vue'
import type { WidgetComponentProps } from '@/architecture/presentation/widgets/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { useFormDataStore } from '@/architecture/runtime/stores/formData'
import type { ValidationEngine, ValidationResult } from '@/architecture/runtime/validation'
import {
  validateFieldValue as validateWidgetFieldValue,
  validateTableWidgetNestedFields,
  type WidgetValidationContext
} from '@/architecture/presentation/widgets/composables/useWidgetValidation'
import { Logger } from '@/architecture/runtime/utils/logger'
import { createPersistedFieldValue } from '@/architecture/runtime/widgetRuntime/persistedFieldValue'
import {
  clearFieldSubtree,
  createClearedFieldValue,
} from '@/architecture/presentation/widgets/utils/tableRowVisibility'

interface EditModeLike {
  editingIndex: { value: number | null | undefined }
  saveRow: (rowData: Record<string, any>) => void
  deleteRow: (index: number) => void
}

interface UseTableWidgetEditActionsOptions {
  tableData: { value: any[] }
  itemFields: { value: FieldConfig[] }
  editMode: EditModeLike
  getRowFieldValue: (index: number, fieldCode: string) => FieldValue
  updateRowFieldValue: (index: number, fieldCode: string, value: FieldValue) => void
  getEditRowFieldPresenceState: (rowIndex: number, field: FieldConfig) => { visible: boolean; excluded: boolean }
}

export function useTableWidgetEditActions(
  props: WidgetComponentProps,
  {
    tableData,
    itemFields,
    editMode,
    getRowFieldValue,
    updateRowFieldValue,
    getEditRowFieldPresenceState,
  }: UseTableWidgetEditActionsOptions
) {
  const formDataStore = useFormDataStore()

  const editingRowStatistics = computed(() => {
    let targetIndex = editMode.editingIndex.value

    if (targetIndex === null || targetIndex === undefined) {
      if (tableData.value.length > 0) {
        targetIndex = tableData.value.length - 1
      } else {
        return {}
      }
    }

    const rowStatistics: Record<string, string> = {}

    itemFields.value.forEach((itemField: any) => {
      const fieldPath = `${props.fieldPath}[${targetIndex}].${itemField.code}`
      const itemValue = formDataStore.getValue(fieldPath)

      if (itemValue?.meta?.statistics && typeof itemValue.meta.statistics === 'object') {
        Object.entries(itemValue.meta.statistics).forEach(([label, expression]) => {
          if (typeof expression === 'string') {
            rowStatistics[label] = expression
          }
        })
      }
    })

    return rowStatistics
  })

  function handleRowFieldModelUpdate(index: number, fieldCode: string, value: FieldValue): void {
    updateRowFieldValue(index, fieldCode, value)
  }

  function getEditRowClassName({ rowIndex }: { rowIndex: number }): string {
    return editMode.editingIndex.value === rowIndex ? 'is-editing-row' : ''
  }

  function handleSave(index: number): void {
    try {
      const rowData: Record<string, any> = {}

      itemFields.value.forEach((itemField) => {
        const fieldPath = `${props.fieldPath}[${index}].${itemField.code}`
        const currentValue = formDataStore.getValue(fieldPath)
        const presenceState = getEditRowFieldPresenceState(index, itemField)

        if (!presenceState.visible) {
          if (!presenceState.excluded) {
            const value = getRowFieldValue(index, itemField.code)
            const fieldValue: FieldValue = value || {
              raw: null,
              display: '',
              meta: {}
            }

            formDataStore.setValue(fieldPath, fieldValue)
            rowData[itemField.code] = fieldValue.raw ?? null
            return
          }

          clearFieldSubtree(formDataStore, fieldPath)
          const clearedFieldValue = createClearedFieldValue(itemField, currentValue?.meta || {})
          formDataStore.setValue(fieldPath, clearedFieldValue)
          rowData[itemField.code] = clearedFieldValue.raw
          return
        }

        const value = getRowFieldValue(index, itemField.code)
        const fieldValue: FieldValue = value || {
          raw: null,
          display: '',
          meta: {}
        }

        formDataStore.setValue(fieldPath, fieldValue)
        rowData[itemField.code] = fieldValue.raw ?? null
      })

      editMode.saveRow(rowData)

      const finalIndex = index

      itemFields.value.forEach((itemField) => {
        const fieldPath = `${props.fieldPath}[${finalIndex}].${itemField.code}`
        const rawValue = rowData[itemField.code]
        const currentValue = formDataStore.getValue(fieldPath)
        const fieldValue = createPersistedFieldValue(itemField, rawValue, currentValue)
        formDataStore.setValue(fieldPath, fieldValue)
      })
    } catch (error) {
      Logger.error('TableWidget', 'handleSave 错误', error)
      throw error
    }
  }

  function handleDelete(index: number): void {
    editMode.deleteRow(index)
  }

  function validate(
    validationEngine: ValidationEngine | null,
    allFields: FieldConfig[],
    fieldErrors: Map<string, ValidationResult[]>
  ): ValidationResult[] {
    const context: WidgetValidationContext = {
      validationEngine,
      allFields,
      fieldErrors,
      formDataStore
    }

    const currentFieldErrors = validateWidgetFieldValue(props.field, props.fieldPath, context)
    updateFieldErrors(props.fieldPath, currentFieldErrors, fieldErrors)

    const nestedErrors = validateTableWidgetNestedFields(props.field, props.fieldPath, context)
    nestedErrors.forEach((errors, path) => {
      updateFieldErrors(path, errors, fieldErrors)
    })

    return currentFieldErrors
  }

  function updateFieldErrors(
    fieldPath: string,
    errors: ValidationResult[],
    fieldErrors: Map<string, ValidationResult[]>
  ): void {
    if (errors.length > 0) {
      fieldErrors.set(fieldPath, errors)
    } else {
      fieldErrors.delete(fieldPath)
    }
  }

  return {
    editingRowStatistics,
    handleRowFieldModelUpdate,
    getEditRowClassName,
    handleSave,
    handleDelete,
    validate
  }
}
