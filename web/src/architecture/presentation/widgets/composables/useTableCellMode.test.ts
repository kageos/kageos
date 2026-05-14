import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useTableCellMode } from './useTableCellMode'
import { useFormDataStore } from '@/architecture/runtime/stores/formData'

function createFieldValue(raw: any) {
  return {
    raw,
    display: raw === null || raw === undefined ? '' : String(raw),
    meta: {},
  }
}

function createProps() {
  return {
    field: {
      code: 'profile',
      name: '资料',
      widget: { type: 'form' },
    },
    value: {
      raw: {
        name: 'Alice',
      },
      display: '',
      meta: {},
    },
    mode: 'table-cell',
    parentMode: 'edit',
    fieldPath: 'items[0].profile',
  } as any
}

describe('useTableCellMode', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    useFormDataStore().clear()
  })

  it('restores subtree values when the drawer is closed without confirm', () => {
    const formDataStore = useFormDataStore()
    formDataStore.setValue('items[0].profile.name', createFieldValue('Alice') as any)

    const tableCellMode = useTableCellMode(createProps())

    tableCellMode.openDrawer()
    formDataStore.setValue('items[0].profile.name', createFieldValue('Bob') as any)
    formDataStore.setValue('items[0].profile.extra.note', createFieldValue('temp') as any)

    tableCellMode.handleDrawerClose()

    expect(formDataStore.getValue('items[0].profile.name').raw).toBe('Alice')
    expect(formDataStore.getAllFieldPaths()).not.toContain('items[0].profile.extra.note')
  })

  it('keeps subtree values after confirm', () => {
    const formDataStore = useFormDataStore()
    formDataStore.setValue('items[0].profile.name', createFieldValue('Alice') as any)

    const tableCellMode = useTableCellMode(createProps())

    tableCellMode.openDrawer()
    formDataStore.setValue('items[0].profile.name', createFieldValue('Bob') as any)

    tableCellMode.confirmDrawer()
    tableCellMode.handleDrawerClose()

    expect(formDataStore.getValue('items[0].profile.name').raw).toBe('Bob')
  })
})
