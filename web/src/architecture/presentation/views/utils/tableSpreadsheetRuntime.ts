import dayjs from 'dayjs'
import type { FieldConfig, TableRow } from '@/architecture/domain/types'

export const TABLE_IMPORT_MAX_ROWS = 500
export const TABLE_IMPORT_MAX_COLUMNS = 120
export const TABLE_IMPORT_MAX_FILE_BYTES = 8 * 1024 * 1024
export const TABLE_EXPORT_MAX_ROWS = 10000

export interface TableImportRow {
  rowNumber: number
  data: Record<string, unknown>
  errors: string[]
}

export interface TableImportPreview {
  rows: TableImportRow[]
  recognizedFields: FieldConfig[]
  ignoredHeaders: string[]
  fatalErrors: string[]
}

/**
 * 导入必须整批通过前端可确定的校验后才能提交。
 * 不允许静默跳过错误行，否则用户看到的文件行数会和实际写入结果不一致。
 */
export const isTableImportPreviewSubmittable = (preview: TableImportPreview): boolean => {
  return preview.fatalErrors.length === 0
    && preview.rows.length > 0
    && preview.rows.every((row) => row.errors.length === 0)
}

interface StaticOption {
  label: string
  value: unknown
}

const UNSUPPORTED_SPREADSHEET_WIDGETS = new Set(['table', 'form', 'list', 'files', 'link'])
const NUMBER_WIDGETS = new Set(['integer', 'float', 'slider', 'rate', 'progress'])
const MULTIPLE_WIDGETS = new Set(['checkbox', 'multiselect', 'users', 'departments'])

const isBlank = (value: unknown): boolean => {
  return value === null || value === undefined || (typeof value === 'string' && value.trim() === '')
}

const normalizeHeader = (value: unknown): string => String(value ?? '').replace(/^\uFEFF/, '').trim()

const getStaticOptions = (field: FieldConfig): StaticOption[] => {
  const options = field.widget?.config?.options
  if (!Array.isArray(options)) return []

  return options.map((option) => {
    if (option && typeof option === 'object' && !Array.isArray(option)) {
      const record = option as Record<string, unknown>
      return {
        label: String(record.label ?? record.name ?? record.value ?? ''),
        value: record.value ?? record.code ?? record.label ?? record.name ?? ''
      }
    }
    return { label: String(option), value: option }
  })
}

const isRequiredField = (field: FieldConfig): boolean => {
  return field.meta?.isRequired === true || String(field.validation || '')
    .split(',')
    .some((rule) => rule.trim() === 'required')
}

export const isTableSpreadsheetFieldSupported = (field: FieldConfig): boolean => {
  return !UNSUPPORTED_SPREADSHEET_WIDGETS.has(String(field.widget?.type || '').toLowerCase())
}

export const isTableSpreadsheetFieldRequired = isRequiredField

export const getTableSpreadsheetHeader = (field: FieldConfig, fields: FieldConfig[]): string => {
  const duplicateName = fields.some((candidate) => candidate !== field && candidate.name === field.name)
  return duplicateName ? `${field.name}（${field.code}）` : field.name
}

const coerceBoolean = (value: unknown): boolean => {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number' && (value === 0 || value === 1)) return value === 1
  const normalized = String(value).trim().toLowerCase()
  if (['true', '1', '是', '开启', '启用', 'yes', 'y'].includes(normalized)) return true
  if (['false', '0', '否', '关闭', '禁用', 'no', 'n'].includes(normalized)) return false
  throw new Error('请填写“是/否”或 true/false')
}

const coerceNumber = (value: unknown, integer: boolean): number => {
  if (typeof value === 'string' && value.trim() === '') throw new Error('不能为空')
  const number = typeof value === 'number' ? value : Number(String(value).replace(/,/g, '').trim())
  if (!Number.isFinite(number)) throw new Error('请填写有效数字')
  if (integer && !Number.isInteger(number)) throw new Error('请填写整数')
  return number
}

