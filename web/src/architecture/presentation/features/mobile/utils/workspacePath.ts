export function resolveMobileWorkspacePath(
  sourcePath?: string,
  fullCodePath?: string,
  parentPath?: string,
  _templateType?: string,
) {
  for (const rawCandidate of [sourcePath, fullCodePath]) {
    const candidate = normalizePath(rawCandidate)
    if (!candidate) continue
    return candidate
  }
  return normalizePath(parentPath)
}

function normalizePath(value?: string) {
  return (value || '').trim().replace(/[?#].*$/, '').replace(/\/+$/, '')
}
