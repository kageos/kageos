/**
 * 从任意「工具/接口返回的 result」中解析出文件组，供工作台工具结果、或其他输出文件展示复用。
 * 约定：任意文件字段返回 bucket/object_key 字符串；多文件用英文逗号分隔。
 */

export interface OutputFileItem {
  ref?: string
  name?: string
  source_name?: string
  download_url?: string
  size?: number
  [key: string]: unknown
}

export interface OutputFileGroup {
  label: string
  files: OutputFileItem[]
}

/**
 * 从 result（字符串或对象）中递归提取所有文件引用字符串，返回文件组列表。
 * 可用于 run_form_submit、run_official_python、ffmpeg 等任意返回 string 的工具结果。
 */
export function extractFileGroupsFromResult(
  result: string | object | undefined
): OutputFileGroup[] {
  if (result == null) return []
  let obj: unknown
  if (typeof result === 'string') {
    try {
      obj = JSON.parse(result) as unknown
    } catch {
      const refs = parseRefs(result)
      return refs.length > 0 ? [{ label: 'Output Files', files: refsToItems(refs) }] : []
    }
  } else if (typeof result === 'object' && result !== null) {
    obj = result
  } else {
    return []
  }

  const groups: OutputFileGroup[] = []
  const visited = new WeakSet<object>()

  const visit = (node: unknown, path: Array<string | number>) => {
    if (!node || typeof node !== 'object') return
    if (visited.has(node)) return
    visited.add(node)

    if (Array.isArray(node)) {
      node.forEach((item, index) => visit(item, [...path, index]))
      return
    }

    for (const [key, value] of Object.entries(node)) {
      if (typeof value === 'string' && isFileLikeKey(key)) {
        const refs = parseRefs(value)
        if (refs.length > 0) {
          groups.push({ label: buildGroupLabel([...path, key]), files: refsToItems(refs) })
          continue
        }
      }
      visit(value, [...path, key])
    }
  }

  visit(obj, [])
  return groups
}

function isFileLikeKey(key: string): boolean {
  return /file|files|attachment|附件/i.test(key)
}

function parseRefs(value: string): string[] {
  return value
    .split(',')
    .map(item => item.trim().replace(/^\/+/, ''))
    .filter(item => item.includes('/'))
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
