import { Workbook } from 'exceljs'
import { describe, expect, it } from 'vitest'

import type { FieldConfig } from '@/architecture/domain/types'
import { parseTableSpreadsheetFile } from './tableSpreadsheetFile'

const fields: FieldConfig[] = [
  {
    code: 'name',
    name: '客户名称',
    validation: 'required',
    widget: { type: 'input' },
    meta: { dataType: 'string', isRequired: true, isReadonly: false }
  },
  {
    code: 'amount',
    name: '预计金额',
    widget: { type: 'float' },
    data: { type: 'float' }
  }
]

describe('tableSpreadsheetFile', () => {
  it('loads xlsx files with the locked ExcelJS dependency tree', async () => {
    const workbook = new Workbook()
    const worksheet = workbook.addWorksheet('客户')
    worksheet.addRow(['客户名称', '预计金额'])
    worksheet.addRow(['千言科技', 12500.5])
    const buffer = await workbook.xlsx.writeBuffer()
    const file = new File(
      [buffer as unknown as BlobPart],
      'customers.xlsx',
      { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }
    )
    Object.defineProperty(file, 'arrayBuffer', {
      value: async () => buffer
    })

    const preview = await parseTableSpreadsheetFile(file, fields)

    expect(preview.fatalErrors).toEqual([])
    expect(preview.rows).toEqual([{
      rowNumber: 2,
      data: { name: '千言科技', amount: 12500.5 },
      errors: []
    }])
  })
})
