import type { TableRow } from '@/architecture/domain/services/TableDomainService'

export type TableActionCommandResult =
  | { type: 'link'; fieldCode: string }
  | { type: 'detail'; initialMode: 'edit' }
  | { type: 'delete' }
  | { type: 'noop' }

export type TableDetailEditAccess = 'unsupported' | 'allowed'

export function hasFunctionCallback(
  callbacks: string[] | null | undefined,
  callbackName: string
): boolean {
  return Array.isArray(callbacks) && callbacks.includes(callbackName)
}

export function resolveTableDetailEditAccess(options: {
  supportsUpdate: boolean
}): TableDetailEditAccess {
  if (!options.supportsUpdate) {
    return 'unsupported'
  }

  return 'allowed'
}

export function resolveTableActionCommand(options: {
  command: string
}): TableActionCommandResult {
  const { command } = options

  if (command.startsWith('link:')) {
    return {
      type: 'link',
      fieldCode: command.slice(5)
    }
  }

  if (command === 'update') {
    return { type: 'detail', initialMode: 'edit' }
  }

  if (command === 'delete') {
    return { type: 'delete' }
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
