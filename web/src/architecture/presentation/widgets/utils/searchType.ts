export function resolveWidgetSearchType(
  explicitSearchType?: string | null,
  fieldSearchType?: string | null
): string {
  return explicitSearchType || fieldSearchType || ''
}
