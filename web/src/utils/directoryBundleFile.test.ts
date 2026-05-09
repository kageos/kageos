import { describe, expect, it } from 'vitest'

import type { DirectoryBundle } from '@/api/service-tree'
import { buildDirectoryBundleFileName } from './directoryBundleFile'

describe('directoryBundleFile', () => {
  it('uses the root code for stable bundle filenames', () => {
    const bundle: DirectoryBundle = {
      schema_version: 1,
      root: {
        code: 'excel_tools',
        name: 'Excel Tools'
      }
    }

    expect(buildDirectoryBundleFileName(bundle, '/system/tools/excel_tools')).toBe('excel_tools.directory-bundle.json')
  })

  it('sanitizes path separators and whitespace', () => {
    const bundle: DirectoryBundle = {
      schema_version: 1,
      root: {
        code: 'sales report/2026',
        name: 'Sales Report'
      }
    }

    expect(buildDirectoryBundleFileName(bundle)).toBe('sales-report-2026.directory-bundle.json')
  })
})
