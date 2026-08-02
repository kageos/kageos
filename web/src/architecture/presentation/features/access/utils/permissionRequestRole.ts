import type { AccessPermissions, AccessRoleCode } from '@/architecture/domain/types'

export type PermissionRequestRole = Exclude<AccessRoleCode, 'owner'>

const requestRoleActions: Record<PermissionRequestRole, Array<keyof AccessPermissions>> = {
  viewer: ['read'],
  member: ['read', 'write', 'update'],
  admin: ['read', 'write', 'update', 'delete', 'admin'],
}

const requestRoleRanks: Record<PermissionRequestRole, number> = {
  viewer: 1,
  member: 2,
  admin: 3,
}

function hasPermissionAction(
  permissions: AccessPermissions | null | undefined,
  action: keyof AccessPermissions,
): boolean {
  if (!permissions) return false
  if (permissions.owner) return true
  if (action !== 'owner' && permissions.admin) return true
  return permissions[action] === true
}

export function permissionSetCoversRequestRole(
  permissions: AccessPermissions | null | undefined,
  role: PermissionRequestRole,
): boolean {
  return requestRoleActions[role].every(action => hasPermissionAction(permissions, action))
}

export function getEffectiveAccessRole(
  permissions: AccessPermissions | null | undefined,
): AccessRoleCode | null {
  if (permissions?.owner) return 'owner'
  if (permissionSetCoversRequestRole(permissions, 'admin')) return 'admin'
  if (permissionSetCoversRequestRole(permissions, 'member')) return 'member'
  if (permissionSetCoversRequestRole(permissions, 'viewer')) return 'viewer'
  return null
}

export function getRecommendedPermissionRequestRole(
  permissions: AccessPermissions | null | undefined,
): PermissionRequestRole | null {
  if (!permissionSetCoversRequestRole(permissions, 'member')) return 'member'
  if (!permissionSetCoversRequestRole(permissions, 'admin')) return 'admin'
  return null
}

export function permissionRequestRoleCovers(
  grantedRole: AccessRoleCode,
  requestedRole: PermissionRequestRole,
): boolean {
  if (grantedRole === 'owner') return true
  return (requestRoleRanks[grantedRole] || 0) >= requestRoleRanks[requestedRole]
}
