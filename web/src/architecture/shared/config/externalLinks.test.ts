import { describe, expect, it, vi } from 'vitest'

import { getKageosHubURL, openExternalURL } from './externalLinks'

describe('externalLinks', () => {
  it('uses the official kageos.com Hub domain', () => {
    expect(getKageosHubURL()).toBe('https://hub.kageos.com')
  })

  it('opens external links without giving the new page an opener', () => {
    const open = vi.spyOn(window, 'open').mockImplementation(() => null)

    openExternalURL(getKageosHubURL())

    expect(open).toHaveBeenCalledWith('https://hub.kageos.com', '_blank', 'noopener,noreferrer')
    open.mockRestore()
  })
})
