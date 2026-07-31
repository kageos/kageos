import { describe, expect, it } from 'vitest'
import {
  findNearestPermissionRequestAncestor,
  getPermissionRequestCheckedPaths,
  getPermissionRequestTargetPaths,
} from './permissionRequestSelection'

describe('permission request selection', () => {
  it('keeps readable and pending resources checked while excluding them from new requests', () => {
    const readablePaths = new Set(['/app/readable'])
    const pendingPaths = new Set(['/app/pending'])

    expect(getPermissionRequestCheckedPaths(
      ['/app/new', '/app/readable'],
      readablePaths,
      pendingPaths,
    )).toEqual([
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

  it('submits only the selected parent and treats descendants as inherited', () => {
    expect(getPermissionRequestTargetPaths(
      ['/app/parent/child', '/app/sibling', '/app/parent'],
      new Set(),
      new Set(),
    )).toEqual(['/app/sibling', '/app/parent'])

    expect(findNearestPermissionRequestAncestor(
      '/app/parent/child/function.form',
      ['/app', '/app/parent'],
    )).toBe('/app/parent')
  })

  it('does not create a child request while its parent request is pending', () => {
    expect(getPermissionRequestTargetPaths(
      ['/app/parent/child'],
      new Set(),
      new Set(['/app/parent']),
    )).toEqual([])
  })
})
