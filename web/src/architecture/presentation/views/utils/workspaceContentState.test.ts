import { describe, expect, it } from 'vitest'
import type { AccessPermissions, ServiceTree } from '@/architecture/domain/types'
import { resolveWorkspaceContentState } from './workspaceContentState'

function createNode(type: ServiceTree['type'], permissions: AccessPermissions): ServiceTree {
  return {
    id: 1,
    name: '测试节点',
    code: 'test',
    type,
    description: '',
    tags: '',
    app_id: 1,
    ref_id: 1,
    full_code_path: '/system/demo/test.form',
    permissions,
    created_at: '',
    updated_at: ''
  }
}

function resolveWithNode(
  currentFunction: ServiceTree | null,
  overrides: Partial<Parameters<typeof resolveWorkspaceContentState>[0]> = {}
) {
  return resolveWorkspaceContentState({
    hasWorkspaceAccessError: false,
    currentFunction,
    queryTab: 'run',
    hasCurrentFunctionDetail: true,
    isRestoringWorkspaceRoute: false,
    ...overrides
  })
}

describe('resolveWorkspaceContentState', () => {
  it.each<ServiceTree['type']>(['function', 'package', 'docs'])(
    'renders the access lock instead of unreadable %s content',
    (type) => {
      expect(resolveWithNode(createNode(type, { read: false }))).toBe('resource-locked')
    }
  )

  it('renders readable function content normally', () => {
    expect(resolveWithNode(createNode('function', { read: true }))).toBe('function')
  })

  it('allows an unreadable resource to open its permission records without exposing content', () => {
    expect(resolveWithNode(createNode('function', { read: false }), {
      panel: 'permission',
    })).toBe('resource-permission')
  })

  it('gives a workspace access error priority over a resource lock', () => {
    expect(resolveWithNode(createNode('function', { read: false }), {
      hasWorkspaceAccessError: true
    })).toBe('workspace-error')
  })

  it('keeps create and edit routes ahead of readable node content', () => {
    const currentFunction = createNode('function', { read: true })

    expect(resolveWithNode(currentFunction, { queryTab: 'create' })).toBe('create')
    expect(resolveWithNode(currentFunction, { queryTab: 'edit' })).toBe('edit')
  })

  it('shows the route restoring state before falling back to empty', () => {
    expect(resolveWithNode(null, { isRestoringWorkspaceRoute: true })).toBe('restoring')
    expect(resolveWithNode(null)).toBe('empty')
  })
})
