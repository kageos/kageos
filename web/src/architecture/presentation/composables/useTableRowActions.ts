import { ElMessage, ElMessageBox } from 'element-plus'
import type { Router } from 'vue-router'
import { normalizeWidgetType, WidgetType } from '@/architecture/domain/constants/widget'
import { convertToFieldValue } from '@/architecture/domain/utils/field'
import { isWidgetConfigFlagEnabled } from '@/architecture/domain/utils/widgetConfigFlag'
import { parseLinkValue, addLinkTypeToUrl } from '@/architecture/shared/routing/linkNavigation'
import { resolveWorkspaceUrl } from '@/architecture/shared/routing/route'
import { RouteSource } from '@/architecture/shared/routing/routeSource'
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
    const type = normalizeWidgetType(field.widget?.type)
    const code = (field.code || '').toLowerCase()
    const name = field.name || ''

    // 智能推断：根据表头中文字符数给一个基础缓冲宽度 (每个汉字大约 14px + 表头排序图标和 padding 大约 40px)
    const nameStr = String(name)
    let charWidth = 0
    for (let i = 0; i < nameStr.length; i++) {
      charWidth += nameStr.charCodeAt(i) > 255 ? 15 : 9
    }
    const headerWidth = charWidth + 40 

    // 动态嗅探数据内容长度
    let maxDataCharLength = 0
    let hasData = false
    const tableData = options.stateManager.getState().data || []
    
    if (Array.isArray(tableData) && tableData.length > 0) {
      for (const row of tableData) {
        let val = row[code]
        if (val !== undefined && val !== null && val !== '') {
          const strVal = String(val).trim()
          if (strVal !== '-' && strVal !== '[]' && strVal !== '{}') {
            hasData = true
            let currLen = 0
            for (let i = 0; i < strVal.length; i++) {
               currLen += strVal.charCodeAt(i) > 255 ? 14 : 8
            }
            if (currLen > maxDataCharLength) {
              maxDataCharLength = currLen
            }
          }
        }
      }
    }

    // 1. 空值倾向很高的字段
    if (/^(remark|desc|description|summary|reason|cause|degrade_reason|.*_reason|source_note)$/.test(code) || 
        /(备注|说明|描述|摘要|原因|理由)/.test(name)) {
      if (!hasData) return Math.max(headerWidth, 80)
    }

    // 2. 系统级字段（ID、创建人、更新时间等）
    if (/^(id|created_at|updated_at|create_time|update_time|creator|updater|modifier|created_by)$/.test(code) || 
        /(ID|创建|更新|修改)(时间|人)/.test(name)) {
      if (type === WidgetType.DATETIME) return Math.max(headerWidth, 175)
      return Math.max(headerWidth, 75)
    }

    if (!hasData) {
      return Math.max(headerWidth, 75)
    }

    const dataWidth = hasData ? Math.min(maxDataCharLength + 32, 350) : 0
    const dataBasedWidth = Math.max(headerWidth, dataWidth)

    let minWidth = 80
    if (type === WidgetType.DATETIME) {
      minWidth = 180
    } else if (type === WidgetType.SWITCH) {
      minWidth = 80
    } else if (type === WidgetType.INTEGER || type === WidgetType.FLOAT) {
      minWidth = Math.max(headerWidth, 80)
    } else if (type === WidgetType.PROGRESS || type === WidgetType.SLIDER) {
      minWidth = 140
    } else if (type === WidgetType.FILES) {
      minWidth = 120
    } else if (type === WidgetType.TEXT_AREA || type === WidgetType.RICH_TEXT) {
      minWidth = Math.max(headerWidth, 120)
    } else if (type === WidgetType.SELECT || type === WidgetType.MULTI_SELECT) {
      minWidth = Math.max(headerWidth, 100)
    }

    return Math.max(headerWidth, Math.min(Math.max(minWidth, dataBasedWidth), 350))
  }

  return {
    handleActionCommand,
    getLinkText,
    getColumnWidth,
    handleDetail,
    handleRowClick
  }
}
