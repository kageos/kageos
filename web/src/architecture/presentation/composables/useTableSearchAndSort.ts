import { computed, ref, watch } from 'vue'
import { WidgetType } from '@/architecture/runtime/constants/widget'
import { hasSearchFieldValue } from '@/architecture/runtime/utils/searchFieldValue'
import { resolveSearchFieldLayoutClass } from '../views/utils/searchFieldLayout'
import type { FunctionDetail, FieldConfig } from '../../domain/types'
import type { SortItem, TableState, TableDomainService } from '../../domain/services/TableDomainService'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import { getTableIdField, getTableListFields } from '@/architecture/runtime/utils/functionSchemaSelectors'

interface UseTableSearchAndSortOptions {
  functionDetail: () => FunctionDetail
  domainService: TableDomainService
  stateManager: IStateManager<TableState>
  searchForm: { value: Record<string, any> }
  sorts: { value: SortItem[] }
  hasManualSort: { value: boolean }
  syncToURL: () => void
  loadTableData: () => Promise<void>
}

const STORAGE_KEY_PREFIX = 'table-search-bar-expanded:'

export function useTableSearchAndSort(options: UseTableSearchAndSortOptions) {
  const searchBarStorageKey = computed(() => {
    const router = options.functionDetail()?.router
    return router ? `${STORAGE_KEY_PREFIX}${router}` : ''
  })

  const loadSearchBarExpanded = (): boolean => {
    const key = searchBarStorageKey.value
    if (!key) return true
    try {
      const raw = localStorage.getItem(key)
      if (raw === 'true') return true
      if (raw === 'false') return false
    } catch (_) {}
    return true
  }

  const searchBarExpanded = ref(true)

  const saveSearchBarExpanded = (value: boolean): void => {
    const key = searchBarStorageKey.value
    if (!key) return
    try {
      localStorage.setItem(key, String(value))
    } catch (_) {}
  }

  watch(
    searchBarStorageKey,
    (key) => {
      if (key) searchBarExpanded.value = loadSearchBarExpanded()
    },
    { immediate: true }
  )

  watch(searchBarExpanded, (value) => saveSearchBarExpanded(value))

  const idField = computed(() => {
    return getTableIdField(options.functionDetail())
  })

  const searchableFields = computed(() => {
    return options.domainService.getSearchableFields(options.functionDetail())
  })

  const activeSearchCount = computed(() => {
    const form = options.stateManager.getState().searchForm
    if (!form || typeof form !== 'object') return 0
    return Object.keys(form).filter((key) => hasSearchFieldValue(form[key])).length
  })

  const visibleFields = computed(() => {
    return getTableListFields(options.functionDetail())
  })

  const getSearchFieldLayoutClass = (field: FieldConfig): string => {
    return resolveSearchFieldLayoutClass(field)
  }

  const linkFields = computed(() => {
    return visibleFields.value.filter((field: FieldConfig) => field.widget?.type === WidgetType.LINK)
  })

  const dataFields = computed(() => {
    return visibleFields.value.filter((field: FieldConfig) =>
      field.widget?.type !== WidgetType.ID && field.widget?.type !== WidgetType.LINK
    )
  })

  const getIdFieldCode = (): string | null => {
    return idField.value?.code || null
  }

  const buildDefaultSorts = (): SortItem[] => {
    const idFieldCode = getIdFieldCode()
    if (idFieldCode) {
      return [{ field: idFieldCode, order: 'desc' }]
    }
    return []
  }

  const sortOrderMap = computed<Record<string, 'ascending' | 'descending' | null>>(() => {
    const map: Record<string, 'ascending' | 'descending' | null> = {}
    options.sorts.value.forEach((sort: SortItem) => {
      map[sort.field] = sort.order === 'asc' ? 'ascending' : 'descending'
    })
    if (options.sorts.value.length === 0 && !options.hasManualSort.value && idField.value) {
      map[idField.value.code] = 'descending'
    }
    return map
  })

  const displaySorts = computed(() => {
    if (options.sorts.value.length > 0) {
      return options.sorts.value
    }
    if (idField.value && !options.hasManualSort.value) {
      return [{ field: idField.value.code, order: 'desc' as const }]
    }
    return []
  })

  const getFieldName = (fieldCode: string): string => {
    const field = visibleFields.value.find((item: FieldConfig) => item.code === fieldCode)
    return field?.name || fieldCode
  }

  const handleRemoveSort = (fieldCode: string): void => {
    options.sorts.value = options.sorts.value.filter((item: SortItem) => item.field !== fieldCode)
    if (options.sorts.value.length === 0) {
      options.hasManualSort.value = false
    }
    options.syncToURL()
    void options.loadTableData()
  }

  const handleClearAllSorts = (): void => {
    options.sorts.value = []
    options.hasManualSort.value = false
    options.syncToURL()
    void options.loadTableData()
  }

  const handleSortChange = (sortInfo: { prop?: string; order?: string }): void => {
    const currentState = options.stateManager.getState()
    let newSorts = [...currentState.sorts]

    if (sortInfo && sortInfo.prop && sortInfo.order && sortInfo.order !== '') {
      const field = sortInfo.prop
      const order = sortInfo.order === 'ascending' ? 'asc' : 'desc'

      const idFieldCode = getIdFieldCode()
      if (idFieldCode) {
        newSorts = newSorts.filter((item: SortItem) => item.field !== idFieldCode)
      }

      const existingIndex = newSorts.findIndex((item: SortItem) => item.field === field)
      if (existingIndex >= 0) {
        const existingSort = newSorts[existingIndex]
        if (existingSort) {
          existingSort.order = order
        }
      } else {
        newSorts.push({ field, order })
      }
    } else if (sortInfo.prop) {
      newSorts = newSorts.filter((item: SortItem) => item.field !== sortInfo.prop)
    }

    options.stateManager.setState({
      ...currentState,
      sorts: newSorts,
      hasManualSort: true
    })

    options.syncToURL()
    void options.loadTableData()
  }

  const getSearchValue = (field: FieldConfig): any => {
    const value = options.searchForm.value[field.code]
    return value === undefined ? null : value
  }

  const updateSearchValue = (field: FieldConfig, value: any, shouldSearch: boolean = false): void => {
    const currentState = options.stateManager.getState()
    const newSearchForm = { ...currentState.searchForm }

    if (
      !hasSearchFieldValue(value)
    ) {
      delete newSearchForm[field.code]
    } else {
      newSearchForm[field.code] = value
    }

    options.stateManager.setState({ ...currentState, searchForm: newSearchForm })
    options.syncToURL()
    if (shouldSearch) {
      void options.loadTableData()
    }
  }

  const handleSearch = (): void => {
    const currentState = options.stateManager.getState()
    options.stateManager.setState({
      ...currentState,
      pagination: {
        ...currentState.pagination,
        currentPage: 1
      }
    })
    options.syncToURL()
    void options.loadTableData()
  }

  const handleReset = (): void => {
    const currentState = options.stateManager.getState()
    options.stateManager.setState({
      ...currentState,
      searchForm: {},
      sorts: [],
      hasManualSort: false,
      pagination: {
        ...currentState.pagination,
        currentPage: 1
      }
    })
    options.syncToURL()
    void options.loadTableData()
  }

  return {
    searchBarExpanded,
    idField,
    searchableFields,
    activeSearchCount,
    visibleFields,
    getSearchFieldLayoutClass,
    linkFields,
    dataFields,
    buildDefaultSorts,
    sortOrderMap,
    displaySorts,
    getFieldName,
    handleRemoveSort,
    handleClearAllSorts,
    handleSortChange,
    getSearchValue,
    updateSearchValue,
    handleSearch,
    handleReset,
  }
}
