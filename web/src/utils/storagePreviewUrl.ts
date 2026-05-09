const THUMBNAIL_SUFFIX = '.thumb'
const THUMBNAIL_EXTENSION = '.webp'

export function deriveThumbnailPreviewUrl(rawUrl?: string): string {
  const value = String(rawUrl || '').trim()
  if (!value || value.startsWith('blob:') || value.startsWith('data:')) {
    return ''
  }

  try {
    const url = new URL(value)
    const pathname = replacePathExtension(url.pathname)
    if (!pathname) {
      return ''
    }
    url.pathname = pathname
    return url.toString()
  } catch {
    // Not an absolute URL; handle it as a proxy path or storage ref.
  }

  const match = value.match(/^([^?#]+)(.*)$/)
  if (!match || !match[1]) {
    return ''
  }
  const pathname = replacePathExtension(match[1])
  return pathname ? `${pathname}${match[2] || ''}` : ''
}

function replacePathExtension(pathname: string): string {
  if (!pathname) {
    return ''
  }
  if (pathname.endsWith(`${THUMBNAIL_SUFFIX}${THUMBNAIL_EXTENSION}`)) {
    return pathname
  }

  return `${pathname}${THUMBNAIL_SUFFIX}${THUMBNAIL_EXTENSION}`
}
