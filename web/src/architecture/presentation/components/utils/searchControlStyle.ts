type SearchControlStyle = Record<string, string | number>

export function buildSearchControlStyle(style?: Record<string, string | number>): SearchControlStyle {
  const normalizedStyle: SearchControlStyle = { ...(style || {}) }

  delete normalizedStyle.width
  delete normalizedStyle.minWidth
  delete normalizedStyle.maxWidth

  return {
    ...normalizedStyle,
    width: '100%',
    minWidth: 0,
    maxWidth: '100%'
  }
}

export function buildSearchRangeFieldStyle(style?: Record<string, string | number>): SearchControlStyle {
  return {
    ...buildSearchControlStyle(style),
    flex: '1 1 0'
  }
}
