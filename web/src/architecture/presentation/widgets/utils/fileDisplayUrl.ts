import type { FileItem } from '../filesWidgetTypes'

function isAbsoluteHttpUrl(value: string): boolean {
  return value.startsWith('http://') || value.startsWith('https://')
}

export function normalizeFileDisplayUrl(rawUrl?: string): string {
  const value = String(rawUrl || '').trim()
  if (!value) {
    return ''
  }

  if (value.startsWith('/')) {
    return value
  }

  if (isAbsoluteHttpUrl(value)) {
    return value
  }

  return `/storage/api/v1/download/${encodeURIComponent(value)}`
}

export function getFileDisplayUrl(file: Pick<FileItem, 'url' | 'server_url'>): string {
  return normalizeFileDisplayUrl(file.url || file.server_url || '')
}
