import { describe, expect, it, vi } from 'vitest'
import { WorkspaceApplicationService } from './WorkspaceApplicationService'

function createEventBus() {
  return {
    on: vi.fn(() => vi.fn()),
    emit: vi.fn(),
    off: vi.fn(),
    once: vi.fn(),
  }
}

describe('WorkspaceApplicationService node access', () => {
  it('does not load function detail when the service-tree node is unreadable', async () => {
    const directory = {
      id: 1,
      type: 'package',
      full_code_path: '/system/democase/hangla_rank',
      children: [],
    }
    const deniedFunction = {
      id: 2,
      type: 'function',
      full_code_path: '/system/democase/hangla_rank/rate.form',
      permissions: { read: false },
    }
    const domainService = {
      loadFunction: vi.fn(),
      getServiceTree: vi.fn(() => [directory]),
      setCurrentDirectory: vi.fn(),
      setCurrentFunction: vi.fn(),
    }
    const service = new WorkspaceApplicationService(domainService as any, createEventBus() as any)

    await service.handleNodeClick(deniedFunction as any)

    expect(domainService.loadFunction).not.toHaveBeenCalled()
    expect(domainService.setCurrentDirectory).toHaveBeenCalledWith(directory, false)
    expect(domainService.setCurrentFunction).toHaveBeenCalledWith(deniedFunction)
  })

  it('loads function detail when the service-tree node is readable', async () => {
    const readableFunction = {
      id: 3,
      type: 'function',
      full_code_path: '/system/democase/hangla_rank/rate.form',
      permissions: { read: true },
    }
    const domainService = {
      loadFunction: vi.fn().mockResolvedValue({ id: 3 }),
      getServiceTree: vi.fn(() => []),
      setCurrentDirectory: vi.fn(),
      setCurrentFunction: vi.fn(),
    }
    const service = new WorkspaceApplicationService(domainService as any, createEventBus() as any)

    await service.handleNodeClick(readableFunction as any)

    expect(domainService.loadFunction).toHaveBeenCalledWith(readableFunction)
    expect(domainService.setCurrentFunction).toHaveBeenCalledWith(readableFunction)
  })
})
