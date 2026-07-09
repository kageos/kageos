import { describe, expect, it } from 'vitest'
import type { FunctionDetail } from '@/architecture/domain/types'
import {
  buildPublicSharePresetValues,
  lockFunctionDetailPresetFields,
} from './publicSharePreset'

describe('publicSharePreset', () => {
  it('keeps only non-empty preset values', () => {
    expect(buildPublicSharePresetValues({
      topic_id: 12,
      title: '',
      enabled: false,
      count: 0,
      tags: [],
      note: 'hello',
    })).toEqual({
      topic_id: 12,
      enabled: false,
      count: 0,
      note: 'hello',
    })
  })

  it('locks request fields present in preset values without mutating the original detail', () => {
    const detail: FunctionDetail = {
      id: 1,
      router: '/alice/demo/vote_submit.form',
      template_type: 'form',
      schema: {
        version: 1,
        type: 'form',
        form: {
          request: [
            {
              code: 'topic_id',
              name: '投票主题',
              widget: { type: 'select', config: { placeholder: '请选择' } },
            },
            {
              code: 'remark',
              name: '备注',
              widget: { type: 'text_area', config: {} },
            },
          ],
          response: [],
        },
      },
    }

    const locked = lockFunctionDetailPresetFields(detail, { topic_id: 12 })

    expect(locked.schema?.form?.request[0]?.widget.config?.disabled).toBe(true)
    expect(locked.schema?.form?.request[1]?.widget.config?.disabled).toBeUndefined()
    expect(detail.schema?.form?.request[0]?.widget.config?.disabled).toBeUndefined()
  })
})
