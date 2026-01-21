/**
 * 主题类型定义
 */

export type ThemeMode = 'light' | 'dark'

export interface ThemeConfig {
  name: string
  mode: ThemeMode
  label: string
}

export const THEME_PRESETS: ThemeConfig[] = [
  {
    name: 'light',
    mode: 'light',
    label: '浅色模式'
  },
  {
    name: 'dark',
    mode: 'dark',
    label: '深色模式'
  }
]

export const DEFAULT_THEME: ThemeConfig = THEME_PRESETS[0]

