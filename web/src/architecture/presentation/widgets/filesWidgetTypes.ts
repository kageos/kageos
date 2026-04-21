export interface FileItem {
  ref: string
  bucket?: string
  key?: string
  name: string
  source_name?: string
  storage?: string
  description?: string
  hash?: string
  size: number
  upload_ts?: number
  content_type?: string
  is_uploaded?: boolean
  download_url?: string
  server_download_url?: string
  downloaded?: boolean
  upload_user?: string
  error?: string
}

export type FilesData = string

export function parseFileRefs(value: unknown): string[] {
  if (typeof value !== 'string') {
    return []
  }
  return value
    .split(',')
    .map(item => item.trim().replace(/^\/+/, ''))
    .filter(Boolean)
}

export function stringifyFileRefs(refs: string[]): string {
  const seen = new Set<string>()
  const normalized: string[] = []
  refs.forEach((ref) => {
    const value = String(ref || '').trim().replace(/^\/+/, '')
    if (!value || seen.has(value)) {
      return
    }
    seen.add(value)
    normalized.push(value)
  })
  return normalized.join(',')
}

export function fileNameFromRef(ref: string): string {
  const clean = ref.trim().split('?')[0] || ref
  const parts = clean.split('/').filter(Boolean)
  return parts[parts.length - 1] || '文件'
}
