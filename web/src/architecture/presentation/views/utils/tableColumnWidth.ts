import { normalizeWidgetType, WidgetType } from '@/architecture/domain/constants/widget'
import type { FieldConfig, TableRow } from '@/architecture/domain/types'

export const resolveTableColumnWidth = (field: FieldConfig, tableData: TableRow[]): number => {
  const type = normalizeWidgetType(field.widget?.type)
  const code = (field.code || '').toLowerCase()
  const name = String(field.name || '')
  let charWidth = 0
  for (let index = 0; index < name.length; index += 1) {
    charWidth += name.charCodeAt(index) > 255 ? 16 : 10
  }
  const headerWidth = charWidth + 56

  let maxDataCharLength = 0
  let hasData = false
  for (const row of tableData) {
    const value = row[code]
    if (value === undefined || value === null || value === '') continue
    const text = String(value).trim()
    if (text === '-' || text === '[]' || text === '{}') continue
    hasData = true
    let currentLength = 0
    for (let index = 0; index < text.length; index += 1) {
      currentLength += text.charCodeAt(index) > 255 ? 14 : 8
    }
    maxDataCharLength = Math.max(maxDataCharLength, currentLength)
  }

  if (
    /^(remark|desc|description|summary|reason|cause|degrade_reason|.*_reason|source_note)$/.test(code)
    || /(备注|说明|描述|摘要|原因|理由)/.test(name)
  ) {
    if (!hasData) return Math.max(headerWidth, 90)
  }

  if (
    /^(id|created_at|updated_at|create_time|update_time|creator|updater|modifier|created_by)$/.test(code)
    || /(ID|创建|更新|修改)(时间|人)/.test(name)
  ) {
    if (type === WidgetType.DATETIME) return Math.max(headerWidth, 180)
    if (type === WidgetType.USER || type === WidgetType.USERS) return Math.max(headerWidth, 140)
    return Math.max(headerWidth, 80)
  }

  if (!hasData) return Math.max(headerWidth, 80)

  const dataBasedWidth = Math.max(headerWidth, Math.min(maxDataCharLength + 32, 350))
  let minWidth = 80
  if (type === WidgetType.DATETIME) minWidth = 180
  else if (type === WidgetType.PROGRESS || type === WidgetType.SLIDER) minWidth = 140
  else if (type === WidgetType.FILES || type === WidgetType.TEXT_AREA || type === WidgetType.RICH_TEXT) minWidth = 120
  else if (type === WidgetType.SELECT || type === WidgetType.MULTI_SELECT) minWidth = 100
  else if (
    type === WidgetType.USER
    || type === WidgetType.USERS
    || type === WidgetType.DEPARTMENT
    || type === WidgetType.DEPARTMENTS
  ) minWidth = 140

  return Math.max(headerWidth, Math.min(Math.max(minWidth, dataBasedWidth), 350))
}
