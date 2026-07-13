export function resolveMobileWorkspacePath(
  sourcePath?: string,
  fullCodePath?: string,
  parentPath?: string,
  templateType?: string,
) {
  const parent = normalizePath(parentPath)
  for (const rawCandidate of [sourcePath, fullCodePath]) {
    const candidate = normalizePath(rawCandidate)
    if (!candidate) continue
    if (isFunctionPath(candidate, templateType)) {
      if (parent) return parent
      const derived = parentDirectory(candidate)
      if (derived) return derived
    }
    return candidate
  }
  return parent
}

function normalizePath(value?: string) {
  return (value || '').trim().replace(/[?#].*$/, '').replace(/\/+$/, '')
}

function isFunctionPath(value: string, templateType?: string) {
  if (/\.(form|table|chart)$/i.test(value)) return true
  const normalizedType = (templateType || '')
    .trim()
    .toLowerCase()
    .replace(/_?template$/, '')
  return ['form', 'table', 'chart'].includes(normalizedType)
}

function parentDirectory(value: string) {
  const index = value.lastIndexOf('/')
  return index > 0 ? value.slice(0, index).replace(/\/+$/, '') : ''
}