const resolveStaticOption = (value: unknown, options: StaticOption[]): unknown => {
  if (options.length === 0) return value
  const normalized = String(value).trim()
  const matched = options.find((option) => (
    String(option.label).trim() === normalized || String(option.value).trim() === normalized
  ))
  if (!matched) {
    throw new Error(`可选值：${options.map((option) => option.label).join('、')}`)
  }
  return matched.value
}

const coerceMultiple = (field: FieldConfig, value: unknown): unknown => {
  const parts = Array.isArray(value)
    ? value
    : String(value).split(/[,，;；\n]/).map((part) => part.trim()).filter(Boolean)
  const options = getStaticOptions(field)
  const normalized = parts.map((part) => resolveStaticOption(part, options))
  return String(field.data?.type || '').startsWith('[]') ? normalized : normalized.join(',')
}

const coerceDateTime = (value: unknown): string => {
  const parsed = dayjs(value as string | number | Date)
  if (!parsed.isValid()) throw new Error('请填写有效日期时间')
  return parsed.format('YYYY-MM-DD HH:mm:ss')
}

const validateBounds = (field: FieldConfig, value: unknown): void => {
  if (typeof value === 'string') {
    if (field.meta?.minLength !== undefined && value.length < field.meta.minLength) {
      throw new Error(`至少填写 ${field.meta.minLength} 个字符`)
    }
    if (field.meta?.maxLength !== undefined && value.length > field.meta.maxLength) {
      throw new Error(`最多填写 ${field.meta.maxLength} 个字符`)
    }
  }
  if (typeof value === 'number') {
    if (field.meta?.min !== undefined && value < field.meta.min) {
      throw new Error(`不能小于 ${field.meta.min}`)
    }
    if (field.meta?.max !== undefined && value > field.meta.max) {
      throw new Error(`不能大于 ${field.meta.max}`)
    }
  }
}

export const coerceTableImportValue = (field: FieldConfig, value: unknown): unknown => {
  const widgetType = String(field.widget?.type || 'input').toLowerCase()
  const dataType = String(field.data?.type || '').toLowerCase()
  let normalized: unknown

  if (widgetType === 'switch' || dataType === 'bool' || dataType === 'boolean') {
    normalized = coerceBoolean(value)
  } else if (MULTIPLE_WIDGETS.has(widgetType) || dataType.startsWith('[]')) {
    normalized = coerceMultiple(field, value)
  } else if (widgetType === 'datetime' || field.data?.format === 'date-time') {
    normalized = coerceDateTime(value)
  } else if (NUMBER_WIDGETS.has(widgetType) || ['int', 'integer', 'float', 'number'].includes(dataType)) {
    normalized = coerceNumber(value, widgetType === 'integer' || ['int', 'integer'].includes(dataType))
  } else if (['select', 'radio'].includes(widgetType)) {
    normalized = resolveStaticOption(value, getStaticOptions(field))
    if (['int', 'integer'].includes(dataType)) normalized = coerceNumber(normalized, true)
    if (['float', 'number'].includes(dataType)) normalized = coerceNumber(normalized, false)
  } else if (typeof value === 'string') {
    normalized = value.trim()
  } else {
    normalized = value
  }

  validateBounds(field, normalized)
  return normalized
}

const createHeaderLookup = (fields: FieldConfig[]): Map<string, FieldConfig> => {
  const lookup = new Map<string, FieldConfig>()
  const nameCounts = new Map<string, number>()
  fields.forEach((field) => nameCounts.set(field.name, (nameCounts.get(field.name) || 0) + 1))

  fields.forEach((field) => {
    lookup.set(field.code, field)
    lookup.set(getTableSpreadsheetHeader(field, fields), field)
    if (nameCounts.get(field.name) === 1) lookup.set(field.name, field)
  })
  return lookup
}

