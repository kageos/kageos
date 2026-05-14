import { describe, expect, it } from 'vitest'
import { buildAppResourcePath, normalizeResourcePath, parseResourcePath } from './resourcePath'

describe('resourcePath', () => {
  it('normalizes resource paths', () => {
    expect(normalizeResourcePath('luobei/demo/report')).toBe('/luobei/demo/report')
    expect(normalizeResourcePath(' /luobei/demo//report/ ')).toBe('/luobei/demo/report')
    expect(normalizeResourcePath('')).toBe('')
  })

  it('parses canonical resource paths', () => {
    expect(parseResourcePath('/luobei/demo/report')).toEqual({
      resourcePath: '/luobei/demo/report',
      user: 'luobei',
      app: 'demo',
      segments: ['luobei', 'demo', 'report']
    })
    expect(parseResourcePath('/luobei/demo/*')).toEqual({
      resourcePath: '/luobei/demo/*',
      user: 'luobei',
      app: 'demo',
      segments: ['luobei', 'demo', '*']
    })
    expect(parseResourcePath('/broken')).toBeNull()
  })

  it('builds app resource paths', () => {
    expect(buildAppResourcePath('luobei', 'demo')).toBe('/luobei/demo')
    expect(buildAppResourcePath('', 'demo')).toBe('')
  })
})
