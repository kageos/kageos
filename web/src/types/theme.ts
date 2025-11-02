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
    mode: 'light',
    name: 'modern-light',
    label: '现代浅色'
  },
  {
    mode: 'dark',
    name: 'modern-dark',
    label: '现代深色'
  }
]

/**
 * 默认主题
 */
export const DEFAULT_THEME = THEME_PRESETS[0]

