function normalizeBaseURL(value?: string): string {
  const trimmed = (value || '').trim()
  if (!trimmed) {
    return ''
  }
  return trimmed.replace(/\/+$/, '')
}

export function getApiBaseURL(): string {
  return normalizeBaseURL(import.meta.env.VITE_API_BASE_URL)
}

export function getWebSocketBaseURL(): string {
  return normalizeBaseURL(import.meta.env.VITE_WS_URL)
}
