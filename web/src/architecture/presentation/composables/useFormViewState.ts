import { computed, type Ref } from 'vue'
import type { FormApplicationService } from '../../application/services/FormApplicationService'
import type { FormDomainService } from '../../domain/services/FormDomainService'
import type { FieldConfig, FieldValue, FunctionDetail } from '../../domain/types'
import type { FormStateManager } from '../../infrastructure/stateManager/FormStateManager'
import { hasAnyRequiredRule } from '@/core/utils/validationUtils'
import { createAutoFieldValue, createEmptyFieldValue, createEmptyRawFieldValue } from '@/core/utils/createFieldValue'
import { FORM_QUESTIONNAIRE_TRIGGER_CHARS } from '../utils/formLayout'

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

  const requestFields = computed(() => (options.functionDetail.value?.request || []) as FieldConfig[])
  const responseFields = computed(() => (options.functionDetail.value?.response || []) as FieldConfig[])

  const requestLabelsOnTop = computed(() =>
    requestFields.value.some((field) => (field.name?.length ?? 0) > FORM_QUESTIONNAIRE_TRIGGER_CHARS)
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
      getSubmitData: () => {
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
      },
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
    return hasAnyRequiredRule(field)
  }

  const handleFieldUpdate = (fieldCode: string, value: FieldValue): void => {
    options.applicationService.updateFieldValue(fieldCode, value)
  }

  return {
    formData,
    requestFields,
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
