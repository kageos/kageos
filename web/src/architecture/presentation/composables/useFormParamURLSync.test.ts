import { ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { RouteSource } from '@/architecture/runtime/utils/routeSource'
import { useFormParamURLSync } from './useFormParamURLSync'
import type { FieldValue, FunctionDetail } from '../../domain/types'

const { route } = vi.hoisted(() => ({
  route: {
    query: {} as Record<string, any>
  }
}))

vi.mock('vue-router', () => ({
  useRoute: () => route
}))

function createFormDetail(overrides: Partial<FunctionDetail> = {}): FunctionDetail {
  return {
    id: 1,
    template_type: 'form',
    method: 'POST',
    router: '/demo/submit.form',
    schema: {
      version: 1,
      type: 'form',
      form: {
        request: [
          {
            code: 'title',
            name: '标题',
            widget: { type: 'input', config: {} },
            data: { type: 'string' }
          }
        ],
        response: []
      }
    },
    ...overrides
  } as FunctionDetail
}

function createFormDataStore(values: Record<string, FieldValue>) {
  return {
    getValue: (fieldCode: string) => values[fieldCode] ?? { raw: null, display: '' },
    getAllValues: () => values
  }
}

describe('useFormParamURLSync', () => {
  let emitSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    route.query = {}
    emitSpy = vi.spyOn(eventBus, 'emit').mockImplementation(() => undefined)
  })

  afterEach(() => {
    emitSpy.mockRestore()
  })

  it('syncs params for standalone form functions without _tab', () => {
    const { syncToURL } = useFormParamURLSync({
      functionDetail: ref(createFormDetail()),
      formDataStore: createFormDataStore({
        title: { raw: '周报', display: '周报' }
      })
    })

    syncToURL()

    expect(emitSpy).toHaveBeenCalledWith(RouteEvent.updateRequested, expect.objectContaining({
      query: {
        title: '周报'
      },
      source: RouteSource.FORM_SYNC,
      preserveParams: expect.objectContaining({
        state: true
      })
    }))
  })

  it('continues syncing table add-row forms', () => {
    route.query = {
      _tab: 'OnTableAddRow'
    }

    const { syncToURL } = useFormParamURLSync({
      functionDetail: ref(createFormDetail({ id: 0 })),
      formDataStore: createFormDataStore({
        title: { raw: '新增记录', display: '新增记录' }
      })
    })

    syncToURL()

    expect(emitSpy).toHaveBeenCalledWith(RouteEvent.updateRequested, expect.objectContaining({
      query: {
        _tab: 'OnTableAddRow',
        title: '新增记录'
      },
      source: RouteSource.FORM_SYNC
    }))
  })

  it('does not sync synthetic edit forms without the add-row tab', () => {
    const { syncToURL } = useFormParamURLSync({
      functionDetail: ref(createFormDetail({ id: 0 })),
      formDataStore: createFormDataStore({
        title: { raw: '编辑记录', display: '编辑记录' }
      })
    })

    syncToURL()

    expect(emitSpy).not.toHaveBeenCalled()
  })

  it('does not sync detail mode form state', () => {
    route.query = {
      _tab: 'detail'
    }

    const { syncToURL } = useFormParamURLSync({
      functionDetail: ref(createFormDetail()),
      formDataStore: createFormDataStore({
        title: { raw: '详情记录', display: '详情记录' }
      })
    })

    syncToURL()

    expect(emitSpy).not.toHaveBeenCalled()
  })
})
