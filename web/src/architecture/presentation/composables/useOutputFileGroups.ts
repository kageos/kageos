/**
 * 从工具结构化结果中解析输出文件组，供工作台展示复用。
 * 约定：任意文件字段返回 bucket/object_key 字符串；多文件用英文逗号分隔。
 * 文件字段必须由 metadata.display_file_fields 显式声明，不从普通结果里猜。
 */

import type { ToolResultMetadata } from '@/architecture/presentation/context/api/workspace'

export interface OutputFileItem {
  ref?: string
  name?: string
  source_name?: string
  download_url?: string
  thumbnail_ref?: string
  thumbnail_url?: string
  content_type?: string
  preview_kind?: string
  size?: number
  [key: string]: unknown
}

export interface OutputFileGroup {
  label: string
  files: OutputFileItem[]
}

/**
 * 从结构化 result（对象或 JSON 字符串）的显式文件字段中提取文件引用字符串。
 * 普通文本结果不做启发式解析，避免把 full_code_path、字体路径、通配符等说明文本误判为输出文件。
 */
export function extractFileGroupsFromResult(
  result: string | object | undefined,
  metadata?: ToolResultMetadata
): OutputFileGroup[] {
  if (result == null) return []
  let obj: unknown
  if (typeof result === 'string') {
    try {
      obj = JSON.parse(result) as unknown
    } catch {
      return []
    }
  } else if (typeof result === 'object' && result !== null) {
    obj = result
  } else {
    return []
  }

  return metadata == null ? [] : extractDeclaredFileGroups(obj, metadata)
}

function extractDeclaredFileGroups(obj: unknown, metadata: ToolResultMetadata): OutputFileGroup[] {
  if (!obj || typeof obj !== 'object') return []

  const fields = Array.isArray(metadata.display_file_fields) ? metadata.display_file_fields : []
  const groups: OutputFileGroup[] = []
  for (const field of fields) {
    const path = field.trim()
    if (!path) continue

    const value = getValueByPath(obj, path)
    if (typeof value !== 'string') continue

    const refs = parseRefs(value)
    if (refs.length === 0) continue
    groups.push({ label: buildGroupLabel(path.split('.')), files: refsToItems(refs) })
  }
  return groups
}

function getValueByPath(obj: unknown, path: string): unknown {
  if (!obj || typeof obj !== 'object') return undefined
  const direct = (obj as Record<string, unknown>)[path]
  if (direct !== undefined) return direct

  let current: unknown = obj
  for (const segment of path.split('.')) {
    if (!current || typeof current !== 'object') return undefined
    current = (current as Record<string, unknown>)[segment]
  }
  return current
}

function parseRefs(value: string): string[] {
  const refs = value
    .split(',')
    .map(normalizeRef)

  if (refs.length === 0 || refs.some(ref => !isValidFileRef(ref))) {
    return []
  }
  return refs
}

function normalizeRef(value: string): string {
  return value.trim().replace(/^\/+/, '')
}

function isValidFileRef(ref: string): boolean {
  if (!ref || ref.includes('*')) return false
  if (/[,\s]/.test(ref)) return false
  const slashIndex = ref.indexOf('/')
  if (slashIndex <= 0 || slashIndex === ref.length - 1) return false

  const bucket = ref.slice(0, slashIndex)
  const key = ref.slice(slashIndex + 1)
  if (!/^[A-Za-z0-9][A-Za-z0-9._-]{0,62}$/.test(bucket)) return false
  if (key.split('/').some(segment => segment.length === 0)) return false
  return true
}

function refsToItems(refs: string[]): OutputFileItem[] {
  return refs.map(ref => ({
    ref,
    name: ref.split('/').pop() || '文件',
  }))
}

function buildGroupLabel(path: Array<string | number>): string {
  if (path.length === 0) return 'Output Files'
  return path.map(formatPathSegment).join(' / ')
}

function formatPathSegment(segment: string | number): string {
  if (typeof segment === 'number') return `#${segment + 1}`

  return segment
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/[_-]+/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase())
}
