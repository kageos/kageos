const STORAGE_PROXY_PREFIXES = ['/ai-agent-os/']
const LOCAL_STORAGE_HOSTS = new Set(['localhost:9000', '127.0.0.1:9000', 'host.containers.internal:9000', 'minio:9000'])

function currentOrigin(): string {
  return typeof window === 'undefined' ? '' : window.location.origin
}

function withCurrentOrigin(path: string): string {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  const origin = currentOrigin()
  return origin ? `${origin}${normalizedPath}` : normalizedPath
}

function isStorageProxyPath(pathname: string): boolean {
  return STORAGE_PROXY_PREFIXES.some(prefix => pathname.startsWith(prefix))
}

export function normalizeStorageFileDisplayUrl(rawUrl?: string): string {
  const value = String(rawUrl || '').trim()
  if (!value) return ''
  if (value.startsWith('blob:') || value.startsWith('data:')) return value

  try {
    const url = new URL(value)
    if (url.protocol === 'http:' || url.protocol === 'https:') {
      if (LOCAL_STORAGE_HOSTS.has(url.host) || isStorageProxyPath(url.pathname)) {
        return withCurrentOrigin(`${url.pathname}${url.search}${url.hash}`)
      }
      return value
    }
    return value
  } catch {
    // Not an absolute URL; treat it as a storage proxy path/ref below.
  }

  return withCurrentOrigin(value)
}
