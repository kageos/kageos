import { describe, expect, it, vi } from 'vitest'
import { TableApplicationService } from './TableApplicationService'

describe('TableApplicationService batch add', () => {
  it('keeps the existing row callback chain and reloads only once', async () => {
    const domainService = {
      beforeRender: vi.fn(),
      addRow: vi.fn()
        .mockResolvedValueOnce({ id: 1 })
        .mockRejectedValueOnce({ response: { data: { msg: '客户联系电话已存在' } } })
        .mockResolvedValueOnce({ id: 3 }),
      loadData: vi.fn().mockResolvedValue({ items: [] })
    }
    const service = new TableApplicationService(
      domainService as any,
      { on: vi.fn(), emit: vi.fn(), off: vi.fn() } as any
    )

    const result = await service.addRows(
      { router: '/opportunities.table' } as any,
      [
        { rowNumber: 2, data: { name: 'A' } },
        { rowNumber: 3, data: { name: 'B' } },
        { rowNumber: 4, data: { name: 'C' } }
      ]
    )

    expect(domainService.addRow).toHaveBeenCalledTimes(3)
    expect(domainService.loadData).toHaveBeenCalledTimes(1)
    expect(result).toEqual({
      createdCount: 2,
      failedCount: 1,
      errors: [{ rowNumber: 3, message: '客户联系电话已存在' }]
    })
  })
})

it('preserves successful writes when refreshing the list fails', async () => {
  const domain = { beforeRender: vi.fn(), addRow: vi.fn().mockResolvedValue({ id: 1 }), loadData: vi.fn().mockRejectedValue(new Error('offline')) }
  const service = new TableApplicationService(domain as any, { on: vi.fn() } as any)
  const result = await service.addRows({ router: '/table' } as any, [{ rowNumber: 2, data: { name: 'A' } }])
  expect(result).toMatchObject({ createdCount: 1, failedCount: 0, errors: [] })
  expect(result.refreshWarning).toContain('不要重复导入')
  expect(domain.addRow).toHaveBeenCalledTimes(1)
})
