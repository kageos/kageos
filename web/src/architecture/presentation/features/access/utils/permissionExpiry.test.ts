import { describe, expect, it } from 'vitest'
import { disablePastPermissionDate, isPermissionExpiryValid } from './permissionExpiry'

describe('permission expiry validation', () => {
  const now = new Date('2026-08-01T12:00:00+08:00')

  it('requires a future date for custom expiration', () => {
    expect(isPermissionExpiryValid(true, null, now)).toBe(true)
    expect(isPermissionExpiryValid(false, null, now)).toBe(false)
    expect(isPermissionExpiryValid(false, new Date('2026-08-01T11:59:59+08:00'), now)).toBe(false)
    expect(isPermissionExpiryValid(false, new Date('2026-08-01T12:00:01+08:00'), now)).toBe(true)
  })

  it('disables calendar days before today while keeping today selectable', () => {
    expect(disablePastPermissionDate(new Date('2026-07-31T00:00:00+08:00'), now)).toBe(true)
    expect(disablePastPermissionDate(new Date('2026-08-01T00:00:00+08:00'), now)).toBe(false)
    expect(disablePastPermissionDate(new Date('2026-08-02T00:00:00+08:00'), now)).toBe(false)
  })
})
