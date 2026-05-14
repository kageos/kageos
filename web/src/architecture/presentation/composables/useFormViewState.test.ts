import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import type { IEventBus } from '@/architecture/domain/interfaces/IEventBus'
import type { IFormGateway } from '@/architecture/domain/interfaces/IFormGateway'
import type { FunctionDetail } from '@/architecture/domain/types'
import { useFormViewState } from './useFormViewState'
import { createFormViewRuntime } from '@/architecture/presentation/views/utils/formViewRuntime'
import { getFormRequestFields } from '@/architecture/runtime/utils/functionSchemaSelectors'

function createMockEventBus(): IEventBus {
  const listeners = new Map<string, Set<(payload?: any) => void>>()

  return {
    emit(event: string, payload?: any) {
      listeners.get(event)?.forEach(handler => handler(payload))
    },
    on(event: string, handler: (payload?: any) => void) {
      if (!listeners.has(event)) {
        listeners.set(event, new Set())
      }
      listeners.get(event)!.add(handler)
      return () => {
        listeners.get(event)?.delete(handler)
      }
    },
    off(event: string, handler: (payload?: any) => void) {
      listeners.get(event)?.delete(handler)
    },
    once(event: string, handler: (payload?: any) => void) {
      const unsubscribe = this.on(event, (payload?: any) => {
        unsubscribe()
        handler(payload)
      })
    }
  }
}

const formGatewayStub: IFormGateway = {
  submitForm: async () => ({})
}

describe('useFormViewState', () => {
  it('filters top-level fields by presence rules and updates required state dynamically', async () => {
    const runtime = createFormViewRuntime({
      eventBus: createMockEventBus(),
      formGateway: formGatewayStub
    })

    const fields = [
      {
        code: 'member_type',
        field_name: 'MemberType',
        name: '会员类型',
        widget: { type: 'input' },
        data: { type: 'string' }
      },
      {
        code: 'card_no',
        name: '卡号',
        widget: { type: 'input' },
        data: { type: 'string' },
        validation: 'required_if=MemberType vip'
      }
    ]
    const functionDetail = ref<FunctionDetail | null>({
      schema: {
        version: 1,
        type: 'form',
        form: {
          request: fields,
          response: []
        }
      }
    })
    const detail = functionDetail.value!
    const requestFields = getFormRequestFields(detail)

    runtime.applicationService.initializeForm(requestFields, { member_type: 'normal', card_no: '' }, true)

    const state = useFormViewState({
      functionDetail,
      stateManager: runtime.stateManager,
      domainService: runtime.domainService,
      applicationService: runtime.applicationService
    })

    expect(state.visibleRequestFields.value.map((field) => field.code)).toEqual(['member_type'])
    expect(state.isFieldRequired(requestFields[1]!)).toBe(false)

    runtime.applicationService.updateFieldValue('member_type', {
      raw: 'vip',
      display: 'vip',
      meta: {}
    })
    await nextTick()

    expect(state.visibleRequestFields.value.map((field) => field.code)).toEqual(['member_type', 'card_no'])
    expect(state.isFieldRequired(requestFields[1]!)).toBe(true)
  })
})
