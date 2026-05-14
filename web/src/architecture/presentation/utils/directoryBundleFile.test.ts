import { describe, expect, it } from 'vitest'

import type { CapabilityBundle } from '@/architecture/domain/types'
import { buildCapabilityBundleFileName, parseCapabilityBundleJson } from './directoryBundleFile'

describe('directoryBundleFile', () => {
  it('uses the capability name for stable bundle filenames', () => {
    const bundle: CapabilityBundle = {
      schema_version: 'capability.bundle.v1',
      name: 'excel_tools',
      packages: [],
      files: []
    }

    expect(buildCapabilityBundleFileName(bundle, '/system/tools/excel_tools')).toBe('excel_tools.capability-bundle.json')
  })

  it('sanitizes path separators and whitespace', () => {
    const bundle: CapabilityBundle = {
      schema_version: 'capability.bundle.v1',
      name: 'sales report/2026',
      packages: [],
      files: []
    }

    expect(buildCapabilityBundleFileName(bundle)).toBe('sales-report-2026.capability-bundle.json')
  })

  it('parses capability bundle json', () => {
    const parsed = parseCapabilityBundleJson(JSON.stringify({
      schema_version: 'capability.bundle.v1',
      name: '消息能力',
      packages: [{ path: 'message', name: '消息' }],
      files: [{ package_path: 'message', path: 'send.go', content: 'package message\n' }]
    }))

    expect(parsed.files[0]).toEqual({
      package_path: 'message',
      path: 'send.go',
      content: 'package message\n'
    })
  })

  it('rejects workspace-bound paths', () => {
    expect(() => parseCapabilityBundleJson(JSON.stringify({
      schema_version: 'capability.bundle.v1',
      packages: [{ path: 'namespace/system/openapi/message' }],
      files: [{ package_path: 'namespace/system/openapi/message', path: 'send.go', content: '' }]
    }))).toThrow('工作空间路径')
  })
})
