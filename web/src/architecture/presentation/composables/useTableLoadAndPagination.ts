import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { TableApplicationService } from '../../application/services/TableApplicationService'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import type { TableDomainService, TableState } from '../../domain/services/TableDomainService'
import type { FunctionDetail } from '../../domain/types'
import {
  buildTableLoadRequest,
  buildTableLoadingState,
  decideTableLoadGuard,
  getTableLoadErrorMessage
} from '../views/utils/tableViewLoadRuntime'

interface UseTableLoadAndPaginationOptions {
  functionDetail: () => FunctionDetail
  stateManager: IStateManager<TableState>
  domainService: TableDomainService
  applicationService: TableApplicationService
  buildDefaultSorts: () => { field: string; order: 'asc' | 'desc' }[]
  syncToURL: () => void
  onPageSizeChange?: (size: number) => void
}

export function useTableLoadAndPagination(options: UseTableLoadAndPaginationOptions) {
  const isMounted = ref(false)
  const skipNextTableLoad = ref(false)

  const loadTableData = async (): Promise<void> => {
    const guardResult = decideTableLoadGuard({
      isMounted: isMounted.value,
      skipNextTableLoad: skipNextTableLoad.value
    })

    if (guardResult === 'skip-unmounted') {
      return
    }

    if (guardResult === 'skip-next-load') {
      skipNextTableLoad.value = false
      const state = options.stateManager.getState()
      options.stateManager.setState(buildTableLoadingState(state, false))
      return
    }

    const stateBeforeLoad = options.stateManager.getState()
    options.stateManager.setState(buildTableLoadingState(stateBeforeLoad, true))

    const currentState = options.stateManager.getState()
    const { searchParams, sortParams, pagination } = buildTableLoadRequest({
      functionDetail: options.functionDetail(),
      state: currentState,
      buildDefaultSorts: options.buildDefaultSorts,
      buildSearchParams: (functionDetail, searchForm) =>
        options.domainService.buildSearchParams(functionDetail, searchForm)
    })

    if (!isMounted.value) {
      return
    }

    try {
      await options.applicationService.loadData(
        options.functionDetail(),
        searchParams,
        sortParams,
        pagination
      )
    } catch (error: any) {
      ElMessage.error(getTableLoadErrorMessage(error))
    }
  }

  const handleSizeChange = (size: number): void => {
    options.onPageSizeChange?.(size)
    const currentState = options.stateManager.getState()
    options.stateManager.setState({
      ...currentState,
      pagination: {
        ...currentState.pagination,
        pageSize: size,
        currentPage: 1
      }
    })
    options.syncToURL()
    void loadTableData()
  }

  const handleCurrentChange = (page: number): void => {
    const currentState = options.stateManager.getState()
    options.stateManager.setState({
      ...currentState,
      pagination: {
        ...currentState.pagination,
        currentPage: page
      }
    })
    options.syncToURL()
    void loadTableData()
  }

  return {
    isMounted,
    skipNextTableLoad,
    loadTableData,
    handleSizeChange,
    handleCurrentChange
  }
}