export const buildTableImportPreview = (
  matrix: unknown[][],
  fields: FieldConfig[]
): TableImportPreview => {
  const supportedFields = fields.filter(isTableSpreadsheetFieldSupported)
  const fatalErrors: string[] = []
  if (matrix.length === 0) {
    return { rows: [], recognizedFields: [], ignoredHeaders: [], fatalErrors: ['文件中没有可读取的数据'] }
  }

  const headerRowIndex = matrix.findIndex((row) => row.some((cell) => !isBlank(cell)))
  if (headerRowIndex < 0) {
    return { rows: [], recognizedFields: [], ignoredHeaders: [], fatalErrors: ['文件中没有表头'] }
  }

  const headers = matrix[headerRowIndex]!.map(normalizeHeader)
  if (headers.length > TABLE_IMPORT_MAX_COLUMNS) {
    fatalErrors.push(`最多支持 ${TABLE_IMPORT_MAX_COLUMNS} 列`)
  }
  const duplicates = headers.filter((header, index) => header && headers.indexOf(header) !== index)
  if (duplicates.length > 0) {
    fatalErrors.push(`表头重复：${[...new Set(duplicates)].join('、')}`)
  }

  const lookup = createHeaderLookup(supportedFields)
  const fieldByColumn = headers.map((header) => lookup.get(header))
  const recognizedFields = [...new Set(fieldByColumn.filter((field): field is FieldConfig => Boolean(field)))]
  const ignoredHeaders = headers.filter((header, index) => header && !fieldByColumn[index])
  if (recognizedFields.length === 0) {
    fatalErrors.push('没有找到可导入的字段，请先下载当前表格的模板')
  }

  const sourceRows = matrix.slice(headerRowIndex + 1)
    .map((row, offset) => ({ row, rowNumber: headerRowIndex + offset + 2 }))
    .filter(({ row }) => row.some((cell) => !isBlank(cell)))
  if (sourceRows.length > TABLE_IMPORT_MAX_ROWS) {
    fatalErrors.push(`一次最多导入 ${TABLE_IMPORT_MAX_ROWS} 行，当前文件有 ${sourceRows.length} 行`)
  }

  const rows = sourceRows.slice(0, TABLE_IMPORT_MAX_ROWS).map(({ row, rowNumber }) => {
    const data: Record<string, unknown> = {}
    const errors: string[] = []

    fieldByColumn.forEach((field, columnIndex) => {
      if (!field) return
      const rawValue = row[columnIndex]
      if (isBlank(rawValue)) return
      try {
        data[field.code] = coerceTableImportValue(field, rawValue)
      } catch (error) {
        errors.push(`${field.name}：${error instanceof Error ? error.message : String(error)}`)
      }
    })

    recognizedFields.forEach((field) => {
      if (isRequiredField(field) && isBlank(data[field.code])) {
        errors.push(`${field.name}：必填`)
      }
    })

    return { rowNumber, data, errors }
  })

  return { rows, recognizedFields, ignoredHeaders, fatalErrors }
}

export const parseCsvText = (source: string): string[][] => {
  const rows: string[][] = []
  let row: string[] = []
  let cell = ''
  let quoted = false
  const input = source.replace(/^\uFEFF/, '')

  for (let index = 0; index < input.length; index += 1) {
    const char = input[index]
    if (quoted) {
      if (char === '"' && input[index + 1] === '"') {
        cell += '"'
        index += 1
      } else if (char === '"') {
        quoted = false
      } else {
        cell += char
      }
    } else if (char === '"') {
      quoted = true
    } else if (char === ',') {
      row.push(cell)
      cell = ''
    } else if (char === '\n') {
      row.push(cell.replace(/\r$/, ''))
      rows.push(row)
      row = []
      cell = ''
    } else {
      cell += char
    }
  }

  if (quoted) throw new Error('CSV 中存在未闭合的引号')
  if (cell !== '' || row.length > 0) {
    row.push(cell.replace(/\r$/, ''))
    rows.push(row)
  }
  return rows
}

