import { TablePermission } from '@/utils/permission'
import type { TableRow } from '@/architecture/domain/services/TableDomainService'

export type TableActionCommandResult =
  | { type: 'link'; fieldCode: string }
  | { type: 'detail'; initialMode: 'edit' }
  | { type: 'delete' }
  | { type: 'no-permission'; action: string }
  | { type: 'noop' }

export type TableDetailEditAccess = 'unsupported' | 'no-permission' | 'allowed'

export function hasFunctionCallback(
  callbacks: string[] | null | undefined,
  callbackName: string
): boolean {
  return Array.isArray(callbacks) && callbacks.includes(callbackName)
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
      : { type: 'no-permission', action: TablePermission.update }
  }

  if (command === 'delete') {
    return canDelete
      ? { type: 'delete' }
      : { type: 'no-permission', action: TablePermission.delete }
  }

  return { type: 'noop' }
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
