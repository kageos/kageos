import { computed, ref, type ComputedRef } from 'vue'
import { useRoute } from 'vue-router'
import type { FieldConfig, FieldValue, FunctionDetail } from '@/architecture/domain/types'
import { convertToFieldValue } from '@/architecture/domain/utils/field'
import { hasAnyRequiredRule } from '@/architecture/domain/utils/validationUtils'
import { useChartParamURLSync } from './useChartParamURLSync'
import { convertValueByFieldType } from '@/architecture/presentation/widgets/utils/typeConverter'
import { getWidgetDefaultValue } from '@/architecture/presentation/widgets/composables/useWidgetDefaultValue'
import { useAuthStore } from '@/architecture/infrastructure/stores/auth'
import { createEmptyFieldValue, createEmptyRawFieldValue } from '@/architecture/domain/utils/createFieldValue'
import { getChartRequestFields } from '@/architecture/domain/utils/functionSchemaSelectors'

interface UseChartFilterStateOptions {
  functionDetail: ComputedRef<FunctionDetail>
  onAutoSearch: () => void
}

export function useChartFilterState(options: UseChartFilterStateOptions) {
  const route = useRoute()

  const requestFields = computed(() => {
    return getChartRequestFields(options.functionDetail.value)
      .filter((field) => field.widget && field.widget.type)
      .map((field) => {
        if (field.widget && (field.widget.type === 'select' || field.widget.type === 'multiselect')) {
          return {
            ...field,
            widget: {
              ...field.widget,
              clearable: true
            }
          }
        }
        return field
      })
  })

  const filterForm = ref<Record<string, any>>({})
  const fieldValues = ref<Record<string, FieldValue>>({})

  const formRendererContext = computed(() => {
    return {
      getFunctionMethod: () => options.functionDetail.value.method || 'GET',
      getFunctionRouter: () => options.functionDetail.value.router || '',
      getSubmitData: () => {
        const data: Record<string, any> = {}
        Object.keys(fieldValues.value).forEach((key) => {
          const value = fieldValues.value[key]
          if (value && value.raw !== null && value.raw !== undefined) {
            data[key] = value.raw
          }
        })
        return data
      },
      registerWidget: () => {},
      unregisterWidget: () => {},
      getFieldError: () => null
    }
  })

  const { watchChartData } = useChartParamURLSync({
    functionDetail: options.functionDetail,
    fieldValues,
    enabled: true,
    debounceMs: 300
  })

  const initializeFieldValues = () => {
    const values: Record<string, FieldValue> = {}
    requestFields.value.forEach((field: FieldConfig) => {
      // Chart 筛选是 sdk-app request 参数，URL 回填只读原始 field.code。
      // 前端/平台状态必须放在 `_` key 下。
      const queryValue = route.query[field.code]
      const value = Array.isArray(queryValue) ? queryValue[0] : queryValue

      if (value !== undefined && value !== null && value !== '') {
        const rawValue = convertValueByFieldType(value, field)
        values[field.code] = convertToFieldValue(rawValue, field)
        filterForm.value[field.code] = rawValue
      } else {
        const defaultValue = getWidgetDefaultValue(field, undefined, () => useAuthStore())
        if (defaultValue.raw !== null && defaultValue.raw !== undefined && defaultValue.raw !== '') {
          values[field.code] = defaultValue
          filterForm.value[field.code] = defaultValue.raw
        } else {
          values[field.code] = createEmptyFieldValue(field)
          filterForm.value[field.code] = null
        }
      }
    })
    fieldValues.value = values
  }

  const getFieldValue = (fieldCode: string): FieldValue => {
    return fieldValues.value[fieldCode] || createEmptyRawFieldValue()
  }

  const getFieldRawValue = (fieldCode: string): any => {
    return getFieldValue(fieldCode).raw ?? null
  }

  const shouldUseChartSearchInput = (field: FieldConfig): boolean => {
    return !field.callbacks?.includes('OnSelectFuzzy')
  }

  const isFieldRequired = (field: FieldConfig): boolean => {
    return hasAnyRequiredRule(field)
  }

  const handleFieldUpdate = (fieldCode: string, value: FieldValue): void => {
    const oldValue = fieldValues.value[fieldCode]
    const oldRaw = oldValue?.raw ?? null
    const newRaw = value?.raw ?? null

    fieldValues.value[fieldCode] = value
    filterForm.value[fieldCode] = value.raw

    const oldIsEmpty = oldRaw == null || oldRaw === ''
    const newIsEmpty = newRaw == null || newRaw === ''
    const valueChanged = oldRaw !== newRaw && (oldIsEmpty !== newIsEmpty || (!oldIsEmpty && !newIsEmpty))

    if (valueChanged) {
      options.onAutoSearch()
    }
  }

  const handleSearchFieldUpdate = (field: FieldConfig, rawValue: any): void => {
    handleFieldUpdate(field.code, convertToFieldValue(rawValue, field))
  }

  const buildRequestParams = (): Record<string, any> => {
    const params: Record<string, any> = {}
    Object.keys(fieldValues.value).forEach((key) => {
      const value = fieldValues.value[key]
      if (value && value.raw !== null && value.raw !== undefined) {
        // 提交给 sdk-app 的数据 key 必须和 schema field.code 对齐，
        // 不要使用平台侧别名。
        params[key] = value.raw
      }
    })
    return params
  }

  const resetFilterValues = (): void => {
    requestFields.value.forEach((field: FieldConfig) => {
      fieldValues.value[field.code] = createEmptyFieldValue(field)
      filterForm.value[field.code] = null
    })
  }

  return {
    requestFields,
    filterForm,
    fieldValues,
    formRendererContext,
    initializeFieldValues,
    watchChartData,
    getFieldValue,
    getFieldRawValue,
    shouldUseChartSearchInput,
    isFieldRequired,
    handleFieldUpdate,
    handleSearchFieldUpdate,
    buildRequestParams,
    resetFilterValues,
  }
}
