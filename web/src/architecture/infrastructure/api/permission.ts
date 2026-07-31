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
