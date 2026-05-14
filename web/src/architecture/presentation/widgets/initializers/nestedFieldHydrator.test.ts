import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useFormDataStore } from '@/architecture/runtime/stores/formData'
import { FormWidgetInitializer } from './FormWidgetInitializer'
import { TableWidgetInitializer } from './TableWidgetInitializer'
import { reindexTableRowFieldPaths } from '../utils/tableFieldPathSync'

function createFunctionDetail() {
  return {
    id: 1,
    router: '/test',
    method: 'POST',
    schema: {
      version: 1,
      type: 'form',
      form: {
        request: [],
        response: []
      }
    }
  } as any
}

describe('nestedFieldHydrator', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    useFormDataStore().clear()
  })

  it('hydrates nested form field values without quick-link metadata', async () => {
    const initializer = new FormWidgetInitializer()
    const formDataStore = useFormDataStore()

    const field = {
      code: 'profile',
      name: '资料',
      widget: { type: 'form' },
      children: [
        { code: 'name', name: '姓名', widget: { type: 'input' }, data: { type: 'string' } },
        { code: 'age', name: '年龄', widget: { type: 'input' }, data: { type: 'int' } }
      ]
    } as any

    const initializedValue = await initializer.initialize({
      field,
      currentValue: {
        raw: { name: 'Alice', age: '18' },
        display: '',
        meta: {}
      },
      allFormData: {},
      functionDetail: createFunctionDetail(),
      initSource: 'default',
      fieldPath: 'profile'
    })

    expect(initializedValue?.raw).toEqual({ name: 'Alice', age: 18 })
    expect(initializedValue?.meta).toEqual({})
    expect(formDataStore.getValue('profile.name').raw).toBe('Alice')
    expect(formDataStore.getValue('profile.age').raw).toBe(18)
  })

  it('hydrates table rows into row field paths', async () => {
    const initializer = new TableWidgetInitializer()
    const formDataStore = useFormDataStore()

    const field = {
      code: 'items',
      name: '明细',
      widget: { type: 'table' },
      children: [
        { code: 'name', name: '名称', widget: { type: 'input' }, data: { type: 'string' } },
        { code: 'qty', name: '数量', widget: { type: 'input' }, data: { type: 'int' } }
      ]
    } as any

    const initializedValue = await initializer.initialize({
      field,
      currentValue: {
        raw: [
          { name: 'Apple', qty: '2' },
          { name: 'Banana', qty: '3' }
        ],
        display: '',
        meta: {}
      },
      allFormData: {},
      functionDetail: createFunctionDetail(),
      initSource: 'default',
      fieldPath: 'items'
    })

    expect(initializedValue?.raw).toEqual([
      { name: 'Apple', qty: 2 },
      { name: 'Banana', qty: 3 }
    ])
    expect(formDataStore.getValue('items[0].name').raw).toBe('Apple')
    expect(formDataStore.getValue('items[0].qty').raw).toBe(2)
    expect(formDataStore.getValue('items[1].name').raw).toBe('Banana')
    expect(formDataStore.getValue('items[1].qty').raw).toBe(3)
  })

  it('hydrates deep form -> form -> table paths for edit rendering and submit extraction', async () => {
    const initializer = new FormWidgetInitializer()
    const formDataStore = useFormDataStore()

    const field = {
      code: 'profile',
      name: '资料',
      widget: { type: 'form' },
      children: [
        {
          code: 'address',
          name: '地址',
          widget: { type: 'form' },
          children: [
            {
              code: 'contacts',
              name: '联系人',
              widget: { type: 'table' },
              children: [
                { code: 'name', name: '姓名', widget: { type: 'input' }, data: { type: 'string' } },
                { code: 'phone', name: '电话', widget: { type: 'input' }, data: { type: 'string' } }
              ]
            }
          ]
        }
      ]
    } as any

    const initializedValue = await initializer.initialize({
      field,
      currentValue: {
        raw: {
          address: {
            contacts: [
              { name: 'Alice', phone: '10086' },
              { name: 'Bob', phone: '10010' }
            ]
          }
        },
        display: '',
        meta: {}
      },
      allFormData: {},
      functionDetail: createFunctionDetail(),
      initSource: 'default',
      fieldPath: 'profile'
    })

    expect(initializedValue?.raw).toEqual({
      address: {
        contacts: [
          { name: 'Alice', phone: '10086' },
          { name: 'Bob', phone: '10010' }
        ]
      }
    })
    expect(formDataStore.getValue('profile.address.contacts[0].name').raw).toBe('Alice')
    expect(formDataStore.getValue('profile.address.contacts[1].phone').raw).toBe('10010')
    expect(formDataStore.getSubmitData([field])).toEqual({
      profile: {
        address: {
          contacts: [
            { name: 'Alice', phone: '10086' },
            { name: 'Bob', phone: '10010' }
          ]
        }
      }
    })
  })

  it('reindexes table row field paths after deleting a row', () => {
    const formDataStore = useFormDataStore()

    formDataStore.setValue('items[0].name', { raw: 'Apple', display: 'Apple', meta: {} } as any)
    formDataStore.setValue('items[1].name', { raw: 'Banana', display: 'Banana', meta: {} } as any)
    formDataStore.setValue('items[1].detail.note', { raw: 'keep', display: 'keep', meta: {} } as any)

    reindexTableRowFieldPaths(formDataStore, 'items', 0)

    expect(formDataStore.getAllFieldPaths()).not.toContain('items[1].name')
    expect(formDataStore.getAllFieldPaths()).not.toContain('items[1].detail.note')
    expect(formDataStore.getValue('items[0].name').raw).toBe('Banana')
    expect(formDataStore.getValue('items[0].detail.note').raw).toBe('keep')
  })
})
