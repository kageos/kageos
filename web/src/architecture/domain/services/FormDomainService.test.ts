import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { FormDomainService } from './FormDomainService'
import { FormStateManager } from '@/architecture/infrastructure/stateManager/FormStateManager'
import { useFormDataStore } from '@/core/stores-v2/formData'

function createEventBus() {
  return {
    emit: () => {},
    on: () => () => {},
    off: () => {}
  } as any
}

describe('FormDomainService nested validation', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    useFormDataStore().clear()
  })

  it('validates nested required_if fields inside form widgets', () => {
    const stateManager = new FormStateManager()
    const service = new FormDomainService(stateManager as any, createEventBus())

    const fields = [
      {
        code: 'profile',
        name: '资料',
        widget: { type: 'form' },
        children: [
          {
            code: 'member_type',
            field_name: 'MemberType',
            name: '会员类型',
            widget: { type: 'input' },
            data: { type: 'string' }
          },
          {
            code: 'card_no',
            field_name: 'CardNo',
            name: '卡号',
            widget: { type: 'input' },
            data: { type: 'string' },
            validation: 'required_if=MemberType vip'
          }
        ]
      }
    ] as any

    stateManager.setState({
      data: new Map([
        ['profile', { raw: { member_type: 'vip', card_no: '' }, display: '', meta: {} }],
        ['profile.member_type', { raw: 'vip', display: 'vip', meta: {} }],
        ['profile.card_no', { raw: '', display: '', meta: {} }]
      ]),
      errors: new Map(),
      submitting: false,
      response: null,
      metadata: null
    } as any)

    const isValid = service.validateForm(fields)

    expect(isValid).toBe(false)
    expect(service.getFieldError('profile.card_no')[0]?.message).toBe('卡号必填')
  })

  it('validates required fields inside table rows', () => {
    const stateManager = new FormStateManager()
    const service = new FormDomainService(stateManager as any, createEventBus())

    const fields = [
      {
        code: 'items',
        name: '明细',
        widget: { type: 'table' },
        children: [
          {
            code: 'name',
            name: '名称',
            widget: { type: 'input' },
            data: { type: 'string' },
            validation: 'required'
          }
        ]
      }
    ] as any

    stateManager.setState({
      data: new Map([
        ['items', { raw: [{ name: '' }], display: '共 1 条', meta: {} }],
        ['items[0].name', { raw: '', display: '', meta: {} }]
      ]),
      errors: new Map(),
      submitting: false,
      response: null,
      metadata: null
    } as any)

    const isValid = service.validateForm(fields)

    expect(isValid).toBe(false)
    expect(service.getFieldError('items[0].name')[0]?.message).toBe('名称必填')
  })

  it('clears response metadata together with form state', () => {
    const stateManager = new FormStateManager()
    const service = new FormDomainService(stateManager as any, createEventBus())

    stateManager.setResponse({ ok: true })
    stateManager.setMetadata({ total_cost_mill: 12 })

    service.clearForm()

    expect(stateManager.getState().data.size).toBe(0)
    expect(stateManager.getState().response).toBeNull()
    expect(stateManager.getState().metadata).toBeNull()
  })
})
