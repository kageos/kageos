import type { AccessAction, AccessPermissions, ServiceTree } from '@/architecture/domain/types'

export function can(permissions: AccessPermissions | null | undefined, action: AccessAction): boolean {
  if (!permissions) {
    return false
  }
  if (permissions.owner) {
    return true
  }
  if (action !== 'owner' && permissions.admin) {
    return true
  }
  return permissions[action] === true
}

export function canRead(node: Pick<ServiceTree, 'permissions'> | null | undefined): boolean {
  return can(node?.permissions, 'read')
}

export function canWrite(node: Pick<ServiceTree, 'permissions'> | null | undefined): boolean {
  return can(node?.permissions, 'write')
}

export function canUpdate(node: Pick<ServiceTree, 'permissions'> | null | undefined): boolean {
  return can(node?.permissions, 'update')
}

export function canDelete(node: Pick<ServiceTree, 'permissions'> | null | undefined): boolean {
  return can(node?.permissions, 'delete')
}

export function canAdmin(node: Pick<ServiceTree, 'permissions'> | null | undefined): boolean {
  return can(node?.permissions, 'admin')
}

export function useAccessControl() {
  return {
    can,
    canRead,
    canWrite,
    canUpdate,
    canDelete,
    canAdmin
  }
}
