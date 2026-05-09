import type { DirectoryBundle } from '@/api/service-tree'

const FALLBACK_DIRECTORY_BUNDLE_FILENAME = 'directory-bundle.json'

function sanitizeFilenamePart(value: string): string {
  return value
    .trim()
    .replace(/[\\/:*?"<>|]+/g, '-')
    .replace(/\s+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 80)
}

export function buildDirectoryBundleFileName(bundle: DirectoryBundle, sourceDirectoryPath?: string): string {
  const pathName = sourceDirectoryPath?.split('/').filter(Boolean).pop()
  const rawName = bundle.root?.code || bundle.root?.name || pathName || 'directory'
  const safeName = sanitizeFilenamePart(rawName)
  if (!safeName) {
    return FALLBACK_DIRECTORY_BUNDLE_FILENAME
  }
  return `${safeName}.directory-bundle.json`
}

export function downloadDirectoryBundleFile(bundle: DirectoryBundle, sourceDirectoryPath?: string): void {
  const blob = new Blob([`${JSON.stringify(bundle, null, 2)}\n`], {
    type: 'application/json;charset=utf-8'
  })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = buildDirectoryBundleFileName(bundle, sourceDirectoryPath)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
