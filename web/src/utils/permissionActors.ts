import type { ServiceTree } from '@/types'

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
