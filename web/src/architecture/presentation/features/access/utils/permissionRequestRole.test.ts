import { describe, expect, it } from 'vitest'
import {
  getEffectiveAccessRole,
  getRecommendedPermissionRequestRole,
  permissionRequestRoleCovers,
  permissionSetCoversRequestRole,
} from './permissionRequestRole'

describe('permissionRequestRole', () => {
  it('recommends member by default and admin after member access exists', () => {
    expect(getRecommendedPermissionRequestRole(undefined)).toBe('member')
    expect(getRecommendedPermissionRequestRole({ read: true })).toBe('member')
    expect(getRecommendedPermissionRequestRole({ read: true, write: true, update: true })).toBe('admin')
    expect(getRecommendedPermissionRequestRole({ admin: true })).toBeNull()
  })

  it('recognizes effective hierarchical roles', () => {
    expect(getEffectiveAccessRole({})).toBeNull()
    expect(getEffectiveAccessRole({ read: true })).toBe('viewer')
    expect(getEffectiveAccessRole({ read: true, write: true, update: true })).toBe('member')
    expect(getEffectiveAccessRole({ admin: true })).toBe('admin')
    expect(getEffectiveAccessRole({ owner: true })).toBe('owner')
  })

  it('treats admin and owner as covering lower request roles', () => {
    expect(permissionSetCoversRequestRole({ admin: true }, 'member')).toBe(true)
    expect(permissionSetCoversRequestRole({ owner: true }, 'admin')).toBe(true)
    expect(permissionSetCoversRequestRole({ read: true, write: true }, 'member')).toBe(false)
  })

  it('only treats a pending parent request as inherited when its role covers the requested role', () => {
    expect(permissionRequestRoleCovers('admin', 'member')).toBe(true)
    expect(permissionRequestRoleCovers('member', 'member')).toBe(true)
    expect(permissionRequestRoleCovers('member', 'admin')).toBe(false)
    expect(permissionRequestRoleCovers('viewer', 'member')).toBe(false)
  })
})
