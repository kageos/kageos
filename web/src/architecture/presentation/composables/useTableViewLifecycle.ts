import { onMounted, onUnmounted, type Ref } from 'vue'
import { eventBus, TableEvent, WorkspaceEvent } from '../../infrastructure/eventBus'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import type { TableState, TableRow } from '../../domain/services/TableDomainService'
import type { FunctionDetail } from '../../domain/types'

interface UseTableViewLifecycleOptions {
  functionDetailId?: number
  isMounted: Ref<boolean>
  clearPermissionError: () => void
  initializeTable: () => Promise<void>
  setupQueryWatch: () => (() => void) | null
  stateManager: IStateManager<TableState>
}

export function useTableViewLifecycle(options: UseTableViewLifecycleOptions) {
  let unsubscribeDataLoaded: (() => void) | null = null
  let unsubscribeFunctionLoaded: (() => void) | null = null
  let unsubscribeTableQueryChanged: (() => void) | null = null

  onMounted(async () => {
    options.clearPermissionError()
    options.isMounted.value = true

    unsubscribeTableQueryChanged = options.setupQueryWatch() || null
    await options.initializeTable()

    unsubscribeDataLoaded = eventBus.on(TableEvent.dataLoaded, async (payload: { data: TableRow[], pagination?: any }) => {
      if (!options.isMounted.value) {
        return
      }

      const currentState = options.stateManager.getState()
      options.stateManager.setState({
        ...currentState,
        pagination: {
          currentPage: payload.pagination?.current_page || currentState.pagination.currentPage,
          pageSize: payload.pagination?.page_size || currentState.pagination.pageSize,
          total: payload.pagination?.total_count || 0
        }
      })
    })

    unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, async (payload: { detail: FunctionDetail }) => {
      if (options.functionDetailId && payload.detail.template_type === 'table' && payload.detail.id === options.functionDetailId) {
        if (options.isMounted.value) {
          await options.initializeTable()
        }
      }
    })
  })

  onUnmounted(() => {
    options.isMounted.value = false

    if (unsubscribeDataLoaded) {
      unsubscribeDataLoaded()
    }
    if (unsubscribeFunctionLoaded) {
      unsubscribeFunctionLoaded()
    }
    if (unsubscribeTableQueryChanged) {
      unsubscribeTableQueryChanged()
    }
  })
}
