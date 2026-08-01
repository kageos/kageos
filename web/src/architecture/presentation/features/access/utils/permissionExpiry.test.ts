import { describe, expect, it } from 'vitest'
import { disablePastPermissionDate, isPermissionExpiryValid } from './permissionExpiry'

describe('permission expiry validation', () => {
  const now = new Date(2026, 7, 1, 12, 0, 0)

  it('requires a future date for custom expiration', () => {
    expect(isPermissionExpiryValid(true, null, now)).toBe(true)
    expect(isPermissionExpiryValid(false, null, now)).toBe(false)
    expect(isPermissionExpiryValid(false, new Date(2026, 7, 1, 11, 59, 59), now)).toBe(false)
    expect(isPermissionExpiryValid(false, new Date(2026, 7, 1, 12, 0, 1), now)).toBe(true)
  })

  it('disables calendar days before today while keeping today selectable', () => {
    expect(disablePastPermissionDate(new Date(2026, 6, 31), now)).toBe(true)
    expect(disablePastPermissionDate(new Date(2026, 7, 1), now)).toBe(false)
    expect(disablePastPermissionDate(new Date(2026, 7, 2), now)).toBe(false)
  })
})
