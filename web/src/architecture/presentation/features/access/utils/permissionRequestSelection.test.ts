import { describe, expect, it } from 'vitest'
import {
  getPermissionRequestCheckedPaths,
  getPermissionRequestTargetPaths,
} from './permissionRequestSelection'

describe('permission request selection', () => {
  it('keeps readable and pending resources checked while excluding them from new requests', () => {
    const readablePaths = new Set(['/app', '/app/readable'])
    const pendingPaths = new Set(['/app/pending'])

    expect(getPermissionRequestCheckedPaths(
      ['/app/new', '/app/readable'],
      readablePaths,
      pendingPaths,
    )).toEqual([
      '/app',
      '/app/readable',
      '/app/pending',
      '/app/new',
    ])

    expect(getPermissionRequestTargetPaths(
      ['/app/new', '/app/readable', '/app/pending', '/app/new'],
      readablePaths,
      pendingPaths,
    )).toEqual(['/app/new'])
  })
})
