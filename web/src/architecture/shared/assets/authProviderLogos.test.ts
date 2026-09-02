import { describe, expect, it } from 'vitest'
import { authProviderLogo, authProviderLogos } from './authProviderLogos'

describe('authProviderLogo', () => {
  it('returns offline brand artwork for every built-in third-party login provider', () => {
    expect(authProviderLogos.google_oauth).toMatch(/^data:image\/svg\+xml,/)
    expect(authProviderLogos.github_oauth).toMatch(/^data:image\/svg\+xml,/)
    expect(authProviderLogos.wechat_open_oauth).toMatch(/^data:image\/svg\+xml,/)
    expect(authProviderLogos.wechat_official).toBe(authProviderLogos.wechat_open_oauth)
    expect(authProviderLogos.kageos_auth).toBe('/brand/kageos-avatar-128.png')
  })

  it('uses a neutral icon for future providers instead of letter placeholders', () => {
    expect(authProviderLogo('custom_oidc')).toMatch(/^data:image\/svg\+xml,/)
  })
})
