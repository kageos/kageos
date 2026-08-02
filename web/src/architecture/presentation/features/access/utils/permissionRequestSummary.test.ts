import { describe, expect, it } from 'vitest'
import type { PermissionRequest } from '@/architecture/presentation/context/api/permission'
import {
  getPermissionRequestWorkspaceRoot,
  summarizePermissionRequests,
} from './permissionRequestSummary'

function request(id: number, resourcePath: string): PermissionRequest {
  return {
    id,
    tenant_user: 'system',
    app: 'demo',
    requester: 'tester',
    resource_path: resourcePath,
    requested_role: 'viewer',
    reason: 'test',
    status: 'pending',
    created_at: '',
    updated_at: '',
  }
}

describe('permission request summary', () => {
  it('counts only direct requests for the current resource and de-duplicates reviewer overlap', () => {
    expect(summarizePermissionRequests(
      '/system/demo/parent',
      [request(1, '/system/demo/parent'), request(2, '/system/demo/parent/child')],
      [request(1, '/system/demo/parent'), request(3, '/system/demo/parent')],
    )).toEqual({
      ownPendingCount: 1,
      reviewPendingCount: 2,
      totalCount: 2,
    })
  })

  it('resolves the workspace root from any resource path', () => {
    expect(getPermissionRequestWorkspaceRoot('/system/demo/parent/child.form')).toBe('/system/demo')
  })
})
