import { computed, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import type { FunctionDetail } from '../../domain/types'
import type { WorkspaceState } from '../../domain/types'
import type { TableApplicationService } from '../../application/services/TableApplicationService'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { RouteSource } from '@/architecture/shared/routing/routeSource'
import {
  buildTableAddDialogOpenRequest,
  buildTableCreateDialogCloseRequest
} from '../views/utils/tableViewRouteRuntime'
import { getTableCreateFields } from '@/architecture/domain/utils/functionSchemaSelectors'

interface UseTableCreateAndPermissionsOptions {
  routeQuery: () => Record<string, any>
  functionDetail: () => FunctionDetail
  workspaceStateManager: IStateManager<WorkspaceState>
  applicationService: TableApplicationService
  createDialogVisible: Ref<boolean>
}

export function useTableCreateAndPermissions(options: UseTableCreateAndPermissionsOptions) {
  const currentFunctionNode = computed(() => {
    return options.workspaceStateManager.getState().currentFunction
  })

  const handleAdd = (): void => {
    options.createDialogVisible.value = true

    eventBus.emit(RouteEvent.updateRequested, {
      ...buildTableAddDialogOpenRequest(options.routeQuery()),
      source: RouteSource.TABLE_ADD_DIALOG_OPEN
    })
  }

  const handleCreateDialogClose = (): void => {
    const functionDetail = options.functionDetail()
    const request = buildTableCreateDialogCloseRequest({
      routeQuery: options.routeQuery(),
      responseFieldCodes: getTableCreateFields(functionDetail).map(field => field.code)
    })

    if (!request) {
      return
    }

    eventBus.emit(RouteEvent.updateRequested, {
      ...request,
      source: RouteSource.TABLE_CREATE_DIALOG_CLOSE
    })
  }

  const handleCreateSubmit = async (data: Record<string, any>): Promise<void> => {
    try {
      await options.applicationService.addRow(options.functionDetail(), data)
      ElMessage.success('新增成功')
      options.createDialogVisible.value = false
      handleCreateDialogClose()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.msg || error?.message || '新增失败'
      ElMessage.error(errorMessage)
    }
  }

  return {
    currentFunctionNode,
    handleAdd,
    handleCreateSubmit,
    handleCreateDialogClose
  }
}
