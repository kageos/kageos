import type { FieldConfig } from '@/architecture/domain/types/field'

function normalizeOptionColorList(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value
      .map(item => String(item).trim())
      .filter(Boolean)
  }

  if (typeof value === 'string') {
    return value
      .split(',')
      .map(item => item.trim())
      .filter(Boolean)
  }

  return []
}

export function getWidgetOptionColors(config?: Record<string, any> | null): string[] {
  if (!config) {
    return []
  }

  const candidates = [
    config.options_colors,
    config.option_colors,
    config.options_color
  ]

  for (const candidate of candidates) {
    const colors = normalizeOptionColorList(candidate)
    if (colors.length > 0) {
      return colors
    }
  }

  return []
}

export function getFieldWidgetOptionColors(field?: FieldConfig | null): string[] {
  const widget = field?.widget as any
  const configColors = getWidgetOptionColors(widget?.config)
  if (configColors.length > 0) {
    return configColors
  }

  return getWidgetOptionColors(widget)
}
