import { SelectFuzzyQueryType } from '@/architecture/domain/constants/select'
import { WidgetType } from '@/architecture/domain/constants/widget'
import type { FieldConfig, FunctionDetail } from '@/architecture/domain/types'
import { createFieldValue } from '@/architecture/domain/utils/createFieldValue'
import { getTableRequestSearchFields } from '@/architecture/domain/utils/functionSchemaSelectors'
import {
  getSearchFieldRawValue,
  hasSearchFieldValue,
  isStoredSearchFieldValue
} from '@/architecture/domain/utils/searchFieldValue'
import { selectFuzzy as defaultSelectFuzzy } from '@/architecture/presentation/context/api/function'
import { Logger } from '@/architecture/shared/logger'

type SelectFuzzyRunner = typeof defaultSelectFuzzy

interface HydrateTableSearchFuzzyOptions {
  functionDetail: FunctionDetail
  searchForm: Record<string, unknown>
  selectFuzzyRunner?: SelectFuzzyRunner
}

function needsDisplayHydration(value: unknown): boolean {
  if (!hasSearchFieldValue(value)) {
    return false
  }

  const raw = getSearchFieldRawValue(value)
  if (!isStoredSearchFieldValue(value)) {
    return true
  }

  return !value.display || String(value.display) === String(raw)
}

function parseMultiRawValue(raw: unknown): unknown[] {
  if (Array.isArray(raw)) {
    return raw
  }

  if (typeof raw === 'string') {
    return raw.split(',').map(item => item.trim()).filter(Boolean)
  }

  if (raw === null || raw === undefined || raw === '') {
    return []
  }

  return [raw]
}

function scalarDataType(valueType: string): string {
  return valueType.replace(/^\[\]/, '').toLowerCase()
}

function convertValueForFuzzyLookup(value: unknown, valueType: string): unknown {
  if (typeof value !== 'string') {
    return value
  }

  const scalarType = scalarDataType(valueType)
  if (scalarType === 'int' || scalarType === 'integer' || scalarType === 'uint') {
    const parsed = parseInt(value, 10)
    return Number.isNaN(parsed) ? value : parsed
  }

  if (scalarType === 'float' || scalarType === 'number' || scalarType === 'double') {
    const parsed = parseFloat(value)
    return Number.isNaN(parsed) ? value : parsed
  }

  return value
}

function getItemDisplayInfo(item: any): unknown {
  return item?.display_info ?? item?.displayInfo
}

function findMatchingItem(items: any[], raw: unknown): any | undefined {
  return items.find(item => String(item?.value) === String(raw))
}

async function hydrateFieldValue(options: {
  field: FieldConfig
  rawValue: unknown
  functionDetail: FunctionDetail
  selectFuzzyRunner: SelectFuzzyRunner
}): Promise<unknown | undefined> {
  const { field, rawValue, functionDetail, selectFuzzyRunner } = options
  const valueType = field.data?.type || 'string'
  const isMultiSelect = field.widget?.type === WidgetType.MULTI_SELECT
  const rawValues = isMultiSelect ? parseMultiRawValue(rawValue) : [rawValue]

  if (rawValues.length === 0) {
    return undefined
  }

  const queryValue = isMultiSelect
    ? rawValues.map(value => convertValueForFuzzyLookup(value, valueType))
    : convertValueForFuzzyLookup(rawValue, valueType)

  const response = await selectFuzzyRunner(
    functionDetail.method || 'GET',
    functionDetail.router || '',
    {
      code: field.code,
      type: isMultiSelect ? SelectFuzzyQueryType.BY_VALUES : SelectFuzzyQueryType.BY_VALUE,
      value: queryValue,
      request: {},
      value_type: valueType
    }
  )

  if (response.error_msg || !Array.isArray(response.items) || response.items.length === 0) {
    return undefined
  }

  if (!isMultiSelect) {
    const matchedItem = findMatchingItem(response.items, rawValue)
    if (!matchedItem) {
      return undefined
    }

    return createFieldValue(
      field,
      rawValue,
      matchedItem.label || String(rawValue),
      {
        displayInfo: getItemDisplayInfo(matchedItem)
      }
    )
  }

  const matchedItems = rawValues.map(raw => findMatchingItem(response.items || [], raw))
  if (!matchedItems.some(Boolean)) {
    return undefined
  }

  const labels = rawValues.map((raw, index) => matchedItems[index]?.label || String(raw))
  const displayInfos = matchedItems
    .map(item => getItemDisplayInfo(item))
    .filter(item => item !== null && item !== undefined)

  return createFieldValue(
    field,
    rawValue,
    labels.join(', '),
    displayInfos.length > 0 ? { displayInfo: displayInfos } : {}
  )
}

export async function hydrateTableSearchFuzzyDisplay(
  options: HydrateTableSearchFuzzyOptions
): Promise<Record<string, unknown>> {
  const { functionDetail, searchForm, selectFuzzyRunner = defaultSelectFuzzy } = options
  const router = functionDetail.router || ''

  if (!router) {
    return searchForm
  }

  const nextSearchForm = { ...searchForm }
  let changed = false
  const fields = getTableRequestSearchFields(functionDetail)

  for (const field of fields) {
    if (!field.callbacks?.includes('OnSelectFuzzy')) {
      continue
    }

    const currentValue = nextSearchForm[field.code]
    if (!needsDisplayHydration(currentValue)) {
      continue
    }

    try {
      const hydratedValue = await hydrateFieldValue({
        field,
        rawValue: getSearchFieldRawValue(currentValue),
        functionDetail,
        selectFuzzyRunner
      })

      if (hydratedValue !== undefined) {
        nextSearchForm[field.code] = hydratedValue
        changed = true
      }
    } catch (error) {
      Logger.warn('[TableSearchFuzzyHydration]', 'OnSelectFuzzy 搜索条件回显失败', {
        fieldCode: field.code,
        error
      })
    }
  }

  return changed ? nextSearchForm : searchForm
}
