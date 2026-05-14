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

  const backgroundRgb = blendWithWhite(rgb, 0.12)
  const borderRgb = blendWithWhite(rgb, 0.28)

  return {
    backgroundColor: rgbToCss(backgroundRgb),
    borderColor: rgbToCss(borderRgb),
    color: resolveReadableTagTextColor(rgb, backgroundRgb)
  }
}

function resolveReadableTagTextColor(
  rgb: { r: number; g: number; b: number },
  backgroundRgb: { r: number; g: number; b: number }
): string {
  for (let factor = 0.72; factor >= 0.18; factor -= 0.06) {
    const candidate = {
      r: Math.round(rgb.r * factor),
      g: Math.round(rgb.g * factor),
      b: Math.round(rgb.b * factor)
    }

    if (contrastRatio(candidate, backgroundRgb) >= 4.5) {
      return rgbToHex(candidate)
    }
  }

  return '#1F2329'
}

function blendWithWhite(rgb: { r: number; g: number; b: number }, alpha: number): { r: number; g: number; b: number } {
  return {
    r: Math.round(rgb.r * alpha + 255 * (1 - alpha)),
    g: Math.round(rgb.g * alpha + 255 * (1 - alpha)),
    b: Math.round(rgb.b * alpha + 255 * (1 - alpha))
  }
}

function contrastRatio(
  foreground: { r: number; g: number; b: number },
  background: { r: number; g: number; b: number }
): number {
  const lighter = Math.max(relativeLuminance(foreground), relativeLuminance(background))
  const darker = Math.min(relativeLuminance(foreground), relativeLuminance(background))
  return (lighter + 0.05) / (darker + 0.05)
}

function relativeLuminance(rgb: { r: number; g: number; b: number }): number {
  const [r, g, b] = [rgb.r, rgb.g, rgb.b].map((channel) => {
    const normalized = channel / 255
    return normalized <= 0.03928
      ? normalized / 12.92
      : ((normalized + 0.055) / 1.055) ** 2.4
  })

  return 0.2126 * (r || 0) + 0.7152 * (g || 0) + 0.0722 * (b || 0)
}

function rgbToHex(rgb: { r: number; g: number; b: number }): string {
  return `#${[rgb.r, rgb.g, rgb.b]
    .map(channel => Math.max(0, Math.min(255, channel)).toString(16).padStart(2, '0'))
    .join('')
    .toUpperCase()}`
}

function rgbToCss(rgb: { r: number; g: number; b: number }): string {
  return `rgb(${rgb.r}, ${rgb.g}, ${rgb.b})`
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
