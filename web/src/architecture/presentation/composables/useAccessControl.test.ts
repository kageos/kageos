import { describe, expect, it } from 'vitest'
import { can, canUseWorkstation } from './useAccessControl'

describe('useAccessControl', () => {
  it('allows owner to do everything', () => {
    expect(can({ owner: true }, 'delete')).toBe(true)
    expect(can({ owner: true }, 'admin')).toBe(true)
  })

  it('allows admin to do non-owner actions', () => {
    expect(can({ admin: true }, 'delete')).toBe(true)
    expect(can({ admin: true }, 'owner')).toBe(false)
  })

  it('does not allow member-like permissions to delete', () => {
    expect(can({ read: true, write: true, update: true }, 'delete')).toBe(false)
  })

  it('allows the workstation for member and above but not viewer', () => {
    expect(canUseWorkstation({ permissions: { read: true } })).toBe(false)
    expect(canUseWorkstation({ permissions: { read: true, write: true, update: true } })).toBe(true)
    expect(canUseWorkstation({ permissions: { admin: true } })).toBe(true)
    expect(canUseWorkstation({ permissions: { owner: true } })).toBe(true)
  })
})
