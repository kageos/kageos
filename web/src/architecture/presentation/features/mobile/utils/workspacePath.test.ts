import { describe, expect, it } from 'vitest'
import { resolveMobileWorkspacePath } from './workspacePath'

describe('resolveMobileWorkspacePath', () => {
  it('keeps the concrete function for a scheduled website monitor notification', () => {
    expect(resolveMobileWorkspacePath(
      '/system/democase/site_monitor/sweep.form',
      '/system/democase/site_monitor/sweep.form',
      '/system/democase/site_monitor',
      'form',
    )).toBe('/system/democase/site_monitor/sweep.form')
  })

  it('removes query parameters without discarding the function path', () => {
    expect(resolveMobileWorkspacePath('/system/democase/site_monitor/sweep.form?from=feishu'))
      .toBe('/system/democase/site_monitor/sweep.form')
  })

  it('keeps a directory source unchanged', () => {
    expect(resolveMobileWorkspacePath('/system/democase/site_monitor', undefined, '/system/democase'))
      .toBe('/system/democase/site_monitor')
  })
})
