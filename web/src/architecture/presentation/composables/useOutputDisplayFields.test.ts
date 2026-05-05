import { describe, expect, it } from 'vitest'
import { extractAllDisplayFields, extractDisplayFieldsFromToolCall } from './useOutputDisplayFields'

describe('extractDisplayFieldsFromToolCall', () => {
  it('ignores null JSON results without throwing', () => {
    expect(extractDisplayFieldsFromToolCall(undefined, 'null')).toEqual([])
  })

  it('ignores non-object result data without throwing', () => {
    expect(extractDisplayFieldsFromToolCall(undefined, undefined, null)).toEqual([])
    expect(extractDisplayFieldsFromToolCall(undefined, '[]')).toEqual([])
    expect(extractDisplayFieldsFromToolCall(undefined, '"done"')).toEqual([])
  })

  it('ignores non-object arguments without throwing', () => {
    const fields = extractDisplayFieldsFromToolCall(
      'null',
      JSON.stringify({ total: 3 })
    )

    expect(fields).toEqual([])
  })

  it('extracts display fields from result metadata', () => {
    const fields = extractDisplayFieldsFromToolCall(
      undefined,
      JSON.stringify({
        summary: 'ok',
        _display_outputs: {
          summary: { label: '摘要' }
        }
      })
    )

    expect(fields).toEqual([
      { label: '摘要', fieldKey: 'summary', value: 'ok', type: 'text' }
    ])
  })

  it('extracts display fields from output_display arguments', () => {
    const fields = extractDisplayFieldsFromToolCall(
      JSON.stringify({ output_display: { 数量: 'count' } }),
      JSON.stringify({ count: 12 })
    )

    expect(fields).toEqual([
      { label: '数量', fieldKey: 'count', value: '12', type: 'number' }
    ])
  })
})

describe('extractAllDisplayFields', () => {
  it('ignores null call entries', () => {
    expect(extractAllDisplayFields([null, { result: 'null' }])).toEqual([])
  })
})
