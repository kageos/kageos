import { computed, effectScope } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { registerWidgetInitializers } from '@/architecture/presentation/widgets/initializers/registerInitializers'
import { useFunctionParamInitialization } from './useFunctionParamInitialization'

const { selectFuzzyMock } = vi.hoisted(() => ({
  selectFuzzyMock: vi.fn()
}))

vi.mock('vue-router', () => ({
  createRouter: () => ({
    beforeEach: vi.fn(),
    afterEach: vi.fn(),
    push: vi.fn(),
    replace: vi.fn()
  }),
  createWebHistory: () => ({}),
  useRoute: () => ({
    query: {}
  }),
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn()
  })
}))

vi.mock('@/api/function', () => ({
  selectFuzzy: (...args: any[]) => selectFuzzyMock(...args)
}))

describe('useFunctionParamInitialization', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    useFormDataStore().clear()
    selectFuzzyMock.mockReset()
    registerWidgetInitializers()
  })

  it('hydrates OnSelectFuzzy fields from raw values when initialData is applied', async () => {
    selectFuzzyMock.mockResolvedValue({
      error_msg: '',
      items: [
        {
          label: '中国银行(9527)',
          value: 1,
          display_info: {
            bank_name: '中国银行'
          }
        }
      ]
    })

    const formDataStore = useFormDataStore()
    formDataStore.setValue('bank_id', {
      raw: 1,
      display: '',
      meta: {
        fromInitialData: true
      }
    } as any)

    const scope = effectScope()

    try {
      const composable = scope.run(() => useFunctionParamInitialization({
        functionDetail: computed(() => ({
          id: 105,
          method: 'GET',
          router: '/liubeiluo/work/component_test/test_all_components.form',
          request: [
            {
              code: 'bank_id',
              name: '银行卡',
              callbacks: ['OnSelectFuzzy'],
              widget: {
                type: 'select',
                config: {}
              },
              data: {
                type: 'int'
              }
            }
          ]
        }) as any),
        formDataStore: {
          getValue: (fieldCode: string) => formDataStore.getValue(fieldCode),
          setValue: (fieldCode: string, value: any) => formDataStore.setValue(fieldCode, value),
          getAllValues: () => {
            const allValues: Record<string, any> = {}
            formDataStore.data.forEach((value, key) => {
              allValues[key] = value
            })
            return allValues
          },
          clear: () => formDataStore.clear()
        }
      }))!

      await composable.hydrateCurrentWidgetDisplays('initialData')

      expect(selectFuzzyMock).toHaveBeenCalledTimes(1)
      expect(formDataStore.getValue('bank_id')).toMatchObject({
        raw: 1,
        display: '中国银行(9527)'
      })
      expect(formDataStore.getValue('bank_id')?.meta?.displayInfo).toEqual({
        bank_name: '中国银行'
      })
    } finally {
      scope.stop()
    }
  })
})
