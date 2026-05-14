import { describe, expect, it } from 'vitest'
import type { ServiceTree } from '@/types'
import {
  canApprovePermissionRequest,
  isServiceTreeNodeAdmin,
  parseUsernameList
} from './permissionActors'

describe('permissionActors', () => {
  it('parses username lists with trim and dedupe', () => {
    expect(parseUsernameList(' alice, bob,alice , ,carol ')).toEqual(['alice', 'bob', 'carol'])
    expect(parseUsernameList([' alice ', 'bob', 'alice'])).toEqual(['alice', 'bob'])
  })

  it('checks permission request approvers against current user', () => {
    expect(canApprovePermissionRequest('alice', ['bob', 'alice'], 'pending')).toBe(true)
    expect(canApprovePermissionRequest('alice', ['bob'], 'pending')).toBe(false)
    expect(canApprovePermissionRequest('alice', ['alice'], 'approved')).toBe(false)
  })

  it('checks node admin access from owner and explicit admin lists', () => {
    expect(isServiceTreeNodeAdmin({ admins: 'bob', owner: 'alice' } as ServiceTree, 'alice')).toBe(true)
    expect(isServiceTreeNodeAdmin({ admins: 'alice,bob' } as ServiceTree, 'alice')).toBe(true)
    expect(isServiceTreeNodeAdmin({ admins: 'bob' } as ServiceTree, 'alice')).toBe(false)
  })

})
