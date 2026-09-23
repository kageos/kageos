import { describe, expect, it, vi } from 'vitest'

import { getKageosDocsURL, getKageosHubURL, getKageosWebsiteURL, openExternalURL } from './externalLinks'

describe('externalLinks', () => {
  it('links localized help to the documentation center, not the website homepage', () => {
    expect(getKageosDocsURL('docs', 'zh-CN')).toBe('https://kageos.com/zh/docs/')
    expect(getKageosDocsURL('data-backup', 'zh-CN')).toBe('https://kageos.com/zh/docs/data-backup/')
    expect(getKageosDocsURL('automation', 'en-US')).toBe('https://kageos.ai/docs/automation/')
    expect(getKageosDocsURL()).toBe('https://kageos.ai/docs/')
  })

  it('uses the official kageos.com Hub domain', () => {
    expect(getKageosHubURL()).toBe('https://hub.kageos.com')
  })

  it('uses the official website for the documentation center entry', () => {
    expect(getKageosWebsiteURL()).toBe('https://kageos.com')
  })

  it('opens external links without giving the new page an opener', () => {
    const open = vi.spyOn(window, 'open').mockImplementation(() => null)

    openExternalURL(getKageosHubURL())

    expect(open).toHaveBeenCalledWith('https://hub.kageos.com', '_blank', 'noopener,noreferrer')
    open.mockRestore()
  })
})
