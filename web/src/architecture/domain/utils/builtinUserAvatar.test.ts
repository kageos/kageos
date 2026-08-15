import { describe, expect, it } from 'vitest'
import type { UserInfo } from '@/architecture/domain/types'
import { DEFAULT_BUILTIN_USER_AVATAR, normalizeBuiltinUserAvatar } from './builtinUserAvatar'

function user(username: string, avatar: string): UserInfo {
  return {
    id: 1,
    username,
    avatar,
    email: `${username}@kageos.local`,
    register_type: 'system',
    email_verified: true,
    status: 'active',
    created_at: '2026-08-15T00:00:00Z',
  }
}

describe('normalizeBuiltinUserAvatar', () => {
  it.each(['system', 'test_user'])('replaces stale %s avatar data', (username) => {
    expect(normalizeBuiltinUserAvatar(user(username, '/legacy-avatar.svg')).avatar)
      .toBe(DEFAULT_BUILTIN_USER_AVATAR)
  })

  it('preserves a regular user avatar', () => {
    const regularUser = user('beiluo', 'https://example.com/avatar.png')

    expect(normalizeBuiltinUserAvatar(regularUser)).toBe(regularUser)
  })
})
