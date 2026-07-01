import { describe, expect, it, vi } from 'vitest'
import type { ServiceTree } from '@/architecture/domain/types'
import { findNodeByPath } from '../utils/workspaceUtils'
import { resolveWorkspaceRootNodeForRoute } from './useWorkspaceViewLifecycle'

function node(patch: Partial<ServiceTree>): ServiceTree {
  return {
    id: patch.id ?? 1,
    name: patch.name ?? 'Node',
    code: patch.code ?? 'node',
    type: patch.type ?? 'package',
    description: '',
    tags: '',
    app_id: 1,
    ref_id: 0,
    full_code_path: patch.full_code_path ?? '/user/app',
    created_at: '',
    updated_at: '',
    children: patch.children,
  }
}

describe('resolveWorkspaceRootNodeForRoute', () => {
  it('selects the exact workspace root package for an app root route', () => {
    const root = node({ id: 10, name: 'Workspace', full_code_path: '/user/app' })
    const child = node({ id: 11, name: 'Directory', full_code_path: '/user/app/directory' })

    expect(resolveWorkspaceRootNodeForRoute(
      '/workspace/user/app',
      [{ ...root, children: [child] }],
      findNodeByPath
    )?.id).toBe(10)
  })

  it('falls back to the first top-level package when the exact root is not present', () => {
    const fallbackRoot = node({ id: 20, name: 'Fallback', full_code_path: '/user/app/main' })

    expect(resolveWorkspaceRootNodeForRoute(
      '/workspace/user/app',
      [fallbackRoot],
      vi.fn().mockReturnValue(null)
    )?.id).toBe(20)
  })

  it('does not select a default root for nested routes', () => {
    const root = node({ id: 30, full_code_path: '/user/app' })

    expect(resolveWorkspaceRootNodeForRoute(
      '/workspace/user/app/directory',
      [root],
      findNodeByPath
    )).toBeNull()
  })
})
