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
})
