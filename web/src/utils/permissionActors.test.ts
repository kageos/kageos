import { describe, expect, it } from 'vitest'
import type { App, ServiceTree } from '@/types'
import {
  canApprovePermissionRequest,
  hasWorkspaceAdminAccess,
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

  it('prefers service tree permissions for workspace admin access and falls back to app admins', () => {
    const currentApp = {
      id: 1,
      user: 'luobei',
      code: 'demo',
      name: 'Demo',
      nats_id: 1,
      host_id: 1,
      status: 'enabled',
      version: 'v1',
      is_public: false,
      admins: 'fallback_admin',
      created_at: '',
      updated_at: ''
    } as App

    const treeWithAppAdmin = [{
      id: 1,
      name: 'root',
      code: 'root',
      type: 'package',
      description: '',
      tags: '',
      app_id: 1,
      ref_id: 1,
      full_code_path: '/luobei/demo',
      permissions: {
        'app:admin': true
      },
      created_at: '',
      updated_at: ''
    }] as ServiceTree[]

    expect(hasWorkspaceAdminAccess({
      currentApp,
      currentUsername: 'alice',
      serviceTree: treeWithAppAdmin
    })).toBe(true)

    expect(hasWorkspaceAdminAccess({
      currentApp,
      currentUsername: 'luobei',
      serviceTree: []
    })).toBe(true)

    expect(hasWorkspaceAdminAccess({
      currentApp,
      currentUsername: 'fallback_admin',
      serviceTree: []
    })).toBe(true)

    expect(hasWorkspaceAdminAccess({
      currentApp,
      currentUsername: 'visitor',
      serviceTree: []
    })).toBe(false)
  })
})
