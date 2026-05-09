import type { App, ServiceTree } from '@/types'
import { hasPermission } from '@/utils/permission'

type UsernameListSource = string | string[] | null | undefined

export function parseUsernameList(value: UsernameListSource): string[] {
  const items = Array.isArray(value) ? value : String(value || '').split(',')
  const seen = new Set<string>()
  const usernames: string[] = []

  for (const item of items) {
    const username = String(item).trim()
    if (!username || seen.has(username)) continue
    seen.add(username)
    usernames.push(username)
  }

  return usernames
}

export function isUsernameInList(value: UsernameListSource, username: string | null | undefined): boolean {
  const normalizedUsername = String(username || '').trim()
  if (!normalizedUsername) return false
  return parseUsernameList(value).includes(normalizedUsername)
}

export function canApprovePermissionRequest(
  currentUsername: string | null | undefined,
  approvers: string[] | null | undefined,
  status: string
): boolean {
  if (status !== 'pending') {
    return false
  }

  return isUsernameInList(approvers, currentUsername)
}

export function isServiceTreeNodeAdmin(
  node: Pick<ServiceTree, 'admins' | 'owner'> | null | undefined,
  currentUsername: string | null | undefined
): boolean {
  if (!node) return false
  const normalizedUsername = String(currentUsername || '').trim()
  if (!normalizedUsername) return false
  return node.owner === normalizedUsername || isUsernameInList(node.admins, normalizedUsername)
}

export function hasWorkspaceAdminAccess(options: {
  currentApp: App | null | undefined
  currentUsername: string | null | undefined
  serviceTree?: ServiceTree[] | null
}): boolean {
  const { currentApp, currentUsername, serviceTree } = options

  if (!currentApp) {
    return false
  }

  const normalizedUsername = String(currentUsername || '').trim()
  if (!normalizedUsername) {
    return false
  }

  const stack = [...(serviceTree || [])]
  while (stack.length > 0) {
    const node = stack.pop()!
    if (node.is_admin === true || hasPermission(node, 'app:admin')) {
      return true
    }
    if (node.children?.length) {
      stack.push(...node.children)
    }
  }

  return currentApp.user === normalizedUsername || isUsernameInList(currentApp.admins, normalizedUsername)
}
