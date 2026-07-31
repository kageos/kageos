import { get, post } from '@/architecture/infrastructure/apiClient/request'
import type { AccessPermissions, AccessRoleCode } from '@/architecture/domain/types'
import { normalizeResourcePath } from '@/architecture/shared/resourcePath'

export type PermissionPrincipalType = 'user' | 'department'

export interface PermissionPrincipal {
  type: PermissionPrincipalType
  key: string
}

export interface RoleAssignment {
  tenant_user: string
  app: string
  principal_type: PermissionPrincipalType
  principal_key: string
  resource_path: string
  role_code: AccessRoleCode
  permissions: AccessPermissions
  source?: 'current' | 'inherited'
  direct?: boolean
  inherited_from?: string
  target_resource?: string
  expires_at?: string
  created_by?: string
  created_at?: string
  updated_at?: string
}

export interface MyPermissions {
  resource_path: string
  role_codes?: AccessRoleCode[]
  permissions: AccessPermissions
  inherited_from?: string
  expires_at?: string
}

export type PermissionRequestStatus = 'pending' | 'approved' | 'rejected' | 'cancelled'

export interface PermissionApprover {
  principal_type: PermissionPrincipalType
  principal_key: string
  role_code: AccessRoleCode
  resource_path: string
  inherited: boolean
}

export interface PermissionRequest {
  id: number
  tenant_user: string
  app: string
  requester: string
  resource_path: string
  requested_role: AccessRoleCode
  reason: string
  status: PermissionRequestStatus
  reviewed_by?: string
  reviewed_at?: string
  review_comment?: string
  requested_expires_at?: string
  created_at: string
  updated_at: string
  approvers?: PermissionApprover[]
}

export function listPermissionAssignments(resourcePath: string) {
  return get<{ assignments: RoleAssignment[] }>('/workspace/api/v1/permissions/assignments', {
    resource_path: normalizeResourcePath(resourcePath)
  })
}

export function grantRole(data: {
  resource_path: string
  principal: PermissionPrincipal
  role_code: AccessRoleCode
  expires_at?: string | null
}) {
  return post('/workspace/api/v1/permissions/grant', {
    ...data,
    resource_path: normalizeResourcePath(data.resource_path)
  })
}

export function batchGrantRoles(data: {
  resource_paths: string[]
  principals: PermissionPrincipal[]
  role_codes: AccessRoleCode[]
  expires_at?: string | null
}) {
  return post('/workspace/api/v1/permissions/batch-grant', {
    ...data,
    resource_paths: data.resource_paths.map(normalizeResourcePath)
  })
}

export function revokeRole(data: {
  resource_path: string
  principal: PermissionPrincipal
  role_code?: AccessRoleCode
}) {
  return post('/workspace/api/v1/permissions/revoke', {
    ...data,
    resource_path: normalizeResourcePath(data.resource_path)
  })
}

export function getMyPermissions(resourcePath: string) {
  return get<MyPermissions>('/workspace/api/v1/permissions/me', {
    resource_path: normalizeResourcePath(resourcePath)
  })
}

export function createPermissionRequest(data: {
  resource_path: string
  role_code: 'viewer' | 'member' | 'admin'
  reason: string
  expires_at?: string | null
}) {
  return post<{ request: PermissionRequest }>('/workspace/api/v1/permissions/requests', {
    ...data,
    resource_path: normalizeResourcePath(data.resource_path)
  })
}

export function listMyPermissionRequests(resourcePath: string, status?: PermissionRequestStatus) {
  return get<{ requests: PermissionRequest[] }>('/workspace/api/v1/permissions/requests/mine', {
    resource_path: normalizeResourcePath(resourcePath),
    ...(status ? { status } : {})
  })
}

export function listPendingPermissionRequests(resourcePath: string) {
  return get<{ requests: PermissionRequest[] }>('/workspace/api/v1/permissions/requests/pending', {
    resource_path: normalizeResourcePath(resourcePath)
  })
}

export function getPendingPermissionRequestCount(resourcePath: string) {
  return get<{ count: number }>('/workspace/api/v1/permissions/requests/pending-count', {
    resource_path: normalizeResourcePath(resourcePath)
  })
}

export function listPermissionRequestHistory(resourcePath: string) {
  return get<{ requests: PermissionRequest[] }>('/workspace/api/v1/permissions/requests/history', {
    resource_path: normalizeResourcePath(resourcePath)
  })
}

export function listPermissionApprovers(resourcePath: string) {
  return get<{ approvers: PermissionApprover[] }>('/workspace/api/v1/permissions/approvers', {
    resource_path: normalizeResourcePath(resourcePath)
  })
}

export function approvePermissionRequest(id: number, comment = '') {
  return post<{ request: PermissionRequest }>(`/workspace/api/v1/permissions/requests/${id}/approve`, {
    comment
  })
}

export function rejectPermissionRequest(id: number, comment: string) {
  return post<{ request: PermissionRequest }>(`/workspace/api/v1/permissions/requests/${id}/reject`, {
    comment
  })
}

export function cancelPermissionRequest(id: number) {
  return post<{ request: PermissionRequest }>(`/workspace/api/v1/permissions/requests/${id}/cancel`, {})
}
