import { describe, expect, it } from 'vitest'

import { getNetworkUnavailableMessage, isLocalKageOSHostname } from './networkUnavailable'

describe('networkUnavailable', () => {
  it.each(['localhost', 'dev.localhost', '127.0.0.1', '::1', '[::1]'])(
    'recognizes %s as a local installation host',
    (hostname) => {
      expect(isLocalKageOSHostname(hostname)).toBe(true)
    }
  )

  it('explains how to recover a local installation', () => {
    expect(getNetworkUnavailableMessage({ hostname: 'localhost', online: true }))
      .toBe('无法连接本地 kageos 服务，请先启动本地服务后重试')
  })

  it('keeps the hosted installation message environment-neutral', () => {
    expect(getNetworkUnavailableMessage({ hostname: 'app.kageos.com', online: true }))
      .toBe('无法连接 kageos 服务，请检查网络连接或稍后重试')
  })

  it('prioritizes the browser offline state', () => {
    expect(getNetworkUnavailableMessage({ hostname: 'localhost', online: false }))
      .toBe('网络已断开，请检查网络连接')
  })
})
