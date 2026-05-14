import { computed, nextTick, ref, type ComputedRef, type Ref } from 'vue'
import { parseCommaSeparatedString } from '@/architecture/domain/utils/stringUtils'
import { SearchComponent } from '@/architecture/domain/constants/search'
import { WidgetType } from '@/architecture/domain/constants/widget'
import { buildSelectionSummary } from '@/architecture/presentation/widgets/utils/selectionSummary'
import {
  getOptionLightPalette,
  getOptionSolidColor,
  normalizeOptionColor,
  type StandardColorType
} from '@/architecture/domain/constants/select'
import { Logger } from '@/architecture/shared/logger'
import { getFieldWidgetOptionColors } from '@/architecture/domain/utils/widgetOptionColors'
import type { FieldConfig } from '@/architecture/domain/types'
import type { SearchInputConfig, SearchOption } from '../utils/searchInputTypes'

interface UseSearchInputFallbackSelectOptions {
  field: FieldConfig
  inputConfig: ComputedRef<SearchInputConfig>
  localValue: Ref<any>
  shouldShowValue: Ref<boolean>
  handleInput: (value: any) => void
}

export function useSearchInputFallbackSelect({
  field,
  inputConfig,
  localValue,
  shouldShowValue,
  handleInput
}: UseSearchInputFallbackSelectOptions) {
  const selectOptions = ref<SearchOption[]>([])
  const selectLoading = ref(false)

  const isFallbackSelect = computed(() => {
    return inputConfig.value.component === SearchComponent.EL_SELECT
  })

  const isSingleFallbackSelect = computed(() => {
    return isFallbackSelect.value && !inputConfig.value.props?.multiple
  })

  const isMultipleFallbackSelect = computed(() => {
    return isFallbackSelect.value && !!inputConfig.value.props?.multiple
  })

  const isMultiselectWidget = computed(() => {
    return field.widget?.type === WidgetType.MULTI_SELECT
  })

  const isSelectWidget = computed(() => {
    return field.widget?.type === WidgetType.SELECT
  })

  const hasOptionColors = computed(() => {
    return optionColors.value.length > 0
  })

  const shouldUseUserFallbackTags = computed(() => {
    return inputConfig.value.props?.popperClass === 'user-select-dropdown-popper'
  })

  const shouldUseColoredFallbackTags = computed(() => {
    return isMultipleFallbackSelect.value && hasOptionColors.value
  })

  const shouldUseCustomFallbackTags = computed(() => {
    return isMultipleFallbackSelect.value
  })

  const shouldUseNeutralFallbackTags = computed(() => {
    return (
      shouldUseCustomFallbackTags.value &&
      !shouldUseUserFallbackTags.value &&
      !shouldUseColoredFallbackTags.value
    )
  })

  const shouldShowColoredMultiFallbackOption = computed(() => {
    return (isMultiselectWidget.value || isSelectWidget.value || field.widget?.type === WidgetType.RADIO) && hasOptionColors.value
  })

  const fallbackTagSummary = computed(() => {
    const values = Array.isArray(localValue.value) ? localValue.value : []
    return buildSelectionSummary(values, 1)
  })

  const optionColors = computed(() => {
    return getFieldWidgetOptionColors(field)
  })

  const staticOptions = computed(() => {
    const inputConfigOptions = inputConfig.value.props?.options
    if (inputConfigOptions && Array.isArray(inputConfigOptions) && inputConfigOptions.length > 0) {
      return inputConfigOptions.map((opt: any) => {
        if (typeof opt === 'string') {
          return { label: opt, value: opt }
        }
        return opt
      })
    }

    const options = field.widget?.config?.options || []
    return options.map((opt: any) => {
      if (typeof opt === 'string') {
        return { label: opt, value: opt }
      }
      return opt
    })
  })

  function getOptionColor(value: any): string | null {
    if (!value) return null
    if (!optionColors.value || optionColors.value.length === 0) return null

    const valueStr = String(value)
    const optionIndex = staticOptions.value.findIndex((opt: any) => {
      const optValue = typeof opt === 'object' ? opt.value : opt
      return String(optValue) === valueStr
    })

    if (optionIndex >= 0 && optionIndex < optionColors.value.length) {
      return normalizeOptionColor(optionColors.value[optionIndex]) || null
    }

    return null
  }

  function getOptionColorType(value: any): StandardColorType | undefined {
    void value
    return undefined
  }

  function getOptionColorValue(value: any): string | undefined {
    return undefined
  }

  function getSelectTagStyle(value: any): Record<string, string> {
    const color = getOptionColor(value)
    if (!color) return {}

    const palette = getOptionLightPalette(color)
    if (!palette) {
      return {}
    }

    return {
      backgroundColor: palette.backgroundColor,
      borderColor: palette.borderColor,
      color: palette.color
    }
  }

  function getOptionColorStyle(value: any): Record<string, string> {
    const color = getOptionColor(value)
    if (!color) return {}

    const backgroundColor = getOptionSolidColor(color)

    const style: Record<string, string> = {
      marginRight: '8px',
      display: 'inline-block',
      width: '12px',
      height: '12px',
      minWidth: '12px',
      minHeight: '12px',
      borderRadius: '2px',
      flexShrink: '0',
      border: 'none',
      verticalAlign: 'middle',
      filter: 'brightness(0.95) saturate(0.9)',
      opacity: '0.9'
    }

    if (backgroundColor) {
      style.backgroundColor = backgroundColor
    }

    return style
  }

  const selectOptionsComputed = computed(() => {
    if (inputConfig.value.component !== SearchComponent.EL_SELECT) {
      return []
    }

    const inputConfigOptions = inputConfig.value.props?.options
    if (inputConfigOptions && inputConfigOptions.length > 0) {
      return inputConfigOptions
    }

    return selectOptions.value
  })

  function getOptionLabel(value: any): string {
    if (value === null || value === undefined) return ''
    const valueStr = String(value)
    const option = selectOptionsComputed.value.find((opt: any) => {
      const optValue = typeof opt === 'object' ? opt.value : opt
      return String(optValue) === valueStr
    })
    if (option) {
      return typeof option === 'object' ? option.label : option
    }
    return valueStr
  }

  function getRenderedOptionValue(option: any): any {
    return typeof option === 'object' ? option.value : option
  }

  function getRenderedOptionLabel(option: any): string {
    return typeof option === 'object' ? option.label : String(option)
  }

  function getRenderedOptionUserInfo(option: any): SearchOption['userInfo'] | null {
    return typeof option === 'object' && option?.userInfo ? option.userInfo : null
  }

  function getUserInfoByValue(value: any): any {
    if (!value) return null
    if (!Array.isArray(selectOptions.value)) return null
    const option = selectOptions.value.find((opt: any) => {
      if (!opt) return false
      const optValue = typeof opt === 'object' ? opt.value : opt
      return String(optValue) === String(value)
    })
    return option?.userInfo || null
  }

  function getUserTagInitial(value: any): string {
    const userInfo = getUserInfoByValue(value)
    if (userInfo?.username) {
      return userInfo.username[0]?.toUpperCase() || 'U'
    }

    const label = getOptionLabel(value)
    return label?.[0]?.toUpperCase() || 'U'
  }

  function handleRemoveTag(valueToRemove: any): void {
    if (!Array.isArray(localValue.value)) {
      return
    }
    const newValues = localValue.value.filter((value: any) => String(value) !== String(valueToRemove))
    localValue.value = newValues
    handleInput(newValues)
  }

  const handleRemoteMethod = async (query: string) => {
    if (inputConfig.value.component !== SearchComponent.EL_SELECT || !inputConfig.value.onRemoteMethod) {
      return
    }

    selectLoading.value = true
    try {
      const options = await inputConfig.value.onRemoteMethod(query)
      const currentValue = localValue.value
      const existingOptions = selectOptions.value || []
      const mergedOptions = [...(options || [])]

      if (currentValue) {
        const valuesToCheck = Array.isArray(currentValue) ? currentValue : [currentValue]
        valuesToCheck.forEach((val: any) => {
          if (
            val &&
            !mergedOptions.find((opt: any) => {
              const optValue = typeof opt === 'object' ? opt.value : opt
              return String(optValue) === String(val)
            })
          ) {
            const existingOption = existingOptions.find((opt: any) => {
              const optValue = typeof opt === 'object' ? opt.value : opt
              return String(optValue) === String(val)
            })
            if (existingOption) {
              mergedOptions.push(existingOption)
            }
          }
        })
      }

      selectOptions.value = mergedOptions
    } catch (error) {
      Logger.error('SearchInput', 'Remote method error', error)
      selectOptions.value = []
    } finally {
      selectLoading.value = false
    }
  }

  const initSelectedOptions = async () => {
    if (inputConfig.value.component !== SearchComponent.EL_SELECT) {
      return
    }

    const currentValue = localValue.value
    if (!currentValue) {
      return
    }

    if (inputConfig.value.onInitOptions) {
      selectLoading.value = true
      shouldShowValue.value = false
      await nextTick()

      try {
        let queryValues: any[] = []
        if (Array.isArray(currentValue) && currentValue.length > 0) {
          queryValues = currentValue
        } else if (typeof currentValue === 'string' && currentValue.includes(',')) {
          queryValues = parseCommaSeparatedString(currentValue)
        } else if (currentValue !== null && currentValue !== undefined && currentValue !== '') {
          queryValues = [currentValue]
        }

        if (queryValues.length === 0) {
          shouldShowValue.value = true
          return
        }

        const options = await inputConfig.value.onInitOptions(
          queryValues.length === 1 ? queryValues[0] : queryValues
        )
        selectOptions.value = options || []

        if (options && options.length > 0) {
          const isMultiple = inputConfig.value.props?.multiple || false

          if (isMultiple) {
            const matchedValues: any[] = []
            queryValues.forEach((queryVal) => {
              const matchedOption = options.find((opt: any) => {
                const optValue = typeof opt === 'object' ? opt.value : opt
                return optValue === queryVal || String(optValue) === String(queryVal)
              })
              if (matchedOption) {
                matchedValues.push(typeof matchedOption === 'object' ? matchedOption.value : matchedOption)
              }
            })
            if (matchedValues.length > 0) {
              localValue.value = matchedValues
            }
          } else {
            const queryValue = queryValues[0]
            const matchedOption = options.find((opt: any) => {
              const optValue = typeof opt === 'object' ? opt.value : opt
              return optValue === queryValue || String(optValue) === String(queryValue)
            })
            if (matchedOption) {
              localValue.value = typeof matchedOption === 'object' ? matchedOption.value : matchedOption
            }
          }
        }

        shouldShowValue.value = true
      } catch (error) {
        Logger.error('SearchInput', 'Init selected options error', error)
        selectOptions.value = []
        shouldShowValue.value = true
      } finally {
        selectLoading.value = false
      }
      return
    }

    if (!inputConfig.value.onRemoteMethod) {
      return
    }

    if (Array.isArray(currentValue) && currentValue.length > 0) {
      selectLoading.value = true
      try {
        const optionPromises = currentValue.map(async (value: any) => {
          if (!value) return null
          const options = await inputConfig.value.onRemoteMethod!(String(value))
          return options?.find((opt: any) => {
            const optValue = typeof opt === 'object' ? opt.value : opt
            return String(optValue) === String(value)
          })
        })

        selectOptions.value = (await Promise.all(optionPromises)).filter(Boolean) as SearchOption[]
      } catch (error) {
        Logger.error('SearchInput', 'Init selected options error', error)
      } finally {
        selectLoading.value = false
      }
    } else if (typeof currentValue === 'string' && currentValue.trim()) {
      selectLoading.value = true
      try {
        const options = await inputConfig.value.onRemoteMethod(String(currentValue))
        const currentOption = options?.find((opt: any) => {
          const optValue = typeof opt === 'object' ? opt.value : opt
          return String(optValue) === String(currentValue)
        })
        if (currentOption) {
          selectOptions.value = [currentOption]
        } else if (options && options.length > 0) {
          selectOptions.value = options
        }
      } catch (error) {
        Logger.error('SearchInput', 'Init selected options error', error)
      } finally {
        selectLoading.value = false
      }
    }
  }

  const handleVisibleChange = (visible: boolean) => {
    if (!visible) {
      return
    }
    const currentValue = localValue.value
    if (currentValue && (Array.isArray(currentValue) ? currentValue.length > 0 : true)) {
      const hasOptions = selectOptions.value && selectOptions.value.length > 0
      if (!hasOptions && inputConfig.value.onInitOptions) {
        nextTick(() => {
          void initSelectedOptions()
        })
      }
    }
  }

  return {
    selectOptions,
    selectLoading,
    isFallbackSelect,
    isSingleFallbackSelect,
    isMultipleFallbackSelect,
    isMultiselectWidget,
    isSelectWidget,
    hasOptionColors,
    shouldUseUserFallbackTags,
    shouldUseColoredFallbackTags,
    shouldUseCustomFallbackTags,
    shouldUseNeutralFallbackTags,
    shouldShowColoredMultiFallbackOption,
    fallbackTagSummary,
    selectOptionsComputed,
    getOptionColorType,
    getOptionColorValue,
    getSelectTagStyle,
    getOptionColorStyle,
    getOptionLabel,
    getRenderedOptionValue,
    getRenderedOptionLabel,
    getRenderedOptionUserInfo,
    getUserInfoByValue,
    getUserTagInitial,
    handleRemoveTag,
    handleRemoteMethod,
    handleVisibleChange,
    initSelectedOptions
  }
}
