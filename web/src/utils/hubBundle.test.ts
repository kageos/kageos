import { describe, expect, it } from 'vitest'
import {
  HUB_DIRECTORY_BUNDLE_SCHEMA_VERSION,
  HUB_DIRECTORY_BUNDLE_TYPE,
  createHubDirectoryBundle,
  parseHubDirectoryBundleJson
} from './hubBundle'

describe('hubBundle', () => {
  it('parses canonical hub directory bundle', () => {
    const bundle = createHubDirectoryBundle({
      hub_directory_name: '投递简历',
      hub_full_code_path: '/hub/minimax/salon_hr/hr_resume_list',
      hub_version_num: 12,
      directory_tree: {
        type: 'package',
        name: '投递简历',
        children: []
      }
    })

    expect(parseHubDirectoryBundleJson(JSON.stringify(bundle))).toEqual({
      schema_version: HUB_DIRECTORY_BUNDLE_SCHEMA_VERSION,
      bundle_type: HUB_DIRECTORY_BUNDLE_TYPE,
      exported_at: undefined,
      directory_tree: {
        type: 'package',
        name: '投递简历',
        children: []
      },
      hub_directory_name: '投递简历',
      hub_full_code_path: '/hub/minimax/salon_hr/hr_resume_list',
      hub_version_num: 12
    })
  })

  it('rejects legacy export field names with directory_tree', () => {
    const legacyBundle = {
      name: '旧版目录',
      full_code_path: '/hub/legacy/example',
      version_num: 'v44',
      directory_tree: {
        type: 'package',
        name: '旧版目录',
        children: []
      }
    }

    expect(() => parseHubDirectoryBundleJson(JSON.stringify(legacyBundle))).toThrow(
      '不支持的安装包类型：undefined'
    )
  })

  it('rejects bare directory_tree payloads', () => {
    const bareTree = {
      type: 'package',
      name: '裸目录树',
      children: []
    }

    expect(() => parseHubDirectoryBundleJson(JSON.stringify(bareTree))).toThrow(
      '不支持的安装包类型：undefined'
    )
  })

  it('rejects unsupported bundle types', () => {
    const invalidBundle = {
      schema_version: HUB_DIRECTORY_BUNDLE_SCHEMA_VERSION,
      bundle_type: 'unknown_bundle',
      directory_tree: {
        type: 'package',
        name: '错误目录',
        children: []
      }
    }

    expect(() => parseHubDirectoryBundleJson(JSON.stringify(invalidBundle))).toThrow(
      '不支持的安装包类型：unknown_bundle'
    )
  })

  it('rejects unsupported schema versions', () => {
    const invalidBundle = {
      schema_version: 2,
      bundle_type: HUB_DIRECTORY_BUNDLE_TYPE,
      directory_tree: {
        type: 'package',
        name: '错误目录',
        children: []
      }
    }

    expect(() => parseHubDirectoryBundleJson(JSON.stringify(invalidBundle))).toThrow(
      '不支持的安装包 schema_version：2'
    )
  })

  it('rejects objects without canonical bundle metadata', () => {
    expect(() => parseHubDirectoryBundleJson(JSON.stringify({
      name: 'not-a-tree'
    }))).toThrow('不支持的安装包类型：undefined')
  })

  it('rejects canonical bundles without valid directory_tree object', () => {
    expect(() => parseHubDirectoryBundleJson(JSON.stringify({
      schema_version: HUB_DIRECTORY_BUNDLE_SCHEMA_VERSION,
      bundle_type: HUB_DIRECTORY_BUNDLE_TYPE,
      directory_tree: 'invalid'
    }))).toThrow('JSON 中缺少有效的 directory_tree 字段')
  })
})
