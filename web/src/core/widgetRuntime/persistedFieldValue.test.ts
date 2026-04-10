import { describe, expect, it } from 'vitest'
import { createPersistedFieldValue } from './persistedFieldValue'

describe('createPersistedFieldValue', () => {
  it('preserves callback labels for scalar fields when saving persisted row values', () => {
    const field = {
      code: 'product_id',
      name: '商品',
      widget: { type: 'select' },
      callbacks: ['OnSelectFuzzy'],
    } as any

    const currentValue = {
      raw: 1,
      display: '苹果汁',
      meta: {
        displayInfo: { price: '12.00' },
      },
    } as any

    expect(createPersistedFieldValue(field, 1, currentValue)).toEqual({
      raw: 1,
      display: '苹果汁',
      dataType: undefined,
      widgetType: 'select',
      meta: {
        displayInfo: { price: '12.00' },
      },
    })
  })

  it('rebuilds container displays for nested form values', () => {
    const field = {
      code: 'profile',
      name: '资料',
      widget: { type: 'form' },
    } as any

    expect(createPersistedFieldValue(field, { note: 'ok' }, {
      raw: { note: 'old' },
      display: '{"note":"old"}',
      meta: { preserved: true },
    } as any)).toEqual({
      raw: { note: 'ok' },
      display: '{"note":"ok"}',
      dataType: undefined,
      widgetType: 'form',
      meta: { preserved: true },
    })
  })
})
