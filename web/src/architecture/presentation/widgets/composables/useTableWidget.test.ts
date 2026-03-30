import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useTableWidget } from './useTableWidget'
import { useFormDataStore } from '@/core/stores-v2/formData'

function createProps() {
  return {
    field: {
      code: 'contacts',
      name: '联系人',
      widget: { type: 'table' },
      children: [
        { code: 'name', name: '姓名', widget: { type: 'input' } },
        { code: 'phone', name: '电话', widget: { type: 'input' } }
      ]
    },
    value: {
      raw: [
        { name: 'Alice', phone: '10086' },
        { name: 'Bob', phone: '10010' }
      ],
      display: '共 2 条',
      meta: {}
    },
    mode: 'table-cell',
    fieldPath: 'profile.address.contacts'
  } as any
}

describe('useTableWidget', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    useFormDataStore().clear()
  })

  it('falls back to raw row values when nested row paths are not hydrated into store', () => {
    const { getRowFieldValue } = useTableWidget(createProps())

    expect(getRowFieldValue(0, 'name')).toEqual({
      raw: 'Alice',
      display: 'Alice',
      meta: {}
    })
    expect(getRowFieldValue(1, 'phone')).toEqual({
      raw: '10010',
      display: '10010',
      meta: {}
    })
  })

  it('prefers store values when a row field has already been edited', () => {
    const formDataStore = useFormDataStore()
    formDataStore.setValue('profile.address.contacts[0].name', {
      raw: 'Alice Updated',
      display: 'Alice Updated',
      meta: {}
    } as any)

    const { getRowFieldValue } = useTableWidget(createProps())

    expect(getRowFieldValue(0, 'name')).toEqual({
      raw: 'Alice Updated',
      display: 'Alice Updated',
      meta: {}
    })
  })
})
