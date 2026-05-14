import { describe, expect, it, vi } from 'vitest'
import { WorkspaceDomainService, type WorkspaceState } from './WorkspaceDomainService'
import { WorkspaceEvent, type IEventBus } from '../interfaces/IEventBus'
import type { IFunctionLoader, FunctionDetail } from '../interfaces/IFunctionLoader'
import type { IStateManager } from '../interfaces/IStateManager'
import type { ServiceTree } from '@/architecture/domain/types'

function createStateManager(initialState: WorkspaceState): IStateManager<WorkspaceState> {
  let state = initialState

  return {
    getState: () => state,
    setState: (nextState) => {
      state = nextState
    },
    subscribe: () => () => {},
    clear: () => {
      state = initialState
    }
  }
}

function createEventBus(): IEventBus {
  return {
    emit: vi.fn(),
    on: () => () => {},
    off: () => {},
    once: () => {}
  }
}

describe('WorkspaceDomainService', () => {
  it('loads function detail by full_code_path only and emits functionLoaded', async () => {
    const detail: FunctionDetail = { name: 'Demo', template_type: 'table' }
    const functionLoader: IFunctionLoader = {
      loadByPath: vi.fn().mockResolvedValue(detail),
      getCached: vi.fn().mockReturnValue(null),
      clearCache: vi.fn()
    }
    const stateManager = createStateManager({
      currentApp: null,
      currentFunction: null,
      currentDirectory: null,
      serviceTree: [],
      loading: false
    })
    const eventBus = createEventBus()
    const service = new WorkspaceDomainService(functionLoader, stateManager, eventBus)
    const node = {
      id: 1,
      name: 'Demo',
      code: 'demo',
      type: 'function',
      description: '',
      tags: '',
      app_id: 1,
      ref_id: 99,
      full_code_path: '/luobei/demo/report.table',
      template_type: 'table',
      created_at: '',
      updated_at: ''
    } as ServiceTree

    const result = await service.loadFunction(node)

    expect(functionLoader.loadByPath).toHaveBeenCalledWith('/luobei/demo/report.table', 'table')
    expect(result).toBe(detail)
    expect((eventBus.emit as any)).toHaveBeenCalledWith(WorkspaceEvent.functionLoaded, { node, detail })
    expect(stateManager.getState().currentFunction).toBe(node)
  })

  it('fails fast when node is missing full_code_path', async () => {
    const functionLoader: IFunctionLoader = {
      loadByPath: vi.fn(),
      getCached: vi.fn().mockReturnValue(null),
      clearCache: vi.fn()
    }
    const stateManager = createStateManager({
      currentApp: null,
      currentFunction: null,
      currentDirectory: null,
      serviceTree: [],
      loading: false
    })
    const service = new WorkspaceDomainService(functionLoader, stateManager, createEventBus())
    const node = {
      id: 1,
      name: 'Broken',
      code: 'broken',
      type: 'function',
      description: '',
      tags: '',
      app_id: 1,
      ref_id: 88,
      created_at: '',
      updated_at: ''
    } as ServiceTree

    await expect(service.loadFunction(node)).rejects.toThrow('节点缺少 full_code_path，无法加载函数详情')
    expect(functionLoader.loadByPath).not.toHaveBeenCalled()
    expect(stateManager.getState().currentFunction).toBe(node)
  })
})
