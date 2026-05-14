import { computed, type Ref } from 'vue'
import type { FormApplicationService } from '../../application/services/FormApplicationService'
import type { FormDomainService } from '../../domain/services/FormDomainService'
import type { FieldConfig, FieldValue, FunctionDetail } from '../../domain/types'
import type { FormStateManager } from '../../infrastructure/stateManager/FormStateManager'
import { getFieldPresenceState } from '@/architecture/runtime/utils/conditionEvaluator'
import { createAutoFieldValue, createEmptyFieldValue, createEmptyRawFieldValue } from '@/architecture/runtime/utils/createFieldValue'
import { FORM_QUESTIONNAIRE_TRIGGER_CHARS } from '../utils/formLayout'
import { getFormRequestFields, getFormResponseFields } from '@/architecture/domain/utils/functionSchemaSelectors'

interface UseFormViewStateOptions {
  functionDetail: Ref<FunctionDetail | null>
  stateManager: FormStateManager
  domainService: FormDomainService
  applicationService: FormApplicationService
}

export function useFormViewState(options: UseFormViewStateOptions) {
  const formData = computed(() => {
    const state = options.stateManager.getState()
    const data: Record<string, any> = {}
    if (state.data) {
      state.data.forEach((value: FieldValue, key: string) => {
        if (value) {
          data[key] = value.raw
        }
      })
    }
    return data
  })

  const requestFields = computed(() => getFormRequestFields(options.functionDetail.value) as FieldConfig[])
  const responseFields = computed(() => getFormResponseFields(options.functionDetail.value) as FieldConfig[])
  const formManager = computed(() => {
    const stateManager = options.stateManager as any
    const formStore = stateManager?.formStore

    return {
      getValue: (fieldPath: string) => options.stateManager.getValue(fieldPath),
      hasValue: (fieldPath: string) => Boolean(formStore?.data?.has(fieldPath) || options.stateManager.getState().data?.has(fieldPath))
    }
  })

  const visibleRequestFields = computed(() =>
    requestFields.value.filter((field) =>
      getFieldPresenceState(field, formManager.value as any, requestFields.value, field.code).visible
    )
  )

  const requestLabelsOnTop = computed(() =>
    visibleRequestFields.value.some((field) => (field.name?.length ?? 0) > FORM_QUESTIONNAIRE_TRIGGER_CHARS)
  )

  const responseLabelsOnTop = computed(() =>
    responseFields.value.some((field) => (field.name?.length ?? 0) > FORM_QUESTIONNAIRE_TRIGGER_CHARS)
  )

  const fieldValues = computed(() => {
    const values: Record<string, FieldValue> = {}
    options.stateManager.getState().data?.forEach((value, key) => {
      if (requestFields.value.some((field: FieldConfig) => field.code === key)) {
        values[key] = value
      }
    })
    requestFields.value.forEach((field: FieldConfig) => {
      if (!values[field.code]) {
        values[field.code] = createEmptyFieldValue(field)
      }
    })
    return values
  })

  const responseFieldValues = computed(() => {
    const state = options.stateManager.getState()
    const values: Record<string, FieldValue> = {}
    responseFields.value.forEach((field: FieldConfig) => {
      const rawValue = state.response?.[field.code]
      values[field.code] = createAutoFieldValue(rawValue, field)
    })
    return values
  })

  const hasResponseData = computed(() => {
    const state = options.stateManager.getState()
    return state.response !== null && state.response !== undefined
  })

  const responseMetadata = computed(() => {
    return options.stateManager.getState().metadata || null
  })

  const submitting = computed(() => {
    return options.stateManager.getState().submitting
  })

  const formRendererContext = computed(() => {
    return {
      getFunctionMethod: () => options.functionDetail.value?.method || 'GET',
      getFunctionRouter: () => options.functionDetail.value?.router || '',
      getSubmitData: () => options.domainService.getSubmitData(requestFields.value),
      registerWidget: () => {},
      unregisterWidget: () => {},
      getFieldError: (fieldPath: string) => {
        const errors = options.domainService.getFieldError(fieldPath)
        return errors[0]?.message || null
      },
      clearFieldErrors: (fieldPath: string, clearOptions?: { includeSubtree?: boolean }) => {
        options.domainService.clearFieldErrors(fieldPath, clearOptions?.includeSubtree || false)
      },
    }
  })

  const getFieldValue = (fieldCode: string): FieldValue => {
    return fieldValues.value[fieldCode] || createEmptyRawFieldValue()
  }

  const getFieldError = (fieldCode: string): string => {
    const errors = options.domainService.getFieldError(fieldCode)
    return errors[0]?.message || ''
  }

  const getResponseFieldValue = (fieldCode: string): FieldValue => {
    return responseFieldValues.value[fieldCode] || createEmptyRawFieldValue()
  }

  const isFieldRequired = (field: FieldConfig): boolean => {
    return getFieldPresenceState(field, formManager.value as any, requestFields.value, field.code).required
  }

  const handleFieldUpdate = (fieldCode: string, value: FieldValue): void => {
    options.applicationService.updateFieldValue(fieldCode, value)
  }

  return {
    formData,
    requestFields,
    visibleRequestFields,
    responseFields,
    requestLabelsOnTop,
    responseLabelsOnTop,
    fieldValues,
    responseFieldValues,
    hasResponseData,
    responseMetadata,
    submitting,
    formRendererContext,
    getFieldValue,
    getFieldError,
    getResponseFieldValue,
    isFieldRequired,
    handleFieldUpdate,
  }
}
