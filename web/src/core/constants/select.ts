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

/**
 * Element Plus 标准颜色类型
 */
export const StandardColors = [
  'success',
  'warning',
  'danger',
  'info',
  'primary'
] as const

export const PresetOptionColors = [
  'default',
  ...StandardColors
] as const

/**
 * 标准颜色类型
 */
export type StandardColorType = typeof StandardColors[number]
export type PresetOptionColorType = typeof PresetOptionColors[number]

/**
 * 标准颜色对应的 CSS 变量映射
 */
export const StandardColorCSSVars: Record<StandardColorType, string> = {
  success: 'var(--el-color-success)',
  warning: 'var(--el-color-warning)',
  danger: 'var(--el-color-danger)',
  info: 'var(--el-color-info)',
  primary: 'var(--el-color-primary)'
}

export const DefaultOptionColorPalette = {
  solidColor: 'var(--el-text-color-placeholder)',
  backgroundColor: 'var(--el-fill-color-light)',
  borderColor: 'var(--el-border-color)',
  textColor: 'var(--el-text-color-regular)'
} as const

/**
 * 检查颜色是否为标准颜色
 */
export function isStandardColor(color: string): boolean {
  return StandardColors.includes(color as StandardColorType)
}

export function isPresetOptionColor(color: string): boolean {
  return PresetOptionColors.includes(color as PresetOptionColorType)
}

/**
 * 获取标准颜色的 CSS 变量值
 */
export function getStandardColorCSSVar(color: StandardColorType): string {
  return StandardColorCSSVars[color] || ''
}

export function normalizeOptionColor(color?: string | null): string | null {
  if (!color) {
    return null
  }

  const trimmedColor = color.trim()
  if (!trimmedColor) {
    return null
  }

  const loweredColor = trimmedColor.toLowerCase()
  if (isPresetOptionColor(loweredColor)) {
    return loweredColor
  }

  if (isSupportedCssColor(trimmedColor)) {
    return trimmedColor
  }

  return 'default'
}

export function getOptionSolidColor(color?: string | null): string {
  const normalizedColor = normalizeOptionColor(color)
  if (!normalizedColor) {
    return ''
  }

  if (normalizedColor === 'default') {
    return DefaultOptionColorPalette.solidColor
  }

  if (isStandardColor(normalizedColor)) {
    return getStandardColorCSSVar(normalizedColor as StandardColorType)
  }

  return normalizedColor
}

export function getOptionLightPalette(color?: string | null): { backgroundColor: string; borderColor: string; color: string } | null {
  const normalizedColor = normalizeOptionColor(color)
  if (!normalizedColor) {
    return null
  }

  if (normalizedColor === 'default') {
    return {
      backgroundColor: DefaultOptionColorPalette.backgroundColor,
      borderColor: DefaultOptionColorPalette.borderColor,
      color: DefaultOptionColorPalette.textColor
    }
  }

  if (isStandardColor(normalizedColor)) {
    return {
      backgroundColor: `var(--el-color-${normalizedColor}-light-9)`,
      borderColor: `var(--el-color-${normalizedColor}-light-5)`,
      color: `var(--el-color-${normalizedColor})`
    }
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

function isSupportedCssColor(color: string): boolean {
  const normalized = color.trim()
  if (!normalized) {
    return false
  }

  if (
    /^#([0-9a-f]{3}){1,2}$/i.test(normalized) ||
    /^rgba?\(([^)]+)\)$/i.test(normalized) ||
    /^hsla?\(([^)]+)\)$/i.test(normalized) ||
    /^var\(--.+\)$/i.test(normalized)
  ) {
    return true
  }

  if (typeof document !== 'undefined') {
    const sample = document.createElement('span')
    sample.style.color = ''
    sample.style.color = normalized
    return sample.style.color !== ''
  }

  return false
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
