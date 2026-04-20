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

  return ''
}

export function getFileDisplayUrl(file: Pick<FileItem, 'download_url' | 'server_download_url' | 'ref'>): string {
  return normalizeFileDisplayUrl(file.download_url || file.server_download_url || file.ref || '')
}
