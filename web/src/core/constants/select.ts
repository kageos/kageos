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

/**
 * 标准颜色类型
 */
export type StandardColorType = typeof StandardColors[number]

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

/**
 * 检查颜色是否为标准颜色
 */
export function isStandardColor(color: string): boolean {
  return StandardColors.includes(color as StandardColorType)
}

/**
 * 获取标准颜色的 CSS 变量值
 */
export function getStandardColorCSSVar(color: StandardColorType): string {
  return StandardColorCSSVars[color] || ''
}

