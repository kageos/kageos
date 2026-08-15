import { beforeEach, describe, expect, it, vi } from 'vitest'

const permissionApi = vi.hoisted(() => ({
  getPermissionRequestSummary: vi.fn(),
}))

vi.mock('@/architecture/presentation/context/api/permission', () => permissionApi)

import {
  getPermissionRequestSummaryState,
  loadPermissionRequestSummary,
  ownPendingPermissionRequestPaths,
  permissionRequestPathSummary,
  seedPermissionRequestSummaryFromTree,
  settlePermissionRequestSummary,
} from './permissionRequestSummaryStore'

describe('permission request summary store', () => {
  beforeEach(() => {
    permissionApi.getPermissionRequestSummary.mockReset()
  })

  it('shares one in-flight summary request across service tree and tab labels', async () => {
    permissionApi.getPermissionRequestSummary.mockResolvedValue({
      paths: {
        '/system/demo/orders': {
          own_pending_count: 1,
          review_pending_count: 2,
        },
      },
      own_pending_count: 1,
      review_pending_count: 2,
    })

    await Promise.all([
      loadPermissionRequestSummary('/system/demo'),
      loadPermissionRequestSummary('/system/demo'),
    ])

    expect(permissionApi.getPermissionRequestSummary).toHaveBeenCalledTimes(1)
    const state = getPermissionRequestSummaryState('/system/demo')
    expect(permissionRequestPathSummary(state, '/system/demo/orders')).toEqual({
      own_pending_count: 1,
      review_pending_count: 2,
    })
    expect([...ownPendingPermissionRequestPaths(state)]).toEqual(['/system/demo/orders'])
  })

  it('seeds badge counts from the initial service tree without another request', () => {
    seedPermissionRequestSummaryFromTree('/alice/ops', [
      {
        full_code_path: '/alice/ops',
        children: [
          {
            full_code_path: '/alice/ops/orders',
            permission_requests: {
              own_pending_count: 1,
              review_pending_count: 2,
            },
          },
        ],
      },
    ])

    expect(permissionApi.getPermissionRequestSummary).not.toHaveBeenCalled()
    expect(permissionRequestPathSummary(
      getPermissionRequestSummaryState('/alice/ops'),
      '/alice/ops/orders',
    )).toEqual({
      own_pending_count: 1,
      review_pending_count: 2,
    })
  })

  it('removes a handled approval from the shared badge state immediately', () => {
    seedPermissionRequestSummaryFromTree('/alice/review', [{
      full_code_path: '/alice/review/orders',
      permission_requests: {
        own_pending_count: 1,
        review_pending_count: 2,
      },
    }])

    settlePermissionRequestSummary('/alice/review', '/alice/review/orders', 'review')

    const state = getPermissionRequestSummaryState('/alice/review')
    expect(permissionRequestPathSummary(state, '/alice/review/orders')).toEqual({
      own_pending_count: 1,
      review_pending_count: 1,
    })
    expect(state.reviewPendingCount).toBe(1)
  })

  it('does not let an older in-flight response restore a handled approval count', async () => {
    let resolveStale: ((value: {
      paths: Record<string, { own_pending_count: number; review_pending_count: number }>
      own_pending_count: number
      review_pending_count: number
    }) => void) | undefined
    permissionApi.getPermissionRequestSummary
      .mockImplementationOnce(() => new Promise(resolve => { resolveStale = resolve }))
      .mockResolvedValueOnce({
        paths: {
          '/alice/race/orders': { own_pending_count: 0, review_pending_count: 1 },
        },
        own_pending_count: 0,
        review_pending_count: 1,
      })

    const staleLoad = loadPermissionRequestSummary('/alice/race', { force: true })
    settlePermissionRequestSummary('/alice/race', '/alice/race/orders', 'review')
    const refreshedLoad = loadPermissionRequestSummary('/alice/race', { force: true })
    resolveStale?.({
      paths: {
        '/alice/race/orders': { own_pending_count: 0, review_pending_count: 2 },
      },
      own_pending_count: 0,
      review_pending_count: 2,
    })
    await Promise.all([staleLoad, refreshedLoad])

    expect(permissionApi.getPermissionRequestSummary).toHaveBeenCalledTimes(2)
    expect(permissionRequestPathSummary(
      getPermissionRequestSummaryState('/alice/race'),
      '/alice/race/orders',
    ).review_pending_count).toBe(1)
  })

  it('does not let an older in-flight response overwrite counts seeded from a refreshed tree', async () => {
    let resolveStale: ((value: {
      paths: Record<string, { own_pending_count: number; review_pending_count: number }>
      own_pending_count: number
      review_pending_count: number
    }) => void) | undefined
    permissionApi.getPermissionRequestSummary.mockImplementationOnce(
      () => new Promise(resolve => { resolveStale = resolve }),
    )

    const staleLoad = loadPermissionRequestSummary('/alice/tree-race', { force: true })
    seedPermissionRequestSummaryFromTree('/alice/tree-race', [{
      full_code_path: '/alice/tree-race/orders',
      permission_requests: {
        own_pending_count: 0,
        review_pending_count: 1,
      },
    }])
    resolveStale?.({
      paths: {
        '/alice/tree-race/orders': { own_pending_count: 0, review_pending_count: 2 },
      },
      own_pending_count: 0,
      review_pending_count: 2,
    })
    await staleLoad

    expect(permissionRequestPathSummary(
      getPermissionRequestSummaryState('/alice/tree-race'),
      '/alice/tree-race/orders',
    ).review_pending_count).toBe(1)
  })

  it('retries the current revision after an older in-flight request fails', async () => {
    let rejectStale: ((reason?: unknown) => void) | undefined
    permissionApi.getPermissionRequestSummary
      .mockImplementationOnce(() => new Promise((_, reject) => { rejectStale = reject }))
      .mockResolvedValueOnce({
        paths: {
          '/alice/retry/orders': { own_pending_count: 0, review_pending_count: 1 },
        },
        own_pending_count: 0,
        review_pending_count: 1,
      })

    const staleLoad = loadPermissionRequestSummary('/alice/retry', { force: true })
    settlePermissionRequestSummary('/alice/retry', '/alice/retry/orders', 'review')
    const refreshedLoad = loadPermissionRequestSummary('/alice/retry', { force: true })
    rejectStale?.(new Error('stale request failed'))
    const results = await Promise.allSettled([staleLoad, refreshedLoad])

    expect(results[0]?.status).toBe('rejected')
    expect(results[1]?.status).toBe('fulfilled')
    expect(permissionApi.getPermissionRequestSummary).toHaveBeenCalledTimes(2)
    expect(permissionRequestPathSummary(
      getPermissionRequestSummaryState('/alice/retry'),
      '/alice/retry/orders',
    ).review_pending_count).toBe(1)
  })
})
