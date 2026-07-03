/**
 * 主题系统类型定义
 */

/**
 * 主题模式
 */
export type ThemeMode = 'light' | 'dark'

/**
 * 主题配置
 */
export interface ThemeConfig {
  /** 主题模式 */
  mode: ThemeMode
  /** 主题名称 */
  name: string
  /** 主题显示标签 */
  label: string
}

/**
 * 预设主题列表
 */
export const THEME_PRESETS: ThemeConfig[] = [
  {
    mode: 'dark',
    name: 'modern-dark',
    label: '经典暗黑'
  },
  {
    mode: 'dark',
    name: 'hub-dark',
    label: 'Kageos Hub'
  },
  {
    mode: 'light',
    name: 'hub-light',
    label: 'Kageos Light'
  }
]

/**
 * 默认主题
 */
export const DEFAULT_THEME: ThemeConfig = THEME_PRESETS[0] ?? {
  mode: 'light',
  name: 'hub-light',
  label: 'Kageos Light'
}
