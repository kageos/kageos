import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useFormDataStore } from '@/architecture/infrastructure/stores/formData'
import {
  clearFieldSubtree,
  createClearedFieldValue,
  getTableRowScopedFieldValue,
  shouldShowTableRowField,
} from './tableRowVisibility'

function createFieldValue(raw: any) {
  return {
    raw,
    display: raw === null || raw === undefined ? '' : String(raw),
    meta: {},
  }
}

describe('tableRowVisibility', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    useFormDataStore().clear()
  })

  it('resolves row-scoped sibling conditions with store values taking precedence', () => {
    const formDataStore = useFormDataStore()
    formDataStore.setValue('items[0].member_type', createFieldValue('vip') as any)

    const allFields = [
      {
        code: 'member_type',
        field_name: 'MemberType',
        name: '会员类型',
        widget: { type: 'input' },
      },
      {
        code: 'card_no',
        name: '卡号',
        widget: { type: 'input' },
        validation: 'required_if=MemberType vip',
      },
    ] as any

    expect(shouldShowTableRowField(
      formDataStore,
      'items',
      0,
      { member_type: 'normal', card_no: '' },
      allFields[1],
      allFields
    )).toBe(true)

    expect(getTableRowScopedFieldValue(
      formDataStore,
      'items',
      0,
      { member_type: 'normal' },
      'member_type'
    )).toEqual({
      raw: 'vip',
      display: 'vip',
      meta: {},
    })
  })

  it('clears a whole field subtree and creates structured empty values', () => {
    const formDataStore = useFormDataStore()
    formDataStore.setValue('items[0].profile', {
      raw: { note: 'old' },
      display: '{"note":"old"}',
      meta: {},
    } as any)
    formDataStore.setValue('items[0].profile.note', createFieldValue('old') as any)
    formDataStore.setValue('items[0].profile.tags[0].label', createFieldValue('tag') as any)

    clearFieldSubtree(formDataStore, 'items[0].profile')

    expect(formDataStore.getAllFieldPaths()).not.toContain('items[0].profile')
    expect(formDataStore.getAllFieldPaths()).not.toContain('items[0].profile.note')
    expect(formDataStore.getAllFieldPaths()).not.toContain('items[0].profile.tags[0].label')

    expect(createClearedFieldValue({
      code: 'profile',
      name: '资料',
      widget: { type: 'form' },
    } as any)).toEqual({
      raw: {},
      display: '',
      dataType: undefined,
      widgetType: 'form',
      meta: {},
    })
  })
})
