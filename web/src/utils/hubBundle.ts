/**
 * Hub 离线目录安装包协议
 *
 * 标准格式：
 * {
 *   "schema_version": 1,
 *   "bundle_type": "hub_directory_bundle",
 *   "exported_at": "2026-03-31T08:00:00Z",
 *   "hub_directory_name": "示例目录",
 *   "hub_full_code_path": "/hub/demo/example",
 *   "hub_version_num": 12,
 *   "directory_tree": { ... }
 * }
 */

export const HUB_DIRECTORY_BUNDLE_SCHEMA_VERSION = 1
export const HUB_DIRECTORY_BUNDLE_TYPE = 'hub_directory_bundle' as const

type UnknownRecord = Record<string, unknown>

export interface HubDirectoryBundle {
  schema_version: number
  bundle_type: typeof HUB_DIRECTORY_BUNDLE_TYPE
  exported_at?: string
  directory_tree: UnknownRecord
  hub_directory_name?: string
  hub_full_code_path?: string
  hub_version_num?: number
}

export interface ParsedHubDirectoryBundle {
  schema_version: number
  bundle_type: typeof HUB_DIRECTORY_BUNDLE_TYPE
  exported_at?: string
  directory_tree: UnknownRecord
  hub_directory_name?: string
  hub_full_code_path?: string
  hub_version_num?: number
}

function asRecord(value: unknown): UnknownRecord | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return null
  }
  return value as UnknownRecord
}

function normalizeOptionalString(value: unknown): string | undefined {
  if (typeof value !== 'string') {
    return undefined
  }
  const trimmed = value.trim()
  return trimmed ? trimmed : undefined
}

function normalizeOptionalVersionNumber(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }

  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) {
      return undefined
    }

    const normalized = trimmed.startsWith('v') || trimmed.startsWith('V')
      ? trimmed.slice(1)
      : trimmed
    const parsed = Number.parseInt(normalized, 10)
    if (!Number.isNaN(parsed)) {
      return parsed
    }
  }

  return undefined
}

export function createHubDirectoryBundle(bundle: Omit<HubDirectoryBundle, 'schema_version' | 'bundle_type'>): HubDirectoryBundle {
  return {
    schema_version: HUB_DIRECTORY_BUNDLE_SCHEMA_VERSION,
    bundle_type: HUB_DIRECTORY_BUNDLE_TYPE,
    ...bundle
  }
}

export function parseHubDirectoryBundleJson(text: string): ParsedHubDirectoryBundle {
  const parsed = JSON.parse(text) as unknown
  const root = asRecord(parsed)

  if (!root) {
    throw new Error('JSON 根节点必须是对象')
  }

  if (root.bundle_type !== HUB_DIRECTORY_BUNDLE_TYPE) {
    throw new Error(`不支持的安装包类型：${String(root.bundle_type)}`)
  }

  if (root.schema_version !== HUB_DIRECTORY_BUNDLE_SCHEMA_VERSION) {
    throw new Error(`不支持的安装包 schema_version：${String(root.schema_version)}`)
  }

  const directoryTree = asRecord(root.directory_tree)
  if (!directoryTree) {
    throw new Error('JSON 中缺少有效的 directory_tree 字段')
  }

  return {
    schema_version: HUB_DIRECTORY_BUNDLE_SCHEMA_VERSION,
    bundle_type: HUB_DIRECTORY_BUNDLE_TYPE,
    exported_at: normalizeOptionalString(root.exported_at),
    directory_tree: directoryTree,
    hub_directory_name: normalizeOptionalString(root.hub_directory_name),
    hub_full_code_path: normalizeOptionalString(root.hub_full_code_path),
    hub_version_num: normalizeOptionalVersionNumber(root.hub_version_num)
  }
}
