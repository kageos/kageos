import { Workbook } from 'exceljs'
import { describe, expect, it } from 'vitest'

import type { FieldConfig } from '@/architecture/domain/types'
import {
  buildTableExportArchiveFileName,
  buildTableExportFileName,
  parseTableSpreadsheetFile
} from './tableSpreadsheetFile'

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
  it('uses distinct and explicit file names for current-page and full filtered exports', () => {
    const date = new Date('2026-08-27T08:00:00.000Z')

    expect(buildTableExportFileName('客户/列表', {
      scope: 'current-page',
      currentPage: 3,
      date
    })).toBe('客户_列表_当前列表_第3页_2026-08-27.xlsx')
    expect(buildTableExportFileName('客户列表', {
      scope: 'all-filtered',
      rangeStart: 10001,
      rangeEnd: 20000,
      date
    })).toBe('客户列表_全部数据_当前筛选_第10001-20000条_2026-08-27.xlsx')
    expect(buildTableExportArchiveFileName('客户/列表', date))
      .toBe('客户_列表_全部数据_当前筛选_2026-08-27.zip')
  })

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