const formatExportValue = (value: unknown): string | number | boolean | Date => {
  if (value === null || value === undefined) return ''
  if (value instanceof Date || ['string', 'number', 'boolean'].includes(typeof value)) {
    return value as string | number | boolean | Date
  }
  if (Array.isArray(value)) return value.map((item) => String(item)).join(', ')
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

export const buildTableExportMatrix = (fields: FieldConfig[], rows: TableRow[]): Array<Array<string | number | boolean | Date>> => {
  const supportedFields = fields.filter(isTableSpreadsheetFieldSupported)
  return [
    supportedFields.map((field) => getTableSpreadsheetHeader(field, supportedFields)),
    ...rows.map((row) => supportedFields.map((field) => formatExportValue(row[field.code])))
  ]
}

export const describeTableSpreadsheetField = (field: FieldConfig): string => {
  const parts: string[] = []
  if (isRequiredField(field)) parts.push('必填')
  const widgetType = String(field.widget?.type || 'input').toLowerCase()
  const dataType = String(field.data?.type || '').toLowerCase()
  if (widgetType === 'switch' || ['bool', 'boolean'].includes(dataType)) {
    parts.push('填写“是”或“否”')
  } else if (widgetType === 'datetime' || field.data?.format === 'date-time') {
    parts.push('格式：YYYY-MM-DD HH:mm:ss')
  } else if (widgetType === 'user') {
    parts.push('填写系统登录账号，不要填写姓名或昵称')
  } else if (widgetType === 'users') {
    parts.push('填写系统登录账号，多个账号用逗号分隔')
  } else if (widgetType === 'department') {
    parts.push('填写部门完整路径')
  } else if (widgetType === 'departments') {
    parts.push('填写部门完整路径，多个部门用逗号分隔')
  } else if (MULTIPLE_WIDGETS.has(widgetType) || dataType.startsWith('[]')) {
    parts.push('多个值用逗号分隔')
  } else if (widgetType === 'integer' || ['int', 'integer'].includes(dataType)) {
    parts.push('填写整数，不要带单位')
  } else if (NUMBER_WIDGETS.has(widgetType) || ['float', 'number'].includes(dataType)) {
    parts.push('填写数字，不要带货币符号或单位')
  }
  const options = getStaticOptions(field)
  if (options.length > 0) parts.push(`可选：${options.map((option) => option.label).join('、')}`)
  if (field.callbacks?.length) parts.push('动态选择字段请填写系统中的原始值或 ID')
  if (field.desc) parts.push(field.desc)
  return parts.join('；') || '按页面新增表单的要求填写'
}

export const getTableSpreadsheetFieldTypeLabel = (field: FieldConfig): string => {
  const widgetType = String(field.widget?.type || 'input').toLowerCase()
  const labels: Record<string, string> = {
    input: '文字', text: '文字', text_area: '长文字', richtext: '长文字',
    integer: '整数', float: '数字', slider: '数字', rate: '数字', progress: '数字',
    switch: '是 / 否', datetime: '日期时间', select: '单选', radio: '单选',
    checkbox: '多选', multiselect: '多选', user: '用户账号', users: '多个用户账号',
    department: '部门路径', departments: '多个部门路径', color: '颜色'
  }
  return labels[widgetType] || field.data?.type || '文字'
}

export const getTableSpreadsheetFieldExample = (field: FieldConfig): string => {
  const widgetType = String(field.widget?.type || 'input').toLowerCase()
  const options = getStaticOptions(field)
  if (field.callbacks?.length) return '123'
  if (options.length > 0) {
    const count = MULTIPLE_WIDGETS.has(widgetType) ? Math.min(2, options.length) : 1
    return options.slice(0, count).map((option) => option.label).join(',')
  }
  if (widgetType === 'switch') return '是'
  if (widgetType === 'datetime') return '2026-07-21 14:30:00'
  if (widgetType === 'user') return 'zhangsan'
  if (widgetType === 'users') return 'zhangsan,lisi'
  if (widgetType === 'department') return '/销售部'
  if (widgetType === 'departments') return '/销售部,/客户成功部'
  if (widgetType === 'integer') return '100'
  if (NUMBER_WIDGETS.has(widgetType)) return '12500.50'
  return field.data?.example || '按业务内容填写'
}

export const getTableSpreadsheetStaticOptionLabels = (field: FieldConfig): string[] => {
  return getStaticOptions(field).map((option) => option.label)
}
