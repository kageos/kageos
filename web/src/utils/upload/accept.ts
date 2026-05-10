export interface AcceptFileLike {
  name: string
  type: string
}

export function isAcceptAll(accept?: string): boolean {
  if (!accept) {
    return true
  }

  const patterns = accept
    .split(',')
    .map(pattern => pattern.trim().toLowerCase())
    .filter(Boolean)

  if (patterns.length === 0) {
    return true
  }

  return patterns
    .some(pattern => pattern === '*' || pattern === '*/*')
}

export function formatAcceptLabel(accept?: string): string {
  return isAcceptAll(accept) ? '任意格式' : (accept || '任意格式')
}

export function isFileAccepted(file: AcceptFileLike, accept?: string): boolean {
  if (isAcceptAll(accept)) {
    return true
  }

  const acceptList = (accept || '')
    .split(',')
    .map(pattern => pattern.trim().toLowerCase())
    .filter(Boolean)
  const fileName = file.name.toLowerCase()
  const fileType = file.type.toLowerCase()

  return acceptList.some((pattern: string) => {
    if (pattern.startsWith('.')) {
      return fileName.endsWith(pattern)
    }
    if (pattern.endsWith('/*')) {
      const prefix = pattern.slice(0, -1)
      return prefix !== '*/' && !!fileType && fileType.startsWith(prefix)
    }
    return !!fileType && fileType === pattern
  })
}
