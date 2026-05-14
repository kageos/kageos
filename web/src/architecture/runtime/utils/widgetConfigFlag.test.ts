import { describe, expect, it } from 'vitest'
import { isWidgetConfigFlagEnabled } from './widgetConfigFlag'

describe('isWidgetConfigFlagEnabled', () => {
  it('accepts boolean and serialized true values', () => {
    expect(isWidgetConfigFlagEnabled(true)).toBe(true)
    expect(isWidgetConfigFlagEnabled('true')).toBe(true)
    expect(isWidgetConfigFlagEnabled(' TRUE ')).toBe(true)
    expect(isWidgetConfigFlagEnabled(1)).toBe(true)
    expect(isWidgetConfigFlagEnabled('1')).toBe(true)
  })

  it('rejects false and unrelated values', () => {
    expect(isWidgetConfigFlagEnabled(false)).toBe(false)
    expect(isWidgetConfigFlagEnabled('false')).toBe(false)
    expect(isWidgetConfigFlagEnabled(0)).toBe(false)
    expect(isWidgetConfigFlagEnabled(undefined)).toBe(false)
  })
})
