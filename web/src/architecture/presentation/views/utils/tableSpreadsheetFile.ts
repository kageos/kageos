import type { FieldConfig, TableRow } from '@/architecture/domain/types'
import {
  buildTableExportMatrix,
  buildTableImportPreview,
  describeTableSpreadsheetField,
  getTableSpreadsheetFieldExample,
  getTableSpreadsheetHeader,
  getTableSpreadsheetStaticOptionLabels,
  isTableSpreadsheetFieldSupported,
  parseCsvText,
  TABLE_IMPORT_MAX_FILE_BYTES,
  type TableImportPreview
} from './tableSpreadsheetRuntime'
import { sanitizeXlsxCommentsForImport } from './sanitizeXlsxForImport'

const MIME_XLSX = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'

const sanitizeFileName = (value: string): string => {
  const normalized = value.trim().replace(/[\\/:*?"<>|]/g, '_')
  return normalized || '表格'
}

const downloadBlob = (blob: Blob, fileName: string): void => {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  link.style.display = 'none'
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

const excelCellValue = (value: unknown): unknown => {
  if (value === null || value === undefined) return ''
  if (value instanceof Date || ['string', 'number', 'boolean'].includes(typeof value)) return value
  if (typeof value !== 'object') return String(value)

  const record = value as Record<string, unknown>
  if (Array.isArray(record.richText)) {
    return record.richText
      .map((item) => typeof item === 'object' && item ? String((item as Record<string, unknown>).text ?? '') : '')
      .join('')
  }
  if ('result' in record) return excelCellValue(record.result)
  if ('text' in record) return String(record.text ?? '')
  if ('error' in record) return String(record.error ?? '')
  return String(value)
}

const loadExcelMatrix = async (file: File): Promise<unknown[][]> => {
  const { Workbook } = await import('exceljs')
  const fileBuffer = await file.arrayBuffer()
  let workbook = new Workbook()
  try {
    await workbook.xlsx.load(fileBuffer as never)
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    if (!message.includes("reading 'comments'") && !message.toLowerCase().includes('comment')) throw error
    const sanitized = await sanitizeXlsxCommentsForImport(fileBuffer)
    workbook = new Workbook()
    await workbook.xlsx.load(sanitized as never)
  }
  const worksheet = workbook.worksheets[0]
  if (!worksheet) return []

  const matrix: unknown[][] = []
  worksheet.eachRow({ includeEmpty: true }, (row) => {
    const values: unknown[] = []
    for (let column = 1; column <= worksheet.actualColumnCount; column += 1) {
      values.push(excelCellValue(row.getCell(column).value))
    }
    matrix.push(values)
  })
  return matrix
}

const workbookBlob = async (workbook: import('exceljs').Workbook): Promise<Blob> => {
  const buffer = await workbook.xlsx.writeBuffer()
  return new Blob([buffer as unknown as BlobPart], { type: MIME_XLSX })
}

const styleHeader = (row: import('exceljs').Row): void => {
  row.height = 24
  row.font = { bold: true, color: { argb: 'FFFFFFFF' } }
  row.fill = { type: 'pattern', pattern: 'solid', fgColor: { argb: 'FF2563EB' } }
  row.alignment = { vertical: 'middle', horizontal: 'center' }
}

export const parseTableSpreadsheetFile = async (
  file: File,
  fields: FieldConfig[]
): Promise<TableImportPreview> => {
  if (file.size > TABLE_IMPORT_MAX_FILE_BYTES) {
    throw new Error(`文件不能超过 ${TABLE_IMPORT_MAX_FILE_BYTES / 1024 / 1024} MB`)
  }
  const extension = file.name.split('.').pop()?.toLowerCase()
  if (!['xlsx', 'csv'].includes(extension || '')) {
    throw new Error('仅支持 .xlsx 和 .csv 文件')
  }

  const matrix = extension === 'csv'
    ? parseCsvText(await file.text())
    : await loadExcelMatrix(file)
  return buildTableImportPreview(matrix, fields)
}

export const downloadTableImportTemplate = async (
  fields: FieldConfig[],
  tableName: string
): Promise<void> => {
  const { Workbook } = await import('exceljs')
  const workbook = new Workbook()
  workbook.creator = 'KageOS'
  workbook.created = new Date()
  const supportedFields = fields.filter(isTableSpreadsheetFieldSupported)
  const worksheet = workbook.addWorksheet('导入数据', {
    views: [{ state: 'frozen', ySplit: 1 }]
  })

  worksheet.columns = supportedFields.map((field) => ({
    header: getTableSpreadsheetHeader(field, supportedFields),
    key: field.code,
    width: Math.min(42, Math.max(14, field.name.length * 2 + 4))
  }))
  styleHeader(worksheet.getRow(1))

  supportedFields.forEach((field, index) => {
    const labels = getTableSpreadsheetStaticOptionLabels(field)
    const formula = `"${labels.join(',').replace(/"/g, '""')}"`
    const widgetType = String(field.widget?.type || '').toLowerCase()
    if (['select', 'radio'].includes(widgetType) && labels.length > 0 && formula.length <= 255) {
      for (let row = 2; row <= 501; row += 1) {
        worksheet.getCell(row, index + 1).dataValidation = {
          type: 'list',
          allowBlank: true,
          formulae: [formula],
          showErrorMessage: true,
          errorTitle: '填写内容不在可选范围',
          error: `请选择：${labels.join('、')}`
        }
      }
    }
  })

  const guide = workbook.addWorksheet('填写说明')
  guide.columns = [
    { header: '字段名称', key: 'name', width: 24 },
    { header: '字段代码', key: 'code', width: 24 },
    { header: '填写要求', key: 'guide', width: 64 },
    { header: '示例', key: 'example', width: 32 }
  ]
  styleHeader(guide.getRow(1))
  supportedFields.forEach((field) => guide.addRow({
    name: field.name,
    code: field.code,
    guide: describeTableSpreadsheetField(field),
    example: getTableSpreadsheetFieldExample(field)
  }))
  guide.getColumn(3).alignment = { wrapText: true, vertical: 'top' }

  downloadBlob(await workbookBlob(workbook), `${sanitizeFileName(tableName)}_导入模板.xlsx`)
}

export const downloadTableData = async (
  fields: FieldConfig[],
  rows: TableRow[],
  tableName: string
): Promise<void> => {
  const { Workbook } = await import('exceljs')
  const workbook = new Workbook()
  workbook.creator = 'KageOS'
  workbook.created = new Date()
  const worksheet = workbook.addWorksheet('数据', {
    views: [{ state: 'frozen', ySplit: 1 }]
  })
  const matrix = buildTableExportMatrix(fields, rows)
  matrix.forEach((values) => worksheet.addRow(values))
  if (matrix.length > 0) styleHeader(worksheet.getRow(1))
  fields.filter(isTableSpreadsheetFieldSupported).forEach((field, index) => {
    worksheet.getColumn(index + 1).width = Math.min(42, Math.max(14, field.name.length * 2 + 4))
  })

  downloadBlob(await workbookBlob(workbook), `${sanitizeFileName(tableName)}_${new Date().toISOString().slice(0, 10)}.xlsx`)
}
