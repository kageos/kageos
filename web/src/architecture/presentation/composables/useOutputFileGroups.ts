/**
 * 从任意「工具/接口返回的 result」中解析出文件组，供工作台工具结果、或其他输出文件展示复用。
 * 约定：任意层级出现 { files: FileItem[] } 或根节点本身是该结构，都视为一组输出文件。
 */

export interface OutputFileItem {
  name?: string
  source_name?: string
  url: string
  server_url?: string
  size?: number
  [key: string]: unknown
}

export interface OutputFileGroup {
  label: string
  files: OutputFileItem[]
}

/**
 * 从 result（字符串或对象）中递归提取所有「含 files 数组」的结构，返回文件组列表。
 * 可用于 run_form_submit、run_official_python、ffmpeg 等任意返回 types.Files 的工具结果。
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
      return []
    }
  } else if (typeof result === 'object' && result !== null) {
    obj = result
  } else {
    return []
  }

  const groups: OutputFileGroup[] = []
  const visited = new WeakSet<object>()

  const collectFiles = (node: unknown): OutputFileItem[] => {
    if (!isFilesGroup(node)) return []
    const list: OutputFileItem[] = []
    for (const f of node.files) {
      if (!f || typeof f !== 'object') continue
      const item = f as Record<string, unknown>
      const url = item.url ?? item.server_url
      if (url && typeof url === 'string') list.push({ ...item, url } as OutputFileItem)
    }
    return list
  }

  const visit = (node: unknown, path: Array<string | number>) => {
    if (!node || typeof node !== 'object') return
    if (visited.has(node)) return
    visited.add(node)

    const files = collectFiles(node)
    if (files.length > 0) {
      groups.push({ label: buildGroupLabel(path), files })
    }

    if (Array.isArray(node)) {
      node.forEach((item, index) => visit(item, [...path, index]))
      return
    }

    for (const [key, value] of Object.entries(node)) {
      if (key === 'files' && files.length > 0) continue
      visit(value, [...path, key])
    }
  }

  visit(obj, [])
  return groups
}

function isFilesGroup(node: unknown): node is { files: unknown[] } {
  return !!node && typeof node === 'object' && !Array.isArray(node) && Array.isArray((node as Record<string, unknown>).files)
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
