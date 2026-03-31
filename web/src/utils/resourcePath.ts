export interface ParsedResourcePath {
  resourcePath: string
  user: string
  app: string
  segments: string[]
}

export function normalizeResourcePath(resourcePath: string | null | undefined): string {
  const trimmed = String(resourcePath || '').trim()
  if (!trimmed) return ''
  const segments = trimmed.split('/').filter(Boolean)
  return segments.length > 0 ? `/${segments.join('/')}` : ''
}

export function parseResourcePath(resourcePath: string | null | undefined): ParsedResourcePath | null {
  const normalizedPath = normalizeResourcePath(resourcePath)
  if (!normalizedPath) return null

  const segments = normalizedPath.split('/').filter(Boolean)
  if (segments.length < 2) return null

  return {
    resourcePath: normalizedPath,
    user: segments[0] || '',
    app: segments[1] || '',
    segments
  }
}

export function buildAppResourcePath(user: string | null | undefined, app: string | null | undefined): string {
  const normalizedUser = String(user || '').trim()
  const normalizedApp = String(app || '').trim()
  if (!normalizedUser || !normalizedApp) return ''
  return `/${normalizedUser}/${normalizedApp}`
}
