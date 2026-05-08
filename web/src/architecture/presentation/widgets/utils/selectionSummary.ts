export interface SelectionSummary<T> {
  visibleValues: T[]
  hiddenCount: number
}

export function buildSelectionSummary<T>(
  values: T[] | null | undefined,
  maxVisible = 1
): SelectionSummary<T> {
  const normalizedValues = Array.isArray(values) ? values : []
  const visibleCount = Math.max(0, maxVisible)

  return {
    visibleValues: normalizedValues.slice(0, visibleCount),
    hiddenCount: Math.max(0, normalizedValues.length - visibleCount)
  }
}
