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

    expect(buildCapabilityBundleFileName(bundle, '/system/tools/excel_tools')).toBe('excel_tools.directory.json')
  })

  it('sanitizes path separators and whitespace', () => {
    const bundle: CapabilityBundle = {
      schema_version: 'capability.bundle.v1',
      name: 'sales report/2026',
      packages: [],
      files: []
    }

    expect(buildCapabilityBundleFileName(bundle)).toBe('sales-report-2026.directory.json')
  })

  it('parses directory json', () => {
    const parsed = parseCapabilityBundleJson(JSON.stringify({
      schema_version: 'capability.bundle.v1',
      name: '消息能力',
      tree_nodes: [
        { relative_path: 'message', type: 'package', code: 'message', name: '消息' },
        { relative_path: 'message/send.form', parent_path: 'message', type: 'function', code: 'send.form', name: '发送消息', template_type: 'form' },
        { relative_path: 'message/readme.docs', parent_path: 'message', type: 'docs', code: 'readme.docs', name: '使用说明' }
      ],
      docs: [{ relative_path: 'message/readme.docs', name: '使用说明', content: '# 使用说明\n', format: 'markdown' }],
      packages: [{ path: 'message', name: '消息' }],
      files: [{ package_path: 'message', path: 'send.go', content: 'package message\n' }]
    }))

    expect(parsed.tree_nodes?.[1]).toMatchObject({
      relative_path: 'message/send.form',
      parent_path: 'message',
      type: 'function',
      template_type: 'form'
    })
    expect(parsed.docs?.[0]).toEqual({
      relative_path: 'message/readme.docs',
      name: '使用说明',
      content: '# 使用说明\n',
      format: 'markdown',
      summary: undefined,
      category: undefined
    })
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
