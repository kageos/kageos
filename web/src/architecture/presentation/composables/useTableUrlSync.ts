import { TEMPLATE_TYPE } from '@/architecture/runtime/utils/functionTypes'
import { isLinkNavigation } from '@/architecture/runtime/utils/linkNavigation'
import { RouteSource } from '@/architecture/runtime/utils/routeSource'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import type { TableState } from '../../domain/services/TableDomainService'
import type { FunctionDetail } from '../../domain/types'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { buildNextTableSyncQuery } from '../views/utils/tableViewURLRuntime'

interface UseTableUrlSyncOptions {
  functionDetail: () => FunctionDetail
  routeQuery: () => Record<string, any>
  stateManager: IStateManager<TableState>
  buildDefaultSorts: () => { field: string; order: 'asc' | 'desc' }[]
}

export function useTableUrlSync(options: UseTableUrlSyncOptions) {
  const syncToURL = (): void => {
    if (options.functionDetail().template_type !== TEMPLATE_TYPE.TABLE) {
      return
    }

    const query = options.routeQuery()
    const isLinkNav = isLinkNavigation(query)
    const newQuery = buildNextTableSyncQuery({
      routeQuery: query,
      functionDetail: options.functionDetail(),
      state: options.stateManager.getState(),
      buildDefaultSorts: options.buildDefaultSorts,
      isLinkNavigation: isLinkNav
    })

    eventBus.emit(RouteEvent.updateRequested, {
      query: newQuery,
      preserveParams: {
        table: true,
        search: true,
        state: true,
        linkNavigation: isLinkNav
      },
      source: RouteSource.TABLE_SYNC
    })
  }

  return {
    syncToURL
  }
}
