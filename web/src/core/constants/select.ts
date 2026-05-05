/**
 * Select 组件相关常量
 */

/**
 * OnSelectFuzzy 回调查询类型
 */
export const SelectFuzzyQueryType = {
  BY_KEYWORD: 'by_keyword',  // 关键字搜索
  BY_VALUE: 'by_value',      // 根据值查找
  BY_VALUES: 'by_values'     // 根据多个值查找
} as const

export type StandardColorType = never

export const DefaultOptionColorPalette = {
  solidColor: 'var(--el-text-color-placeholder)',
  backgroundColor: 'var(--el-fill-color-light)',
  borderColor: 'var(--el-border-color)',
  textColor: 'var(--el-text-color-regular)'
} as const

export function normalizeOptionColor(color?: string | null): string | null {
  if (!color) {
    return null
  }

  const trimmedColor = color.trim()
  if (!trimmedColor) {
    return null
  }

  if (/^[0-9a-f]{6}$/i.test(trimmedColor)) {
    return `#${trimmedColor.toUpperCase()}`
  }

  return null
}

function normalizeOptionCssColor(color?: string | null): string | null {
  if (!color) {
    return null
  }

  const trimmedColor = color.trim()
  if (/^#[0-9a-f]{6}$/i.test(trimmedColor)) {
    return `#${trimmedColor.slice(1).toUpperCase()}`
  }

  return normalizeOptionColor(trimmedColor)
}

export function getOptionSolidColor(color?: string | null): string {
  const normalizedColor = normalizeOptionCssColor(color)
  if (!normalizedColor) {
    return ''
  }

  return normalizedColor
}

export function getOptionLightPalette(color?: string | null): { backgroundColor: string; borderColor: string; color: string } | null {
  const normalizedColor = normalizeOptionCssColor(color)
  if (!normalizedColor) {
    return null
  }

  const rgb = parseColorToRgb(normalizedColor)
  if (!rgb) {
    return {
      backgroundColor: DefaultOptionColorPalette.backgroundColor,
      borderColor: normalizedColor,
      color: normalizedColor
    }
  }

  return {
    backgroundColor: `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.12)`,
    borderColor: `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.28)`,
    color: resolveReadableTagTextColor(normalizedColor, rgb)
  }
}

function resolveReadableTagTextColor(optionColor: string, rgb: { r: number; g: number; b: number }): string {
  const brightness = (rgb.r * 299 + rgb.g * 587 + rgb.b * 114) / 1000
  return brightness >= 170 ? '#1f2329' : optionColor
}

function parseColorToRgb(color: string): { r: number; g: number; b: number } | null {
  const normalized = color.trim().toLowerCase()

  if (/^#([0-9a-f]{3}){1,2}$/.test(normalized)) {
    const hex = normalized.slice(1)
    const expanded = hex.length === 3
      ? hex.split('').map(part => part + part).join('')
      : hex

    return {
      r: Number.parseInt(expanded.slice(0, 2), 16),
      g: Number.parseInt(expanded.slice(2, 4), 16),
      b: Number.parseInt(expanded.slice(4, 6), 16)
    }
  }

  const rgbMatch = normalized.match(/^rgba?\(([^)]+)\)$/)
  if (!rgbMatch) {
    return null
  }

  const channels = (rgbMatch[1] ?? '')
    .split(',')
    .slice(0, 3)
    .map(part => Number.parseFloat(part.trim()))

  if (channels.length !== 3 || channels.some(Number.isNaN)) {
    return null
  }

  return {
    r: channels[0]!,
    g: channels[1]!,
    b: channels[2]!
  }
}
