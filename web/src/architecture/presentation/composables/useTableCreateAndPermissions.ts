import { computed, type Ref } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import type { FunctionDetail } from '../../domain/types'
import type { WorkspaceState } from '../../domain/services/WorkspaceDomainService'
import type { TableApplicationService } from '../../application/services/TableApplicationService'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { RouteSource } from '@/utils/routeSource'
import {
  buildTableAddDialogOpenRequest,
  buildTableCreateDialogCloseRequest
} from '../views/utils/tableViewRouteRuntime'
import { hasPermission, TablePermission } from '@/utils/permission'
import { usePermissionErrorStore } from '@/stores/permissionError'
import { getTableCreateFields } from '@/utils/functionSchemaSelectors'

interface UseTableCreateAndPermissionsOptions {
  routeQuery: () => Record<string, any>
  functionDetail: () => FunctionDetail
  workspaceStateManager: IStateManager<WorkspaceState>
  applicationService: TableApplicationService
  createDialogVisible: Ref<boolean>
}

export function useTableCreateAndPermissions(options: UseTableCreateAndPermissionsOptions) {
  const permissionErrorStore = usePermissionErrorStore()

  const currentFunctionNode = computed(() => {
    return options.workspaceStateManager.getState().currentFunction
  })

  const canCreate = computed(() => {
    const node = currentFunctionNode.value
    if (!node) return false
    return hasPermission(node, TablePermission.write)
  })

  const canUpdate = computed(() => {
    const node = currentFunctionNode.value
    if (!node) return false
    return hasPermission(node, TablePermission.update)
  })

  const canDelete = computed(() => {
    const node = currentFunctionNode.value
    if (!node) return false
    return hasPermission(node, TablePermission.delete)
  })

  const permissionError = computed(() => permissionErrorStore.currentError)

  const clearPermissionError = (): void => {
    permissionErrorStore.clearError()
  }

  const handleApplyPermissionForAction = (action: string): void => {
    ElMessage.warning(`当前用户暂无 ${action} 权限`)
  }

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
    const node = currentFunctionNode.value
    if (!node) {
      ElMessage.error('无法获取函数节点信息，无法验证权限')
      return
    }

    if (!hasPermission(node, TablePermission.write)) {
      ElNotification.warning({
        title: '权限不足',
        message: '您没有新增该表格记录的权限',
        duration: 3000
      })
      return
    }

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
    canCreate,
    canUpdate,
    canDelete,
    permissionError,
    clearPermissionError,
    handleApplyPermissionForAction,
    handleAdd,
    handleCreateSubmit,
    handleCreateDialogClose
  }
}
