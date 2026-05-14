import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useFormWidget } from './useFormWidget'
import { useFormDataStore } from '@/architecture/infrastructure/stores/formData'

function createFieldValue(raw: any) {
  return {
    raw,
    display: raw === null || raw === undefined ? '' : String(raw),
    meta: {},
  }
}

function createProps(overrides: Record<string, any> = {}) {
  return {
    field: {
      code: 'profile',
      name: '资料',
      widget: { type: 'form' },
      children: [
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
          depend_on: 'member_type',
          validation: 'required_if=MemberType vip',
        },
        {
          code: 'attachments',
          name: '附件',
          widget: { type: 'form' },
          depend_on: 'member_type',
          children: [
            {
              code: 'note',
              name: '备注',
              widget: { type: 'input' },
            },
          ],
        },
      ],
    },
    value: {
      raw: {
        member_type: 'vip',
        card_no: 'A001',
        attachments: {
          note: 'old',
        },
      },
      display: '',
      meta: {},
    },
    mode: 'table-cell',
    parentMode: 'edit',
    fieldPath: 'profile',
    ...overrides,
  } as any
}

describe('useFormWidget', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    useFormDataStore().clear()
  })

  it('falls back to raw values in table-cell edit context and then prefers store values', () => {
    const { getSubFieldValue, updateSubFieldValue } = useFormWidget(createProps())

    expect(getSubFieldValue('member_type')).toEqual({
      raw: 'vip',
      display: 'vip',
      meta: {},
    })

    updateSubFieldValue('member_type', createFieldValue('svip') as any)

    expect(getSubFieldValue('member_type')).toEqual({
      raw: 'svip',
      display: 'svip',
      meta: {},
    })
    expect(getSubFieldValue('card_no').raw).toBeNull()
    expect(getSubFieldValue('card_no').display).toBe('')
  })

  it('filters nested fields by conditional visibility rules within the current container scope', () => {
    const formDataStore = useFormDataStore()
    formDataStore.setValue('profile.member_type', createFieldValue('normal') as any)

    const hiddenResult = useFormWidget(createProps({
      mode: 'edit',
      value: {
        raw: {
          member_type: 'normal',
          card_no: '',
        },
        display: '',
        meta: {},
      },
    }))

    expect(hiddenResult.visibleSubFields.value.map((field) => field.code)).toEqual([
      'member_type',
      'attachments',
    ])

    formDataStore.setValue('profile.member_type', createFieldValue('vip') as any)

    const visibleResult = useFormWidget(createProps({
      mode: 'edit',
      value: {
        raw: {
          member_type: 'vip',
          card_no: '',
        },
        display: '',
        meta: {},
      },
    }))

    expect(visibleResult.visibleSubFields.value.map((field) => field.code)).toEqual([
      'member_type',
      'card_no',
      'attachments',
    ])
  })

  it('clears dependent sibling subtrees inside the current form scope', () => {
    const formDataStore = useFormDataStore()
    formDataStore.setValue('profile.card_no', createFieldValue('A001') as any)
    formDataStore.setValue('profile.attachments', {
      raw: { note: 'old' },
      display: '{"note":"old"}',
      meta: { preserved: true },
    } as any)
    formDataStore.setValue('profile.attachments.note', createFieldValue('old') as any)

    const { updateSubFieldValue } = useFormWidget(createProps({ mode: 'edit' }))

    updateSubFieldValue('member_type', createFieldValue('normal') as any)

    expect(formDataStore.getValue('profile.card_no').raw).toBeNull()
    expect(formDataStore.getValue('profile.card_no').display).toBe('')
    expect(formDataStore.getValue('profile.attachments')).toEqual({
      raw: {},
      display: '',
      dataType: undefined,
      widgetType: 'form',
      meta: { preserved: true },
    })
    expect(formDataStore.getAllFieldPaths()).not.toContain('profile.attachments.note')
  })

  it('syncs the current form container value after nested child updates', () => {
    const formDataStore = useFormDataStore()
    formDataStore.setValue('items[0].profile', {
      raw: { member_type: 'vip', card_no: 'A001' },
      display: '{"member_type":"vip","card_no":"A001"}',
      meta: { preserved: true },
    } as any)
    formDataStore.setValue('items[0].profile.member_type', createFieldValue('vip') as any)
    formDataStore.setValue('items[0].profile.card_no', createFieldValue('A001') as any)

    const { updateSubFieldValue } = useFormWidget(createProps({
      fieldPath: 'items[0].profile',
      mode: 'edit',
    }))

    updateSubFieldValue('card_no', createFieldValue('B002') as any)

    expect(formDataStore.getValue('items[0].profile')).toEqual({
      raw: {
        member_type: 'vip',
        card_no: 'B002',
        attachments: {
          note: 'old',
        },
      },
      display: '{"member_type":"vip","card_no":"B002","attachments":{"note":"old"}}',
      dataType: undefined,
      widgetType: 'form',
      meta: { preserved: true },
    })
  })
})
