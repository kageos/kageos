export const managementActions = [
  'timer.task.created', 'timer.task.updated', 'timer.task.paused', 'timer.task.resumed',
  'timer.task.cancelled', 'timer.task.deleted', 'timer.task.run_now',
  'workspace.deleted', 'directory.installed', 'directory.copied',
  'public_share.created', 'public_share.disabled',
] as const

export function isManagementAction(action: string): boolean {
  return (managementActions as readonly string[]).includes(action)
}

export function managementActionKey(action: string): string {
  return `managementLog.actions.${action.replaceAll('.', '_')}`
}

export function managementChanges(before: Record<string, unknown> | null | undefined, after: Record<string, unknown> | null | undefined) {
  return [...new Set([...Object.keys(before || {}), ...Object.keys(after || {})])]
    .filter(key => key !== 'message' && JSON.stringify(before?.[key]) !== JSON.stringify(after?.[key]))
    .map(key => ({ key, before: before?.[key], after: after?.[key] }))
}
