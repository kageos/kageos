import { computed, nextTick, onMounted, ref, watch, type ComputedRef, type Ref } from 'vue'
import { normalizeSearchValue } from '@/utils/searchValueNormalizer'
import { SearchComponent, SearchConfig, SearchType } from '@/architecture/runtime/constants/search'
import { WidgetType } from '@/architecture/runtime/constants/widget'
import { parseCommaSeparatedString } from '@/utils/stringUtils'
import type { FieldConfig } from '@/architecture/domain/types'
import type { SearchInputConfig } from '../utils/searchInputTypes'

function debounce<T extends (...args: any[]) => any>(func: T, wait: number): T {
  let timeout: ReturnType<typeof setTimeout> | null = null
  return ((...args: any[]) => {
    if (timeout) clearTimeout(timeout)
    timeout = setTimeout(() => {
      func(...args)
    }, wait)
  }) as T
}

interface UseSearchInputModelStateOptions {
  field: FieldConfig
  searchType: string
  modelValue: Ref<any>
  inputConfig: ComputedRef<SearchInputConfig>
  shouldUseWidgetSearchRenderer: ComputedRef<boolean>
  emitUpdate: (value: any) => void
  initSelectedOptions: () => Promise<void>
}

export function useSearchInputModelState({
  field,
  searchType,
  modelValue,
  inputConfig,
  shouldUseWidgetSearchRenderer,
  emitUpdate,
  initSelectedOptions
}: UseSearchInputModelStateOptions) {
  const localValue = ref(modelValue.value)
  const shouldShowValue = ref(true)
  const isInternalUpdate = ref(false)
  const dateRangeValue = ref<[number | string | null, number | string | null] | null>(null)
  const rangeValue = ref<any>({
    min: undefined,
    max: undefined
  })

  const selectValue = computed({
    get: () => {
      if (!shouldShowValue.value) {
        return inputConfig.value.props?.multiple ? [] : null
      }
      return localValue.value
    },
    set: (value: any) => {
      localValue.value = value
    }
  })

  const handleInputDebounced = debounce((value: any) => {
    const normalizedValue = normalizeSearchValue(value, {
      widgetType: field.widget?.type,
      searchType,
      field
    })

    emitUpdate(normalizedValue)
  }, SearchConfig.DEBOUNCE_DELAY)

  const handleInput = (value: any) => {
    isInternalUpdate.value = true
    localValue.value = value
    handleInputDebounced(value)
    setTimeout(() => {
      isInternalUpdate.value = false
    }, SearchConfig.INTERNAL_UPDATE_DELAY)
  }

  const handleClear = () => {
    localValue.value = null
    dateRangeValue.value = null
    rangeValue.value = { min: undefined, max: undefined }
    emitUpdate(null)
  }

  const handleRangeChange = () => {
    const min = rangeValue.value.min
    const max = rangeValue.value.max

    if (
      (min === undefined || min === null || min === '') &&
      (max === undefined || max === null || max === '')
    ) {
      emitUpdate(null)
      return
    }

    emitUpdate({
      min: min === '' ? undefined : min,
      max: max === '' ? undefined : max
    })
  }

  const handleDateRangeChange = (
    value: [number | string | null, number | string | null] | null
  ) => {
    dateRangeValue.value = value
    emitUpdate(value)
  }

  const syncRemoteSelectIfNeeded = () => {
    if (
      inputConfig.value.component === SearchComponent.EL_SELECT &&
      inputConfig.value.props?.remote &&
      localValue.value &&
      (Array.isArray(localValue.value) ? localValue.value.length > 0 : true)
    ) {
      if (inputConfig.value.onInitOptions) {
        shouldShowValue.value = false
      }
      nextTick(() => {
        void initSelectedOptions()
      })
    } else {
      shouldShowValue.value = true
    }
  }

  watch(
    () => modelValue.value,
    (newValue: any, oldValue: any) => {
      if (shouldUseWidgetSearchRenderer.value) {
        return
      }

      if (isInternalUpdate.value) {
        return
      }

      if (JSON.stringify(newValue) === JSON.stringify(oldValue)) {
        return
      }

      const isRangeSearch = searchType?.includes('gte') && searchType?.includes('lte')
      const isSliderWidget = field.widget?.type === WidgetType.SLIDER
      const isRangeInput =
        inputConfig.value.component === SearchComponent.NUMBER_RANGE_INPUT ||
        inputConfig.value.component === SearchComponent.RANGE_INPUT

      if ((isSliderWidget || isRangeInput) && isRangeSearch) {
        if (Array.isArray(newValue)) {
          dateRangeValue.value = [newValue[0] || null, newValue[1] || null]
          rangeValue.value = {
            min: newValue[0] || undefined,
            max: newValue[1] || undefined
          }
        } else if (
          newValue &&
          typeof newValue === 'object' &&
          !Array.isArray(newValue) &&
          ('min' in newValue || 'max' in newValue)
        ) {
          rangeValue.value = {
            min: newValue.min !== undefined && newValue.min !== null ? newValue.min : undefined,
            max: newValue.max !== undefined && newValue.max !== null ? newValue.max : undefined
          }
          dateRangeValue.value = null
        } else if (newValue === null || newValue === undefined) {
          rangeValue.value = { min: undefined, max: undefined }
          dateRangeValue.value = null
        }
        return
      }

      if (isRangeSearch && inputConfig.value.component === SearchComponent.EL_DATE_PICKER) {
        dateRangeValue.value = Array.isArray(newValue)
          ? [newValue[0] || null, newValue[1] || null]
          : null
        return
      }

      const isMultiselectContains =
        field.widget?.type === WidgetType.MULTI_SELECT && searchType?.includes(SearchType.CONTAINS)

      if (isMultiselectContains) {
        let newLocalValue: any[] = []
        if (newValue === null || newValue === undefined || newValue === '') {
          newLocalValue = []
        } else if (Array.isArray(newValue)) {
          newLocalValue = newValue
        } else if (typeof newValue === 'string') {
          newLocalValue = newValue
            ? newValue.split(',').map((value) => value.trim()).filter((value) => value)
            : []
        } else {
          newLocalValue = [newValue]
        }

        if (JSON.stringify(localValue.value) !== JSON.stringify(newLocalValue)) {
          localValue.value = newLocalValue
        }
      } else if (
        inputConfig.value.component === SearchComponent.EL_SELECT &&
        inputConfig.value.props?.multiple
      ) {
        if (newValue === null || newValue === undefined || newValue === '') {
          localValue.value = []
        } else if (Array.isArray(newValue)) {
          localValue.value = newValue
        } else if (typeof newValue === 'string') {
          localValue.value = parseCommaSeparatedString(newValue)
        } else {
          localValue.value = [newValue]
        }
      } else {
        localValue.value = newValue
      }

      syncRemoteSelectIfNeeded()
    },
    { immediate: true }
  )

  watch(
    () => inputConfig.value,
    (newConfig, oldConfig) => {
      if (shouldUseWidgetSearchRenderer.value || newConfig === oldConfig) {
        return
      }

      if (
        newConfig.component === SearchComponent.EL_SELECT &&
        newConfig.props?.remote &&
        localValue.value &&
        (Array.isArray(localValue.value) ? localValue.value.length > 0 : true)
      ) {
        nextTick(() => {
          void initSelectedOptions()
        })
      }
    }
  )

  onMounted(() => {
    if (shouldUseWidgetSearchRenderer.value) {
      return
    }
    syncRemoteSelectIfNeeded()
  })

  return {
    localValue,
    shouldShowValue,
    isInternalUpdate,
    selectValue,
    dateRangeValue,
    rangeValue,
    handleInput,
    handleClear,
    handleRangeChange,
    handleDateRangeChange
  }
}
