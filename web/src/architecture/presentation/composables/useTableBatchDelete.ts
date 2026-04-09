import { ref, type Ref } from 'vue'
import type { Router } from 'vue-router'
import type { ElTable } from 'element-plus'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { buildBatchDeleteIds, buildTablePermissionApplyURL } from '../views/utils/tableViewActionRuntime'
import { hasPermission, TablePermission } from '@/utils/permission'
import type { FunctionDetail, FieldConfig } from '../../domain/types'
import type { TableRow } from '../../domain/services/TableDomainService'

interface PermissionNodeLike {
  full_code_path?: string
  template_type?: string
}

interface UseTableBatchDeleteOptions {
  router: Router
  functionDetail: () => FunctionDetail
  currentFunctionNode: () => PermissionNodeLike | null
  idField: () => FieldConfig | undefined
  loadTableData: () => Promise<void>
}

export function useTableBatchDelete(options: UseTableBatchDeleteOptions) {
  const isBatchDeleteMode = ref(false)
  const selectedRows = ref<TableRow[]>([])
  const tableRef = ref<InstanceType<typeof ElTable> | null>(null)

  const clearSelection = () => {
    selectedRows.value = []
    if (tableRef.value) {
      tableRef.value.clearSelection()
    }
  }

  const enterBatchDeleteMode = (): void => {
    isBatchDeleteMode.value = true
    clearSelection()
  }

  const exitBatchDeleteMode = (): void => {
    isBatchDeleteMode.value = false
    clearSelection()
  }

  const handleSelectionChange = (selection: TableRow[]): void => {
    selectedRows.value = selection
  }

  const checkSelectable = (): boolean => {
    return true
  }

  const handleBatchDelete = async (): Promise<void> => {
    if (selectedRows.value.length === 0) {
      ElMessage.warning('请先选择要删除的记录')
      return
    }

    const node = options.currentFunctionNode()
    if (!node) {
      ElMessage.error('无法获取函数节点信息，无法验证权限')
      return
    }

    if (!hasPermission(node as any, TablePermission.delete)) {
      ElNotification.warning({
        title: '权限不足',
        message: '您没有删除该表格记录的权限',
        duration: 3000
      })
      const applyUrl = buildTablePermissionApplyURL(node, TablePermission.delete)
      if (applyUrl) {
        options.router.push(applyUrl)
      }
      return
    }

    try {
      await ElMessageBox.confirm(
        `确定要删除选中的 ${selectedRows.value.length} 条记录吗？`,
        '批量删除确认',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      const ids = buildBatchDeleteIds(selectedRows.value, options.idField()?.code)
      if (ids.length === 0) {
        ElMessage.error('无法获取记录 ID，删除失败')
        return
      }

      const { tableDeleteRows } = await import('@/api/function')
      const functionDetail = options.functionDetail()
      const functionRouter = functionDetail.router ?? ''
      if (!functionRouter) {
        ElMessage.error('函数路由缺失，无法执行批量删除')
        return
      }

      await tableDeleteRows(functionDetail.method || 'GET', functionRouter, ids)

      ElNotification.success({
        title: '删除成功',
        message: `已成功删除 ${ids.length} 条记录`,
        duration: 3000,
        position: 'top-right'
      })

      clearSelection()
      isBatchDeleteMode.value = false
      await options.loadTableData()
    } catch (error: any) {
      if (error !== 'cancel') {
        const errorMessage = error?.response?.data?.msg || error?.message || '批量删除失败'
        ElNotification.error({
          title: '删除失败',
          message: errorMessage,
          duration: 5000,
          position: 'top-right'
        })
      }
    }
  }

  return {
    isBatchDeleteMode,
    selectedRows,
    tableRef,
    enterBatchDeleteMode,
    exitBatchDeleteMode,
    handleSelectionChange,
    checkSelectable,
    handleBatchDelete,
  }
}
