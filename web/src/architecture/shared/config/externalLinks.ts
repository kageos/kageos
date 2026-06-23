import type { SupportedLocale } from '@/architecture/shared/i18n'

export type KageosDocSlug =
  | 'docs'
  | 'architecture'
  | 'connectors'
  | 'login'
  | 'runtime'
  | 'permissions'
  | 'automation'
  | 'hub'
  | 'operations'
  | 'api'

function normalizeBaseURL(value: unknown, fallback: string): string {
  const trimmed = typeof value === 'string' ? value.trim() : ''
  return (trimmed || fallback).replace(/\/+$/, '')
}

const kageosDocsBaseURL = normalizeBaseURL(
  import.meta.env.VITE_KAGEOS_DOCS_BASE_URL || import.meta.env.VITE_KAGEOS_WEBSITE_URL,
  'https://kageos.com'
)

export function getKageosDocsURL(slug: KageosDocSlug = 'docs', locale?: SupportedLocale | string): string {
  const docsPrefix = locale?.toLowerCase().startsWith('zh') ? '/zh/docs' : '/docs'
  const suffix = slug === 'docs' ? '' : `/${slug}`
  return `${kageosDocsBaseURL}${docsPrefix}${suffix}`
}

export function openExternalURL(url: string): void {
  window.open(url, '_blank', 'noopener,noreferrer')
}
