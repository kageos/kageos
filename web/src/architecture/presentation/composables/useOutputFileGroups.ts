/**
 * 从任意「工具/接口返回的 result」中解析出文件组，供工作台工具结果、或其他输出文件展示复用。
 * 约定：result 为 JSON 对象时，顶层任意 key 的值为 { files: FileItem[] } 即视为一组输出文件。
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
 * 从 result（字符串或对象）中提取所有「含 files 数组」的顶层字段，返回文件组列表。
 * 可用于 run_form_submit 等工具返回、或任意包含 { files: [] } 结构的 JSON。
 */
export function extractFileGroupsFromResult(
  result: string | object | undefined
): OutputFileGroup[] {
  if (result == null) return []
  let obj: Record<string, unknown>
  if (typeof result === 'string') {
    try {
      obj = JSON.parse(result) as Record<string, unknown>
    } catch {
      return []
    }
  } else if (typeof result === 'object' && result !== null) {
    obj = result as Record<string, unknown>
  } else {
    return []
  }
  const groups: OutputFileGroup[] = []
  for (const [key, val] of Object.entries(obj)) {
    if (!val || typeof val !== 'object' || !Array.isArray((val as Record<string, unknown>).files))
      continue
    const files = (val as { files: unknown[] }).files
    const list: OutputFileItem[] = []
    for (const f of files) {
      if (!f || typeof f !== 'object') continue
      const item = f as Record<string, unknown>
      const url = item.url ?? item.server_url
      if (url && typeof url === 'string') list.push({ ...item, url } as OutputFileItem)
    }
    if (list.length > 0) {
      const label = key.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
      groups.push({ label, files: list })
    }
  }
  return groups
}
