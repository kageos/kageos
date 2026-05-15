import type { FieldValue } from '@/architecture/domain/types/field'

export function isStoredSearchFieldValue(value: unknown): value is FieldValue {
  return !!value && typeof value === 'object' && 'raw' in value && 'display' in value
}

export function getSearchFieldRawValue<T = unknown>(value: unknown): T {
  return isStoredSearchFieldValue(value) ? (value.raw as T) : (value as T)
}

export function getSearchFieldHideValue(value: unknown): string {
  if (!isStoredSearchFieldValue(value)) {
    return ''
  }

  return typeof value.display === 'string' ? value.display : String(value.display ?? '')
}

export function hasSearchFieldValue(value: unknown): boolean {
  const rawValue = getSearchFieldRawValue(value)

  if (rawValue === null || rawValue === undefined) {
    return false
  }

  if (Array.isArray(rawValue)) {
    return rawValue.length > 0
  }

  if (typeof rawValue === 'string') {
    return rawValue.trim() !== ''
  }

  return true
}

export function createStoredSearchFieldValue(raw: unknown, display?: string | null, meta?: Record<string, unknown>): unknown {
  if (display === undefined || display === null || display === '') {
    return raw
  }

  return {
    raw,
    display: String(display),
    meta: meta || {}
  }
}
