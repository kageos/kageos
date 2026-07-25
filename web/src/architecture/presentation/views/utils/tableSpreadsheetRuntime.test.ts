import { describe, expect, it } from 'vitest'
import type { FieldConfig } from '@/architecture/domain/types'
import {
  buildTableImportPreview,
  coerceTableImportValue,
  describeTableSpreadsheetField,
  isTableImportPreviewSubmittable,
  isTableSpreadsheetFieldSupported,
  parseCsvText
} from './tableSpreadsheetRuntime'

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
  },
  {
    code: 'status',
    name: '状态',
    widget: {
      type: 'select',
      config: { options: [{ label: '跟进中', value: 'active' }, { label: '已结束', value: 'closed' }] }
    }
  }
]

describe('tableSpreadsheetRuntime', () => {
  it('maps display headers and option labels to callback payload values', () => {
    const preview = buildTableImportPreview([
      ['客户名称', '预计金额', '状态', '无关列'],
      ['千言科技', '12,500.50', '跟进中', 'ignore']
    ], fields)

    expect(preview.fatalErrors).toEqual([])
    expect(preview.ignoredHeaders).toEqual(['无关列'])
    expect(preview.rows).toEqual([{
      rowNumber: 2,
      data: { name: '千言科技', amount: 12500.5, status: 'active' },
      errors: []
    }])
  })

  it('keeps invalid rows visible instead of silently sending them', () => {
    const preview = buildTableImportPreview([
      ['客户名称', '预计金额'],
      ['', 'not-a-number']
    ], fields)

    expect(preview.rows[0]!.errors).toEqual([
      '预计金额：请填写有效数字',
      '客户名称：必填'
    ])
    expect(isTableImportPreviewSubmittable(preview)).toBe(false)
  })

  it('only allows submission when every parsed row passes validation', () => {
    const preview = buildTableImportPreview([
      ['客户名称', '预计金额', '状态'],
      ['千言科技', '12500.50', '跟进中'],
      ['北洛科技', '6800', '已结束']
    ], fields)

    expect(isTableImportPreviewSubmittable(preview)).toBe(true)

    preview.rows[1]!.errors.push('客户名称：必填')
    expect(isTableImportPreviewSubmittable(preview)).toBe(false)
  })

  it('rejects values outside static options', () => {
    expect(() => coerceTableImportValue(fields[2]!, '未知')).toThrow('可选值')
  })

  it('parses quoted CSV cells, commas and line breaks', () => {
    expect(parseCsvText('客户名称,备注\r\n"千言,科技","第一行\n第二行"')).toEqual([
      ['客户名称', '备注'],
      ['千言,科技', '第一行\n第二行']
    ])
  })

  it('explains special field formats from schema and excludes uploads', () => {
    expect(describeTableSpreadsheetField({
      code: 'owners',
      name: '协作人',
      widget: { type: 'users' },
      data: { type: '[]string' }
    })).toContain('系统登录账号，多个账号用逗号分隔')
    expect(describeTableSpreadsheetField({
      code: 'enabled',
      name: '是否启用',
      widget: { type: 'switch' }
    })).toContain('填写“是”或“否”')
    expect(isTableSpreadsheetFieldSupported({
      code: 'files',
      name: '附件',
      widget: { type: 'files' }
    })).toBe(false)
  })
})
