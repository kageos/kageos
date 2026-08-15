import { afterEach, describe, expect, it } from 'vitest'

import { getCurrentPublicShareId } from './publicShare'

describe('getCurrentPublicShareId', () => {
  const originalPath = window.location.pathname

  afterEach(() => {
    window.history.replaceState({}, '', originalPath)
  })

  it('reads the share id from the canonical short route', () => {
    window.history.replaceState({}, '', '/s/ps_abc123')

    expect(getCurrentPublicShareId()).toBe('ps_abc123')
  })

  it('does not recognize the removed legacy route', () => {
    window.history.replaceState({}, '', '/public/s/ps_abc123')

    expect(getCurrentPublicShareId()).toBe('')
  })

  it('does not capture nested paths', () => {
    window.history.replaceState({}, '', '/s/ps_abc123/extra')

    expect(getCurrentPublicShareId()).toBe('')
  })
})
