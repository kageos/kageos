import type { UserInfo } from '@/architecture/domain/types'

export const DEFAULT_BUILTIN_USER_AVATAR = '/brand/kageos-avatar-128.png'
export const BRAND_LOGO_192_URL = '/brand/kageos-logo-192.png'

const BUILTIN_USERNAMES = new Set(['system', 'test_user'])

export function normalizeBuiltinUserAvatar(user: UserInfo): UserInfo {
  if (!BUILTIN_USERNAMES.has(user.username) || user.avatar === DEFAULT_BUILTIN_USER_AVATAR) {
    return user
  }

  return {
    ...user,
    avatar: DEFAULT_BUILTIN_USER_AVATAR,
  }
}
