import { buildPermissionApplyURL, TablePermission } from '@/utils/permission'
import type { TableRow } from '@/architecture/domain/services/TableDomainService'

export type TableActionCommandResult =
  | { type: 'link'; fieldCode: string }
  | { type: 'detail'; initialMode: 'edit' }
  | { type: 'delete' }
  | { type: 'apply-permission'; action: string }
  | { type: 'noop' }

interface PermissionNodeLike {
  full_code_path?: string
  template_type?: string
}

export type TableDetailEditAccess = 'unsupported' | 'no-permission' | 'allowed'

export function hasFunctionCallback(
  callbacks: string[] | string | null | undefined,
  callbackName: string
): boolean {
  if (Array.isArray(callbacks)) {
    return callbacks.includes(callbackName)
  }

  if (typeof callbacks !== 'string') {
    return false
  }

  return callbacks
    .split(',')
    .map(item => item.trim())
    .filter(Boolean)
    .includes(callbackName)
}

export function resolveTableDetailEditAccess(options: {
  supportsUpdate: boolean
  canUpdate: boolean
}): TableDetailEditAccess {
  if (!options.supportsUpdate) {
    return 'unsupported'
  }

  return options.canUpdate ? 'allowed' : 'no-permission'
}

export function resolveTableActionCommand(options: {
  command: string
  canUpdate: boolean
  canDelete: boolean
}): TableActionCommandResult {
  const { command, canUpdate, canDelete } = options

  if (command.startsWith('link:')) {
    return {
      type: 'link',
      fieldCode: command.slice(5)
    }
  }

  if (command === 'update') {
    return canUpdate
      ? { type: 'detail', initialMode: 'edit' }
      : { type: 'apply-permission', action: TablePermission.update }
  }

  if (command === 'delete') {
    return canDelete
      ? { type: 'delete' }
      : { type: 'apply-permission', action: TablePermission.delete }
  }

  return { type: 'noop' }
}

export function buildTablePermissionApplyURL(
  node: PermissionNodeLike | null | undefined,
  action: string
): string | null {
  if (!node?.full_code_path) {
    return null
  }

  return buildPermissionApplyURL(node.full_code_path, action, node.template_type)
}

export function buildBatchDeleteIds(
  rows: TableRow[],
  idFieldCode?: string | null
): number[] {
  return rows
    .map(row => {
      if (typeof row.id === 'number') {
        return row.id
      }

      if (idFieldCode && typeof row[idFieldCode] === 'number') {
        return row[idFieldCode]
      }

      return null
    })
    .filter((id): id is number => id !== null)
}
