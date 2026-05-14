export function isWidgetConfigFlagEnabled(value: unknown): boolean {
  if (value === true || value === 1) {
    return true
  }
  if (typeof value !== 'string') {
    return false
  }
  const normalized = value.trim().toLowerCase()
  return normalized === 'true' || normalized === '1'
}
