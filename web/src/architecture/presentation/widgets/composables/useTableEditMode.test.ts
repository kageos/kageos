import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useTableEditMode } from './useTableEditMode'
import { useFormDataStore } from '@/architecture/presentation/context/formRuntimeContext'

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
      code: 'items',
      name: '明细',
      widget: { type: 'table' },
      children: [
        { code: 'name', name: '名称', widget: { type: 'input' } },
        { code: 'profile', name: '资料', widget: { type: 'form' } },
      ],
    },
    value: {
      raw: [
        {
          name: 'Apple',
          profile: {
            note: 'old',
          },
        },
      ],
      display: '共 1 条',
      meta: {},
    },
    mode: 'edit',
    fieldPath: 'items',
  } as any
}

describe('useTableEditMode', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    useFormDataStore().clear()
  })

  it('restores the whole edited row subtree on cancel', () => {
    const formDataStore = useFormDataStore()
    formDataStore.setValue('items', createFieldValue([
      {
        name: 'Apple',
        profile: {
          note: 'old',
        },
      },
    ]) as any)
    formDataStore.setValue('items[0].name', createFieldValue('Apple') as any)
    formDataStore.setValue('items[0].profile', {
      raw: { note: 'old' },
      display: '{"note":"old"}',
      meta: {},
    } as any)
    formDataStore.setValue('items[0].profile.note', createFieldValue('old') as any)

    const editMode = useTableEditMode(createProps())

    editMode.startEditing(0)

    formDataStore.setValue('items[0].name', createFieldValue('Banana') as any)
    formDataStore.setValue('items[0].profile', {
      raw: { note: 'new' },
      display: '{"note":"new"}',
      meta: {},
    } as any)
    formDataStore.setValue('items[0].profile.note', createFieldValue('new') as any)
    formDataStore.setValue('items[0].profile.tags[0].label', createFieldValue('temp') as any)

    editMode.cancelEditing()

    expect(formDataStore.getValue('items[0].name').raw).toBe('Apple')
    expect(formDataStore.getValue('items[0].profile.note').raw).toBe('old')
    expect(formDataStore.getAllFieldPaths()).not.toContain('items[0].profile.tags[0].label')
  })
})
