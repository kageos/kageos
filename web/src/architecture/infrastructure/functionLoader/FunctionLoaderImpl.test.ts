import { afterEach, describe, expect, it, vi } from 'vitest'
import { FunctionLoaderImpl } from './FunctionLoaderImpl'

describe('FunctionLoaderImpl', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('always fetches fresh function detail instead of returning stale cached schema', async () => {
    const get = vi
      .fn()
      .mockResolvedValueOnce({
        router: '/demo/form',
        request: [{ code: 'progress', widget: { type: 'input' } }]
      })
      .mockResolvedValueOnce({
        router: '/demo/form',
        request: [{ code: 'progress', widget: { type: 'slider' } }]
      })

    const cacheManager = {
      get: vi.fn().mockReturnValue({
        router: '/demo/form',
        request: [{ code: 'progress', widget: { type: 'input' } }]
      }),
      set: vi.fn(),
      delete: vi.fn(),
      getKeys: vi.fn().mockReturnValue([]),
      clear: vi.fn()
    }

    const loader = new FunctionLoaderImpl({ get } as any, cacheManager as any, 0)

    const first = await loader.loadByPath('/demo/form', 'form')
    const second = await loader.loadByPath('/demo/form', 'form')

    expect(get).toHaveBeenCalledTimes(2)
    expect(first.request?.[0]?.widget?.type).toBe('input')
    expect(second.request?.[0]?.widget?.type).toBe('slider')
    expect(cacheManager.get).not.toHaveBeenCalled()
    expect(cacheManager.set).not.toHaveBeenCalled()
  })

  it('deduplicates concurrent requests for the same path while one request is in flight', async () => {
    vi.useFakeTimers()

    let resolveRequest!: (value: any) => void
    const get = vi.fn().mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveRequest = resolve
        })
    )

    const cacheManager = {
      get: vi.fn(),
      set: vi.fn(),
      delete: vi.fn(),
      getKeys: vi.fn().mockReturnValue([]),
      clear: vi.fn()
    }

    const loader = new FunctionLoaderImpl({ get } as any, cacheManager as any, 0)

    const pendingA = loader.loadByPath('/demo/form', 'form')
    const pendingB = loader.loadByPath('/demo/form', 'form')

    await vi.advanceTimersByTimeAsync(0)

    resolveRequest({
      router: '/demo/form',
      request: [{ code: 'progress', widget: { type: 'slider' } }]
    })

    const [resultA, resultB] = await Promise.all([pendingA, pendingB])

    expect(get).toHaveBeenCalledTimes(1)
    expect(resultA).toEqual(resultB)
  })
})
