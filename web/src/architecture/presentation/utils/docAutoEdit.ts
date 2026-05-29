const DOC_AUTO_EDIT_STORAGE_PREFIX = 'kageos:auto-edit-doc:'

function normalizeDocPath(fullCodePath: string): string {
  const path = fullCodePath.trim()
  if (!path) return ''
  return path.startsWith('/') ? path : `/${path}`
}

export function markDocForAutoEdit(fullCodePath: string | undefined): void {
  if (typeof window === 'undefined' || !fullCodePath) return

  const normalizedPath = normalizeDocPath(fullCodePath)
  if (!normalizedPath) return

  try {
    window.sessionStorage.setItem(`${DOC_AUTO_EDIT_STORAGE_PREFIX}${normalizedPath}`, '1')
  } catch {
    // Session storage can be unavailable in privacy modes; creation should still succeed.
  }
}

export function consumeDocAutoEdit(fullCodePath: string | undefined): boolean {
  if (typeof window === 'undefined' || !fullCodePath) return false

  const normalizedPath = normalizeDocPath(fullCodePath)
  if (!normalizedPath) return false

  const key = `${DOC_AUTO_EDIT_STORAGE_PREFIX}${normalizedPath}`
  try {
    const shouldAutoEdit = window.sessionStorage.getItem(key) === '1'
    if (shouldAutoEdit) {
      window.sessionStorage.removeItem(key)
    }
    return shouldAutoEdit
  } catch {
    return false
  }
}
