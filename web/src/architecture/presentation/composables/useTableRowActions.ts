import { ElMessage, ElMessageBox } from 'element-plus'
import type { Router } from 'vue-router'
import { WidgetType } from '@/architecture/domain/constants/widget'
import { convertToFieldValue } from '@/architecture/domain/utils/field'
import { isWidgetConfigFlagEnabled } from '@/architecture/domain/utils/widgetConfigFlag'
import { parseLinkValue, addLinkTypeToUrl } from '@/architecture/runtime/utils/linkNavigation'
import { resolveWorkspaceUrl } from '@/architecture/runtime/utils/route'
import { RouteSource } from '@/architecture/runtime/utils/routeSource'
import type { TableApplicationService } from '../../application/services/TableApplicationService'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import type { TableState, TableRow } from '../../domain/types'
import type { FieldConfig, FunctionDetail } from '../../domain/types'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import {
  buildTableDetailRowPayload,
  buildTableLinkRouteRequest
} from '../views/utils/tableViewRouteRuntime'
import { resolveTableActionCommand } from '../views/utils/tableViewActionRuntime'

interface UseTableRowActionsOptions {
  functionDetail: () => FunctionDetail
  router: Router
  stateManager: IStateManager<TableState>
  applicationService: TableApplicationService
  linkFields: () => FieldConfig[]
  skipNextTableLoad: { value: boolean }
}

export function useTableRowActions(options: UseTableRowActionsOptions) {
  const handleDetail = (row: TableRow, initialMode: 'read' | 'edit' = 'read'): void => {
    options.skipNextTableLoad.value = true

    const tableData = options.stateManager.getState().data || []

    eventBus.emit('table:detail-row', buildTableDetailRowPayload({
      row,
      tableData,
      initialMode
    }))
  }

  const handleRowClick = (row: TableRow, _column: unknown, event: Event): void => {
    const target = event.target as HTMLElement
    if (target?.closest?.('.action-column, .el-dropdown, .el-checkbox')) return
    handleDetail(row)
  }

  const getLinkText = (linkField: FieldConfig, rawValue: unknown): string => {
    const value = convertToFieldValue(rawValue, linkField)
    const url = value?.raw || ''
    if (!url) return linkField.name || '链接'

    const match = url.match(/^\[([^\]]+)\](.+)$/)
    if (match) {
      return match[1]
    }

    return linkField.widget?.config?.text || linkField.name || '链接'
  }

  const handleLinkClick = (fieldCode: string, row: TableRow): void => {
    const linkField = options.linkFields().find((field) => field.code === fieldCode)
    if (!linkField) return

    const value = convertToFieldValue(row[fieldCode], linkField)
    const raw = value?.raw || ''
    if (!raw) return

    const parsedLink = parseLinkValue(raw)
    const linkConfig = linkField.widget?.config || {}
    const target = linkConfig.target || '_self'
    const resolvedUrl = resolveWorkspaceUrl(parsedLink.url, options.router.currentRoute.value)
    const isExternal = resolvedUrl.startsWith('http://') || resolvedUrl.startsWith('https://')

    if (isExternal) {
      window.open(resolvedUrl, '_blank')
      return
    }

    const finalUrl = addLinkTypeToUrl(resolvedUrl, parsedLink.type)

    if (target === '_blank') {
      window.open(finalUrl, '_blank')
      return
    }

    eventBus.emit(RouteEvent.updateRequested, {
      ...buildTableLinkRouteRequest(finalUrl),
      source: RouteSource.TABLE_LINK_CLICK
    })
  }

  const handleDelete = async (row: TableRow): Promise<void> => {
    try {
      await ElMessageBox.confirm('确定要删除该行数据吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })
      await options.applicationService.deleteRow(options.functionDetail(), row.id)
      ElMessage.success('删除成功')
    } catch (error) {
      if (error !== 'cancel' && error !== 'close') {
        ElMessage.error('删除失败')
      }
    }
  }

  const handleActionCommand = (command: string, row: TableRow): void => {
    const action = resolveTableActionCommand({
      command
    })

    if (action.type === 'link') {
      handleLinkClick(action.fieldCode, row)
      return
    }

    if (action.type === 'detail') {
      handleDetail(row, action.initialMode)
      return
    }

    if (action.type === 'delete') {
      void handleDelete(row)
      return
    }
  }

  const getColumnWidth = (field: FieldConfig): number => {
    if (field.widget?.type === WidgetType.DATETIME) return 180
    if (field.widget?.type === WidgetType.TEXT_AREA) return 300
    if (field.widget?.type === WidgetType.FILES && isWidgetConfigFlagEnabled(field.widget?.config?.list_preview)) return 180
    if (field.widget?.type === 'department' || field.widget?.type === 'departments') return 300
    if (field.widget?.type === 'user' || field.widget?.type === 'users') return 250
    return 150
  }

  return {
    handleActionCommand,
    getLinkText,
    getColumnWidth,
    handleDetail,
    handleRowClick
  }
}
