import { get, post } from '@/architecture/infrastructure/apiClient/request'
import type { AccessPermissions, AccessRoleCode } from '@/architecture/domain/types'
import { normalizeResourcePath } from '@/architecture/shared/resourcePath'

export interface TeamMemberAccess {
  tenant_user: string
  app: string
  username: string
  resource_path: string
  role_code: AccessRoleCode
  permissions: AccessPermissions
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

export function listTeamMembers(resourcePath: string) {
  return get<{ members: TeamMemberAccess[] }>('/workspace/api/v1/team_access/members', {
    resource_path: normalizeResourcePath(resourcePath)
  })
}

export function assignTeamRole(data: {
  resource_path: string
  username: string
  role_code: AccessRoleCode
  expires_at?: string | null
}) {
  return post('/workspace/api/v1/team_access/assign', {
    ...data,
    resource_path: normalizeResourcePath(data.resource_path)
  })
}

export function batchAssignTeamRoles(data: {
  resource_paths: string[]
  usernames: string[]
  role_codes: AccessRoleCode[]
  expires_at?: string | null
}) {
  return post('/workspace/api/v1/team_access/batch_assign', {
    ...data,
    resource_paths: data.resource_paths.map(normalizeResourcePath)
  })
}

export function removeTeamRole(data: {
  resource_path: string
  username: string
  role_code?: AccessRoleCode
}) {
  return post('/workspace/api/v1/team_access/remove', {
    ...data,
    resource_path: normalizeResourcePath(data.resource_path)
  })
}

export function getMyPermissions(resourcePath: string) {
  return get<MyPermissions>('/workspace/api/v1/team_access/my_permissions', {
    resource_path: normalizeResourcePath(resourcePath)
  })
}
