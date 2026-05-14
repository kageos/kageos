import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useTableWidget } from './useTableWidget'
import { useFormDataStore } from '@/architecture/runtime/stores-v2/formData'

function createFieldValue(raw: any, meta: Record<string, any> = {}) {
  return {
    raw,
    display: raw === null || raw === undefined ? '' : String(raw),
    meta,
  }
}

function createProps() {
  return {
    field: {
      code: 'items',
      name: '明细',
      widget: { type: 'table' },
      children: [
        { code: 'topic_id', name: '主题', widget: { type: 'input' } },
        { code: 'option_ids', name: '选项', widget: { type: 'multiselect' }, depend_on: 'topic_id' },
        {
          code: 'extra',
          name: '扩展',
          widget: { type: 'form' },
          depend_on: 'topic_id',
          children: [
            { code: 'note', name: '备注', widget: { type: 'input' } },
          ],
        },
      ],
    },
    value: {
      raw: [
        {
          topic_id: 't1',
          option_ids: ['o1'],
          extra: {
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

describe('useTableWidget', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    useFormDataStore().clear()
  })

  it('clears dependent fields within the edited table row scope', () => {
    const formDataStore = useFormDataStore()
    formDataStore.setValue('items', createFieldValue([
      {
        topic_id: 't1',
        option_ids: ['o1'],
        extra: {
          note: 'old',
        },
      },
    ], { preserved: true }) as any)
    formDataStore.setValue('items[0].option_ids', createFieldValue(['o1'], { preserved: true }) as any)
    formDataStore.setValue('items[0].extra', {
      raw: { note: 'old' },
      display: '{"note":"old"}',
      meta: { preserved: true },
    } as any)
    formDataStore.setValue('items[0].extra.note', createFieldValue('old') as any)

    const { updateRowFieldValue } = useTableWidget(createProps())

    updateRowFieldValue(0, 'topic_id', createFieldValue('t2') as any)

    expect(formDataStore.getValue('items[0].option_ids')).toEqual({
      raw: null,
      display: '',
      dataType: undefined,
      widgetType: 'multiselect',
      meta: { preserved: true },
    })
    expect(formDataStore.getValue('items[0].extra')).toEqual({
      raw: {},
      display: '',
      dataType: undefined,
      widgetType: 'form',
      meta: { preserved: true },
    })
    expect(formDataStore.getValue('items')).toEqual({
      raw: [
        {
          topic_id: 't2',
          option_ids: null,
          extra: {},
        },
      ],
      display: '共 1 条',
      dataType: undefined,
      widgetType: 'table',
      meta: { preserved: true },
    })
    expect(formDataStore.getAllFieldPaths()).not.toContain('items[0].extra.note')
  })
})
